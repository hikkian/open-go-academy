package httpapi

import (
	"academy/internal/assessment"
	"academy/internal/runner"
	"academy/internal/store"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

const examTrustNotice = "Локальная учебная проверка версионированных критериев Go Academy: без прокторинга и проверки личности; не подтверждает должность или стаж."

type examInput struct {
	Revision *int64            `json:"revision"`
	Token    string            `json:"attempt_token"`
	Answers  map[string]string `json:"answers"`
	Codes    map[string]string `json:"codes"`
}

func tokenHash(token string) string {
	b := sha256.Sum256([]byte(token))
	return hex.EncodeToString(b[:])
}
func validateExamInput(e *assessment.Exam, q examInput) error {
	if q.Revision == nil || *q.Revision < 0 {
		return errors.New("нужна актуальная ревизия попытки; обновите её с сервера")
	}
	if len(q.Token) != 64 {
		return errors.New("ключ попытки утрачен; отмените её и начните заново")
	}
	if len(q.Answers) > len(e.Questions) || len(q.Codes) > len(e.Tasks) {
		return errors.New("неизвестные поля экзамена")
	}
	for id, value := range q.Answers {
		known := false
		for _, question := range e.Questions {
			if question.ID == id {
				for _, option := range question.Options {
					if option.ID == value {
						known = true
					}
				}
			}
		}
		if !known {
			return errors.New("неизвестный вопрос или вариант ответа")
		}
	}
	for id, source := range q.Codes {
		known := false
		for _, task := range e.Tasks {
			known = known || task.ID == id
		}
		if !known || len(source)+len(e.Driver(id)) > runner.MaxSource {
			return errors.New("неизвестное задание или исходник превышает лимит 64 КиБ вместе с проверяющим адаптером")
		}
	}
	return nil
}
func examStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrExamPrerequisites):
		fail(w, 409, "сначала подтвердите выполнение обязательных упражнений")
	case errors.Is(err, store.ErrRankEvidenceRequired):
		fail(w, 409, "для этого ранга обязательны проверенные проектные и аттестационные evidence; автоматическая выдача пока не настроена")
	case errors.Is(err, store.ErrExamAttemptNotFound), errors.Is(err, store.ErrProfileNotFound), errors.Is(err, store.ErrExamNotFound):
		fail(w, 404, "попытка или экзамен не найдены")
	case errors.Is(err, store.ErrExamToken):
		fail(w, 403, "ключ попытки недействителен")
	case errors.Is(err, store.ErrExamAttemptConflict):
		fail(w, 409, "состояние попытки изменилось; обновите историю")
	default:
		fail(w, 500, "не удалось сохранить состояние экзамена")
	}
}
func (a *API) examRoutes(mux *http.ServeMux) {
	for _, e := range assessment.All() {
		if err := a.Store.RegisterExam(e.ID, e.Version, e.Rank, e.Hash()); err != nil {
			panic(fmt.Errorf("immutable exam registration: %w", err))
		}
	}
	mux.HandleFunc("GET /api/exams", func(w http.ResponseWriter, r *http.Request) {
		p, err := a.Store.Progress()
		if err != nil {
			examStoreError(w, err)
			return
		}
		awards, err := a.Store.RankAwards("local")
		if err != nil {
			examStoreError(w, err)
			return
		}
		ranks := map[string]bool{}
		for _, award := range awards {
			ranks[award.Rank] = true
		}
		catalog := []map[string]any{}
		for _, e := range assessment.All() {
			missing := []string{}
			for _, id := range e.Prerequisites {
				if p.Evidence[id] == 0 {
					missing = append(missing, id)
				}
			}
			if e.RequiredRank != "" && !ranks[e.RequiredRank] {
				missing = append(missing, "rank:"+e.RequiredRank)
			}
			entry := map[string]any{"id": e.ID, "version": e.Version, "title": e.Title, "rank": e.Rank, "description": e.Description, "estimated_minutes": e.EstimatedMinutes, "criteria": e.Criteria}
			policyStatus, policyErr := a.Store.RankPolicyEligibility("local", e.ID, e.Version)
			if policyErr == nil {
				entry["rank_evidence"] = policyStatus
				for _, requirement := range policyStatus.Requirements {
					if requirement.Status != "verified" {
						missing = append(missing, "evidence:"+requirement.RequirementID)
					}
				}
			} else if errors.Is(policyErr, store.ErrRankPolicyNotFound) && advancedRank(e.Rank) {
				entry["rank_evidence"] = map[string]any{"status": "unavailable", "reason": "rank_evidence_policy_unconfigured"}
				missing = append(missing, "rank_evidence_policy")
			} else if !errors.Is(policyErr, store.ErrRankPolicyNotFound) {
				examStoreError(w, policyErr)
				return
			}
			entry["eligible"] = len(missing) == 0
			entry["missing_prerequisites"] = missing
			catalog = append(catalog, entry)
		}
		write(w, 200, catalog)
	})
	mux.HandleFunc("GET /api/exams/{id}", func(w http.ResponseWriter, r *http.Request) {
		e, ok := assessment.Find(r.PathValue("id"), r.URL.Query().Get("version"))
		if !ok {
			fail(w, 404, "экзамен не найден")
			return
		}
		write(w, 200, e)
	})
	mux.HandleFunc("POST /api/exams/{id}/attempts", func(w http.ResponseWriter, r *http.Request) {
		e, ok := assessment.Find(r.PathValue("id"), "")
		if !ok {
			fail(w, 404, "экзамен не найден")
			return
		}
		var q struct{}
		if decode(w, r, &q) != nil {
			fail(w, 400, "некорректный запрос начала экзамена")
			return
		}
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			fail(w, 500, "не удалось создать ключ попытки")
			return
		}
		token := hex.EncodeToString(raw)
		requiredRanks := []string{}
		if e.RequiredRank != "" {
			requiredRanks = append(requiredRanks, e.RequiredRank)
		}
		if advancedRank(e.Rank) {
			eligibility, eligibilityErr := a.Store.RankPolicyEligibility("local", e.ID, e.Version)
			if errors.Is(eligibilityErr, store.ErrRankPolicyNotFound) {
				examStoreError(w, store.ErrRankEvidenceRequired)
				return
			}
			if eligibilityErr != nil {
				examStoreError(w, eligibilityErr)
				return
			}
			if !eligibility.Eligible {
				examStoreError(w, store.ErrExamPrerequisites)
				return
			}
		}
		attempt, err := a.Store.StartExam("local", e.ID, e.Version, tokenHash(token), e.Prerequisites, requiredRanks...)
		if err != nil {
			examStoreError(w, err)
			return
		}
		write(w, 201, map[string]any{"id": attempt.ID, "exam_id": attempt.ExamID, "version": attempt.Version, "status": attempt.Status, "attempt_token": token, "revision": attempt.Revision})
	})
	mux.HandleFunc("GET /api/exams/attempts", func(w http.ResponseWriter, r *http.Request) {
		attempts, err := a.Store.ExamAttempts("local")
		if err != nil {
			examStoreError(w, err)
			return
		}
		for i := range attempts {
			attempts[i].AnswersJSON = ""
			attempts[i].CodesJSON = ""
		}
		write(w, 200, attempts)
	})
	mux.HandleFunc("GET /api/exams/attempts/{id}", func(w http.ResponseWriter, r *http.Request) {
		attempt, err := a.Store.ExamAttempt("local", r.PathValue("id"))
		if err != nil {
			examStoreError(w, err)
			return
		}
		write(w, 200, attempt)
	})
	mux.HandleFunc("GET /api/rank-awards", func(w http.ResponseWriter, r *http.Request) {
		awards, err := a.Store.RankAwards("local")
		if err != nil {
			examStoreError(w, err)
			return
		}
		write(w, 200, awards)
	})
	mux.HandleFunc("POST /api/exams/attempts/{id}/cancel", func(w http.ResponseWriter, r *http.Request) {
		var q struct{}
		if decode(w, r, &q) != nil {
			fail(w, 400, "некорректный запрос")
			return
		}
		if err := a.Store.CancelExamAttempt("local", r.PathValue("id")); err != nil {
			examStoreError(w, err)
			return
		}
		write(w, 200, map[string]bool{"cancelled": true})
	})
	mux.HandleFunc("PUT /api/exams/attempts/{id}/draft", func(w http.ResponseWriter, r *http.Request) {
		var q examInput
		if decode(w, r, &q) != nil {
			fail(w, 400, "некорректный черновик")
			return
		}
		existing, err := a.Store.ExamAttempt("local", r.PathValue("id"))
		if err != nil {
			examStoreError(w, err)
			return
		}
		e, ok := assessment.Find(existing.ExamID, existing.Version)
		if !ok || existing.CriteriaHash != e.Hash() {
			fail(w, 409, "версия экзамена недоступна; начните новую попытку")
			return
		}
		if err := validateExamInput(e, q); err != nil {
			fail(w, 422, err.Error())
			return
		}
		answers, _ := json.Marshal(q.Answers)
		codes, _ := json.Marshal(q.Codes)
		updated, err := a.Store.SaveExamDraft("local", r.PathValue("id"), tokenHash(q.Token), *q.Revision, string(answers), string(codes))
		if err != nil {
			examStoreError(w, err)
			return
		}
		write(w, 200, map[string]any{"saved": true, "attempt": updated})
	})
	mux.HandleFunc("POST /api/exams/attempts/{id}/submit", func(w http.ResponseWriter, r *http.Request) {
		var q examInput
		if decode(w, r, &q) != nil {
			fail(w, 400, "оценку, ранг и результат назначает только сервер; передайте ответы и код")
			return
		}
		existing, err := a.Store.ExamAttempt("local", r.PathValue("id"))
		if err != nil {
			examStoreError(w, err)
			return
		}
		e, ok := assessment.Find(existing.ExamID, existing.Version)
		if !ok {
			fail(w, 409, "версия экзамена недоступна; начните новую попытку")
			return
		}
		if err := validateExamInput(e, q); err != nil {
			fail(w, 422, err.Error())
			return
		}

		if existing.ExamID != e.ID || existing.Version != e.Version || existing.CriteriaHash != e.Hash() {
			fail(w, 409, "версия экзамена обновлена; начните новую попытку")
			return
		}
		answers, _ := json.Marshal(q.Answers)
		codes, _ := json.Marshal(q.Codes)
		claimed, err := a.Store.ClaimExamAttempt("local", e.ID, existing.ID, tokenHash(q.Token), *q.Revision, string(answers), string(codes))
		if err != nil {
			examStoreError(w, err)
			return
		}
		finalized := false
		defer func() {
			if !finalized {
				a.Store.InterruptExamAttempt("local", claimed.ID)
			}
		}()
		ctx, cancel := context.WithTimeout(r.Context(), 180*time.Second)
		defer cancel()
		checks := e.QuizChecks(q.Answers)
		quizPassed := true
		for _, check := range checks {
			quizPassed = quizPassed && check.Passed
		}
		for _, task := range e.Tasks {
			source := q.Codes[task.ID]
			sourceErr := e.ValidateTaskSource(task.ID, source)
			if spec := e.FunctionSpec(task.ID); spec != nil {
				actual := runner.Result{}
				if quizPassed && sourceErr == nil {
					request, requestErr := spec.Request(map[string]string{"solution.go": source}, true)
					if requestErr != nil {
						sourceErr = requestErr
					} else {
						if a.Runner == nil {
							fail(w, 503, "исполнитель недоступен; попытка не засчитана")
							return
						}
						actual, err = a.Runner.Execute(ctx, request)
						if err != nil {
							fail(w, 503, "исполнитель прервал проверку; начните новую попытку")
							return
						}
					}
				}
				for i, c := range spec.Check(actual, true) {
					check := assessment.Check{ID: fmt.Sprintf("%s-%d", task.ID, i+1), Name: task.Title + ": " + c.Name, Passed: quizPassed && sourceErr == nil && c.Passed}
					if !quizPassed {
						check.Detail = "Не запускалась: сначала исправьте теоретические ответы."
					} else if sourceErr != nil {
						check.Detail = "Проверка исходника: " + boundedExamDetail(sourceErr.Error())
					} else if !check.Passed {
						check.Detail = "Результат не соответствует критериям. Закрытые входы и вывод не раскрываются."
					}
					checks = append(checks, check)
				}
				continue
			}
			failedCase := false
			for i, tc := range e.TaskCases(task.ID) {
				check := assessment.Check{ID: fmt.Sprintf("%s-%d", task.ID, i+1), Name: fmt.Sprintf("%s: проверка %d", task.Title, i+1), Passed: false}
				if !quizPassed {
					check.Detail = "Не запускалась: сначала исправьте теоретические ответы."
				} else if sourceErr != nil {
					check.Detail = "Проверка исходника: " + boundedExamDetail(sourceErr.Error())
				} else if failedCase {
					check.Detail = "Не запускалась после первой ошибки в этом задании."
				}
				if quizPassed && sourceErr == nil && !failedCase {
					if a.Runner == nil {
						fail(w, 503, "исполнитель недоступен; попытка не засчитана, можно повторить")
						return
					}
					actual, err := a.Runner.Execute(ctx, runner.Request{Mode: "run", Files: map[string]string{"solution.go": source, "main.go": e.Driver(task.ID)}, Args: tc.Args})
					if err != nil {
						fail(w, 503, "исполнитель прервал проверку; попытка не засчитана, начните новую")
						return
					}
					check.Passed = actual.Status == "ok" && actual.ExitCode == 0 && actual.Stderr == "" && actual.Stdout == tc.Expected+"\n"
					failedCase = !check.Passed
					if !check.Passed {
						check.Detail = "Вывод или завершение программы не соответствует критериям."
					}
				}
				checks = append(checks, check)
			}
		}
		successful := 0
		for _, check := range checks {
			if check.Passed {
				successful++
			}
		}
		score := successful * 100 / len(checks)
		passed := successful == len(checks)
		encoded, _ := json.Marshal(checks)
		completed, err := a.Store.FinishExamAttempt("local", claimed.ID, score, passed, string(encoded))
		if err != nil {
			examStoreError(w, err)
			return
		}
		finalized = true
		write(w, 200, map[string]any{"attempt": completed, "score": score, "passed": passed, "checks": checks, "rank_awarded": passed, "rank": e.Rank, "trust_notice": examTrustNotice})
	})
}

func advancedRank(rank string) bool {
	switch rank {
	case "Middle", "Middle+", "Senior":
		return true
	default:
		return false
	}
}

func boundedExamDetail(s string) string {
	if len(s) > 2048 {
		return s[:2048] + "…"
	}
	return s
}
