#!/usr/bin/env python3
"""Corroborate runner policy cgroup limits using fixed trusted Go fixtures.

This probe is intentionally not an API benchmark. It builds fixed local fixtures,
feeds binaries over stdin (never bind mounts), and runs them in the pinned runner
image with policy flags read from internal/runner/runner.go.
"""
from __future__ import annotations

import argparse
import io
import os
import re
import shlex
import shutil
import subprocess
import sys
import tarfile
import time
import uuid
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RUNNER_SOURCE = ROOT / "internal/runner/runner.go"
IMAGE_RE = re.compile(r'^const Image = "([^"]+)"', re.M)
POLICY_RE = re.compile(r'func policy\(name string\) \[\]string \{\s*return \[\]string\{(.*?)\}\s*\}', re.S)
STRING_RE = re.compile(r'"((?:\\.|[^"\\])*)"')
POLICY_TOKEN_RE = re.compile(r'"((?:\\.|[^"\\])*)"|([A-Za-z_][A-Za-z0-9_]*)|(,)')

# Static, reviewed fixtures. The PID fixture creates only children of itself and
# kills/reaps all of them before returning; no arbitrary source is accepted.
FIXTURES = {
    "memory": r'''package main
import ("fmt"; "runtime")
func main() {
 var blocks [][]byte
 for { b := make([]byte, 8<<20); for i := 0; i < len(b); i += 4096 { b[i] = 1 }; blocks = append(blocks, b); runtime.KeepAlive(blocks); fmt.Printf("allocated=%dMiB\n", len(blocks)*8) }
}
''',
    "pids": r'''package main
import ("errors"; "fmt"; "os"; "os/exec"; "syscall"; "time")
func main() {
 if len(os.Args) > 1 && os.Args[1] == "child" { time.Sleep(time.Hour); return }
 var children []*exec.Cmd
 var eagain bool
 for { c := exec.Command("/proc/self/exe", "child"); if err := c.Start(); err != nil { fmt.Printf("spawn_error=%v\n", err); eagain = errors.Is(err, syscall.EAGAIN); break }; children = append(children, c) }
 for _, c := range children { _ = c.Process.Kill() }; for _, c := range children { _ = c.Wait() }
 fmt.Printf("children_cleaned=%d\n", len(children))
 if len(children) == 0 { os.Exit(2) }
 // Keep the resource-exhaustion signature machine-readable and explicit.
 fmt.Println("eagain=" + fmt.Sprint(eagain)); if !eagain { os.Exit(3) }
}
''',
}

EXPECTED_POLICY_TOKENS = (
    "run", "--rm", "--pull=never", "--name", "__dynamic_name__", "--label", "academy.runner=true",
    "--network=none", "--http-proxy=false", "--stop-timeout=0", "--read-only", "--cap-drop=ALL",
    "--security-opt=no-new-privileges", "--user=65532:65532", "--memory=768m", "--memory-swap=768m",
    "--pids-limit=128", "--cpus=1", "--tmpfs=/tmp:rw,noexec,nosuid,nodev,size=128m",
    "--tmpfs=/work:rw,nosuid,nodev,size=192m", "--workdir=/work", "--env=CGO_ENABLED=0",
    "--env=GOTOOLCHAIN=local", "--env=GOPROXY=off", "--env=GOSUMDB=off",
    "--tmpfs=/cache:rw,nosuid,nodev,size=192m", "--shm-size=16m", "--env=GOCACHE=/cache",
    "--env=HOME=/tmp", "--env=GOMAXPROCS=2",
)


def policy_flags(source: str) -> tuple[str, list[str]]:
    image_match = IMAGE_RE.search(source)
    body_match = POLICY_RE.search(source)
    if not image_match or not body_match:
        raise ValueError("Could not parse Image and policy() from internal/runner/runner.go")
    image = image_match.group(1)
    if not re.fullmatch(r"[^@\s]+@sha256:[0-9a-f]{64}", image):
        raise ValueError("runner image must be pinned to an immutable sha256 digest; refusing mutable image reference")
    body = body_match.group(1)
    tokens: list[str] = []
    previous = 0
    for match in POLICY_TOKEN_RE.finditer(body):
        if body[previous:match.start()].strip():
            raise ValueError("runner policy contains unsupported syntax; refusing probe")
        previous = match.end()
        if match.group(1) is not None:
            tokens.append(bytes(match.group(1), "utf-8").decode("unicode_escape"))
        elif match.group(2) is not None:
            if match.group(2) != "name":
                raise ValueError("runner policy contains an unknown dynamic value; refusing probe")
            tokens.append("__dynamic_name__")
        else:
            continue
    if body[previous:].strip() or tuple(tokens) != EXPECTED_POLICY_TOKENS:
        raise ValueError("runner policy literals or dynamic marker changed; refusing probe")
    flags = [token for token in tokens if token != "__dynamic_name__"]
    return image, flags


