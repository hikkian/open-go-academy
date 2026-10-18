import importlib.util
import re
import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import patch

SCRIPT = Path(__file__).with_name("m12_sandbox_limit_probe.py")
spec = importlib.util.spec_from_file_location("m12_probe", SCRIPT)
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)


class ProbeTests(unittest.TestCase):
    def test_reads_pinned_image_and_exact_policy_flags(self):
        source = probe.RUNNER_SOURCE.read_text(encoding="utf-8")
        image, flags = probe.policy_flags(source)
        self.assertTrue(image.startswith("docker.io/library/golang@sha256:"))
        self.assertEqual(flags[0], "run")
        self.assertIn("--memory=768m", flags)
        self.assertIn("--network=none", flags)
        unsafe_changes = (
            source.replace('"--network=none"', '"--network=none", "--network=host"'),
            source.replace('"--network=none"', '"--network=none", "--network=none"'),
            source.replace('"--read-only"', '"--read-only", "--volume=/:/host"'),
            source.replace('"--pids-limit=128"', '"--pids-limit=256"'),
            source.replace('"--name", name', '"--name", "--label", name'),
        )
        for changed in unsafe_changes:
            with self.subTest(changed=changed[-120:]), self.assertRaisesRegex(ValueError, "policy literals or dynamic marker"):
                probe.policy_flags(changed)
        with self.assertRaisesRegex(ValueError, "immutable sha256 digest"):
            probe.policy_flags(source.replace(image, "docker.io/library/golang:latest"))
        with self.assertRaisesRegex(ValueError, "immutable sha256 digest"):
            probe.policy_flags(source.replace(image, "docker.io/library/golang@sha256:abc"))

    def test_validates_memory_and_pid_event_evidence(self):
        memory = """child_exit=137
PROBE_EVENTS_BEGIN
memory_events:
low 0
high 0
max 1
oom 1
oom_kill 1
oom_group_kill 0
pids_events:
max 0
PROBE_EVENTS_END
"""
        pids = """spawn_error=fork/exec: resource temporarily unavailable
children_cleaned=126
eagain=true
child_exit=0
PROBE_EVENTS_BEGIN
memory_events:
low 0
max 0
oom_kill 0
pids_events:
max 1
PROBE_EVENTS_END
"""
        probe.validate_output("memory", memory, 0)
        probe.validate_output("pids", pids, 0)
        with self.assertRaisesRegex(ValueError, "oom_kill"):
            probe.validate_output("memory", memory.replace("oom_kill 1", "oom_kill 0"), 0)
        with self.assertRaisesRegex(ValueError, "pids.events max"):
            probe.validate_output("pids", pids.replace("max 1\nPROBE_EVENTS_END", "max 0\nPROBE_EVENTS_END"), 0)
        with self.assertRaisesRegex(ValueError, "EAGAIN"):
            probe.validate_output("pids", pids.replace("temporarily unavailable", "permission denied"), 0)
        with self.assertRaisesRegex(ValueError, "child_exit=0"):
            probe.validate_output("pids", pids.replace("child_exit=0\n", ""), 0)
        with self.assertRaisesRegex(ValueError, "child_exit=0"):
            probe.validate_output("pids", pids.replace("child_exit=0", "child_exit=137"), 0)

    def test_cleanup_is_exactly_scoped_to_generated_probe_name(self):
        name = "oga-m12-probe-" + "a" * 32
        self.assertEqual(probe.cleanup_argv(name), ["podman", "rm", "--force", "--time=0", "--ignore", name])
        with self.assertRaisesRegex(ValueError, "refusing cleanup"):
            probe.cleanup_argv("unrelated-container")

    def test_full_podman_argv_restores_name_scope_and_unique_labels(self):
        source = probe.RUNNER_SOURCE.read_text(encoding="utf-8")
        image, flags = probe.policy_flags(source)
        name = "oga-m12-probe-" + "b" * 32
        probe_id = "b" * 32
        argv = probe.container_argv(flags, image, name, probe_id, "echo probe")
        self.assertEqual(argv[0:2], ["podman", "run"])
        name_index = argv.index("--name")
        self.assertEqual(argv[name_index + 1], name)
        self.assertEqual(argv[argv.index("--name") + 2], "--label")
        self.assertEqual(argv[argv.index("--label") + 1], "academy.runner=true")
        self.assertIn("--label=academy.install=m12-probe", argv)
        self.assertIn(f"--label=academy.probe.id={probe_id}", argv)
        self.assertEqual(argv[argv.index(image) + 1:argv.index(image) + 4], ["sh", "-c", "echo probe"])
        self.assertIn("--network=none", argv)
        self.assertIn("--read-only", argv)
        self.assertFalse(any(x in ("-v", "--volume") or x.startswith("--volume=") for x in argv))
        self.assertFalse(any(x == "academy.install=" or x == "name" for x in argv))

    def test_build_uses_temporary_fixed_fixture_and_mocked_compiler(self):
        with tempfile.TemporaryDirectory() as td, patch.object(probe.subprocess, "run") as run:
            run.return_value = subprocess.CompletedProcess(["go"], 0, "", "")
            binary = probe._build("/fake/go", "pids", Path(td))
            self.assertEqual(binary, Path(td) / "pids")
            self.assertIn("syscall.EAGAIN", (Path(td) / "pids.go").read_text())
            argv = run.call_args.args[0]
            self.assertEqual(argv[:2], ["/fake/go", "build"])
            self.assertEqual(run.call_args.kwargs["env"]["GOARCH"], "amd64")

    def test_default_compiler_prefers_repository_toolchain_then_path(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            bundled = root / ".tools/go/bin/go"
            bundled.parent.mkdir(parents=True)
            bundled.touch()
            with patch.object(probe.subprocess, "run", return_value=subprocess.CompletedProcess([], 0, "go version go1.27.2 linux/amd64\n", "")) as run:
                self.assertEqual(probe.select_go_compiler("", root), str(bundled))
                run.assert_called_once_with([str(bundled), "version"], text=True, stdout=subprocess.PIPE,
                                            stderr=subprocess.STDOUT, timeout=15)
            bundled.unlink()
            with patch.object(probe.shutil, "which", return_value="/usr/bin/go"), patch.object(
                probe.subprocess, "run", return_value=subprocess.CompletedProcess([], 0, "go version go1.27.2 linux/amd64\n", "")
            ) as run:
                self.assertEqual(probe.select_go_compiler("", root), "/usr/bin/go")
                self.assertEqual(run.call_args.args[0], ["/usr/bin/go", "version"])

    def test_explicit_or_default_wrong_go_version_is_rejected_without_download(self):
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            bundled = root / ".tools/go/bin/go"
            bundled.parent.mkdir(parents=True)
            bundled.touch()
            with patch.object(probe.subprocess, "run", return_value=subprocess.CompletedProcess([], 0, "go version go1.26.0 linux/amd64\n", "")) as run:
                with self.assertRaisesRegex(RuntimeError, "exactly go1.27.2"):
                    probe.select_go_compiler("", root)
                self.assertEqual(run.call_args.args[0], [str(bundled), "version"])
                with self.assertRaisesRegex(RuntimeError, "exactly go1.27.2"):
                    probe.select_go_compiler("/custom/go", root)
                self.assertEqual(run.call_args.args[0], ["/custom/go", "version"])
                self.assertFalse(any("download" in str(c).lower() for c in run.call_args_list))

    def test_run_probe_uses_policy_no_mounts_and_exact_cleanup(self):
        fake_bin = Path(tempfile.gettempdir()) / "fake-oga-probe-bin"
        fake_bin.write_bytes(b"fixture")
        event_output = ("child_exit=137\nPROBE_EVENTS_BEGIN\nmemory_events:\nmax 1\noom_kill 1\n"
                        "pids_events:\nmax 0\nPROBE_EVENTS_END\n")
        pids_output = ("spawn_error=resource temporarily unavailable\nchildren_cleaned=126\neagain=true\nchild_exit=0\n"
                       "PROBE_EVENTS_BEGIN\nmemory_events:\nmax 0\noom_kill 0\npids_events:\nmax 1\nPROBE_EVENTS_END\n")
        runs = 0
        def fake_run(argv, **kwargs):
            nonlocal runs
            if argv[-1:] == ["version"]:
                return subprocess.CompletedProcess(argv, 0, "go version go1.27.2 linux/amd64\n", "")
            if argv[:2] == ["podman", "info"]:
                return subprocess.CompletedProcess(argv, 0, "true v2\n", "")
            if argv[:3] == ["podman", "image", "exists"]:
                return subprocess.CompletedProcess(argv, 0, "", "")
            if argv[:2] == ["podman", "rm"]:
                return subprocess.CompletedProcess(argv, 0, "", "")
            if argv[:2] == ["podman", "run"]:
                runs += 1
                self.assertFalse(any("volume" in x or x == "-v" for x in argv))
                self.assertIn("--network=none", argv)
                self.assertIn("--read-only", argv)
                return subprocess.CompletedProcess(argv, 0, (event_output if runs == 1 else pids_output).encode(), b"")
            self.fail(f"unexpected subprocess: {argv}")
        with patch.object(probe, "_build", return_value=fake_bin), patch.object(probe.subprocess, "run", side_effect=fake_run) as run:
            report = probe.run_probe("/fake/go", None)
        self.assertIn("not an end-to-end API benchmark", report)
        self.assertEqual(sum(1 for c in run.call_args_list if c.args[0][:2] == ["podman", "rm"]), 2)
        fake_bin.unlink()

    def test_timeout_runs_exact_cleanup_and_propagates(self):
        fake_bin = Path(tempfile.gettempdir()) / "fake-oga-probe-timeout-bin"
        fake_bin.write_bytes(b"fixture")
        calls = []
        def fake_run(argv, **kwargs):
            calls.append(argv)
            if argv[-1:] == ["version"]:
                return subprocess.CompletedProcess(argv, 0, "go version go1.27.2 linux/amd64\n", "")
            if argv[:2] == ["podman", "info"]:
                return subprocess.CompletedProcess(argv, 0, "true v2\n", "")
            if argv[:3] == ["podman", "image", "exists"]:
                return subprocess.CompletedProcess(argv, 0, "", "")
            if argv[:2] == ["podman", "run"]:
                raise subprocess.TimeoutExpired(argv, 1)
            if argv[:2] == ["podman", "rm"]:
                return subprocess.CompletedProcess(argv, 0, "", "")
            self.fail(f"unexpected subprocess: {argv}")
        with patch.object(probe.shutil, "which", return_value="podman"), patch.object(probe, "_build", return_value=fake_bin), patch.object(
            probe.subprocess, "run", side_effect=fake_run
        ):
            with self.assertRaises(subprocess.TimeoutExpired):
                probe.run_probe("/fake/go", None, timeout=1)
        cleanup = [x for x in calls if x[:2] == ["podman", "rm"]]
        self.assertEqual(len(cleanup), 1)
        self.assertEqual(cleanup[0][:5], ["podman", "rm", "--force", "--time=0", "--ignore"])
        self.assertRegex(cleanup[0][5], r"^oga-m12-probe-[0-9a-f]{32}$")
        run_argv = next(x for x in calls if x[:2] == ["podman", "run"])
        self.assertEqual(cleanup[0][5], run_argv[run_argv.index("--name") + 1])
        fake_bin.unlink()

    def test_cleanup_failure_fails_probe(self):
        fake_bin = Path(tempfile.gettempdir()) / "fake-oga-probe-cleanup-bin"
        fake_bin.write_bytes(b"fixture")
        output = ("child_exit=137\nPROBE_EVENTS_BEGIN\nmemory_events:\nmax 1\noom_kill 1\n"
                  "pids_events:\nmax 0\nPROBE_EVENTS_END\n")
        def fake_run(argv, **kwargs):
            if argv[-1:] == ["version"]:
                return subprocess.CompletedProcess(argv, 0, "go version go1.27.2 linux/amd64\n", "")
            if argv[:2] == ["podman", "info"]:
                return subprocess.CompletedProcess(argv, 0, "true v2\n", "")
            if argv[:3] == ["podman", "image", "exists"]:
                return subprocess.CompletedProcess(argv, 0, "", "")
            if argv[:2] == ["podman", "run"]:
                return subprocess.CompletedProcess(argv, 0, output.encode(), b"")
            if argv[:2] == ["podman", "rm"]:
                return subprocess.CompletedProcess(argv, 1, b"", b"cleanup denied")
            self.fail(f"unexpected subprocess: {argv}")
        with patch.object(probe.shutil, "which", return_value="podman"), patch.object(probe, "_build", return_value=fake_bin), patch.object(
            probe.subprocess, "run", side_effect=fake_run
        ):
            with self.assertRaisesRegex(RuntimeError, "scoped cleanup failed"):
                probe.run_probe("/fake/go", None)
        fake_bin.unlink()


if __name__ == "__main__":
    unittest.main()
