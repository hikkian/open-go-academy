#!/usr/bin/env python3
"""Bounded rootless Podman host-boundary probe for the reviewed runner policy.

Only static local fixtures are executed. Host sentinels/listeners live in a
TemporaryDirectory; no host paths are mounted into the container.
"""
from __future__ import annotations

import argparse
import io
import os
import re
import shlex
import shutil
import socket
import stat
import subprocess
import sys
import tarfile
import tempfile
import threading
import time
import uuid
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RUNNER_SOURCE = ROOT / "internal/runner/runner.go"
IMAGE_RE = re.compile(r'^const Image = "([^\"]+)"', re.M)
POLICY_RE = re.compile(r'func policy\(name string\) \[\]string \{\s*return \[\]string\{(.*?)\}\s*\}', re.S)
TOKEN_RE = re.compile(r'"((?:\\.|[^"\\])*)"|([A-Za-z_][A-Za-z0-9_]*)|(,)')
EXPECTED = (
    "run", "--rm", "--pull=never", "--name", "__dynamic_name__", "--label", "academy.runner=true",
    "--network=none", "--http-proxy=false", "--stop-timeout=0", "--read-only", "--cap-drop=ALL",
    "--security-opt=no-new-privileges", "--user=65532:65532", "--memory=768m", "--memory-swap=768m",
    "--pids-limit=128", "--cpus=1", "--tmpfs=/tmp:rw,noexec,nosuid,nodev,size=128m",
    "--tmpfs=/work:rw,nosuid,nodev,size=192m", "--workdir=/work", "--env=CGO_ENABLED=0",
    "--env=GOTOOLCHAIN=local", "--env=GOPROXY=off", "--env=GOSUMDB=off",
    "--tmpfs=/cache:rw,nosuid,nodev,size=192m", "--shm-size=16m", "--env=GOCACHE=/cache",
    "--env=HOME=/tmp", "--env=GOMAXPROCS=2",
)

FIXTURE = r'''package main
import (
 "fmt"
 "net"
 "os"
 "strings"
 "time"
)
func readDenied(path string) bool { _, err := os.ReadFile(path); return err != nil }
func writeDenied(path string) bool {
 f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0)
 if err != nil { return true }
 _, err = f.Write([]byte("PROBE_WRITE"))
 _ = f.Close()
 return err != nil
}
func unavailable(path string) bool { _, err := os.Stat(path); return err != nil }
func main() {
 if len(os.Args) != 7 { os.Exit(2) }
 sentinel, issuer, tcpAddr, udpAddr, unixAddr, hostEnvMarker := os.Args[1], os.Args[2], os.Args[3], os.Args[4], os.Args[5], os.Args[6]
 tcp, tcpErr := net.DialTimeout("tcp", tcpAddr, 250*time.Millisecond)
 if tcp != nil { _ = tcp.Close() }
 udpSent := false
 udp, udpErr := net.DialTimeout("udp", udpAddr, 250*time.Millisecond)
 if udpErr == nil { _ = udp.SetDeadline(time.Now().Add(250*time.Millisecond)); _, udpErr = udp.Write([]byte("bounded-probe")); udpSent = udpErr == nil; _ = udp.Close() }
 ux, unixErr := net.DialTimeout("unix", unixAddr, 250*time.Millisecond)
 if ux != nil { _ = ux.Close() }
 procSentinel := "/proc/1/root" + sentinel
 mount, _ := os.ReadFile("/proc/self/mountinfo")
 environ, _ := os.ReadFile("/proc/1/environ")
 fmt.Printf("direct_sentinel_denied=%t\n", readDenied(sentinel))
 fmt.Printf("sentinel_write_denied=%t\n", writeDenied(sentinel))
 fmt.Printf("proc_root_sentinel_denied=%t\n", readDenied(procSentinel))
 fmt.Printf("proc_root_write_denied=%t\n", writeDenied("/proc/1/root"+sentinel))
 fmt.Printf("proc1_root_visible=%t\n", func() bool { _, e := os.Stat("/proc/1/root"); return e == nil }())
 selfNS, _ := os.Readlink("/proc/self/ns/pid")
 proc1NS, _ := os.Readlink("/proc/1/ns/pid")
 fmt.Printf("proc1_pid_namespace_matches_self=%t\n", selfNS != "" && selfNS == proc1NS)
 fmt.Printf("host_only_env_in_container=%t\n", os.Getenv("OGA_M12_HOST_ONLY_SENTINEL") == hostEnvMarker)
 fmt.Printf("host_only_env_in_proc1_environ=%t\n", strings.Contains(string(environ), hostEnvMarker))
 fmt.Printf("host_sentinel_in_mountinfo=%t\n", strings.Contains(string(mount), sentinel))
 fmt.Printf("issuer_fixture_denied=%t\n", readDenied(issuer))
 fmt.Printf("issuer_write_denied=%t\n", writeDenied(issuer))
 fmt.Printf("issuer_fixture_in_mountinfo=%t\n", strings.Contains(string(mount), issuer))
 fmt.Printf("docker_socket_unavailable=%t\n", unavailable("/var/run/docker.sock"))
 fmt.Printf("podman_socket_unavailable=%t\n", unavailable("/run/podman/podman.sock"))
 fmt.Printf("mountinfo_lines=%d\n", len(strings.Split(strings.TrimSpace(string(mount)), "\n")))
 fmt.Printf("tcp_connect_denied=%t\n", tcpErr != nil)
 fmt.Printf("udp_write_unavailable=%t\n", udpErr != nil)
 fmt.Printf("udp_write_attempted=%t\n", udpSent)
 fmt.Printf("unix_connect_denied=%t\n", unixErr != nil)
}
'''


