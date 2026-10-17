package content

import (
	"academy/internal/grading"
	"academy/internal/runner"
	"strings"
	"testing"
)

// These regressions run trusted references and deliberate mutants through the
// same production driver and vector checker used for learner submissions.
func TestMiddle20HardProductionVectorsRejectConfirmFirst(t *testing.T) {
	spec, reference := middle20_22BoundSpec(t, "middle-20-hard")
	if !middle20_22Passes(t, spec, reference) {
		t.Fatal("production driver rejected middle-20-hard reference")
	}
	confirmFirst := `package main
func PublishOutcomeHard(mandatory, brokerConfirm, returned bool) string {
	if !brokerConfirm { return "unknown" }
	if returned && !mandatory { return "invalid" }
	if returned { return "unrouted" }
	return "confirmed"
}
`
	if middle20_22Passes(t, spec, confirmFirst) {
		t.Error("production driver accepted confirm-first mutant")
	}
}

func TestMiddle22HardProductionVectorsRejectMutants(t *testing.T) {
	spec, reference := middle20_22BoundSpec(t, "middle-22-hard")
	if !middle20_22Passes(t, spec, reference) {
		t.Fatal("production driver rejected middle-22-hard reference")
	}
	mutants := map[string]string{
		"ignores elapsed": `package main
func RetryAfterSecondsHard(available, capacity, elapsedSeconds, refillPerSecond, cost int64) int64 {
 if capacity<0 || available<0 || available>capacity || elapsedSeconds<0 || refillPerSecond<0 || cost<=0 || cost>capacity { return -1 }
 if available>=cost { return 0 }; if refillPerSecond==0 { return -1 }
 deficit:=cost-available; q:=deficit/refillPerSecond; if deficit%refillPerSecond!=0 { q++ }; return q
}`,
		"ignores capacity":          strings.Replace(reference, "\tif cost > capacity {\n\t\treturn -1\n\t}\n", "", 1),
		"overflowing ceil addition": strings.Replace(reference, "return ceilDivNonnegative(cost-available, refillPerSecond)", "deficit := cost - available\n\treturn (deficit + refillPerSecond - 1) / refillPerSecond", 1),
		"missing invalid-input validation": strings.Replace(strings.Replace(reference,
			"\tif capacity < 0 || available < 0 || available > capacity || elapsedSeconds < 0 || refillPerSecond < 0 || cost <= 0 {\n\t\treturn -1\n\t}\n", "", 1),
			"\tif cost > capacity {\n\t\treturn -1\n\t}\n", "", 1),
	}
	for name, mutant := range mutants {
		t.Run(name, func(t *testing.T) {
			if mutant == reference {
				t.Fatal("failed to construct mutant")
			}
			if middle20_22Passes(t, spec, mutant) {
				t.Errorf("production driver accepted %s mutant", name)
			}
		})
	}
}

func middle20_22BoundSpec(t *testing.T, id string) (*grading.Spec, string) {
	t.Helper()
	specs, err := grading.LoadFor([]string{id})
	if err != nil {
		t.Fatal(err)
	}
	spec := specs[id]
	if spec == nil {
		t.Fatalf("missing production spec %s", id)
	}
	raw, err := solutions.ReadFile("solutions/" + id + "/main.go")
	if err != nil {
		t.Fatalf("read trusted reference %s: %v", id, err)
	}
	reference := string(raw)
	if err := spec.BindReference(reference); err != nil {
		t.Fatalf("bind trusted contract %s: %v", id, err)
	}
	return spec, reference
}

func middle20_22Passes(t *testing.T, spec *grading.Spec, source string) bool {
	t.Helper()
	request, err := spec.Request(map[string]string{"main.go": source}, true)
	if err != nil {
		t.Fatalf("build production grader request: %v", err)
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
