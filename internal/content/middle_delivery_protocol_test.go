package content

import (
	"academy/internal/grading"
	"academy/internal/runner"
	"strings"
	"testing"
)

func TestMiddleDeliveryHardContractsAndMutants(t *testing.T) {
	ids := []string{"middle-16-hard", "middle-17-hard", "middle-18-hard"}
	specs, err := grading.LoadFor(ids)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			referenceBytes, err := solutions.ReadFile("solutions/" + id + "/main.go")
			if err != nil {
				t.Fatal(err)
			}
			reference := string(referenceBytes)
			assertDeliveryProtocolChecks(t, specs[id], reference, true)

			mutants := []struct {
				name   string
				source string
			}{}
			switch id {
			case "middle-16-hard":
				mutants = append(mutants,
					struct {
						name   string
						source string
					}{"reapply-known-event", strings.Replace(reference, "if !alreadyProcessed {", "if true {", 1)},
					struct {
						name   string
						source string
					}{"ack-before-effect", strings.Replace(reference,
						"if !alreadyProcessed {\n\t\tif err := apply(eventID); err != nil { return \"effect_failed\" }\n\t}\n\tif err := ack(eventID); err != nil { return \"effect_committed_ack_unknown\" }",
						"if err := ack(eventID); err != nil { return \"effect_committed_ack_unknown\" }\n\tif !alreadyProcessed {\n\t\tif err := apply(eventID); err != nil { return \"effect_failed\" }\n\t}", 1)},
					struct {
						name   string
						source string
					}{"ignore-replay-ack-error", strings.Replace(reference,
						"if err := ack(eventID); err != nil { return \"effect_committed_ack_unknown\" }",
						"if alreadyProcessed { _ = ack(eventID); return \"acknowledged\" }\n\tif err := ack(eventID); err != nil { return \"effect_committed_ack_unknown\" }", 1)})
			case "middle-17-hard":
				mutants = append(mutants,
					struct {
						name   string
						source string
					}{"mark-before-publish", strings.Replace(reference,
						"if !publish(eventID) { return \"retry_publish\" }\n\tif !markPublished(eventID) { return \"duplicate_risk\" }",
						"if !markPublished(eventID) { return \"duplicate_risk\" }\n\tif !publish(eventID) { return \"retry_publish\" }", 1)},
					struct {
						name   string
						source string
					}{"publish-before-commit", strings.Replace(reference,
						"if !committed { return \"discard\" }\n\tif !publish(eventID) { return \"retry_publish\" }",
						"if !publish(eventID) { return \"retry_publish\" }\n\tif !committed { return \"discard\" }", 1)})
			case "middle-18-hard":
				mutants = append(mutants, struct {
					name   string
					source string
				}{"retry-unknown-commit", strings.Replace(reference, `if commitState == "unknown" { return "reconcile" }`, `if commitState == "unknown" { return "retry" }`, 1)})
				mutants = append(mutants,
					struct {
						name   string
						source string
					}{"accept-invalid-state", strings.Replace(reference, `if commitState != "committed" && commitState != "uncommitted" && commitState != "unknown" { return "invalid" }`, `if false { return "invalid" }`, 1)},
					struct {
						name   string
						source string
					}{"validate-budget-before-state", strings.Replace(reference, `if commitState == "committed" { return "ack" }`, "if attempts < 0 || maxAttempts <= 0 { return \"invalid\" }\n\tif commitState == \"committed\" { return \"ack\" }", 1)})
			}
			for _, mutant := range mutants {
				t.Run(mutant.name, func(t *testing.T) {
					if mutant.source == reference {
						t.Fatal("mutant fixture did not alter the reference")
					}
					assertDeliveryProtocolChecks(t, specs[id], mutant.source, false)
				})
			}
		})
	}
}

func assertDeliveryProtocolChecks(t *testing.T, spec *grading.Spec, source string, wantPass bool) {
	t.Helper()
	if err := spec.BindReference(source); err != nil {
		t.Fatal(err)
	}
	request, err := spec.Request(map[string]string{"main.go": source}, true)
	if err != nil {
		t.Fatal(err)
	}
	stdout := trustedRun(t, request.Files, request.Args)
	checks := spec.Check(runner.Result{Status: "ok", ExitCode: 0, Stdout: stdout}, true)
	if len(checks) == 0 {
		t.Fatal("grader returned no checks")
	}
	passed := true
	for _, check := range checks {
		passed = passed && check.Passed
	}
	if passed != wantPass {
		t.Fatalf("all checks passed=%v, want %v; stdout=%q", passed, wantPass, stdout)
	}
}
