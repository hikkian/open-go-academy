#!/usr/bin/env python3
"""Bounded namespace, route, mount and cleanup probe for the pinned rootless runner."""
from __future__ import annotations

import argparse
import hashlib
import importlib.util
import json
import os
import re
import shutil
import subprocess
import sys
import tarfile
import tempfile
import time
import uuid
from pathlib import Path
from typing import Any, Callable

ROOT = Path(__file__).resolve().parents[1]
ACCESS_HELPER_PATH = ROOT / "scripts/m12_sandbox_access_probe.py"
LIMIT_HELPER_PATH = ROOT / "scripts/m12_sandbox_limit_probe.py"
RUNNER_SOURCE = ROOT / "internal/runner/runner.go"
IMAGE = "docker.io/library/golang@sha256:3b15ba438c60ab8fabe5fee3b11025c0049e358b31feff384648f11fc8e335f8"
PROBE_NAME_RE = re.compile(r"^oga-m12-access-[0-9a-f]{32}$")
PROBE_ID_RE = re.compile(r"^[0-9a-f]{32}$")
CGROUP_EVENTS_RE = re.compile(r"(?m)^populated\s+([01])$")
OBSERVATION_MARKER = "OGA_M12_NAMESPACE="


def _load_access_helper():
    spec = importlib.util.spec_from_file_location("m12_sandbox_access_probe", ACCESS_HELPER_PATH)
    if spec is None or spec.loader is None:
        raise RuntimeError("could not load existing M12 access-probe policy helpers")
    module = importlib.util.module_from_spec(spec)
    sys.modules[spec.name] = module
    spec.loader.exec_module(module)
    return module


ACCESS = _load_access_helper()
_limit_spec = importlib.util.spec_from_file_location("m12_sandbox_limit_probe", LIMIT_HELPER_PATH)
if _limit_spec is None or _limit_spec.loader is None:
    raise RuntimeError("could not load existing M12 toolchain helper")
LIMIT = importlib.util.module_from_spec(_limit_spec)
sys.modules[_limit_spec.name] = LIMIT
_limit_spec.loader.exec_module(LIMIT)

# Static fixture: no arguments, files, sockets, lookups, host paths or mounts.
# A one-second sleep keeps the sandbox cgroup populated while its events are read.
FIXTURE = r'''package main
import (
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "os"
    "sort"
    "strings"
    "time"
)
type Observation struct {
    PIDSelfNS string `json:"pid_self_ns"`
    PID1NS string `json:"pid1_ns"`
    MountSelfNS string `json:"mount_self_ns"`
    MountPID1NS string `json:"mount_pid1_ns"`
    MountEntries []string `json:"mount_entries"`
    Routes4 []string `json:"routes4"`
    Routes6 []string `json:"routes6"`
    ResolvConfPresent bool `json:"resolv_conf_present"`
    CgroupEvents string `json:"cgroup_events"`
    CgroupEventsError bool `json:"cgroup_events_error"`
}
func link(path string) string {
    value, err := os.Readlink(path)
    if err != nil { return "inaccessible" }
    return value
}
func mountEntries() []string {
    data, err := os.ReadFile("/proc/self/mountinfo")
    if err != nil { return nil }
    entries := make(map[string]bool)
    for _, line := range strings.Split(string(data), "\n") {
        fields := strings.Fields(line)
        separator := -1
        for i, field := range fields { if field == "-" { separator = i; break } }
        if separator < 0 || len(fields) < 6 || separator+1 >= len(fields) { continue }
        key := fields[4] + "\x00" + fields[separator+1]
        digest := sha256.Sum256([]byte(key))
        entries[hex.EncodeToString(digest[:])] = true
    }
    result := make([]string, 0, len(entries))
    for entry := range entries { result = append(result, entry) }
    sort.Strings(result)
    return result
}
func routeRows(path string, ipv4 bool) []string {
    data, err := os.ReadFile(path)
    if err != nil { return nil }
    result := []string{}
    for i, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
        if line == "" || (ipv4 && i == 0) { continue }
        result = append(result, strings.TrimSpace(line))
    }
    return result
}
func main() {
    time.Sleep(time.Second)
    _, resolvErr := os.Stat("/etc/resolv.conf")
    events, eventsErr := os.ReadFile("/sys/fs/cgroup/cgroup.events")
    o := Observation{
        PIDSelfNS: link("/proc/self/ns/pid"), PID1NS: link("/proc/1/ns/pid"),
        MountSelfNS: link("/proc/self/ns/mnt"), MountPID1NS: link("/proc/1/ns/mnt"),
        MountEntries: mountEntries(), Routes4: routeRows("/proc/net/route", true),
        Routes6: routeRows("/proc/net/ipv6_route", false), ResolvConfPresent: resolvErr == nil,
        CgroupEvents: strings.TrimSpace(string(events)), CgroupEventsError: eventsErr != nil,
    }
    encoded, err := json.Marshal(o)
    if err != nil { os.Exit(2) }
    fmt.Printf("OGA_M12_NAMESPACE=%s\n", encoded)
}
'''


