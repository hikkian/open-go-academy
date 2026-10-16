#!/usr/bin/env python3
"""Exercise SQLite backup/restore and API restart using disposable local data."""

import http.cookiejar
import json
import os
import pathlib
import socket
import sqlite3
import subprocess
import tempfile
import time
import urllib.error
import urllib.request


ROOT = pathlib.Path(__file__).resolve().parent.parent


def free_loopback_port():
    with socket.socket() as listener:
        listener.bind(("127.0.0.1", 0))
        return listener.getsockname()[1]


def main():
    with tempfile.TemporaryDirectory(prefix="open-go-academy-recovery-") as directory:
        data_dir = pathlib.Path(directory)
        database = data_dir / "academy.db"
        backup = data_dir / "academy-backup.sqlite"
        port = free_loopback_port()
        address = f"127.0.0.1:{port}"
        base_url = f"http://{address}/api"
        environment = {
            key: value for key, value in os.environ.items() if not key.startswith("ACADEMY_")
        }
        environment.update(
            ACADEMY_ADDR=address,
            ACADEMY_DB=str(database),
            ACADEMY_RUNNER_SOCKET=str(data_dir / "missing-runner.sock"),
            ACADEMY_PROJECT_RUNNER_SOCKET=str(data_dir / "missing-project-runner.sock"),
        )
        client = urllib.request.build_opener(
            urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar())
        )

        def request(path, body=None, method=None, token=""):
            data = None if body is None else json.dumps(body).encode()
            req = urllib.request.Request(
                base_url + path,
                data=data,
                method=method,
                headers={"Content-Type": "application/json", "X-CSRF-Token": token},
            )
            with client.open(req, timeout=4) as response:
                return json.load(response)

        def start_server():
            process = subprocess.Popen(
                ["sh", "scripts/go.sh", "run", "./cmd/api"],
                cwd=ROOT,
                env=environment,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
            for _ in range(120):
                if process.poll() is not None:
                    raise RuntimeError("temporary Academy API exited during startup")
                try:
                    request("/health")
                    return process
                except (OSError, urllib.error.URLError):
                    time.sleep(0.1)
            process.terminate()
            process.wait(timeout=5)
            raise RuntimeError("temporary Academy API startup timed out")

        def stop_server(process):
            if process.poll() is None:
                process.terminate()
                try:
                    process.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=5)

        process = start_server()
        try:
            token = request("/profile")["csrf_token"]
            request("/profile", {"name": "Recovery learner", "theme": "dark"}, "POST", token)
            request(
                "/drafts/hello",
                {"files": {"main.go": "package main\nfunc main(){}"}},
                "PUT",
                token,
            )
        finally:
            stop_server(process)

        with sqlite3.connect(database) as source, sqlite3.connect(backup) as target:
            source.backup(target)
        with sqlite3.connect(backup) as snapshot:
            integrity = snapshot.execute("PRAGMA integrity_check").fetchone()[0]
            if integrity != "ok":
                raise RuntimeError(f"backup integrity check failed: {integrity}")

        displaced_database = data_dir / "academy-before-restore.db"
        database.replace(displaced_database)
        for suffix in ("-wal", "-shm"):
            pathlib.Path(str(database) + suffix).unlink(missing_ok=True)
        with sqlite3.connect(backup) as source, sqlite3.connect(database) as target:
            source.backup(target)

        process = start_server()
        try:
            profile = request("/profile")["profile"]
            draft = request("/drafts/hello")["files"]
            if profile["name"] != "Recovery learner" or profile["theme"] != "dark":
                raise RuntimeError("restored profile/theme did not match the saved record")
            if draft["main.go"] != "package main\nfunc main(){}":
                raise RuntimeError("restored lesson draft did not match the saved record")
        finally:
            stop_server(process)

    print("PASS: disposable SQLite backup restored; API restart and migrations preserved profile/theme/draft")


if __name__ == "__main__":
    main()
