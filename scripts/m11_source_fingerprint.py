#!/usr/bin/env python3
"""Identify the source inputs of the M11 acceptance gate, independent of mtimes."""
import hashlib
from pathlib import Path
import re


def acceptance_matches(evidence: str, fingerprint: str) -> bool:
    recorded = re.search(r"^Source fingerprint: (sha256:[0-9a-f]{64})$", evidence, re.MULTILINE)
    finished = re.search(r"^Finished source fingerprint: (sha256:[0-9a-f]{64})$", evidence, re.MULTILINE)
    return bool(re.search(r"^Result: PASS$", evidence, re.MULTILINE)
                and re.search(r"^Scope: all 88 advanced lessons$", evidence, re.MULTILINE)
                and recorded and finished and recorded.group(1) == fingerprint == finished.group(1))


def source_fingerprint(root: Path) -> str:
    paths = []
    for directory in ("internal/content", "internal/grading", "internal/runner", "internal/competency"):
        paths.extend(path for path in (root / directory).rglob("*")
                     if path.is_file() and path.suffix in (".go", ".json", ".md", ".txt", ".sql"))
    paths.extend(root / name for name in (
        "go.mod", "go.sum", "scripts/go.sh", "scripts/m11_source_fingerprint.py",
        "scripts/check_m11_acceptance.py"))
    digest = hashlib.sha256(b"open-go-academy:m11-source:v1\0")
    for path in sorted(paths, key=lambda item: item.relative_to(root).as_posix()):
        if path.is_symlink():
            raise ValueError(f"acceptance source must not be a symlink: {path}")
        relative = path.relative_to(root).as_posix().encode("utf-8")
        content = path.read_bytes()
        digest.update(relative + b"\0" + len(content).to_bytes(8, "big") + content)
    return "sha256:" + digest.hexdigest()


if __name__ == "__main__":
    print(source_fingerprint(Path(__file__).resolve().parents[1]))
