package projects

import (
	"strings"
	"testing"
)

func TestLoadP1(t *testing.T) {
	p := Load()
	if p.ID != "p1" || p.Title != "Go CLI Toolkit" || p.Rank != "Junior" {
		t.Fatalf("unexpected project metadata: %#v", p)
	}
	if len(p.Requirements) < 5 || len(p.Milestones) != 3 || len(p.PublicExamples) != 2 {
		t.Fatalf("incomplete project content: %#v", p)
	}
	if len(p.StarterFiles) != 4 {
		t.Fatalf("got starter files %v", keys(p.StarterFiles))
	}
	if _, ok := p.StarterFiles["toolkit.go"]; !ok {
		t.Fatal("starter toolkit.go missing")
	}
	if err := ValidateFiles(p.StarterFiles); err != nil {
		t.Fatalf("embedded starter violates project file policy: %v", err)
	}
}

func TestValidateFiles(t *testing.T) {
	valid := map[string]string{"main.go": "package main", "go.mod": "module cli-toolkit\n\ngo 1.27.2\n", "README.md": "# sample", "fixture.json": "[]"}
	if err := ValidateFiles(valid); err != nil {
		t.Fatalf("valid files rejected: %v", err)
	}
	for _, goMod := range []string{"\tmodule  cli-toolkit\r\n\r\n  go\t1.27.2 \r\n"} {
		variant := cloneFiles(valid)
		variant["go.mod"] = goMod
		if err := ValidateFiles(variant); err != nil {
			t.Errorf("accepted go.mod whitespace variant as invalid: %v", err)
		}
	}
	for _, name := range []string{"../escape.go", "/absolute.go", "folder/main.go", `folder\main.go`, "script.sh", ".", "1main.go", ".hidden.go", "odd name.go", "bad\x00name.go", "bad\nname.json"} {
		if err := ValidateFiles(map[string]string{name: "x"}); err == nil {
			t.Errorf("accepted unsafe file %q", name)
		}
	}
	tooMany := map[string]string{}
	for i := 0; i < 17; i++ {
		tooMany[itoa(i)+".go"] = "x"
	}
	if err := ValidateFiles(tooMany); err == nil {
		t.Fatal("accepted more than 16 files")
	}
	if err := ValidateFiles(map[string]string{"main.go": strings.Repeat("x", 128*1024+1)}); err == nil {
		t.Fatal("accepted more than 128 KiB")
	}
	base := map[string]string{"main.go": "package main", "go.mod": requiredGoMod, "README.md": "# sample"}
	tooMuchGo := cloneFiles(base)
	tooMuchGo["extra.go"] = strings.Repeat("x", 64*1024)
	if err := ValidateFiles(tooMuchGo); err == nil {
		t.Fatal("accepted Go sources over 64 KiB")
	}
	tooMuchJSON := cloneFiles(base)
	tooMuchJSON["data.json"] = strings.Repeat("x", 64*1024+1)
	if err := ValidateFiles(tooMuchJSON); err == nil {
		t.Fatal("accepted JSON over 64 KiB")
	}
	tooManyJSON := cloneFiles(base)
	for i := 0; i < 9; i++ {
		tooManyJSON[string(rune('a'+i))+".json"] = "[]"
	}
	if err := ValidateFiles(tooManyJSON); err == nil {
		t.Fatal("accepted more than 8 JSON files")
	}
	for _, tc := range []struct {
		name  string
		files map[string]string
	}{
		{"missing README", map[string]string{"main.go": "package main", "go.mod": requiredGoMod}},
		{"empty README", map[string]string{"main.go": "package main", "go.mod": requiredGoMod, "README.md": " \n\t"}},
		{"missing Go source", map[string]string{"main_test.go": "package main", "go.mod": requiredGoMod, "README.md": "# sample"}},
		{"wrong module", map[string]string{"main.go": "package main", "go.mod": "module attacker\n\ngo 1.27.2\n", "README.md": "# sample"}},
		{"single-line go.mod", map[string]string{"main.go": "package main", "go.mod": "module cli-toolkit go 1.27.2", "README.md": "# sample"}},
		{"extra module directive", map[string]string{"main.go": "package main", "go.mod": "module cli-toolkit\n\ngo 1.27.2\n\nrequire example.com/x v1.0.0\n", "README.md": "# sample"}},
	} {
		if err := ValidateFiles(tc.files); err == nil {
			t.Errorf("accepted %s", tc.name)
		}
	}
	if err := ValidateFiles(nil); err == nil {
		t.Fatal("accepted an empty project")
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
func cloneFiles(files map[string]string) map[string]string {
	copy := make(map[string]string, len(files))
	for name, contents := range files {
		copy[name] = contents
	}
	return copy
}
func itoa(n int) string { return string(rune('a' + n)) }

func TestModuleSyntaxRejectsNonGoWhitespace(t *testing.T) {
	for _, space := range []string{"\v", "\f", "\u00a0", "\r"} {
		p := Load()
		p.StarterFiles["go.mod"] = "module" + space + "cli-toolkit\n\ngo 1.27.2\n"
		if ValidateFiles(p.StarterFiles) == nil {
			t.Fatalf("accepted invalid module whitespace %q", space)
		}
	}
}
