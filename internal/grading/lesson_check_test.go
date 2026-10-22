package grading

import (
	"strings"
	"testing"

	"academy/internal/runner"
)

func TestCheckShowsExpectedAndActualOnlyForPublicExamples(t *testing.T) {
	s := demo(t)
	checks := s.Check(runner.Result{Stdout: "3\n4\n300\n", ExitCode: 0, Status: "ok"}, true)
	if len(checks) != 3 {
		t.Fatalf("want three checks, got %#v", checks)
	}
	if checks[1].Passed || checks[1].Expected != "3" || checks[1].Actual != "4" || checks[1].Hint == "" {
		t.Fatalf("a failed public example says what was expected and what came out: %#v", checks[1])
	}
	closed := checks[2]
	if closed.Expected != "" || closed.Actual != "" || !strings.HasPrefix(closed.Name, "закрытый пример") || closed.Hint == "" {
		t.Fatalf("a closed example never shows its values: %#v", closed)
	}
}
