#!/usr/bin/env python3
"""Build a deterministic, cache-only inventory of declared license metadata.

This reports source observations only. It does not interpret SPDX expressions,
determine legal obligations, or contact package registries.
"""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shlex
import sys
import tarfile
import zipfile
from pathlib import Path
from typing import Any


LICENSE_NAMES = {"license", "licence", "copying", "notice"}


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def license_name(path: str) -> bool:
    base = Path(path).name.lower()
    return any(base == name or base.startswith(name + ".") or base.startswith(name + "-") for name in LICENSE_NAMES)


def go_escape(value: str) -> str:
    return re.sub(r"[A-Z]", lambda m: "!" + m.group(0).lower(), value)


def parse_go_mod(path: Path) -> tuple[str, list[tuple[str, str]]]:
    module = ""
    requires: list[tuple[str, str]] = []
    in_require = False
    try:
        lines = path.read_text(encoding="utf-8").splitlines()
    except (OSError, UnicodeError):
        return module, requires
    for raw in lines:
        # Go permits quoted strings in module directives. Strip comments only
        # outside a quoted string, then let shlex decode the quoted tokens.
        quote = False
        escaped = False
        end = len(raw)
        for index, char in enumerate(raw):
            if escaped:
                escaped = False
            elif char == "\\" and quote:
                escaped = True
            elif char == '"':
                quote = not quote
            elif not quote and raw.startswith("//", index):
                end = index
                break
        line = raw[:end].strip()
        if not line:
            continue
        try:
            bits = shlex.split(line, posix=True)
        except ValueError:
            continue
        if bits and bits[0] == "module" and len(bits) == 2:
            module = bits[1]
        if bits == ["require", "("]:
            in_require = True
            continue
        if in_require and bits == [")"]:
            in_require = False
            continue
        if (in_require and len(bits) >= 2) or (bits and bits[0] == "require" and len(bits) >= 3):
            offset = 0 if in_require else 1
            requires.append((bits[offset], bits[offset + 1]))
    return module, sorted(set(requires))


def discover_go_mods(repo: Path) -> list[Path]:
    """Find source manifests without traversing generated or ignored roots.

    Git archives do not contain .git, so discovery must be filesystem based;
    explicitly pruning local state/build roots keeps an extracted source tree
    and the working checkout's source inventory equivalent.
    """
    excluded = {".git", ".data", ".tools", "artifacts", "vendor", "node_modules", "testdata"}
    found: list[Path] = []
    for current, dirs, files in os.walk(repo):
        dirs[:] = sorted(d for d in dirs if d not in excluded)
        if "go.mod" in files:
            found.append(Path(current) / "go.mod")
    return sorted(found)


def read_license_files(root: Path) -> list[dict[str, str]]:
    found: list[dict[str, str]] = []
    if not root.is_dir():
        return found
    for current, dirs, files in os.walk(root):
        dirs[:] = sorted(d for d in dirs if d not in {".git", ".data", ".tools", "artifacts", "vendor", "node_modules", "testdata"})
        for filename in sorted(files):
            candidate = Path(current) / filename
            relative = candidate.relative_to(root).as_posix()
            if license_name(relative):
                try:
                    digest = sha256(candidate.read_bytes())
                except OSError:
                    continue
                found.append({"path": relative, "sha256": digest})
    return found


def package_license(root: Path) -> str:
    manifest = root / "package.json"
    try:
        data = json.loads(manifest.read_text(encoding="utf-8"))
        value = data.get("license")
        if isinstance(value, str) and value.strip():
            return value.strip()
        if isinstance(value, dict) and isinstance(value.get("type"), str):
            return value["type"].strip() or "UNKNOWN"
        if isinstance(data.get("licenses"), list):
            values = sorted({str(x.get("type", "")).strip() for x in data["licenses"] if isinstance(x, dict) and x.get("type")})
            if len(values) == 1:
                return values[0]
            if values:
                return "AMBIGUOUS: " + ", ".join(values)
    except (OSError, UnicodeError, json.JSONDecodeError):
        pass
    return "UNKNOWN"


