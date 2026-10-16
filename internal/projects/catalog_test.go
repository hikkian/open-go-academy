package projects

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogAndP2Starter(t *testing.T) {
	all := All()
	if len(all) != 2 || all[0].ID != "p1" || all[1].ID != "p2" {
		t.Fatalf("catalog order: %#v", all)
	}
	withTaskFlow := AllWithTaskFlow(true)
	if len(withTaskFlow) != 3 || withTaskFlow[2].ID != "p3" || len(AllWithTaskFlow(false)) != 2 {
		t.Fatalf("conditional P3 catalog: %#v", withTaskFlow)
	}
	p, ok := Find("p2")
	if !ok || p.Rank != "Junior+" || len(p.Milestones) != 4 || len(p.Requirements) < 5 {
		t.Fatalf("incomplete P2 metadata: %+v", p)
	}
	if err := ValidateFor("p2", p.StarterFiles); err != nil {
		t.Fatalf("starter rejected: %v", err)
	}
	if _, ok := p.StarterFiles["auth.go"]; ok {
		t.Fatal("reference business implementation leaked into starter")
	}
	mod, sum, ok := ModulePolicy("p2")
	if !ok || p.StarterFiles["go.mod"] != mod || p.StarterFiles["go.sum"] != sum {
		t.Fatal("starter module policy differs")
	}
	if _, _, ok := ModulePolicy("p1"); ok {
		t.Fatal("unexpected P1 module policy")
	}
	if _, ok := Find("missing"); ok {
		t.Fatal("found unknown project")
	}
}

func TestP3CatalogIsRunnableAndUsesReferenceLocks(t *testing.T) {
	p, ok := Find("p3")
	if !ok || p.Rank != "Middle" || p.Version != "p3-taskflow-v1" || len(p.Milestones) < 3 {
		t.Fatalf("incomplete P3 metadata: %+v", p)
	}
	criteriaHash, ok := CriteriaHash("p3")
	if !ok || len(criteriaHash) != len("sha256:")+64 {
		t.Fatalf("invalid versioned P3 criteria hash: %q", criteriaHash)
	}
	if repeated, ok := CriteriaHash("p3"); !ok || repeated != criteriaHash {
		t.Fatal("P3 criteria hash is not stable")
	}
	if err := ValidateFor("p3", p.StarterFiles); err != nil {
		t.Fatalf("P3 starter rejected: %v", err)
	}
	if _, ok = p.StarterFiles["store.go"]; ok {
		t.Fatal("P3 reference implementation leaked into starter")
	}
	mod, sum, ok := ModulePolicy("p3")
	if !ok || p.StarterFiles["go.mod"] != mod || p.StarterFiles["go.sum"] != sum {
		t.Fatal("P3 starter module lock differs from server policy")
	}
	for repo, expected := range map[string]string{"../../content/projects/p3/reference/go.mod": mod, "../../content/projects/p3/reference/go.sum": sum} {
		actual, err := os.ReadFile(repo)
		if err != nil {
			t.Fatal(err)
		}
		if string(actual) != expected {
			t.Fatalf("P3 policy differs from trusted reference asset: %s", repo)
		}
	}
	if _, _, ok := ModulePolicy("p1"); ok {
		t.Fatal("unexpected P1 module policy")
	}
}

func TestValidateP3FilesRejectsConfigAndLockSubstitution(t *testing.T) {
	base := loadP3().StarterFiles
	for _, name := range []string{"../escape.go", `a\\b.go`, "main.env", "go.sum.bak"} {
		f := cloneFiles(base)
		f[name] = "untrusted"
		if err := ValidateFor("p3", f); err == nil {
			t.Errorf("accepted unsafe/configuration file %q", name)
		}
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		f := cloneFiles(base)
		f[name] += "\nreplace example.invalid/x => /host\n"
		if ValidateFor("p3", f) == nil {
			t.Errorf("accepted mutated %s", name)
		}
	}
	if err := ValidateFor("unknown", base); err == nil {
		t.Fatal("accepted unknown project")
	}
}

