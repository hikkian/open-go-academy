package assessment

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPublicExamDoesNotSerializePrivateGradingMaterial(t *testing.T) {
	e := Load()
	if e.ID != "zero-core" || e.Version != "1.0.0" || len(e.Questions) != 12 || len(e.Tasks) != 2 {
		t.Fatalf("unexpected exam definition: id=%q version=%q questions=%d tasks=%d", e.ID, e.Version, len(e.Questions), len(e.Tasks))
	}
	if strings.Join(e.Prerequisites, ",") != "hello,repair,two-lines" || !strings.Contains(e.Criteria[0], "все 12") {
		t.Fatalf("unexpected eligibility or pass threshold: prerequisites=%v criteria=%v", e.Prerequisites, e.Criteria)
	}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	serialized := string(b)
	for _, secret := range []string{"correctIDs", "cases", "drivers", `"Expected"`, `"Args"`} {
		if strings.Contains(serialized, secret) {
			t.Errorf("public DTO contains private grading data %q", secret)
		}
	}
	for _, task := range e.Tasks {
		for _, example := range task.PublicExamples {
			if !strings.Contains(serialized, `"input":"`+example.Input+`"`) {
				t.Errorf("task %s public example missing", task.ID)
			}
		}
	}
	answers := make(map[string]string, len(e.correctIDs))
	positions := map[string]bool{}
	for id, answer := range e.correctIDs {
		answers[id] = answer
		positions[answer] = true
	}
	if len(positions) < 3 {
		t.Fatalf("correct option IDs should vary with displayed order, got %v", positions)
	}
	checks := e.QuizChecks(answers)
	if len(checks) != 12 {
		t.Fatalf("got %d quiz checks, want 12", len(checks))
	}
	for _, check := range checks {
		if !check.Passed {
			t.Errorf("correct response for %s did not pass", check.ID)
		}
	}
	delete(answers, "q-byte")
	checks = e.QuizChecks(answers)
	if checks[10].Passed {
		t.Fatal("missing answer passed the perfect-score objective exam")
	}
	answers["q-byte"] = "wrong"
	checks = e.QuizChecks(answers)
	if checks[10].Passed {
		t.Fatal("incorrect answer passed the perfect-score objective exam")
	}
}

func TestValidateTaskSourceAllowsHelpersAndRejectsStructuralEscapes(t *testing.T) {
	e := Load()
	good := map[string]string{
		"coding":    "package main\nfunc helper(x int) int { return x }\nfunc sumEven(limit int) int { return helper(limit) }",
		"debugging": "package main\nfunc clampPercent(value int) int { return value }",
	}
	for task, source := range good {
		if err := e.ValidateTaskSource(task, source); err != nil {
			t.Errorf("valid %s source rejected: %v", task, err)
		}
	}
	bad := []struct{ task, source string }{
		{"coding", "package other\nfunc sumEven(limit int) int{return limit}"},
		{"coding", "package main\nfunc sumEven(limit string) int{return 0}"},
		{"coding", "package main\nfunc sumEven(limit int) int{return 0}\nfunc main(){}"},
		{"debugging", "package main\nfunc clampPercent(value int) int{return value}\nfunc TestEscape(){}"},
		{"debugging", "package main\nfunc helper(value int) int{return value}"},
		{"coding", "package main\nimport \"os\"\nfunc sumEven(limit int) int{return 0}"},
		{"coding", "package main\nfunc init(){ os.Exit(0) }\nfunc sumEven(limit int) int{return 0}"},
		{"coding", "package main\nvar leak = func() int { panic(\"ran at init\") }()\nfunc sumEven(limit int) int{return 0}"},
		{"coding", "package main\ntype Number = int\nfunc sumEven(limit int) int{return 0}"},
		{"coding", "package main\nfunc main() {}\nfunc sumEven(limit int) int{return 0}"},
		{"unknown", "package main"},
	}
	for _, item := range bad {
		if err := e.ValidateTaskSource(item.task, item.source); err == nil {
			t.Errorf("accepted invalid source: %s", item.source)
		}
	}
	maxSource := 64*1024 - len(e.Driver("coding"))
	if err := e.ValidateTaskSource("coding", "package main\nfunc sumEven(limit int) int{return 0}"+strings.Repeat(" ", maxSource)); err == nil {
		t.Fatal("accepted source+driver over 64 KiB")
	}
}

func TestTrustedReferencesRunAgainstHiddenVectors(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repo := filepath.Clean(filepath.Join(filepath.Dir(here), "..", ".."))
	goBin := "go"
	if candidate := filepath.Join(repo, ".tools", "go", "bin", "go"); fileExists(candidate) {
		goBin = candidate
	}
	for _, task := range []string{"coding", "debugging"} {
		t.Run(task, func(t *testing.T) {
			dir := t.TempDir()
			ref := filepath.Join(repo, "content", "exams", "zero-core", "reference", task, "main.go")
			source, err := os.ReadFile(ref)
			if err != nil {
				t.Fatal(err)
			}
			if err := Load().ValidateTaskSource(task, string(source)); err != nil {
				t.Fatalf("trusted reference rejected by source policy: %v", err)
			}
			if err := os.WriteFile(filepath.Join(dir, "solution.go"), source, 0600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "driver.go"), []byte(Load().Driver(task)), 0600); err != nil {
				t.Fatal(err)
			}
			for _, tc := range Load().TaskCases(task) {
				args := append([]string{"run", "solution.go", "driver.go"}, tc.Args...)
				cmd := exec.Command(goBin, args...)
				cmd.Dir = dir
				cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
				out, err := cmd.Output()
				if err != nil {
					t.Fatalf("go run args %v: %v", tc.Args, err)
				}
				if strings.TrimSpace(string(out)) != tc.Expected {
					t.Errorf("args %v: got %q want %q", tc.Args, strings.TrimSpace(string(out)), tc.Expected)
				}
			}
		})
	}
}

func TestCriteriaHashIncludesPrivateEvaluationInputs(t *testing.T) {
	e := Load()
	original := e.Hash()
	if e.Hash() != original {
		t.Fatal("hash is not stable")
	}
	e.correctIDs["q-computer"] = "b"
	if e.Hash() == original {
		t.Fatal("answer change did not change criteria hash")
	}
	e = Load()
	original = e.Hash()
	e.cases["coding"][0].Expected = "9"
	if e.Hash() == original {
		t.Fatal("hidden vector change did not change criteria hash")
	}
	e = Load()
	original = e.Hash()
	e.drivers["coding"] += "// version change"
	if e.Hash() == original {
		t.Fatal("driver change did not change criteria hash")
	}
	e = Load()
	original = e.Hash()
	e.Rank = "Junior"
	if e.Hash() == original {
		t.Fatal("rank change did not change criteria hash")
	}
	e = Load()
	original = e.Hash()
	e.Prerequisites = append(e.Prerequisites, "unverified")
	if e.Hash() == original {
		t.Fatal("prerequisite change did not change criteria hash")
	}
}

func fileExists(path string) bool { _, err := os.Stat(path); return err == nil }
