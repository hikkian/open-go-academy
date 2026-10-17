package content

import (
	"academy/internal/grading"
	"academy/internal/runner"
	"strings"
	"testing"
)

// Run trusted references and representative wrong implementations through the
// production request builder, runner result shape, and vector checker.
func TestSenior13To23ProductionVectorsAcceptReferencesAndRejectOmittedBranches(t *testing.T) {
	ids := []string{
		"senior-13-easy", "senior-13-medium", "senior-13-hard",
		"senior-14-easy", "senior-14-medium", "senior-14-hard",
		"senior-15-easy", "senior-15-medium", "senior-15-hard",
		"senior-16-easy", "senior-16-medium", "senior-16-hard",
		"senior-17-easy", "senior-17-medium", "senior-17-hard",
		"senior-18-easy", "senior-18-medium", "senior-18-hard",
		"senior-19-easy", "senior-19-medium", "senior-19-hard",
		"senior-20-easy", "senior-20-medium", "senior-20-hard",
		"senior-21-easy", "senior-21-medium", "senior-21-hard",
		"senior-22-easy", "senior-22-medium", "senior-22-hard",
		"senior-23-easy", "senior-23-medium", "senior-23-hard",
	}
	specs, err := grading.LoadFor(ids)
	if err != nil { t.Fatal(err) }
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			raw, err := solutions.ReadFile("solutions/" + id + "/main.go")
			if err != nil { t.Fatal(err) }
			spec := specs[id]
			if err := spec.BindReference(string(raw)); err != nil { t.Fatal(err) }
			if !senior13_23Passes(t, spec, string(raw)) { t.Fatal("production grader rejected trusted reference") }
		})
	}
	for _, fixture := range []struct{id, mutant string}{
		{"senior-16-medium", `package main
func CapacityHeadroom(capacity, demand int) string {
 if demand >= capacity-capacity/5 { return "low_headroom" }; return "headroom"
}`},
		{"senior-17-easy", `package main
func MigrationPhase(expanded, backfillVerified, consumersMoved bool) string {
 if !expanded { return "expand" }; return "contract_candidate"
}`},
		{"senior-20-medium", `package main
func CanExpand(errorPercent, threshold, sampleCount int) bool { return errorPercent <= threshold }
`},
		{"senior-23-hard", strings.Replace(string(mustSeniorReference(t, "senior-23-hard")), "if openRisks < 0 || reviewerSigned < 0 || reviewerSigned > 1 {", "if false {", 1)},
	} {
		t.Run("reject_"+fixture.id, func(t *testing.T) {
			if senior13_23Passes(t, specs[fixture.id], fixture.mutant) { t.Error("production vectors accepted representative mutant") }
		})
	}
}

func mustSeniorReference(t *testing.T, id string) string {
	t.Helper(); raw, err := solutions.ReadFile("solutions/"+id+"/main.go"); if err != nil { t.Fatal(err) }; return string(raw)
}

func senior13_23Passes(t *testing.T, spec *grading.Spec, source string) bool {
	t.Helper()
	request, err := spec.Request(map[string]string{"main.go": source}, true)
	if err != nil { t.Fatalf("build production grader request: %v", err) }
	stdout := trustedRun(t, request.Files, request.Args)
	checks := spec.Check(runner.Result{Status: "ok", ExitCode: 0, Stdout: stdout}, true)
	if len(checks) == 0 { t.Fatal("production grader returned no vector checks") }
	for _, check := range checks { if !check.Passed { return false } }
	return true
}
