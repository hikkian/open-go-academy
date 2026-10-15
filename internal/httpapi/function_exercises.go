package httpapi

import (
	"context"
	"net/http"
	"time"
)

// functionExercise never runs source on the API host. Expected values stay here;
// only the trusted adapter and selected inputs enter the confined runner.
func (a *API) functionExercise(w http.ResponseWriter, r *http.Request, id, action string, files map[string]string) {
	spec, ok := a.Content.Grading(id)
	if !ok {
		fail(w, 503, "проверка задания недоступна")
		return
	}
	private := action == "submit"
	request, err := spec.Request(files, private)
	if err != nil {
		fail(w, 422, "Проверь объявление функции и разрешённые конструкции: "+err.Error())
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 55*time.Second)
	defer cancel()
	result, err := a.Runner.Execute(ctx, request)
	if err != nil {
		fail(w, 503, "исполнитель не завершил проверку; повтори позже")
		return
	}
	if private {
		result.Checks = spec.Check(result, true)
		result.Passed = len(result.Checks) > 0
		for _, check := range result.Checks {
			result.Passed = result.Passed && check.Passed
		}
		// User panic/println can echo hidden inputs. Do not return or store raw
		// private-run output/diagnostics; public Run provides real compiler messages.
		result.Stdout = "Проверены открытые и закрытые сценарии."
		result.Stderr = ""
		result.Files = nil
		if _, err := a.Store.RecordAttempt(id, files, result.Passed, result.Stdout); err != nil {
			fail(w, 500, "не удалось сохранить результат; повтори отправку")
			return
		}
	}
	write(w, 200, result)
}