def go_cache_module(cache: Path, module: str, version: str) -> tuple[str, Path | None, list[dict[str, str]]]:
    escaped = f"{go_escape(module)}@{go_escape(version)}"
    extracted = cache / escaped
    if extracted.is_dir():
        return "extracted-cache", extracted, read_license_files(extracted)
    zip_path = cache / "cache" / "download" / go_escape(module) / "@v" / f"{go_escape(version)}.zip"
    if not zip_path.is_file():
        # GOMODCACHE itself is commonly GOPATH/pkg/mod; download cache may be there.
        zip_path = cache / "download" / go_escape(module) / "@v" / f"{go_escape(version)}.zip"
    files: list[dict[str, str]] = []
    if zip_path.is_file():
        try:
            with zipfile.ZipFile(zip_path) as archive:
                for item in sorted(archive.infolist(), key=lambda x: x.filename):
                    parts = item.filename.split("/", 1)
                    if len(parts) == 2 and license_name(parts[1]):
                        files.append({"path": f"<module-archive>!/{parts[1]}", "sha256": sha256(archive.read(item))})
            return "download-cache-zip", None, files
        except (OSError, zipfile.BadZipFile):
            pass
    return "not-cached", None, []


def npm_cache_blob(cache: Path, integrity: str) -> Path | None:
    match = re.fullmatch(r"sha512-([A-Za-z0-9+/=]+)", integrity or "")
    if not match:
        return None
    import base64

    try:
        digest = base64.b64decode(match.group(1), validate=True).hex()
    except (ValueError, base64.binascii.Error):
        return None
    candidate = cache / "_cacache" / "content-v2" / "sha512" / digest[:2] / digest[2:4] / digest[4:]
    if candidate.is_file():
        return candidate
    # Direct cache root can point at _cacache itself.
    candidate = cache / "content-v2" / "sha512" / digest[:2] / digest[2:4] / digest[4:]
    return candidate if candidate.is_file() else None


def npm_cache_metadata(cache: Path, integrity: str) -> tuple[str, list[dict[str, str]]]:
    blob = npm_cache_blob(cache, integrity)
    if blob is None:
        return "UNKNOWN", []
    try:
        with tarfile.open(blob, mode="r:gz") as archive:
            package_member = next((m for m in archive.getmembers() if m.name.rstrip("/") == "package/package.json"), None)
            declared = "UNKNOWN"
            if package_member:
                stream = archive.extractfile(package_member)
                if stream:
                    data = json.load(stream)
                    value = data.get("license")
                    if isinstance(value, str) and value.strip():
                        declared = value.strip()
                    elif isinstance(value, dict) and value.get("type"):
                        declared = str(value["type"]).strip() or "UNKNOWN"
                    elif isinstance(data.get("licenses"), list):
                        types = sorted({str(x.get("type", "")).strip() for x in data["licenses"] if isinstance(x, dict) and x.get("type")})
                        declared = types[0] if len(types) == 1 else ("AMBIGUOUS: " + ", ".join(types) if types else "UNKNOWN")
            texts = []
            for member in sorted(archive.getmembers(), key=lambda x: x.name):
                name = member.name.split("/", 1)[-1]
                if member.isfile() and license_name(name):
                    stream = archive.extractfile(member)
                    if stream:
                        texts.append({"path": f"<npm-tarball>!/{member.name}", "sha256": sha256(stream.read())})
            return declared, texts
    except (OSError, tarfile.TarError, json.JSONDecodeError, UnicodeError):
        return "UNKNOWN", []


