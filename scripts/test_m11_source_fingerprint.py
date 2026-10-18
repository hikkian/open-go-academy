import os
from pathlib import Path
import tempfile
import unittest

from m11_source_fingerprint import acceptance_matches, source_fingerprint


class SourceFingerprintTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        for name in ("go.mod", "go.sum", "scripts/go.sh", "scripts/m11_source_fingerprint.py", "scripts/check_m11_acceptance.py",
                     "internal/content/data/example.json", "internal/grading/spec.go",
                     "internal/runner/runner.go"):
            path = self.root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(name, encoding="utf-8")

    def test_checkout_timestamps_do_not_invalidate_unchanged_source(self):
        before = source_fingerprint(self.root)
        for path in self.root.rglob("*"):
            if path.is_file():
                os.utime(path, (1_900_000_000, 1_900_000_000))
        self.assertEqual(before, source_fingerprint(self.root))

    def test_grader_change_invalidates_content_evidence(self):
        before = source_fingerprint(self.root)
        (self.root / "internal/grading/spec.go").write_text("changed grader")
        self.assertNotEqual(before, source_fingerprint(self.root))

    def test_source_path_is_part_of_identity(self):
        before = source_fingerprint(self.root)
        source = self.root / "internal/content/data/example.json"
        source.rename(source.with_name("renamed.json"))
        self.assertNotEqual(before, source_fingerprint(self.root))

    def test_external_symlink_is_rejected(self):
        source = self.root / "internal/content/data/example.json"
        source.unlink()
        source.symlink_to(self.root / "go.mod")
        with self.assertRaises(ValueError):
            source_fingerprint(self.root)

    def test_acceptance_requires_exact_source_and_success(self):
        fingerprint = source_fingerprint(self.root)
        evidence = f"Result: PASS\nScope: all 88 advanced lessons\nSource fingerprint: {fingerprint}\nFinished source fingerprint: {fingerprint}\n"
        self.assertTrue(acceptance_matches(evidence, fingerprint))
        self.assertFalse(acceptance_matches("Result: PASS\nStarted epoch: 1\n", fingerprint))
        self.assertFalse(acceptance_matches(evidence.replace("PASS", "FAIL"), fingerprint))
        self.assertFalse(acceptance_matches(evidence.replace("Result:", "Historical result:"), fingerprint))
        self.assertFalse(acceptance_matches(evidence, "sha256:" + "0" * 64))
        self.assertFalse(acceptance_matches(evidence.replace("all 88 advanced lessons", "middle-26"), fingerprint))
        self.assertFalse(acceptance_matches(evidence.replace("Finished source fingerprint: " + fingerprint,
                                                         "Finished source fingerprint: sha256:" + "0" * 64), fingerprint))


if __name__ == "__main__":
    unittest.main()
