import importlib.util
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("m12_sandbox_access_probe.py")
spec = importlib.util.spec_from_file_location("m12_access_probe", SCRIPT)
probe = importlib.util.module_from_spec(spec)
spec.loader.exec_module(probe)


class AccessProbeTests(unittest.TestCase):
    def test_policy_is_literal_pinned_and_network_is_none(self):
        source = probe.RUNNER_SOURCE.read_text(encoding="utf-8")
        image, flags = probe.policy_flags(source)
        self.assertIn("@sha256:", image)
        self.assertIn("--network=none", flags)
        self.assertIn("--read-only", flags)
        for changed in (
            source.replace('"--network=none"', '"--network=host"'),
            source.replace('"--read-only"', '"--read-only", "--volume=/:/host"'),
            source.replace('"--memory=768m"', '"--memory=769m"'),
            source.replace(image, "docker.io/library/golang:latest"),
        ):
            with self.subTest(change=changed[-90:]), self.assertRaises(ValueError):
                probe.policy_flags(changed)

    def test_argv_has_no_mount_and_cleanup_is_scoped(self):
        source = probe.RUNNER_SOURCE.read_text(encoding="utf-8")
        image, flags = probe.policy_flags(source)
        suffix = "a" * 32
        argv = probe.container_argv(flags, image, "oga-m12-access-" + suffix, suffix, "echo probe")
        self.assertIn("--network=none", argv)
        self.assertFalse(any(x == "-v" or "volume" in x for x in argv))
        self.assertEqual(argv[argv.index("--name") + 1], "oga-m12-access-" + suffix)
        self.assertEqual(probe.cleanup_argv("oga-m12-access-" + suffix)[-1], "oga-m12-access-" + suffix)
        with self.assertRaises(ValueError):
            probe.cleanup_argv("other-container")
        with self.assertRaises(ValueError):
            probe.container_argv(flags, image, "unrelated", suffix, "true")

    def test_accepts_denials_with_udp_send_not_assumed_to_fail(self):
        output = """direct_sentinel_denied=true
sentinel_write_denied=true
proc_root_sentinel_denied=true
proc_root_write_denied=true
proc1_root_visible=true
proc1_pid_namespace_matches_self=true
host_only_env_in_container=false
host_only_env_in_proc1_environ=false
host_sentinel_in_mountinfo=false
issuer_fixture_denied=true
issuer_write_denied=true
issuer_fixture_in_mountinfo=false
docker_socket_unavailable=true
podman_socket_unavailable=true
mountinfo_lines=18
tcp_connect_denied=true
udp_write_unavailable=false
udp_write_attempted=true
unix_connect_denied=true
"""
        probe.validate_output(output, 0, (0, 0, 0))
        with self.assertRaisesRegex(ValueError, "direct_sentinel_denied"):
            probe.validate_output(output.replace("direct_sentinel_denied=true", "direct_sentinel_denied=false"), 0, (0, 0, 0))
        with self.assertRaisesRegex(ValueError, "listener evidence"):
            probe.validate_output(output, 0, (1, 0, 0))
        with self.assertRaisesRegex(ValueError, "exit/listener"):
            probe.validate_output(output, 137, (0, 0, 0))


if __name__ == "__main__":
    unittest.main()