def cleanup_argv(name: str) -> list[str]:
    if not PROBE_NAME_RE.fullmatch(name):
        raise ValueError("refusing cleanup for a non-generated namespace-probe name")
    return ACCESS.cleanup_argv(name)


def container_argv(flags: list[str], image: str, name: str, probe_id: str, script: str) -> list[str]:
    if image != IMAGE or not PROBE_ID_RE.fullmatch(probe_id):
        raise ValueError("refusing mutable image or invalid probe id")
    if not PROBE_NAME_RE.fullmatch(name):
        raise ValueError("refusing non-generated namespace-probe name")
    argv = ACCESS.container_argv(flags, image, name, probe_id, script)
    if "--network=none" not in argv or "--read-only" not in argv:
        raise ValueError("runner policy is missing required network/read-only isolation")
    if any(arg in ("-v", "--volume", "--mount") or arg.startswith(("--volume=", "--mount=")) for arg in argv):
        raise ValueError("host mounts are forbidden in the namespace probe")
    return argv


def _mount_keys(lines: list[str]) -> set[str]:
    keys: set[str] = set()
    for line in lines:
        fields = line.split()
        try:
            separator = fields.index("-")
        except ValueError:
            continue
        if len(fields) < 6 or separator + 1 >= len(fields):
            continue
        key = fields[4] + "\0" + fields[separator + 1]
        keys.add(hashlib.sha256(key.encode()).hexdigest())
    return keys


def observer_snapshot() -> dict[str, Any]:
    def readlink(path: str) -> str:
        try:
            return os.readlink(path)
        except OSError:
            return "inaccessible"

    try:
        mountinfo = Path("/proc/self/mountinfo").read_text(encoding="utf-8").splitlines()
    except OSError:
        mountinfo = []
    return {
        "pid_ns": readlink("/proc/self/ns/pid"),
        "mount_ns": readlink("/proc/self/ns/mnt"),
        "mount_entries": sorted(_mount_keys(mountinfo)),
    }


def parse_observation(output: str) -> dict[str, Any]:
    for line in output.splitlines():
        if line.startswith(OBSERVATION_MARKER):
            try:
                value = json.loads(line[len(OBSERVATION_MARKER):])
            except json.JSONDecodeError as exc:
                raise ValueError("malformed namespace fixture JSON") from exc
            if not isinstance(value, dict):
                raise ValueError("namespace fixture did not return an object")
            return value
    raise ValueError("namespace fixture output is missing its observation marker")


def validate_observation(value: dict[str, Any], observer: dict[str, Any]) -> dict[str, Any]:
    required_strings = ("pid_self_ns", "pid1_ns", "mount_self_ns", "mount_pid1_ns", "cgroup_events")
    if any(not isinstance(value.get(key), str) for key in required_strings):
        raise ValueError("namespace fixture is missing namespace/cgroup observations")
    for key in ("mount_entries", "routes4", "routes6"):
        if not isinstance(value.get(key), list) or any(not isinstance(item, str) for item in value[key]):
            raise ValueError(f"malformed {key} observation")
    if not isinstance(value.get("cgroup_events_error"), bool) or not isinstance(value.get("resolv_conf_present"), bool):
        raise ValueError("malformed cgroup/resolv.conf observation")
    if value["cgroup_events_error"] or not CGROUP_EVENTS_RE.search(value["cgroup_events"]):
        raise ValueError("container cgroup.events was unavailable or malformed during the live probe")

    def route_iface(row: str, ipv4: bool) -> str:
        fields = row.split()
        if not fields:
            return ""
        return fields[0] if ipv4 else fields[-1]

    non_loopback = [row for row in value["routes4"] if route_iface(row, True) != "lo"]
    non_loopback += [row for row in value["routes6"] if route_iface(row, False) != "lo"]
    if non_loopback:
        raise ValueError("--network=none fixture observed a non-loopback route")
    observer_mounts = set(observer.get("mount_entries", []))
    container_mounts = set(value["mount_entries"])
    return {
        "pid_namespace_differs": observer.get("pid_ns") != value["pid_self_ns"],
        "mount_namespace_differs": observer.get("mount_ns") != value["mount_self_ns"],
        "observer_mount_entries": len(observer_mounts),
        "container_mount_entries": len(container_mounts),
        "observer_only_mount_entries": len(observer_mounts - container_mounts),
        "container_only_mount_entries": len(container_mounts - observer_mounts),
        "shared_mount_entries": len(observer_mounts & container_mounts),
        "non_loopback_routes": False,
        "route4_rows": len(value["routes4"]),
        "route6_rows": len(value["routes6"]),
        "resolv_conf_present_observation_only": value["resolv_conf_present"],
        "cgroup_events_live": value["cgroup_events"],
        "proc1_pid_ns_inaccessible": value["pid1_ns"] == "inaccessible",
    }


