package httpapi

import (
	"strings"

	"academy/internal/content"
	"academy/internal/runner"
)

// lessonChecks is the verdict of "Проверить" for a program exercise. The checks are the same as before; each one now says what
// was expected and what came out, why it likely failed, and which criterion of the exercise it covers (docs/LESSON_UI_SPEC.md
// §2.1–2.2). Criteria are matched by position: 0 runs, 1 prints the public output, 2 is the exact comparison, the last one is the
// structure rule. A check that passed carries no hint.
func lessonChecks(ex content.Exercise, result runner.Result, files map[string]string) []runner.Check {
	ran := result.Status == "ok" && result.ExitCode == 0
	sameWords := strings.Join(strings.Fields(result.Stdout), " ") == strings.Join(strings.Fields(ex.PublicOutput), " ")
	outputOK := strings.TrimSpace(result.Stdout) == strings.TrimSpace(ex.PublicOutput)
	exact := result.Stdout == ex.PublicOutput && result.Stderr == ""

	run := runner.Check{Name: "Программа успешно завершилась", Passed: ran, Criterion: criterionAt(ex, 0)}
	output := runner.Check{Name: "Публичный пример вывода", Passed: outputOK, Expected: ex.PublicOutput, Actual: result.Stdout,
		Criterion: criterionAt(ex, 1)}
	strict := runner.Check{Name: "Точная проверка строк и отсутствия лишнего вывода", Passed: exact, Expected: ex.PublicOutput,
		Actual: result.Stdout, Criterion: criterionAt(ex, 2)}
	structureCheck := structure(ex.ID, files)
	structureCheck.Criterion = criterionAt(ex, len(ex.Criteria)-1)

	switch {
	case ran:
	case result.Stderr != "" || result.ExitCode != 0:
		run.Hint = "сначала почини ошибку, которая показана в выводе выше"
	default:
		run.Hint = "программа не завершилась нормально (статус: " + result.Status + ")"
	}
	switch {
	case outputOK:
	case sameWords:
		output.Hint = "совпадают слова, но не совпадают пробелы или переводы строк: включи показ невидимых символов"
	default:
		output.Hint = "вывод должен совпасть буква в букву, включая знаки препинания и регистр"
	}
	switch {
	case exact:
	case result.Stderr != "":
		strict.Hint = "в потоке ошибок есть текст: убери лишний вывод в stderr"
	case outputOK:
		strict.Hint = "нет или лишний перевод строки в конце: см. ⏎ в блоках «ждали» и «получили»"
	default:
		strict.Hint = "есть лишние строки или пробелы: см. блоки «ждали» и «получили»"
	}
	if !structureCheck.Passed {
		structureCheck.Hint = "решение не похоже на требуемую структуру: проверь вызовы в main по условию задания"
	}
	return []runner.Check{run, output, strict, structureCheck}
}

// criterionAt returns a pointer to the criterion index, or nil when the exercise has no such criterion.
func criterionAt(ex content.Exercise, i int) *int {
	if i < 0 || i >= len(ex.Criteria) {
		return nil
	}
	return &i
}