def cleanup_argv(name: str) -> list[str]:
    if not re.fullmatch(r"oga-m12-probe-[0-9a-f]{32}", name):
        raise ValueError("refusing cleanup for non-probe container name")
    return ["podman", "rm", "--force", "--time=0", "--ignore", name]


def container_argv(flags: list[str], image: str, name: str, probe_id: str, script: str) -> list[str]:
    """Reinsert dynamic Go policy values omitted by literal-only extraction."""
    if not re.fullmatch(r"oga-m12-probe-[0-9a-f]{32}", name):
        raise ValueError("refusing to run with non-probe container name")
    if not re.fullmatch(r"[0-9a-f]{32}", probe_id):
        raise ValueError("invalid probe id")
    restored: list[str] = []
    index = 0
    while index < len(flags):
        flag = flags[index]
        if flag == "--name":
            if index + 1 >= len(flags) or flags[index + 1] != "--label":
                raise ValueError("could not restore dynamic runner policy name")
            restored.extend(("--name", name))
        else:
            restored.append(flag)
        index += 1
    if restored.count("--name") != 1 or restored.count("--label") != 1:
        raise ValueError("unexpected runner policy dynamic/static marker layout")
    return [
        "podman", *restored,
        "--label=academy.install=m12-probe",
        f"--label=academy.probe.id={probe_id}",
        "-i", image, "sh", "-c", script,
    ]


def _build(go_bin: str, kind: str, directory: Path) -> Path:
    src = directory / f"{kind}.go"
    out = directory / kind
    src.write_text(FIXTURES[kind], encoding="utf-8")
    env = dict(os.environ, GOOS="linux", GOARCH="amd64", CGO_ENABLED="0", GOTOOLCHAIN="local")
    result = subprocess.run([go_bin, "build", "-trimpath", "-o", str(out), str(src)], cwd=directory,
                            env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=120)
    if result.returncode:
        raise RuntimeError(f"host Go fixture build failed ({kind}): {result.stdout.strip()}")
    return out


def select_go_compiler(go_bin: str = "", repo_root: Path = ROOT) -> str:
    """Select the pinned repository toolchain first, then PATH, and verify it."""
    if go_bin:
        selected = go_bin
    else:
        bundled = repo_root / ".tools/go/bin/go"
        selected = str(bundled) if bundled.is_file() else (shutil.which("go") or "")
    if not selected:
        raise RuntimeError("Required Go toolchain go1.27.2 not found (.tools/go/bin/go or PATH); no download attempted")
    try:
        result = subprocess.run([selected, "version"], text=True, stdout=subprocess.PIPE,
                                stderr=subprocess.STDOUT, timeout=15)
    except (OSError, subprocess.SubprocessError) as exc:
        raise RuntimeError(f"Could not verify host Go compiler {selected}: {exc}") from exc
    version_output = result.stdout.strip()
    match = re.match(r"^go version (\S+)(?:\s|$)", version_output)
    if result.returncode or not match or match.group(1) != "go1.27.2":
        raise RuntimeError(f"Required Go toolchain is exactly go1.27.2; {selected} reported {version_output or 'no version'}; no download attempted")
    return selected


def _tar_binary(path: Path, name: str) -> bytes:
    buf = io.BytesIO()
    with tarfile.open(fileobj=buf, mode="w") as tf:
        data = path.read_bytes()
        info = tarfile.TarInfo(name)
        info.size, info.mode = len(data), 0o500
        tf.addfile(info, io.BytesIO(data))
    return buf.getvalue()


def validate_output(kind: str, output: str, returncode: int) -> None:
    if "PROBE_EVENTS_BEGIN" not in output or "PROBE_EVENTS_END" not in output:
        raise ValueError(f"{kind}: trusted wrapper did not capture cgroup event files")
    memory = re.search(r"^memory_events:\s*\n?(.+?)(?=^pids_events:|\Z)", output, re.M | re.S)
    pids = re.search(r"^pids_events:\s*\n?(.+?)(?=^PROBE_EVENTS_END|\Z)", output, re.M | re.S)
    if not memory or not pids:
        raise ValueError(f"{kind}: missing memory.events or pids.events output")
    oom = re.search(r"^oom_kill\s+(\d+)$", memory.group(1), re.M)
    max_count = re.search(r"^max\s+(\d+)$", pids.group(1), re.M)
    if not oom or not max_count:
        raise ValueError(f"{kind}: malformed cgroup event counters")
    exits = re.findall(r"^child_exit=(\d+)$", output, re.M)
    expected_exit = "137" if kind == "memory" else "0"
    if len(exits) != 1 or exits[0] != expected_exit:
        raise ValueError(f"{kind}: expected exactly one child_exit={expected_exit} result")
    if kind == "memory":
        if int(oom.group(1)) < 1:
            raise ValueError("memory: expected memory.events oom_kill > 0")
    else:
        if int(max_count.group(1)) < 1:
            raise ValueError("pids: expected pids.events max > 0")
        if "spawn_error=" not in output or "temporarily unavailable" not in output:
            raise ValueError("pids: expected process creation failure EAGAIN")
        cleaned = re.findall(r"^children_cleaned=([1-9][0-9]*)$", output, re.M)
        eagain = re.findall(r"^eagain=true$", output, re.M)
        if len(cleaned) != 1 or len(eagain) != 1:
            raise ValueError("pids: fixture did not report EAGAIN and child cleanup")
    if returncode != 0:
        raise ValueError(f"{kind}: wrapper/container exited {returncode}")


