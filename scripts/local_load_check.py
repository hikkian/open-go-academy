#!/usr/bin/env python3
"""Bounded local API load/persistence check using only the Python stdlib.

This starts a disposable loopback API instance and never executes user code.
"""
from __future__ import annotations

import argparse
import concurrent.futures
import http.cookiejar
import json
import os
import pathlib
import platform
import socket
import sqlite3
import subprocess
import tempfile
import time
import urllib.error
import urllib.request


ROOT = pathlib.Path(__file__).resolve().parent.parent
READ_PATHS = ("/api/lessons", "/api/learning", "/api/progress", "/api/drafts/hello")


def percentile(values: list[float], p: float) -> float:
    ordered = sorted(values)
    return round(ordered[min(len(ordered) - 1, int((len(ordered) - 1) * p))], 3)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--report", type=pathlib.Path, help="write a non-secret JSON summary")
    args = parser.parse_args()
    report: dict[str, object] = {
        "scope": "local disposable API only; no code execution, external services, or SLO assertion",
        "platform": platform.platform(),
        "python": platform.python_version(),
        "read_burst": {"requested": 120, "workers": 16, "status_counts": {}, "errors": 0},
        "draft_put_burst": {"requested": 140, "workers": 16, "status_counts": {}, "errors": 0},
    }
    with tempfile.TemporaryDirectory(prefix="oga-m12-load-") as td:
        work = pathlib.Path(td)
        work.chmod(0o700)
        binary = work / "academy-api"
        subprocess.run(["sh", "scripts/go.sh", "build", "-o", str(binary), "./cmd/api"], cwd=ROOT, check=True)
        go_version = subprocess.run(["sh", "scripts/go.sh", "version"], cwd=ROOT, check=True,
                                    capture_output=True, text=True).stdout.strip()
        report["go_version"] = go_version
        os.chmod(binary, 0o700)
        with socket.socket() as sock:
            sock.bind(("127.0.0.1", 0))
            port = sock.getsockname()[1]
        addr = f"127.0.0.1:{port}"
        base = f"http://{addr}"
        db = work / "disposable.db"
        env = {k: v for k, v in os.environ.items() if not k.startswith("ACADEMY_")}
        env.update({
            "ACADEMY_ADDR": addr,
            "ACADEMY_DB": str(db),
            "ACADEMY_ISSUER_KEY": str(work / "disposable-issuer.key"),
            "ACADEMY_RUNNER_SOCKET": str(work / "unreachable-runner.sock"),
            "ACADEMY_PROJECT_RUNNER_SOCKET": str(work / "unreachable-project-runner.sock"),
            "ACADEMY_PROJECT_REVIEWERS_FILE": "",
            "ACADEMY_TUTOR_PROVIDER_CONFIG": str(work / "disposable-tutor.json"),
            "ACADEMY_WEB": str(work / "no-static-assets"),
            "GOTOOLCHAIN": "go1.27.2",
        })
        cookies = http.cookiejar.CookieJar()
        client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(cookies))
        processes: list[subprocess.Popen] = []

        def start() -> subprocess.Popen:
            p = subprocess.Popen([str(binary)], cwd=ROOT, env=env, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            processes.append(p)
            for _ in range(160):
                if p.poll() is not None:
                    raise RuntimeError("disposable API exited during startup")
                try:
                    status, _, body, _ = request("/api/health", timeout=1)
                    if status == 200 and json.loads(body).get("status") == "ok":
                        return p
                except (OSError, urllib.error.URLError, TimeoutError):
                    time.sleep(0.05)
            raise RuntimeError("disposable API startup timed out")

        def stop(p: subprocess.Popen) -> None:
            if p.poll() is None:
                p.terminate()
            p.wait(timeout=8)

        def request(path: str, method: str = "GET", payload: object | None = None,
                    token: str | None = None, timeout: float = 8,
                    headers: dict[str, str] | None = None, host: str | None = None) -> tuple[int, dict[str, str], bytes, float]:
            body = None if payload is None else json.dumps(payload, separators=(",", ":")).encode()
            request_headers = {"Content-Type": "application/json"}
            if token is not None:
                request_headers["X-CSRF-Token"] = token
            if headers:
                request_headers.update(headers)
            req = urllib.request.Request(base + path, data=body, method=method, headers=request_headers)
            if host is not None:
                req.add_header("Host", host)
            t0 = time.perf_counter()
            try:
                with client.open(req, timeout=timeout) as response:
                    raw = response.read()
                    return response.status, dict(response.headers.items()), raw, (time.perf_counter() - t0) * 1000
            except urllib.error.HTTPError as exc:
                raw = exc.read()
                return exc.code, dict(exc.headers.items()), raw, (time.perf_counter() - t0) * 1000

        def counted_get(path: str) -> tuple[int, float, str | None]:
            try:
                status, _, _, elapsed = request(path)
                return status, elapsed, None
            except Exception as exc:  # summarize; never emit request details or secrets
                return 0, 0.0, type(exc).__name__

        def counted_put(index: int) -> tuple[int, float, str | None, bool]:
            marker = f"m12-load-{index:03d}"
            bundle = {"files": {
                "main.go": f"package main\n// {marker}\nfunc main() {{}}\n",
                "helper.go": f"package main\n// {marker}\nconst loadMarker = \"{marker}\"\n",
            }}
            try:
                status, headers, _, elapsed = request("/api/drafts/hello", "PUT", bundle, token)
                return status, elapsed, None, status == 429 and "Retry-After" in headers
            except Exception as exc:
                return 0, 0.0, type(exc).__name__, False

        try:
            p = start()
            # Unauthenticated protected read must fail before a profile exists.
            unauth, _, _, _ = request("/api/lessons", host=addr)
            report["unauthenticated_status"] = unauth
            if unauth != 401:
                raise AssertionError(f"unauthenticated protected read returned {unauth}")

            status, headers, raw, _ = request("/api/profile")
            if status != 200:
                raise AssertionError("profile bootstrap failed")
            session = next((c.value for c in cookies if c.name == "academy_session"), None)
            token = json.loads(raw)["csrf_token"]
            if not session or not any(c.name == "academy_session" and c.has_nonstandard_attr("HttpOnly") for c in cookies):
                # CookieJar does not consistently retain HttpOnly metadata; header is authoritative.
                set_cookie = headers.get("Set-Cookie", "")
                if "HttpOnly" not in set_cookie:
                    raise AssertionError("profile bootstrap omitted HttpOnly session cookie")
            if session != token:
                raise AssertionError("session cookie and CSRF token mismatch")

            status, _, _, _ = request("/api/profile", "POST", {"name": "M12 disposable", "theme": "light"}, token)
            if status != 200:
                raise AssertionError(f"disposable profile creation returned {status}")
            profile_status, _, profile_raw, _ = request("/api/profile")
            learning_status, _, learning_raw, _ = request("/api/learning")
            passport_status, _, passport_raw, _ = request("/api/passport")
            baseline_profile = json.loads(profile_raw).get("profile") if profile_status == 200 else None
            baseline_learning = json.loads(learning_raw) if learning_status == 200 else {}
            baseline_passport = json.loads(passport_raw) if passport_status == 200 else {}
            if not isinstance(baseline_profile, dict) or baseline_learning.get("current_rank") != "Zero" or baseline_learning.get("verified") is not False or baseline_passport.get("awards") != []:
                raise AssertionError("disposable baseline profile/learning/awards state is unexpected")

            no_csrf, _, _, _ = request("/api/drafts/hello", "PUT", {"files": {"main.go": "package main"}}, token="")
            foreign_origin, _, _, _ = request("/api/lessons", headers={"Origin": "https://invalid.example"})
            invalid_host, _, _, _ = request("/api/lessons", host="invalid.example")
            report["security_statuses"] = {"missing_csrf": no_csrf, "foreign_origin": foreign_origin, "invalid_host": invalid_host}
            if (no_csrf, foreign_origin, invalid_host) != (403, 403, 403):
                raise AssertionError("local boundary probe returned unexpected status")

            read_paths = [READ_PATHS[i % len(READ_PATHS)] for i in range(120)]
            with concurrent.futures.ThreadPoolExecutor(max_workers=16) as pool:
                read_results = list(pool.map(counted_get, read_paths))
            read_counts: dict[str, int] = {}
            for status, _, _ in read_results:
                key = str(status)
                read_counts[key] = read_counts.get(key, 0) + 1
            latencies = [elapsed for status, elapsed, _ in read_results if status == 200]
            read_errors = sum(error is not None or status != 200 for status, _, error in read_results)
            report["read_burst"] = {
                "requested": len(read_results), "workers": 16, "status_counts": read_counts,
                "errors": read_errors,
                "latency_ms": {"p50": percentile(latencies, .50), "p95": percentile(latencies, .95), "max": round(max(latencies), 3)} if latencies else {},
            }
            if read_errors:
                raise AssertionError(f"read burst had {read_errors} non-200/error responses")

            with concurrent.futures.ThreadPoolExecutor(max_workers=16) as pool:
                put_results = list(pool.map(counted_put, range(140)))
            put_counts: dict[str, int] = {}
            for status, _, _, _ in put_results:
                key = str(status)
                put_counts[key] = put_counts.get(key, 0) + 1
            put_errors = sum(error is not None or status not in (200, 429) for status, _, error, _ in put_results)
            retry_after = sum(has_retry for _, _, _, has_retry in put_results)
            report["draft_put_burst"] = {
                "requested": len(put_results), "workers": 16, "status_counts": put_counts,
                "errors": put_errors, "rate_limited_with_retry_after": retry_after,
                "latency_ms": {"p50": percentile([x[1] for x in put_results if x[0] in (200, 429)], .50),
                               "p95": percentile([x[1] for x in put_results if x[0] in (200, 429)], .95),
                               "max": round(max(x[1] for x in put_results), 3)},
            }
            if put_errors or not put_counts.get("200") or not put_counts.get("429") or retry_after != put_counts["429"]:
                raise AssertionError("draft burst did not show expected successful and rate-limited writes")

            status, _, raw, _ = request("/api/drafts/hello")
            saved = json.loads(raw).get("files", {}) if status == 200 else {}
            markers = {line.removeprefix("// ") for content in saved.values() for line in content.splitlines() if line.startswith("// m12-load-")}
            if len(saved) != 2 or len(markers) != 1:
                raise AssertionError("final draft is not exactly one complete submitted bundle")
            chosen = next(iter(markers))
            try:
                successful_index = int(chosen.removeprefix("m12-load-"))
            except ValueError as exc:
                raise AssertionError("final bundle marker has an invalid request index") from exc
            if successful_index not in [i for i, result in enumerate(put_results) if result[0] == 200]:
                raise AssertionError("final draft marker came from a rate-limited or failed request")
            expected = {"main.go": f"package main\n// {chosen}\nfunc main() {{}}\n",
                        "helper.go": f"package main\n// {chosen}\nconst loadMarker = \"{chosen}\"\n"}
            if saved != expected:
                raise AssertionError("final draft files do not match one whole submitted bundle")
            report["draft_integrity"] = "one complete two-file submitted bundle; no mixed markers"
            status, _, raw, _ = request("/api/progress")
            initial_progress = json.loads(raw) if status == 200 else {}
            initial_profile, _, prof_raw, _ = request("/api/profile")
            initial_profile_data = json.loads(prof_raw).get("profile")
            learning_status, _, learning_raw, _ = request("/api/learning")
            passport_status, _, passport_raw, _ = request("/api/passport")
            learning_after = json.loads(learning_raw) if learning_status == 200 else {}
            passport_after = json.loads(passport_raw) if passport_status == 200 else {}
            if initial_profile != 200 or initial_profile_data != baseline_profile or initial_progress.get("xp") != 0:
                raise AssertionError("draft saves unexpectedly granted XP or profile is inconsistent")
            if learning_status != 200 or learning_after.get("current_rank") != "Zero" or learning_after.get("verified") is not False or passport_status != 200 or passport_after.get("awards") != []:
                raise AssertionError("draft saves unexpectedly changed verified rank or rank awards")
            stop(p)
            processes.remove(p)
            p = start()
            status, _, raw, _ = request("/api/drafts/hello")
            restarted_files = json.loads(raw).get("files", {}) if status == 200 else {}
            prof_status, _, prof_raw, _ = request("/api/profile")
            profile_after = json.loads(prof_raw).get("profile") if prof_status == 200 else None
            prog_status, _, prog_raw, _ = request("/api/progress")
            progress_after = json.loads(prog_raw) if prog_status == 200 else {}
            learning_after_restart = json.loads(request("/api/learning")[2])
            passport_after_restart = json.loads(request("/api/passport")[2])
            if restarted_files != expected or profile_after != baseline_profile:
                raise AssertionError("draft/profile changed across API restart")
            if progress_after.get("xp") != 0 or profile_after.get("xp") != 0:
                raise AssertionError("draft persistence unexpectedly granted XP")
            if learning_after_restart.get("current_rank") != "Zero" or learning_after_restart.get("verified") is not False or passport_after_restart.get("awards") != []:
                raise AssertionError("rank/award state changed across API restart")
            report["restart"] = {"draft_persisted": True, "profile_unchanged": True, "xp_after_saves": 0,
                                  "rank_awards": 0, "current_rank": "Zero", "verified": False}
            stop(p)
            processes.remove(p)

            con = sqlite3.connect(f"file:{db}?mode=ro", uri=True)
            try:
                integrity = con.execute("PRAGMA integrity_check").fetchall()
                foreign_keys = con.execute("PRAGMA foreign_key_check").fetchall()
                award_count = con.execute("SELECT COUNT(*) FROM rank_awards").fetchone()[0]
            finally:
                con.close()
            report["sqlite"] = {"integrity_check": integrity[0][0] if integrity else "no result", "foreign_key_violations": len(foreign_keys), "rank_awards": award_count}
            if integrity != [("ok",)] or foreign_keys or award_count != 0:
                raise AssertionError("disposable SQLite integrity checks failed")
            report["result"] = "PASS"
        finally:
            for proc in list(processes):
                try:
                    stop(proc)
                except Exception:
                    proc.kill()
                    proc.wait(timeout=3)
            # TemporaryDirectory removes DB, issuer key, sockets, binary, and config.
    if args.report:
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report, ensure_ascii=False, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