def _read_after_rm(name: str, timeout: int = 5) -> str:
    result = subprocess.run(["podman", "inspect", "--format", "{{.State.CgroupPath}}", name],
                            text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=timeout)
    if result.returncode:
        return "unavailable: container disappeared under --rm before post-exit cgroup.events could be read; populated=0 not observed"
    cgroup_path = result.stdout.strip()
    root = Path("/sys/fs/cgroup").resolve()
    candidate = (root / cgroup_path.lstrip("/")).resolve()
    if candidate != root and root not in candidate.parents:
        return "unavailable: reported cgroup path escaped the mounted cgroup root"
    try:
        events = (candidate / "cgroup.events").read_text(encoding="utf-8").strip()
    except OSError:
        return "unavailable: cgroup.events disappeared before literal populated=0; no zero inferred"
    match = CGROUP_EVENTS_RE.search(events)
    if not match:
        return "unavailable: cgroup.events had no literal populated value"
    return f"observed after run: {events}"


def scoped_names(probe_id: str, runner: Callable[..., subprocess.CompletedProcess] = subprocess.run) -> set[str]:
    if not PROBE_ID_RE.fullmatch(probe_id):
        raise ValueError("invalid probe id for exact-label query")
    result = runner(["podman", "ps", "--all", "--filter", "label=academy.install=m12-access-probe",
                     "--filter", f"label=academy.probe.id={probe_id}", "--format", "{{.Names}}"],
                    text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=5)
    if result.returncode:
        raise RuntimeError("could not query exact namespace-probe labels: " + result.stderr.strip())
    return {line.strip() for line in result.stdout.splitlines() if line.strip()}


def cleanup_exact(name: str, probe_id: str, runner: Callable[..., subprocess.CompletedProcess] = subprocess.run) -> str:
    if not PROBE_NAME_RE.fullmatch(name) or not PROBE_ID_RE.fullmatch(probe_id):
        raise ValueError("refusing cleanup outside the generated name/id scope")
    names = scoped_names(probe_id, runner)
    if names - {name}:
        raise RuntimeError("exact probe label unexpectedly matched another container; refusing cleanup")
    if name not in names:
        exists = runner(["podman", "container", "exists", name],
                        text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=5)
        if exists.returncode == 0:
            raise RuntimeError("generated name exists without both exact installation labels; refusing cleanup")
        if exists.returncode != 1:
            raise RuntimeError("could not verify exact generated container absence")
        return "container already removed by --rm; exact labels have no remaining container"
    removed = runner(cleanup_argv(name), text=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=10)
    if removed.returncode:
        raise RuntimeError("exact generated container cleanup failed: " + removed.stderr.strip())
    remaining = scoped_names(probe_id, runner)
    if remaining:
        raise RuntimeError("exact probe label still has a container after cleanup")
    return "removed exact generated name after exact install/probe-label match"


def _build_fixture(go_bin: str, directory: Path) -> Path:
    src = directory / "namespace_probe.go"
    out = directory / "namespace_probe"
    src.write_text(FIXTURE, encoding="utf-8")
    env = dict(os.environ, GOOS="linux", GOARCH="amd64", CGO_ENABLED="0", GOTOOLCHAIN="local")
    built = subprocess.run([go_bin, "build", "-trimpath", "-o", str(out), str(src)], cwd=directory,
                           env=env, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=60)
    if built.returncode:
        raise RuntimeError("static namespace fixture build failed: " + built.stdout.strip())
    return out


def _fixture_tar(binary: Path) -> bytes:
    import io
    from io import BytesIO

    stream = BytesIO()
    with tarfile.open(fileobj=stream, mode="w") as archive:
        data = binary.read_bytes()
        info = tarfile.TarInfo("namespace-probe")
        info.size, info.mode = len(data), 0o500
        archive.addfile(info, io.BytesIO(data))
    return stream.getvalue()


