package httpapi

import (
	"strings"
	"testing"

	"academy/internal/content"
	"academy/internal/runner"
)

func helloExercise() content.Exercise {
	return content.Exercise{ID: "hello", PublicOutput: "Go готов!\n",
		Criteria: []string{"Программа компилируется.", "Вывод ровно «Go готов!».", "Лишнего вывода нет.", "В main есть fmt.Println."}}
}

func TestLessonChecksShowWhatWasExpectedAndWhatCameOut(t *testing.T) {
	files := map[string]string{"main.go": "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"Привет, Go!\") }\n"}
	checks := lessonChecks(helloExercise(), runner.Result{Stdout: "Привет, Go!\n", ExitCode: 0, Status: "ok"}, files)
	if len(checks) != 4 {
		t.Fatalf("want four checks, got %d", len(checks))
	}
	out := checks[1]
	if out.Passed || out.Expected != "Go готов!\n" || out.Actual != "Привет, Go!\n" || out.Hint == "" || out.Criterion == nil || *out.Criterion != 1 {
		t.Fatalf("output check: %#v", out)
	}
	if !checks[0].Passed || checks[0].Hint != "" || checks[0].Expected != "" {
		t.Fatalf("a passed check carries no hint and no expected value: %#v", checks[0])
	}
	if checks[3].Criterion == nil || *checks[3].Criterion != 3 {
		t.Fatalf("structure check maps to the last criterion: %#v", checks[3])
	}
}

func TestLessonChecksNameAMissingLineBreakAsInvisible(t *testing.T) {
	checks := lessonChecks(helloExercise(), runner.Result{Stdout: "Go готов!", ExitCode: 0, Status: "ok"}, map[string]string{"main.go": "package main\n"})
	if !checks[1].Passed {
		t.Fatalf("the words match once trimmed: %#v", checks[1])
	}
	if checks[2].Passed || !strings.Contains(checks[2].Hint, "перевод строки") {
		t.Fatalf("the exact check should name the line break: %#v", checks[2])
	}
}

func TestLessonChecksPointAtStderrAndCompileErrors(t *testing.T) {
	stderr := lessonChecks(helloExercise(), runner.Result{Stdout: "Go готов!\n", Stderr: "warn", ExitCode: 0, Status: "ok"}, map[string]string{"main.go": "package main\n"})
	if stderr[2].Passed || !strings.Contains(stderr[2].Hint, "stderr") {
		t.Fatalf("stderr text should be named: %#v", stderr[2])
	}
	failed := lessonChecks(helloExercise(), runner.Result{Stderr: "syntax error", ExitCode: 1, Status: "error"}, map[string]string{"main.go": "package main\n"})
	if failed[0].Passed || !strings.Contains(failed[0].Hint, "почини ошибку") {
		t.Fatalf("a compile error is named in the run check: %#v", failed[0])
	}
}
