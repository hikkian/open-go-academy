package assessment

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"academy/internal/runner"
)

func TestAdvancedCatalogsAreCompleteAndVersioned(t *testing.T) {
	catalogs := []struct {
		exam               *Exam
		id, rank, required string
		questions, tasks   int
		hash               string
	}{
		{loadMiddle(), "middle-exam", "Middle", "Junior+", 15, 3, "769db5107eba8e957e3645c4b56eaff5bf4e946705668ac58216d63a85bfdc9c"},
		{loadMiddlePlus(), "middleplus-exam", "Middle+", "Middle", 15, 3, "4e2bc1f4d4e230f9b52432f2a47e24a9b6f76b9e4bdf8f10be107bb240a951fa"},
		{loadSenior(), "senior-exam", "Senior", "Middle+", 15, 3, "beac22567bc0cf6c1c56ddb527f0c87897bf4e799e7db1c17a19a1a6eb424860"},
	}
	for _, tc := range catalogs {
		t.Run(tc.id, func(t *testing.T) {
			e := tc.exam
			if e.ID != tc.id || e.Version != "1.0.0" || e.Rank != tc.rank || e.RequiredRank != tc.required {
				t.Fatalf("unexpected immutable identity: %+v", e)
			}
			if len(e.Questions) != tc.questions || len(e.Tasks) != tc.tasks {
				t.Fatalf("incomplete catalog: %d questions, %d tasks", len(e.Questions), len(e.Tasks))
			}
			if strings.Contains(strings.ToLower(e.Description), "присва") && !strings.Contains(strings.ToLower(e.Description), "не присва") {
				t.Fatal("exam description grants rank authority")
			}
			if !strings.Contains(strings.Join(e.Criteria, " "), "не присваивает") {
				t.Fatal("rank limitation must be explicit in criteria")
			}
			if e.Hash() != tc.hash || e.Hash() != e.Hash() {
				t.Fatalf("immutable content hash changed: got %s, want %s", e.Hash(), tc.hash)
			}
			seen := map[string]bool{}
			for _, q := range e.Questions {
				if q.ID == "" || seen[q.ID] || len(q.Options) != 3 {
					t.Fatalf("invalid or duplicate question: %+v", q)
				}
				seen[q.ID] = true
				if strings.TrimSpace(q.Prompt) == "" {
					t.Fatalf("empty question %s", q.ID)
				}
				for _, option := range q.Options {
					if strings.TrimSpace(option.Text) == "" {
						t.Fatalf("empty option %s", q.ID)
					}
				}
				if e.correctIDs[q.ID] == "" {
					t.Fatalf("question %s lacks private answer", q.ID)
				}
			}
			checks := e.QuizChecks(e.correctIDs)
			if len(checks) != tc.questions {
				t.Fatalf("quiz checks=%d", len(checks))
			}
			for _, check := range checks {
				if !check.Passed {
					t.Errorf("trusted answer for %s failed", check.ID)
				}
			}
			for _, task := range e.Tasks {
				if task.ID == "" || strings.TrimSpace(task.Description) == "" || task.StarterCode == "" || len(task.PublicExamples) == 0 {
					t.Fatalf("incomplete task: %+v", task)
				}
				if e.FunctionSpec(task.ID) == nil {
					t.Fatalf("task %s lacks trusted grading policy", task.ID)
				}
				if err := e.FunctionSpec(task.ID).ValidateSource(map[string]string{"main.go": advancedReferences[e.FunctionSpec(task.ID).ExerciseID()]}); err != nil {
					t.Fatalf("trusted reference for %s rejected: %v", task.ID, err)
				}
			}
		})
	}
}

func TestAdvancedTrustedReferencesPassActualPublicAndPrivateExamVectors(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repo := filepath.Clean(filepath.Join(filepath.Dir(here), "..", ".."))
	goBin := "go"
	if candidate := filepath.Join(repo, ".tools", "go", "bin", "go"); exists(candidate) {
		goBin = candidate
	}
	for _, exam := range []*Exam{loadMiddle(), loadMiddlePlus(), loadSenior()} {
		for _, task := range exam.Tasks {
			exam, task := exam, task
			t.Run(exam.ID+"/"+task.ID, func(t *testing.T) {
				spec := exam.FunctionSpec(task.ID)
				if spec == nil {
					t.Fatal("advanced task lacks grading spec")
				}
				reference := advancedReferences[spec.ExerciseID()]
				request, err := spec.Request(map[string]string{"main.go": reference}, true)
				if err != nil {
					t.Fatalf("reference rejected by exam spec: %v", err)
				}
				dir := t.TempDir()
				files := make([]string, 0, len(request.Files))
				for name, source := range request.Files {
					fullPath := filepath.Join(dir, name)
					if err := os.WriteFile(fullPath, []byte(source), 0600); err != nil {
						t.Fatal(err)
					}
					files = append(files, name)
				}
				sort.Strings(files)
				args := append([]string{"run"}, files...)
				args = append(args, request.Args...)
				cmd := exec.Command(goBin, args...)
				cmd.Dir = dir
				cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off")
				out, err := cmd.Output()
				if err != nil {
					t.Fatalf("trusted exam driver failed: %v", err)
				}
				for _, check := range spec.Check(runner.Result{Status: "ok", ExitCode: 0, Stdout: string(out)}, true) {
					if !check.Passed {
						t.Errorf("actual grader vector failed: %s", check.Name)
					}
				}
			})
		}
	}
}
func exists(path string) bool { _, err := os.Stat(path); return err == nil }