def policy_flags(source: str) -> tuple[str, list[str]]:
    im, body = IMAGE_RE.search(source), POLICY_RE.search(source)
    if not im or not body:
        raise ValueError("could not parse pinned Image and literal policy()")
    image = im.group(1)
    if not re.fullmatch(r"[^@\s]+@sha256:[0-9a-f]{64}", image):
        raise ValueError("runner image is not immutable sha256-pinned")
    raw = body.group(1)
    tokens: list[str] = []
    pos = 0
    for m in TOKEN_RE.finditer(raw):
        if raw[pos:m.start()].strip():
            raise ValueError("unsupported runner policy syntax; refusing probe")
        pos = m.end()
        if m.group(1) is not None:
            tokens.append(bytes(m.group(1), "utf-8").decode("unicode_escape"))
        elif m.group(2) is not None:
            if m.group(2) != "name":
                raise ValueError("unknown runner policy dynamic value")
            tokens.append("__dynamic_name__")
    if raw[pos:].strip() or tuple(tokens) != EXPECTED:
        raise ValueError("runner policy literals/dynamic marker changed; refusing probe")
    return image, [x for x in tokens if x != "__dynamic_name__"]


def container_argv(flags: list[str], image: str, name: str, probe_id: str, script: str) -> list[str]:
    if not re.fullmatch(r"oga-m12-access-[0-9a-f]{32}", name) or not re.fullmatch(r"[0-9a-f]{32}", probe_id):
        raise ValueError("invalid generated probe identity")
    restored: list[str] = []
    i = 0
    while i < len(flags):
        if flags[i] == "--name":
            if i + 1 >= len(flags) or flags[i + 1] != "--label":
                raise ValueError("unexpected dynamic name policy position")
            restored.extend(("--name", name))
        else:
            restored.append(flags[i])
        i += 1
    if restored.count("--name") != 1:
        raise ValueError("unexpected runner name flag count")
    return ["podman", *restored, "--label=academy.install=m12-access-probe",
            f"--label=academy.probe.id={probe_id}", "-i", image, "sh", "-c", script, "--"]


def cleanup_argv(name: str) -> list[str]:
    if not re.fullmatch(r"oga-m12-access-[0-9a-f]{32}", name):
        raise ValueError("refusing unscoped cleanup")
    return ["podman", "rm", "--force", "--time=0", "--ignore", name]


