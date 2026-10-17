package content

import (
	"academy/internal/grading"
	"academy/internal/runner"
	"strings"
	"testing"
)

func TestMiddle28HardProductionVectorsObserveComposition(t *testing.T) {
	specs, err := grading.LoadFor([]string{"middle-28-hard"})
	if err != nil {
		t.Fatal(err)
	}
	spec := specs["middle-28-hard"]
	if spec == nil {
		t.Fatal("missing middle-28-hard spec")
	}
	raw, err := solutions.ReadFile("solutions/middle-28-hard/main.go")
	if err != nil {
		t.Fatal(err)
	}
	reference := string(raw)
	if err := spec.BindReference(reference); err != nil {
		t.Fatal(err)
	}
	if !middle28HardPasses(t, spec, reference) {
		t.Fatal("production vectors rejected trusted reference")
	}
	mutants := map[string]string{
		"drops repository error": strings.Replace(reference, "return s.repo.Get(ctx, id)", "task, _ := s.repo.Get(ctx, id)\n\treturn task, nil", 1),
		"rewrites requested ID":  strings.Replace(reference, "return s.repo.Get(ctx, id)", "return s.repo.Get(ctx, \"other\")", 1),
		"calls repository twice": strings.Replace(reference, "return s.repo.Get(ctx, id)", "_, _ = s.repo.Get(ctx, id)\n\treturn s.repo.Get(ctx, id)", 1),
	}
	for name, mutant := range mutants {
		t.Run(name, func(t *testing.T) {
			if mutant == reference {
				t.Fatal("failed to construct mutant")
			}
			if middle28HardPasses(t, spec, mutant) {
				t.Errorf("production vectors accepted %s mutant", name)
			}
		})
	}
}

func middle28HardPasses(t *testing.T, spec *grading.Spec, source string) bool {
	t.Helper()
	request, err := spec.Request(map[string]string{"main.go": source}, true)
	if err != nil {
		t.Fatal(err)
	}
	stdout := trustedRun(t, request.Files, request.Args)
	checks := spec.Check(runner.Result{Status: "ok", ExitCode: 0, Stdout: stdout}, true)
	if len(checks) == 0 {
		t.Fatal("production grader returned no vector checks")
	}
	for _, check := range checks {
		if !check.Passed {
			return false
		}
	}
	return true
}