def make_inventory(repo: Path, go_cache: Path, npm_cache: Path, locks: list[Path]) -> dict[str, Any]:
    records: list[dict[str, Any]] = []
    seen: set[tuple[str, str, str]] = set()
    go_mods = discover_go_mods(repo)
    cache_queue: list[tuple[str, str]] = []
    for modfile in go_mods:
        module, requires = parse_go_mod(modfile)
        rel = modfile.relative_to(repo).as_posix()
        records.append({"ecosystem": "go", "name": module or "UNKNOWN", "version": "local", "manifest": rel,
                        "cache": "workspace", "declared_license": "UNKNOWN", "license_files": read_license_files(modfile.parent)})
        cache_queue.extend(requires)
    visited: set[tuple[str, str]] = set()
    while cache_queue:
        module, version = cache_queue.pop(0)
        key = (module, version)
        if key in visited:
            continue
        visited.add(key)
        status, root, files = go_cache_module(go_cache, module, version)
        declared = package_license(root) if root else "UNKNOWN"
        records.append({"ecosystem": "go", "name": module, "version": version, "manifest": "go.mod dependency graph",
                        "cache": status, "declared_license": declared, "license_files": files})
        if root and (root / "go.mod").is_file():
            _, deps = parse_go_mod(root / "go.mod")
            cache_queue.extend(deps)

    for lock in sorted(set(locks)):
        try:
            data = json.loads(lock.read_text(encoding="utf-8"))
        except (OSError, UnicodeError, json.JSONDecodeError):
            continue
        for install_path, item in sorted(data.get("packages", {}).items()):
            if not install_path.startswith("node_modules/") or not isinstance(item, dict):
                continue
            name = install_path[len("node_modules/"):]
            version = str(item.get("version", "UNKNOWN"))
            declared = item.get("license")
            declared = str(declared).strip() if isinstance(declared, str) and declared.strip() else "UNKNOWN"
            files: list[dict[str, str]] = []
            cache_status = "lock-metadata"
            installed = lock.parent / install_path
            if installed.is_dir() and not installed.is_symlink():
                if declared == "UNKNOWN":
                    declared = package_license(installed)
                files = read_license_files(installed)
                cache_status = "installed-cache"
            else:
                cached_declared, cached_files = npm_cache_metadata(npm_cache, str(item.get("integrity", "")))
                if declared == "UNKNOWN":
                    declared = cached_declared
                files = cached_files
                if cached_files or cached_declared != "UNKNOWN":
                    cache_status = "npm-content-cache"
            records.append({"ecosystem": "npm", "name": name, "version": version, "manifest": lock.relative_to(repo).as_posix(),
                            "cache": cache_status, "declared_license": declared or "UNKNOWN", "license_files": files})
    records.sort(key=lambda r: (r["ecosystem"], r["name"], r["version"], r["manifest"]))
    return {"format": "open-go-academy-license-inventory-v1", "mode": "cache-only", "records": records}


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo-root", type=Path, default=Path(__file__).resolve().parents[1])
    parser.add_argument("--go-mod-cache", type=Path, default=Path(os.environ.get("GOMODCACHE", str(Path.home() / "go/pkg/mod"))))
    parser.add_argument("--npm-cache", type=Path, default=Path(os.environ.get("npm_config_cache", str(Path.home() / ".npm"))))
    parser.add_argument("--npm-lock", type=Path, action="append", help="lockfile path (repeatable; defaults to root and web package-lock.json)")
    parser.add_argument("--output", type=Path, help="write JSON report here; otherwise stdout")
    args = parser.parse_args(argv)
    root = args.repo_root.resolve()
    locks = [p.resolve() for p in args.npm_lock] if args.npm_lock else [p for p in (root / "package-lock.json", root / "web/package-lock.json") if p.is_file()]
    inventory = make_inventory(root, args.go_mod_cache.resolve(), args.npm_cache.resolve(), locks)
    rendered = json.dumps(inventory, ensure_ascii=False, sort_keys=True, indent=2) + "\n"
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(rendered, encoding="utf-8")
    else:
        sys.stdout.write(rendered)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
