package content

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCatalogAndReferenceSolutions(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	lesson, ok := c.FindLesson("zero-01")
	if !ok {
		t.Fatal("lesson missing")
	}
	if lesson.Title != "Что такое компьютер, программа и Go" || len(lesson.Prerequisites) != 0 || len(lesson.Exercises) != 3 {
		t.Fatalf("unexpected lesson metadata: %+v", lesson)
	}
	for _, id := range []string{"hello", "repair", "two-lines"} {
		if _, ok := c.FindExercise(id); !ok {
			t.Fatalf("exercise %s absent", id)
		}
	}
	goBin, err := filepath.Abs(filepath.Join("..", "..", ".tools", "go", "bin", "go"))
	if _, err := os.Stat(goBin); err != nil {
		goBin, err = exec.LookPath("go")
		if err != nil {
			t.Fatalf("Go toolchain required: %v", err)
		}
	}
	for _, tc := range []struct{ id, want string }{{"hello", "Go готов!\n"}, {"repair", "42\n"}, {"two-lines", "Имя: Ada\nЯзык: Go\n"}} {
		dir := filepath.Join("..", "..", "content", "reference", tc.id)
		src, err := os.ReadFile(filepath.Join(dir, "main.go"))
		if err != nil {
			t.Fatal(err)
		}
		private, ok := c.Reference(tc.id)
		if !ok || private != string(src) {
			t.Fatalf("private reference %s is missing or differs from its source", tc.id)
		}
		tmp := t.TempDir()
		if err = os.WriteFile(filepath.Join(tmp, "main.go"), src, 0600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(goBin, "run", "main.go")
		cmd.Dir = tmp
		cmd.Env = append(os.Environ(), "GOTOOLCHAIN=go1.27.2")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("reference %s failed: %v\n%s", tc.id, err, out)
		}
		if string(out) != tc.want {
			t.Errorf("reference %s output %q want %q", tc.id, out, tc.want)
		}
	}
	if strings.TrimSpace(lesson.Body) == "" {
		t.Fatal("missing body")
	}
}

func TestAllGoCodeBlocksRunAndSelfCheckIsAvailable(t *testing.T) {
	catalog, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	lesson, ok := catalog.FindLesson("zero-01")
	if !ok {
		t.Fatal("lesson missing")
	}
	if len(lesson.SelfCheck) != 5 {
		t.Fatalf("got %d lesson self-checks, want 5", len(lesson.SelfCheck))
	}
	for _, check := range lesson.SelfCheck {
		if check.Question == "" || check.Answer == "" {
			t.Fatalf("incomplete lesson self-check: %+v", check)
		}
	}
	goBin, err := filepath.Abs(filepath.Join("..", "..", ".tools", "go", "bin", "go"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(goBin); err != nil {
		goBin, err = exec.LookPath("go")
		if err != nil {
			t.Fatalf("Go toolchain required: %v", err)
		}
	}
	blocks := goCodeBlocks(lesson.Body)
	if len(blocks) != 2 {
		t.Fatalf("got %d Go blocks, want 2", len(blocks))
	}
	for i, source := range blocks {
		tmp := t.TempDir()
		if err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(goBin, "run", "main.go")
		cmd.Dir = tmp
		cmd.Env = append(os.Environ(), "GOTOOLCHAIN=go1.27.2")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("lesson Go block %d failed: %v\n%s", i+1, err, out)
		}
	}
}

func goCodeBlocks(markdown string) []string {
	var blocks []string
	var current strings.Builder
	inGo := false
	for _, line := range strings.Split(markdown, "\n") {
		if !inGo {
			if strings.TrimSpace(line) == "```go" {
				inGo = true
				current.Reset()
			}
			continue
		}
		if strings.TrimSpace(line) == "```" {
			blocks = append(blocks, current.String())
			inGo = false
			continue
		}
		current.WriteString(line)
		current.WriteByte('\n')
	}
	return blocks
}

func TestInternalMarkdownLinksRejectMissingTargets(t *testing.T) {
	catalog, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	lesson, ok := catalog.FindLesson("zero-01")
	if !ok {
		t.Fatal("lesson missing")
	}
	lesson.Body += "\n[missing](exercise:no-such-exercise)\n"
	if err := catalog.validateMarkdownLinks(lesson); err == nil {
		t.Fatal("expected missing internal exercise link to fail validation")
	}
	lesson.Body = "[missing](#no-such-heading)"
	if err := catalog.validateMarkdownLinks(lesson); err == nil {
		t.Fatal("expected missing internal heading link to fail validation")
	}
}

func TestGenericCodeCallsAreNotMarkdownNavigation(t *testing.T) {
	c := &Catalog{}
	lesson := Lesson{ID: "example", Body: "```go\npackage main\nfunc main(){ _ = Sum[int](values) }\n```\nUse `Apply[T](values)` in code.\n"}
	if err := c.validateMarkdownLinks(lesson); err != nil {
		t.Fatal(err)
	}
	lesson.Body += "\n[broken](lesson:not-published)\n"
	if err := c.validateMarkdownLinks(lesson); err == nil {
		t.Fatal("real missing link was ignored")
	}
}
