#!/usr/bin/env python3
"""Run the real M11 gate and record source-bound, scope-explicit evidence."""
import argparse
from datetime import datetime, timezone
import hashlib
import os
from pathlib import Path
import shlex
import signal
import subprocess
import sys
import time

from m11_source_fingerprint import source_fingerprint


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--lessons", help="comma-separated draft IDs; omitted means all 88")
    parser.add_argument("--report", type=Path, required=True)
    parser.add_argument("--log", type=Path, required=True)
    parser.add_argument("--cache-dir", type=Path, required=True)
    args = parser.parse_args()
    root = Path(__file__).resolve().parents[1]
    report = args.report.resolve()
    log = args.log.resolve()
    if report == log:
        parser.error("report and raw log must be different paths")
    report.parent.mkdir(parents=True, exist_ok=True)
    log.parent.mkdir(parents=True, exist_ok=True)
    args.cache_dir.mkdir(parents=True, exist_ok=True)
    before = source_fingerprint(root)
    started = int(time.time())
    env = os.environ.copy()
    settings = {"ACADEMY_ACCEPT_M11_DRAFTS": "1", "GOTOOLCHAIN": "local",
                "GOPROXY": "off", "GOSUMDB": "off", "GOFLAGS": "-p=1",
                "GOMAXPROCS": "2", "GOCACHE": str(args.cache_dir.resolve())}
    env.pop("ACADEMY_M11_LESSON_IDS", None)
    if args.lessons:
        settings["ACADEMY_M11_LESSON_IDS"] = args.lessons
    env.update(settings)
    command = ["sh", "scripts/go.sh", "test", "-count=1", "-timeout=3h",
               "./internal/content", "-run", "^TestM11DraftCatalogAcceptance$", "-v"]
    scope = "all 88 drafts" if not args.lessons else args.lessons
    print(f"M11 acceptance started: {scope}\nSource fingerprint: {before}\nRaw log: {log}", flush=True)
    with log.open("wb") as output:
        process = subprocess.Popen(command, cwd=root, env=env, stdout=output,
                                   stderr=subprocess.STDOUT, start_new_session=True)
        try:
            code = process.wait()
        except KeyboardInterrupt:
            os.killpg(process.pid, signal.SIGTERM)
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                process.wait()
            code = 130
    after = source_fingerprint(root)
    elapsed = time.time() - started
    status = "PASS" if code == 0 and before == after else ("STALE" if code == 0 else ("INTERRUPTED" if code == 130 else "FAIL"))
    log_hash = hashlib.sha256(log.read_bytes()).hexdigest()
    invocation = " ".join(f"{key}={shlex.quote(value)}" for key, value in settings.items()) + " " + shlex.join(command)
    report.write_text(
        f"M11 acceptance\nResult: {status}\nScope: {scope}\nStarted epoch: {started}\n"
        f"Started UTC: {datetime.fromtimestamp(started, timezone.utc).isoformat()}\n"
        f"Elapsed seconds: {elapsed:.3f}\nExit code: {code}\nSource fingerprint: {before}\n"
        f"Finished source fingerprint: {after}\nRaw log: {log}\nRaw log SHA-256: {log_hash}\n"
        f"Command: {invocation}\n\n"
        "This gate executes repository-owned examples/starters and reference solutions through the pinned isolated runner.\n"
        "A passing automatic gate does not establish pedagogical completeness, reviewer identity or public sandbox safety.\n",
        encoding="utf-8")
    print(f"Result: {status}; elapsed {elapsed:.3f}s; report {report}", flush=True)
    return 0 if status == "PASS" else (code if code > 0 else 1)


if __name__ == "__main__":
    sys.exit(main())
