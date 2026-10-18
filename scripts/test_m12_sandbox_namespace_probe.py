import importlib.util
import subprocess
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("m12_sandbox_namespace_probe.py")
spec = importlib.util.spec_from_file_location("m12_namespace_probe", SCRIPT)
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)


class NamespaceProbeTests(unittest.TestCase):
    def test_reads_exact_pinned_runner_policy_and_network_none(self):
        source = probe.RUNNER_SOURCE.read_text(encoding="utf-8")
        image, flags = probe.ACCESS.policy_flags(source)
        self.assertEqual(image, probe.IMAGE)
        self.assertIn("--network=none", flags)
        self.assertIn("--read-only", flags)
        with self.assertRaises(ValueError):
            probe.ACCESS.policy_flags(source.replace('"--network=none"', '"--network=host"'))

    def test_observation_parser_and_mount_route_cgroup_validation(self):
        encoded = (
            '{"pid_self_ns":"pid:[200]","pid1_ns":"inaccessible",'
            '"mount_self_ns":"mnt:[200]","mount_pid1_ns":"mnt:[200]",'
            '"mount_entries":["abc"],"routes4":["lo 0000007F 00000000 00000003"],'
            '"routes6":[],"resolv_conf_present":true,"cgroup_events":"populated 1\\nfrozen 0",'
            '"cgroup_events_error":false}'
        )
        value = probe.parse_observation("prefix\n" + probe.OBSERVATION_MARKER + encoded + "\n")
        observer = {"pid_ns": "pid:[100]", "mount_ns": "mnt:[100]", "mount_entries": ["xyz"]}
        summary = probe.validate_observation(value, observer)
        self.assertTrue(summary["proc1_pid_ns_inaccessible"])
        self.assertTrue(summary["non_loopback_routes"] is False)
        self.assertEqual(summary["cgroup_events_live"], "populated 1\nfrozen 0")
        value["routes4"] = ["eth0 00000000 0100000A 00000003"]
        with self.assertRaisesRegex(ValueError, "non-loopback route"):
            probe.validate_observation(value, observer)
        with self.assertRaisesRegex(ValueError, "missing its observation marker"):
            probe.parse_observation("no marker")

    def test_generated_name_and_cleanup_refuse_other_containers(self):
        probe_id = "a" * 32
        name = "oga-m12-access-" + probe_id
        self.assertEqual(probe.cleanup_argv(name)[-1], name)
        with self.assertRaisesRegex(ValueError, "non-generated"):
            probe.cleanup_argv("unrelated-container")

        calls = []

        def absent_runtime(argv, **kwargs):
            calls.append(argv)
            if argv[1] == "ps":
                return subprocess.CompletedProcess(argv, 0, stdout="", stderr="")
            if argv[1:3] == ["container", "exists"]:
                return subprocess.CompletedProcess(argv, 1, stdout="", stderr="")
            raise AssertionError(f"unexpected cleanup command: {argv}")

        outcome = probe.cleanup_exact(name, probe_id, absent_runtime)
        self.assertIn("already removed by --rm", outcome)
        self.assertEqual(calls[0][calls[0].index("--filter") + 1], "label=academy.install=m12-access-probe")
        self.assertIn(f"label=academy.probe.id={probe_id}", calls[0])
        self.assertEqual(calls[1][-1], name)

        def foreign_label_match(argv, **kwargs):
            calls.append(argv)
            return subprocess.CompletedProcess(argv, 0, stdout="other-container\n", stderr="")

        before = len(calls)
        with self.assertRaisesRegex(RuntimeError, "matched another container"):
            probe.cleanup_exact(name, probe_id, foreign_label_match)
        self.assertEqual(len(calls), before + 1)

    def test_argv_has_no_host_mounts_and_fixed_probe_scope(self):
        source = probe.RUNNER_SOURCE.read_text(encoding="utf-8")
        image, flags = probe.ACCESS.policy_flags(source)
        probe_id = "b" * 32
        name = "oga-m12-access-" + probe_id
        argv = probe.container_argv(flags, image, name, probe_id, "sleep 1")
        self.assertIn("--network=none", argv)
        self.assertIn("--label=academy.install=m12-access-probe", argv)
        self.assertIn(f"--label=academy.probe.id={probe_id}", argv)
        self.assertFalse(any(arg in ("-v", "--volume", "--mount") for arg in argv))
        with self.assertRaisesRegex(ValueError, "non-generated"):
            probe.container_argv(flags, image, "bad-name", probe_id, "true")


if __name__ == "__main__":
    unittest.main()
