import base64
import hashlib
import json
import tarfile
import tempfile
import unittest
from io import BytesIO
from pathlib import Path

import m13_license_inventory as inventory


class LicenseInventoryTests(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()
        self.root = Path(self.tmp.name) / "repo"
        self.cache = Path(self.tmp.name) / "gomod"
        self.npm = Path(self.tmp.name) / "npm"
        self.root.mkdir()
        self.cache.mkdir()
        self.npm.mkdir()

    def tearDown(self):
        self.tmp.cleanup()

    def test_go_root_and_nested_module_graph_resolves_only_local_cache(self):
        (self.root / "go.mod").write_text("module example/root\n\ngo 1.24\nrequire (\n example/dep v1.2.3\n)\n", encoding="utf-8")
        nested = self.root / "content/project/go.mod"
        nested.parent.mkdir(parents=True)
        nested.write_text("module example/nested\n\nrequire example/dep v1.2.3\n", encoding="utf-8")
        dep = self.cache / "example/dep@v1.2.3"
        dep.mkdir(parents=True)
        (dep / "go.mod").write_text("module example/dep\n\nrequire example/transitive v0.2.0\n", encoding="utf-8")
        (dep / "LICENSE").write_text("sample license\n", encoding="utf-8")
        transitive = self.cache / "example/transitive@v0.2.0"
        transitive.mkdir(parents=True)
        (transitive / "go.mod").write_text("module example/transitive\n", encoding="utf-8")
        result = inventory.make_inventory(self.root, self.cache, self.npm, [])
        deps = [r for r in result["records"] if r["ecosystem"] == "go" and r["version"] != "local"]
        self.assertEqual([(r["name"], r["version"]) for r in deps], [
            ("example/dep", "v1.2.3"), ("example/transitive", "v0.2.0")])
        self.assertEqual(deps[0]["declared_license"], "UNKNOWN")
        self.assertEqual(deps[0]["license_files"][0]["sha256"], hashlib.sha256(b"sample license\n").hexdigest())
        self.assertEqual(deps[1]["cache"], "extracted-cache")

    def test_npm_cache_tarball_supplies_metadata_and_exact_license_digest(self):
        license_bytes = b"fixture license text"
        archive_bytes = BytesIO()
        with tarfile.open(fileobj=archive_bytes, mode="w:gz") as tar:
            for name, data in (("package/package.json", b'{"name":"pkg","version":"1.0.0","license":"MIT"}'),
                               ("package/LICENSE", license_bytes)):
                info = tarfile.TarInfo(name)
                info.size = len(data)
                tar.addfile(info, BytesIO(data))
        blob = archive_bytes.getvalue()
        b64 = base64.b64encode(hashlib.sha512(blob).digest()).decode()
        digest = hashlib.sha512(blob).hexdigest()
        blob_path = self.npm / "_cacache/content-v2/sha512" / digest[:2] / digest[2:4] / digest[4:]
        blob_path.parent.mkdir(parents=True)
        blob_path.write_bytes(blob)
        lock = self.root / "package-lock.json"
        lock.write_text(json.dumps({"packages": {"node_modules/pkg": {
            "version": "1.0.0", "integrity": "sha512-" + b64}}}), encoding="utf-8")
        result = inventory.make_inventory(self.root, self.cache, self.npm, [lock])
        row = next(r for r in result["records"] if r["ecosystem"] == "npm")
        self.assertEqual(row["declared_license"], "MIT")
        self.assertEqual(row["license_files"][0]["sha256"], hashlib.sha256(license_bytes).hexdigest())
        self.assertTrue(row["license_files"][0]["path"].endswith("!/package/LICENSE"))

    def test_missing_or_ambiguous_metadata_stays_unknown_or_ambiguous(self):
        (self.root / "go.mod").write_text("module example/root\nrequire example/missing v1.0.0\n", encoding="utf-8")
        ambiguous = self.cache / "example/ambiguous@v1.0.0"
        ambiguous.mkdir(parents=True)
        (ambiguous / "package.json").write_text(json.dumps({"licenses": [{"type": "MIT"}, {"type": "Apache-2.0"}]}), encoding="utf-8")
        lock = self.root / "package-lock.json"
        lock.write_text(json.dumps({"packages": {
            "node_modules/missing": {"version": "2.0.0"},
            "node_modules/ambiguous": {"version": "3.0.0", "license": ""}}}), encoding="utf-8")
        records = inventory.make_inventory(self.root, self.cache, self.npm, [lock])["records"]
        self.assertIn("UNKNOWN", [r["declared_license"] for r in records])
        self.assertEqual(inventory.package_license(ambiguous), "AMBIGUOUS: Apache-2.0, MIT")
        self.assertTrue(all(r["license_files"] == [] for r in records if r["cache"] == "not-cached"))

    def test_output_order_and_rendering_are_deterministic(self):
        (self.root / "go.mod").write_text("module example/root\nrequire (\nz/z v1.0.0\na/a v1.0.0\n)\n", encoding="utf-8")
        first = inventory.make_inventory(self.root, self.cache, self.npm, [])
        second = inventory.make_inventory(self.root, self.cache, self.npm, [])
        self.assertEqual(json.dumps(first, sort_keys=True), json.dumps(second, sort_keys=True))
        self.assertEqual([r["name"] for r in first["records"]], ["a/a", "example/root", "z/z"])

    def test_quoted_go_module_paths_are_unquoted(self):
        manifest = self.root / "go.mod"
        manifest.write_text(
            'module "example.com/course root"\n'
            'require (\n "example.com/quoted dep" v1.2.3 // inline comment\n)\n',
            encoding="utf-8",
        )
        module, requires = inventory.parse_go_mod(manifest)
        self.assertEqual(module, "example.com/course root")
        self.assertEqual(requires, [("example.com/quoted dep", "v1.2.3")])

    def test_ignored_local_state_does_not_change_source_manifest_inventory(self):
        (self.root / "go.mod").write_text("module example/root\n", encoding="utf-8")
        before = inventory.make_inventory(self.root, self.cache, self.npm, [])

        generated = self.root / ".data/fresh-source"
        generated.mkdir(parents=True)
        (generated / "go.mod").write_text("module example/ignored\n", encoding="utf-8")
        (generated / "LICENSE").write_text("local generated data\n", encoding="utf-8")
        # Build output roots are excluded as well, including in source archives
        # where no Git index exists to identify tracked paths.
        artifact = self.root / "artifacts/fresh-source"
        artifact.mkdir(parents=True)
        (artifact / "go.mod").write_text("module example/build-output\n", encoding="utf-8")

        after = inventory.make_inventory(self.root, self.cache, self.npm, [])
        self.assertEqual(after, before)

    def test_report_paths_hide_absolute_cache_roots_and_match_across_roots(self):
        def build(base):
            repo = base / "repo"
            cache = base / "gomod"
            npm = base / "npm"
            repo.mkdir(parents=True)
            cache.mkdir()
            npm.mkdir()
            (repo / "go.mod").write_text("module example/root\nrequire example/dep v1.0.0\n", encoding="utf-8")
            dep = cache / "example/dep@v1.0.0"
            dep.mkdir(parents=True)
            (dep / "go.mod").write_text("module example/dep\n", encoding="utf-8")
            (dep / "LICENSE").write_text("stable license", encoding="utf-8")
            return inventory.make_inventory(repo, cache, npm, [])

        with tempfile.TemporaryDirectory() as first_dir, tempfile.TemporaryDirectory() as second_dir:
            first = build(Path(first_dir) / "root-a")
            second = build(Path(second_dir) / "root-b")
            first_json = json.dumps(first, sort_keys=True)
            second_json = json.dumps(second, sort_keys=True)
            self.assertEqual(first_json, second_json)
            self.assertNotIn(first_dir, first_json)
            self.assertNotIn(second_dir, first_json)
            dep_row = next(r for r in first["records"] if r["name"] == "example/dep")
            self.assertEqual(dep_row["license_files"][0]["path"], "LICENSE")


if __name__ == "__main__":
    unittest.main()