def _fixture_tar() -> bytes:
    with tempfile.TemporaryDirectory(prefix="oga-m12-src-") as td:
        src = Path(td) / "probe.go"
        src.write_text(FIXTURE, encoding="utf-8")
        go = ROOT / ".tools/go/bin/go"
        if not go.is_file():
            go = Path(shutil.which("go") or "")
        if not str(go) or not go.is_file():
            raise RuntimeError("pinned Go toolchain unavailable; no download attempted")
        env = dict(os.environ, GOOS="linux", GOARCH="amd64", CGO_ENABLED="0", GOTOOLCHAIN="local")
        built = subprocess.run([str(go), "build", "-trimpath", "-o", str(Path(td)/"probe"), str(src)],
                               cwd=td, env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                               text=True, timeout=60)
        if built.returncode:
            raise RuntimeError("static fixture build failed: " + built.stdout[:1000])
        binary = (Path(td)/"probe").read_bytes()
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode="w") as tf:
        info = tarfile.TarInfo("probe")
        info.size, info.mode = len(binary), 0o500
        import io as _io
        tf.addfile(info, _io.BytesIO(binary))
    return buf.getvalue()


def _listener_counts(tcp: socket.socket, udp: socket.socket, ux: socket.socket) -> tuple[int, int, int]:
    accepted = 0
    tcp.settimeout(0.05)
    while True:
        try:
            conn, _ = tcp.accept()
            accepted += 1
            conn.close()
        except (TimeoutError, socket.timeout):
            break
    udp.settimeout(0.05)
    udp_count = 0
    try:
        udp.recvfrom(256)
        udp_count = 1
    except (TimeoutError, socket.timeout):
        pass
    ux.settimeout(0.05)
    ux_count = 0
    try:
        conn, _ = ux.accept()
        ux_count += 1
        conn.close()
    except (TimeoutError, socket.timeout):
        pass
    return accepted, udp_count, ux_count


def validate_output(output: str, rc: int, counts: tuple[int, int, int]) -> None:
    expected = {
        "direct_sentinel_denied": "true", "proc_root_sentinel_denied": "true",
        "sentinel_write_denied": "true", "proc_root_write_denied": "true",
        "proc1_pid_namespace_matches_self": "true", "host_sentinel_in_mountinfo": "false",
        "issuer_fixture_denied": "true", "issuer_write_denied": "true", "issuer_fixture_in_mountinfo": "false",
        "host_only_env_in_container": "false", "host_only_env_in_proc1_environ": "false",
        "docker_socket_unavailable": "true", "podman_socket_unavailable": "true",
        "tcp_connect_denied": "true", "unix_connect_denied": "true",
    }
    fields = dict(re.findall(r"^([a-z0-9_]+)=([^\n]+)$", output, re.M))
    for key, value in expected.items():
        if fields.get(key) != value:
            raise ValueError(f"expected {key}={value}; got {fields.get(key, 'missing')}")
    if not re.fullmatch(r"[1-9][0-9]*", fields.get("mountinfo_lines", "")):
        raise ValueError("mountinfo snapshot missing")
    if fields.get("proc1_root_visible") not in ("true", "false"):
        raise ValueError("proc root visibility missing")
    if fields.get("udp_write_attempted") not in ("true", "false") or fields.get("udp_write_unavailable") not in ("true", "false"):
        raise ValueError("UDP attempt status missing")
    if rc != 0 or counts != (0, 0, 0):
        raise ValueError(f"probe exit/listener evidence mismatch (exit={rc}, listeners={counts})")


