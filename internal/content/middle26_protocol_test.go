package content

import (
	"academy/internal/grading"
	"academy/internal/runner"
	"strings"
	"testing"
)

// This regression uses only repository-owned references and source mutations.
// Student submissions continue to run through the isolated production runner.
func TestMiddle26ProductionProtocolDriversRejectMutants(t *testing.T) {
	specs, err := grading.LoadFor([]string{"middle-26-medium", "middle-26-hard"})
	if err != nil {
		t.Fatal(err)
	}
	references := make(map[string]string, len(specs))
	for id, spec := range specs {
		b, err := solutions.ReadFile("solutions/" + id + "/main.go")
		if err != nil {
			t.Fatalf("read private reference %s: %v", id, err)
		}
		references[id] = string(b)
		if err := spec.BindReference(references[id]); err != nil {
			t.Fatalf("bind trusted contract for %s: %v", id, err)
		}
	}
	check := func(id, source string) bool {
		t.Helper()
		spec, ok := specs[id]
		if !ok {
			t.Fatalf("missing grading spec %s", id)
		}
		request, err := spec.Request(map[string]string{"main.go": source}, true)
		if err != nil {
			t.Fatalf("build production request for %s: %v", id, err)
		}
		output := trustedRun(t, request.Files, request.Args)
		checks := spec.Check(runner.Result{Status: "ok", ExitCode: 0, Stdout: output}, true)
		if len(checks) == 0 {
			t.Fatalf("production grader returned no checks for %s", id)
		}
		for _, c := range checks {
			if !c.Passed {
				return false
			}
		}
		return true
	}

	for _, id := range []string{"middle-26-medium", "middle-26-hard"} {
		reference := references[id]
		if !check(id, reference) {
			t.Errorf("production driver rejected reference %s", id)
		}
	}

	medium := references["middle-26-medium"]
	withoutRollback := strings.Replace(medium, "tx.Rollback()\n\t\treturn \"conflict\"", "return \"conflict\"", 1)
	if withoutRollback == medium {
		t.Fatal("failed to construct missing-rollback mutant")
	}
	if check("middle-26-medium", withoutRollback) {
		t.Error("medium driver accepted a conflict path without Rollback")
	}
	extraCommit := strings.Replace(medium, "tx.Rollback()\n\t\treturn \"conflict\"", "tx.Commit()\n\t\ttx.Rollback()\n\t\treturn \"conflict\"", 1)
	if extraCommit == medium {
		t.Fatal("failed to construct conflict-commit mutant")
	}
	if check("middle-26-medium", extraCommit) {
		t.Error("medium driver accepted Commit after conflict")
	}

	hard := references["middle-26-hard"]
	unknownCommitted := strings.Replace(hard, "default:\n\t\ttx.Rollback()\n\t\treturn \"internal_error\"", "default:\n\t\treturn \"committed\"", 1)
	if unknownCommitted == hard {
		t.Fatal("failed to construct unknown-Apply mutant")
	}
	if check("middle-26-hard", unknownCommitted) {
		t.Error("hard driver accepted unknown Apply result as committed")
	}
}