def run_probe(go_bin: str, output_path: Path | None, timeout: int = 90) -> str:
    if timeout <= 0:
        raise ValueError("timeout must be a positive number of seconds")
    source = RUNNER_SOURCE.read_text(encoding="utf-8")
    image, flags = policy_flags(source)
    podman = shutil.which("podman")
    if not podman:
        raise RuntimeError("Prerequisite unavailable: podman executable is not on PATH")
    info = subprocess.run(["podman", "info", "--format", "{{.Host.Security.Rootless}} {{.Host.CgroupsVersion}}"],
                          text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=15)
    if info.returncode or info.stdout.strip() != "true v2":
        raise RuntimeError("Prerequisite unavailable: rootless Podman with cgroup v2 required; got " + info.stdout.strip())
    # Verify pinned image is already present. Never pull, install, or use network.
    image_check = subprocess.run(["podman", "image", "exists", image], stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=15)
    if image_check.returncode:
        raise RuntimeError("Pinned runner image is not present locally; probe will not pull it: " + image)
    go = select_go_compiler(go_bin)
    lines = ["Open Go Academy M12 sandbox cgroup evidence probe",
             "Scope: corroborates the same policy flags as internal/runner; not an end-to-end API benchmark.",
             f"Image: {image}", "Policy flags (read from current policy()): " + " ".join(shlex.quote(x) for x in flags),
             "Prerequisites: rootless=true, cgroups=v2; local pinned image present; no network or host mounts."]
    with __import__("tempfile").TemporaryDirectory(prefix="oga-m12-probe-") as tmp:
        temp = Path(tmp)
        for kind in ("memory", "pids"):
            binary = _build(go, kind, temp)
            probe_id = uuid.uuid4().hex
            name = "oga-m12-probe-" + probe_id
            script = ("set +e; test \"$(cat /sys/fs/cgroup/memory.max)\" = 805306368 || exit 90; "+
                      "test \"$(cat /sys/fs/cgroup/memory.swap.max)\" = 0 || exit 91; "+
                      "test \"$(cat /sys/fs/cgroup/pids.max)\" = 128 || exit 92; "+
                      "test \"$(cat /sys/fs/cgroup/cpu.max)\" = '100000 100000' || exit 93; "+
                      "test \"$(id -u)\" = 65532 || exit 94; "+
                      "grep -q '^Seccomp:[[:space:]]*2' /proc/self/status || exit 95; "+
                      "grep -q '^NoNewPrivs:[[:space:]]*1' /proc/self/status || exit 96; "+
                      "tar -xf - -C /work; /work/probe; rc=$?; "+
                      "printf 'child_exit=%s\\nPROBE_EVENTS_BEGIN\\n' \"$rc\"; "+
                      "printf 'memory_events:\\n'; cat /sys/fs/cgroup/memory.events; "+
                      "printf 'pids_events:\\n'; cat /sys/fs/cgroup/pids.events; "+
                      "printf 'PROBE_EVENTS_END\\n'; exit 0")
            argv = container_argv(flags, image, name, probe_id, script)
            cleanup = cleanup_argv(name)
            started = time.monotonic()
            try:
                result = subprocess.run(argv, input=_tar_binary(binary, "probe"), stdout=subprocess.PIPE,
                                        stderr=subprocess.STDOUT, timeout=timeout)
                output = result.stdout.decode("utf-8", errors="replace")
                validate_output(kind, output, result.returncode)
                lines.append(f"{kind}: PASS ({time.monotonic()-started:.1f}s)")
                lines.extend("  " + line for line in output.rstrip().splitlines())
            finally:
                # Exact UUID-scoped name only; never enumerate or clean other containers.
                cleaned = subprocess.run(cleanup, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=15)
                if cleaned.returncode:
                    raise RuntimeError(f"scoped cleanup failed for {name}: {cleaned.stderr.decode(errors='replace')}")
    report = "\n".join(lines) + "\n"
    if output_path:
        output_path.write_text(report, encoding="utf-8")
    return report


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--go-bin", default="", help="host Go compiler path (default: go from PATH)")
    parser.add_argument("--output", type=Path, help="write reproducible text evidence here")
    parser.add_argument("--timeout", type=int, default=90, help="per-container timeout seconds")
    args = parser.parse_args()
    try:
        report = run_probe(args.go_bin, args.output, args.timeout)
    except (OSError, subprocess.SubprocessError, RuntimeError, ValueError) as exc:
        print(f"M12 sandbox cgroup probe FAILED: {exc}", file=sys.stderr)
        return 1
    print(report, end="")
    return 0

if __name__ == "__main__":
    raise SystemExit(main())
