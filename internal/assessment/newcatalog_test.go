package assessment

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"academy/internal/runner"
)

func TestCatalogAndFrozenZeroCriteria(t *testing.T) {
	if got := Load().Hash(); got != "bd2cc6db2c954ef3410db69039bcc12cb15085098194686e9883d1e7b70f2d53" {
		t.Fatalf("legacy Zero hash changed: %s", got)
	}
	legacyChanged := Load()
	legacyChanged.RequiredRank = "Zero"
	if legacyChanged.Hash() == Load().Hash() {
		t.Fatal("new rank requirement not bound to legacy policy hash")
	}
	all := All()
	if len(all) != 6 {
		t.Fatalf("catalog has %d exams", len(all))
	}
	for _, e := range all[1:3] {
		if e.Version != "1.0.0" || len(e.Questions) != 20 || len(e.Tasks) != 3 {
			t.Errorf("incomplete exam %s: %s questions=%d tasks=%d", e.ID, e.Version, len(e.Questions), len(e.Tasks))
		}
	}
	advancedIDs := []string{"middle-exam", "middleplus-exam", "senior-exam"}
	for i, id := range advancedIDs {
		e := all[i+3]
		wantVersion := "1.0.0"
		if id == "middle-exam" {
			wantVersion = "1.0.1"
		}
		if e.ID != id || e.Version != wantVersion || len(e.Questions) != 15 || len(e.Tasks) != 3 {
			t.Errorf("incomplete advanced exam %s: %+v", id, e)
		}
		for _, exerciseID := range e.Prerequisites {
			if strings.Contains(exerciseID, ":") {
				t.Errorf("advanced prerequisite %q is not an exercise ID", exerciseID)
			}
		}
	}
	if e, ok := Find("junior-core", ""); !ok || e.RequiredRank != "Zero" {
		t.Fatal("latest Junior lookup or eligibility failed")
	}
	if _, ok := Find("juniorplus-api", "9.9.9"); ok {
		t.Fatal("unknown immutable version resolved")
	}
	if all[1].Hash() == all[2].Hash() {
		t.Fatal("different assessment policies share a hash")
	}
	b, err := json.Marshal(all[1])
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, secret := range []string{"private_input", "private_expected", "allowed_imports", "assessment_refs", "driver", "correctIDs"} {
		if strings.Contains(text, secret) {
			t.Errorf("public assessment leaks %q", secret)
		}
	}
	changed := *all[1]
	changed.RequiredRank = "Other"
	if changed.Hash() == all[1].Hash() {
		t.Fatal("required rank is not part of new exam hash")
	}
}

func TestNewExamTrustedReferencesPassProductionPrivateRequests(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	repo := filepath.Clean(filepath.Join(filepath.Dir(here), "..", ".."))
	goBin := filepath.Join(repo, ".tools", "go", "bin", "go")
	if _, err := os.Stat(goBin); err != nil {
		goBin = "go"
	}
	for _, exam := range All()[1:3] {
		for _, task := range exam.Tasks {
			exam, task := exam, task
			t.Run(exam.ID+"/"+task.ID, func(t *testing.T) {
				spec := exam.FunctionSpec(task.ID)
				if spec == nil {
					t.Fatal("task has no trusted function spec")
				}
				dir := t.TempDir()
				refPath := filepath.Join(repo, "content", "exams", exam.ID, "reference", map[string]string{"transitions": "countTransitions", "range": "clampRange", "unicode": "firstUniqueRune", "redirect": "safeRedirectPath", "retry": "retryableStatus", "access": "canAccessResource"}[task.ID], "main.go")
				ref, err := os.ReadFile(refPath)
				if err != nil {
					t.Fatal(err)
				}
				if err = spec.ValidateSource(map[string]string{"main.go": string(ref)}); err != nil {
					t.Fatalf("production source policy rejected reference: %v", err)
				}
				req, err := spec.Request(map[string]string{"main.go": string(ref)}, true)
				if err != nil {
					t.Fatal(err)
				}
				for name, source := range req.Files {
					if err = os.WriteFile(filepath.Join(dir, name), []byte(source), 0600); err != nil {
						t.Fatal(err)
					}
				}
				cmd := exec.Command(goBin, append([]string{"run", "main.go", "academy_driver.go"}, req.Args...)...)
				cmd.Dir = dir
				cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")
				out, err := cmd.Output()
				if err != nil {
					t.Fatalf("trusted host run failed: %v", err)
				}
				checks := spec.Check(runner.Result{Status: "ok", ExitCode: 0, Stdout: string(out)}, true)
				if len(checks) == 0 {
					t.Fatal("production checker returned no checks")
				}
				for _, check := range checks {
					if !check.Passed {
						t.Errorf("reference failed production check %q; output=%s", check.Name, out)
					}
				}
			})
		}
	}
}

// This ordinary, repository-owned fixture computes the correct result but
// changes its input. The trusted driver must reject that contract violation.
func TestTransitionsRejectsMutationWithCorrectAnswer(t *testing.T) {
	e, _ := Find("junior-core", "")
	spec := e.FunctionSpec("transitions")
	source := "package main\nfunc CountTransitions(v []int)int{ n:=0;for i:=1;i<len(v);i++{if v[i]!=v[i-1]{n++}};if len(v)>0{v[0]++};return n }\n"
	req, err := spec.Request(map[string]string{"main.go": source}, true)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for name, data := range req.Files {
		if err = os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(filepath.Join(runtime.GOROOT(), "bin", "go"), append([]string{"run", "main.go", "academy_driver.go"}, req.Args...)...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal("trusted mutation fixture did not run", err)
	}
	rejected := false
	for _, check := range spec.Check(runner.Result{Status: "ok", Stdout: string(out)}, true) {
		rejected = rejected || !check.Passed
	}
	if !rejected {
		t.Fatal("correct answers with changed input were accepted")
	}
}