def run_probe(output_path: Path | None = None, timeout: int = 90) -> str:
    if not 1 <= timeout <= 120:
        raise ValueError("timeout must be between 1 and 120 seconds")
    image, flags = policy_flags(RUNNER_SOURCE.read_text(encoding="utf-8"))
    if not shutil.which("podman"):
        raise RuntimeError("podman is unavailable")
    info = subprocess.run(["podman", "info", "--format", "{{.Host.Security.Rootless}} {{.Host.CgroupsVersion}}"],
                          capture_output=True, text=True, timeout=15)
    if info.returncode or info.stdout.strip() != "true v2":
        raise RuntimeError("requires rootless Podman and cgroup v2: " + info.stdout.strip())
    exists = subprocess.run(["podman", "image", "exists", image], capture_output=True, timeout=15)
    if exists.returncode:
        raise RuntimeError("pinned image absent locally; probe will not pull")
    tar_input = _fixture_tar()
    lines = ["M12 S01/S02 bounded rootless host-boundary probe", f"runner image: {image}",
             "policy: exact current runner policy() literals; no mounts, --network=none, read-only, cap-drop=ALL",
             "scope: disposable temp sentinel/key fixture and loopback TCP/UDP/unix listeners; no external network or DNS query.",
             "proc semantics: /proc/1 is checked to be in this container's PID namespace; this does not inspect host /proc/1/environ."]
    with tempfile.TemporaryDirectory(prefix="oga-m12-host-", dir="/tmp") as tmp:
        root = Path(tmp)
        os.chmod(root, 0o755)
        sentinel = root / "host-sentinel.txt"
        marker = "DISPOSABLE_SENTINEL_" + uuid.uuid4().hex
        sentinel.write_text(marker, encoding="ascii")
        os.chmod(sentinel, 0o644)
        issuer = root / "issuer-key-fixture.pem"
        key_marker = "DISPOSABLE_ISSUER_FIXTURE_" + uuid.uuid4().hex
        issuer.write_text(key_marker, encoding="ascii")
        os.chmod(issuer, 0o600)
        host_env_marker = "HOST_ONLY_ENV_" + uuid.uuid4().hex
        sock_path = str(root / "host.sock")
        tcp = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        udp = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        ux = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        sockets = (tcp, udp, ux)
        try:
            tcp.bind(("127.0.0.1", 0)); tcp.listen(4)
            udp.bind(("127.0.0.1", 0))
            ux.bind(sock_path); ux.listen(4)
            tcp_addr = f"127.0.0.1:{tcp.getsockname()[1]}"
            udp_addr = f"127.0.0.1:{udp.getsockname()[1]}"
            probe_id = uuid.uuid4().hex
            name = "oga-m12-access-" + probe_id
            script = ("set -eu; tar -xf - -C /work; /work/probe \"$1\" \"$2\" \"$3\" \"$4\" \"$5\" \"$6\"; "
                      "test \"$(id -u)\" = 65532; grep -q '^Seccomp:[[:space:]]*2' /proc/self/status; "
                      "grep -q '^NoNewPrivs:[[:space:]]*1' /proc/self/status")
            argv = container_argv(flags, image, name, probe_id, script) + [str(sentinel), str(issuer), tcp_addr, udp_addr, sock_path, host_env_marker]
            started = time.monotonic()
            try:
                host_env = dict(os.environ, OGA_M12_HOST_ONLY_SENTINEL=host_env_marker)
                result = subprocess.run(argv, input=tar_input, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
                                        timeout=timeout, env=host_env)
            finally:
                cleanup = subprocess.run(cleanup_argv(name), stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=15)
                if cleanup.returncode:
                    raise RuntimeError("scoped probe container cleanup failed")
            output = result.stdout.decode("utf-8", errors="replace")
            counts = _listener_counts(tcp, udp, ux)
            try:
                validate_output(output, result.returncode, counts)
            except ValueError as exc:
                raise ValueError(f"{exc}; container output={output[-1200:]!r}") from exc
            if sentinel.read_text(encoding="ascii") != marker or issuer.read_text(encoding="ascii") != key_marker:
                raise RuntimeError("host fixture changed")
            if stat.S_IMODE(sentinel.stat().st_mode) != 0o644 or stat.S_IMODE(issuer.stat().st_mode) != 0o600:
                raise RuntimeError("host fixture permissions changed")
            lines.append(f"result: PASS; elapsed={time.monotonic()-started:.2f}s; host listeners tcp/udp/unix={counts}")
            lines.append("host sentinel (0644) and issuer-key fixture (0600) unchanged; contents redacted")
            lines.extend("  " + x for x in output.rstrip().splitlines())
            lines.append("interpretation: this local probe does not establish DNS/external-network blocking or other-host behavior.")
        finally:
            for sock in sockets:
                sock.close()
    report = "\n".join(lines) + "\n"
    if output_path:
        output_path.parent.mkdir(parents=True, exist_ok=True)
        output_path.write_text(report, encoding="utf-8")
    return report


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--output", type=Path)
    parser.add_argument("--timeout", type=int, default=90)
    args = parser.parse_args()
    try:
        print(run_probe(args.output, args.timeout), end="")
    except (OSError, subprocess.SubprocessError, RuntimeError, ValueError) as exc:
        print("M12 host-boundary probe NOT PASS: " + str(exc), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