func TestValidateP2Files(t *testing.T) {
	base := LoadP2ForTest()
	for _, name := range []string{"internal/handler.go", "migrations/0002_tags.up.sql", "docs/notes.md", "api/openapi.yaml"} {
		f := cloneFiles(base)
		f[name] = "content"
		if err := ValidateFor("p2", f); err != nil {
			t.Errorf("valid nested path %q: %v", name, err)
		}
	}
	for _, name := range []string{"../escape.go", "a/../b.go", `a\b.go`, "a//b.go", "a/./b.go", "a/b/c/d.go", "main.exe", "bad\nname.go", "é.go"} {
		f := cloneFiles(base)
		f[name] = "x"
		if err := ValidateFor("p2", f); err == nil {
			t.Errorf("accepted unsafe path %q", name)
		}
	}
	for _, tc := range []struct {
		name   string
		mutate func(map[string]string)
	}{
		{"wrong module", func(f map[string]string) { f["go.mod"] += "\nreplace example.com/x => ../x\n" }},
		{"wrong sum", func(f map[string]string) { f["go.sum"] += "example.com/x v1.0.0 h1:abc=\n" }},
		{"missing lock", func(f map[string]string) { delete(f, "go.sum") }},
		{"missing source", func(f map[string]string) { delete(f, "main.go") }},
		{"empty readme", func(f map[string]string) { f["README.md"] = "  " }},
	} {
		f := cloneFiles(base)
		tc.mutate(f)
		if err := ValidateFor("p2", f); err == nil {
			t.Errorf("accepted %s", tc.name)
		}
	}
	tooMany := cloneFiles(base)
	for i := 0; i < P2MaxFiles; i++ {
		tooMany["extra"+strings.Repeat("x", i)+".md"] = "x"
	}
	if ValidateFor("p2", tooMany) == nil {
		t.Fatal("accepted file quota overflow")
	}
	largeGo := cloneFiles(base)
	largeGo["extra.go"] = strings.Repeat("x", P2MaxGoBytes+1)
	if ValidateFor("p2", largeGo) == nil {
		t.Fatal("accepted Go source quota overflow")
	}
	largeFile := cloneFiles(base)
	largeFile["large.md"] = strings.Repeat("x", P2MaxFileBytes+1)
	if ValidateFor("p2", largeFile) == nil {
		t.Fatal("accepted per-file overflow")
	}
	largeTotal := cloneFiles(base)
	largeTotal["large.md"] = strings.Repeat("x", P2MaxBytes)
	if ValidateFor("p2", largeTotal) == nil {
		t.Fatal("accepted total quota overflow")
	}
	if ValidateFor("unknown", base) == nil {
		t.Fatal("accepted unknown project")
	}
	if err := ValidateFor("p1", Load().StarterFiles); err != nil {
		t.Fatalf("P1 validation changed: %v", err)
	}
}

func LoadP2ForTest() map[string]string { return loadP2().StarterFiles }

func TestP4P5CatalogBundlesAndPinnedModules(t *testing.T) {
	for _, tc := range []struct {
		id, rank, module, source string
	}{
		{"p4", "Middle+", p4Module, "content/projects/p4/starter/aggregate.go"},
		{"p5", "Senior", p5Module, "content/projects/p5/starter/queue.go"},
	} {
		project, ok := Find(tc.id)
		if !ok || project.ID != tc.id || project.Rank != tc.rank || project.Version == "" || len(project.StarterFiles) == 0 {
			t.Fatalf("project %s missing/incomplete: %+v, ok=%v", tc.id, project, ok)
		}
		mod, sum, ok := ModulePolicy(tc.id)
		if !ok || mod != tc.module || sum != "" {
			t.Fatalf("%s module policy=(%q,%q,%v), want locked stdlib module", tc.id, mod, sum, ok)
		}
		moduleSource, err := os.ReadFile("../../content/projects/" + tc.id + "/starter/go.mod")
		if err != nil || string(moduleSource) != mod {
			t.Fatalf("%s embedded lock differs from authored starter module: %v", tc.id, err)
		}
		source, err := os.ReadFile("../../" + tc.source)
		if err != nil || project.StarterFiles[filepath.Base(tc.source)] != string(source) {
			t.Fatalf("%s embedded starter differs from authored source %s: %v", tc.id, tc.source, err)
		}
		if err := ValidateFor(tc.id, cloneFiles(project.StarterFiles)); err != nil {
			t.Fatalf("%s bundled starter invalid: %v", tc.id, err)
		}
		if _, ok := CriteriaHash(tc.id); !ok {
			t.Fatalf("%s criteria hash unavailable", tc.id)
		}
		assertProjectTreeMatches(t, filepath.Join("../../content/projects", tc.id, "starter"), project.StarterFiles)
		mutated := cloneFiles(project.StarterFiles)
		mutated["go.mod"] += "\nreplace example.invalid/x => /host\n"
		if err := ValidateFor(tc.id, mutated); err == nil {
			t.Fatalf("%s accepted modified module lock", tc.id)
		}
	}
	for projectID, checks := range map[string]map[string]string{
		"p4": {"docs/project.md": "README.md", "docs/rubric.md": "rubric.md"},
		"p5": {"docs/project.md": "README.md", "docs/rubric.md": "rubric.md", "docs/threat-model.md": "threat-model.md"},
	} {
		project, _ := Find(projectID)
		for starterPath, sourcePath := range checks {
			source, err := os.ReadFile(filepath.Join("../../content/projects", projectID, sourcePath))
			if err != nil || project.StarterFiles[starterPath] != string(source) {
				t.Fatalf("%s embedded %s differs from authored %s: %v", projectID, starterPath, sourcePath, err)
			}
		}
	}
	if got := AllWithProfiles(false, false, false); len(got) != 2 {
		t.Fatalf("projects without optional runner profiles=%v", projectIDs(got))
	}
	if got := AllWithProfiles(false, true, true); len(got) != 4 {
		t.Fatalf("projects with verified P4/P5 profiles=%v", projectIDs(got))
	}
}

func assertProjectTreeMatches(t *testing.T, root string, bundled map[string]string) {
	t.Helper()
	err := filepath.WalkDir(root, func(name string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(relative)
		contents, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		if bundled[key] != string(contents) {
			return fmt.Errorf("embedded starter mismatch for %s", key)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func projectIDs(projects []Project) []string {
	ids := make([]string, len(projects))
	for i, project := range projects {
		ids[i] = project.ID
	}
	return ids
}