def run_probe(timeout: int = 30) -> str:
    if timeout < 5 or timeout > 60:
        raise ValueError("probe timeout must be between 5 and 60 seconds")
    source = RUNNER_SOURCE.read_text(encoding="utf-8")
    image, flags = ACCESS.policy_flags(source)
    if image != IMAGE:
        raise ValueError("runner image differs from this probe's reviewed pinned image")
    podman = shutil.which("podman")
    if not podman:
        raise RuntimeError("podman is unavailable; no installation or image pull attempted")
    info = subprocess.run([podman, "info", "--format", "{{.Host.Security.Rootless}} {{.Host.CgroupsVersion}}"],
                          text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, timeout=10)
    if info.returncode or info.stdout.strip() != "true v2":
        raise RuntimeError("requires rootless Podman with cgroup v2; got " + info.stdout.strip())
    cached = subprocess.run([podman, "image", "exists", image], stdout=subprocess.PIPE,
                            stderr=subprocess.PIPE, timeout=10)
    if cached.returncode:
        raise RuntimeError("pinned image is not cached locally; probe will not pull it")

    probe_id = uuid.uuid4().hex
    name = "oga-m12-access-" + probe_id
    observer = observer_snapshot()
    script = "tar -xf - -C /work && /work/namespace-probe"
    argv = container_argv(flags, image, name, probe_id, script)
    output = ""
    result_code = -1
    post_cgroup = "not observed"
    cleanup_result = "not completed"
    try:
        with tempfile.TemporaryDirectory(prefix="oga-m12-namespace-") as temp:
            binary = _build_fixture(LIMIT.select_go_compiler(), Path(temp))
            completed = subprocess.run(argv, input=_fixture_tar(binary), stdout=subprocess.PIPE,
                                        stderr=subprocess.STDOUT, timeout=timeout)
            output = completed.stdout.decode("utf-8", errors="replace")
            result_code = completed.returncode
            post_cgroup = _read_after_rm(name)
    finally:
        cleanup_result = cleanup_exact(name, probe_id)

    if result_code:
        raise RuntimeError(f"namespace fixture exited {result_code}: {output[-2000:]}")
    observation = parse_observation(output)
    summary = validate_observation(observation, observer)
    if not summary["pid_namespace_differs"] or not summary["mount_namespace_differs"]:
        raise RuntimeError("sandbox PID/mount namespace inode matched observer; see probe output")

    report = [
        "M12 bounded rootless namespace/route/cleanup probe: PASS",
        f"image: {image}",
        "network policy: --network=none; no DNS lookup was performed",
        "host mounts: none; only runner tmpfs policy mounts are present",
        f"probe name: {name}",
        f"observer PID namespace: {observer['pid_ns']}",
        f"container PID namespace: {observation['pid_self_ns']} (differs={summary['pid_namespace_differs']})",
        ("container /proc/1/ns/pid: inaccessible" if summary["proc1_pid_ns_inaccessible"]
         else f"container /proc/1/ns/pid: {observation['pid1_ns']}"),
        f"observer mount namespace: {observer['mount_ns']}",
        f"container mount namespace: {observation['mount_self_ns']} (differs={summary['mount_namespace_differs']})",
        f"mountinfo entry counts: observer={summary['observer_mount_entries']} container={summary['container_mount_entries']} shared={summary['shared_mount_entries']} observer-only={summary['observer_only_mount_entries']} container-only={summary['container_only_mount_entries']}",
        f"container route table rows: IPv4={json.dumps(observation['routes4'])} IPv6={json.dumps(observation['routes6'])}",
        f"non-loopback routes: {summary['non_loopback_routes']}",
        f"/etc/resolv.conf present: {summary['resolv_conf_present_observation_only']} (presence only; not read or queried)",
        f"cgroup.events while fixture was alive: {summary['cgroup_events_live']!r}",
        f"cgroup.events after --rm returned: {post_cgroup}",
        f"cleanup: {cleanup_result}",
    ]
    return "\n".join(report) + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--timeout", type=int, default=30, help="bounded container run timeout in seconds (5–60)")
    args = parser.parse_args()
    try:
        sys.stdout.write(run_probe(args.timeout))
        return 0
    except (OSError, RuntimeError, ValueError, subprocess.SubprocessError) as exc:
        print(f"M12 namespace probe failed: {exc}", file=sys.stderr)
        return 1


if __name__ == "__main__":
    raise SystemExit(main())
