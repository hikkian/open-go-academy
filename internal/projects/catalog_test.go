package projects

import (
	"strings"
	"testing"
)

func TestCatalogAndP2Starter(t *testing.T) {
	all := All()
	if len(all) != 2 || all[0].ID != "p1" || all[1].ID != "p2" {
		t.Fatalf("catalog order: %#v", all)
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
