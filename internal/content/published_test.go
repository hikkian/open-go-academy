package content

import (
	"academy/internal/runner"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Only repository-owned examples and reference solutions run on the test host.
// Learner submissions are never passed to this helper.
func trustedRun(t *testing.T, files map[string]string, args []string) string {
	return trustedCommand(t, files, args, "run")
}
func trustedCommand(t *testing.T, files map[string]string, args []string, mode string) string {
	t.Helper()
	goBin, _ := filepath.Abs(filepath.Join("..", "..", ".tools", "go", "bin", "go"))
	if _, err := os.Stat(goBin); err != nil {
		goBin = "go"
	}
	tmp := t.TempDir()
	for name, source := range files {
		if filepath.IsAbs(name) || filepath.Clean(name) != name || strings.HasPrefix(name, "..") {
			t.Fatal("unsafe trusted fixture path")
		}
		target := filepath.Join(tmp, name)
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module example.local/academy\n\ngo 1.27.2\n"), 0600); err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "CGO_ENABLED=0")
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	command := []string{"test", "-count=1", "-timeout=20s", "./..."}
	if mode == "compile" {
		command = []string{"test", "-run=^$", "./..."}
	}
	binary := filepath.Join(tmp, "example-program")
	if mode == "run" {
		command = []string{"build", "-o", binary, "."}
	}
	cmd := exec.CommandContext(ctx, goBin, command...)
	cmd.Dir = tmp
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("trusted fixture build/test: %v\n%s", err, output)
	}
	if mode != "run" {
		return string(output)
	}
	runContext, stop := context.WithTimeout(context.Background(), 20*time.Second)
	defer stop()
	program := exec.CommandContext(runContext, binary, args...)
	program.Dir = tmp
	program.Env = env
	output, err = program.CombinedOutput()
	if err != nil {
		t.Fatalf("trusted fixture execution: %v\n%s", err, output)
	}
	return string(output)
}
func TestEveryPublishedGoExampleAndFunctionReference(t *testing.T) {
	catalog, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, lesson := range catalog.AllLessons() {
		t.Run(lesson.ID, func(t *testing.T) {
			if len(lesson.SelfCheck) == 0 || len(lesson.Glossary) == 0 || len(lesson.Exercises) == 0 {
				t.Fatal("published lesson lacks practice or knowledge checks")
			}
			groups, err := documentationExamples(lesson.Body)
			if err != nil {
				t.Fatal(err)
			}
			if len(groups) == 0 {
				t.Fatal("published lesson has no Go example")
			}
			for _, group := range groups {
				t.Run(group.ID, func(t *testing.T) { trustedCommand(t, group.Files, nil, group.Mode) })
			}
			for _, exercise := range lesson.Exercises {
				if exercise.Kind != "function" {
					continue
				}
				t.Run(exercise.ID, func(t *testing.T) {
					spec, ok := catalog.Grading(exercise.ID)
					if !ok {
						t.Fatal("no grader")
					}
					ref, ok := catalog.Reference(exercise.ID)
					if !ok {
						t.Fatal("no private reference")
					}
					request, err := spec.Request(map[string]string{"main.go": ref}, true)
					if err != nil {
						t.Fatal(err)
					}
					out := trustedRun(t, request.Files, request.Args)
					checks := spec.Check(runner.Result{Status: "ok", ExitCode: 0, Stdout: out}, true)
					if len(checks) == 0 {
						t.Fatal("no reference checks")
					}
					for _, check := range checks {
						if !check.Passed {
							t.Fatalf("trusted reference failed %s; stdout=%q", check.Name, out)
						}
					}
					if strings.TrimSpace(exercise.PublicOutput) == "" {
						t.Fatal("no public expected result")
					}
				})
			}
		})
	}
}
func TestPublishedOrderAndPrivateGradingBoundary(t *testing.T) {
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	lessons := c.AllLessons()
	if lessons[0].ID != "zero-01" {
		t.Fatal("beginner entry is no longer first")
	}
	positions := map[string]int{}
	for i, l := range lessons {
		positions[l.ID] = i
	}
	for _, l := range lessons {
		for _, dep := range l.Prerequisites {
			if positions[dep] >= positions[l.ID] {
				t.Fatalf("%s precedes prerequisite %s", l.ID, dep)
			}
		}
	}
}
