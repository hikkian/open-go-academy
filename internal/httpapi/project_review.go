package httpapi

import (
	"academy/internal/projectreview"
	"academy/internal/projects"
	"academy/internal/store"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type manualReviewRubric struct {
	Version      string               `json:"version"`
	Hash         string               `json:"hash"`
	ProjectID    string               `json:"project_id"`
	Project      string               `json:"project"`
	Requirements []string             `json:"requirements"`
	Milestones   []projects.Milestone `json:"milestones"`
	RubricFiles  map[string]string    `json:"rubric_files,omitempty"`
}

func buildManualReviewRubric(project projects.Project) manualReviewRubric {
	files := map[string]string{}
	for _, name := range []string{"docs/project.md", "docs/rubric.md", "docs/threat-model.md"} {
		if body := project.StarterFiles[name]; body != "" {
			files[name] = body
		}
	}
	result := manualReviewRubric{
		Version:      project.Version + "-manual-review-v1",
		ProjectID:    project.ID,
		Project:      project.Title,
		Requirements: append([]string(nil), project.Requirements...),
		Milestones:   append([]projects.Milestone(nil), project.Milestones...),
		RubricFiles:  files,
	}
	if project.ID == "p3" {
		result.Version = project.Version + "-manual-review-v2"
		result.Requirements = append(result.Requirements, p3ManualOutcomeRequirements()...)
	}
	canonical, _ := json.Marshal(struct {
		Version      string
		ProjectID    string
		Project      string
		Requirements []string
		Milestones   []projects.Milestone
		RubricFiles  map[string]string
	}{result.Version, result.ProjectID, result.Project, result.Requirements, result.Milestones, result.RubricFiles})
	result.Hash = projectreview.Hash(canonical)
	return result
}

// These criteria extend the human review, not the automatic P3 checker. Approval
// attests to the complete exact-version rubric bound into the signed decision.
func p3ManualOutcomeRequirements() []string {
	return []string{
		"[M27-access-boundaries] Проверь матрицу actor/tenant/role и отрицательные HTTP/SQL сценарии IDOR. В exact artifact должны быть исходники тестов, команды и результаты, а не только отметка о наличии тестов.",
		"[M28-dependency-graph] Проверь типизированный граф handler → service → repository и composition root: конкретные экземпляры, владение ресурсами и cleanup при ошибке сборки. Потребуй fake-тесты точных вызовов/аргументов и объяснение границ интерфейсов.",
		"[M29-request-boundaries] Проверь порядок middleware, передачу deadline/cancellation, безопасное отображение ошибок и границы тела запроса. Воспроизведи хотя бы отменённый запрос и отказ зависимости; логи не должны раскрывать секреты или payload пользователя.",
		"[M30-32-operations] Проверь readiness/liveness, bounded shutdown/drain и реальные очищенные logs/metrics/traces. Укажи воспроизводимый сценарий завершения активной работы, ограничения cardinality и источник каждого наблюдаемого сигнала.",
		"[M33-slo-error-budget] Проверь документированный SLI: числитель, знаменатель, окно, отсутствующие сигналы, SLO и расчёт error budget. Приложи результат фиксированного workload с ошибками/latency и параметрами машины; не принимай выдуманные измерения за evidence.",
		"[M34-35-fault-resilience] Проверь retry budget, timeout и обоснованный выбор circuit breaker либо альтернативы. Требуется воспроизводимая fault-injection проверка отказа/восстановления, идемпотентности, освобождения ресурсов и разбор наблюдений/ограничений.",
		"[M36-40-release-evidence] Проверь версионированный acceptance record: exact artifact/hash, schema/API, test/vet/race/build, фиксированная нагрузка, failure/recovery, rollback-план и совместимость данных. Решение о выпуске должно перечислять оставшиеся риски, причины и границы утверждений.",
		"Решение approve по этой rubric означает проверку всех обязательных критериев на данном artifact. Если evidence отсутствует или не воспроизводится, верни changes_requested либо reject и укажи конкретный пробел; автоматический pass не заменяет ручную оценку.",
	}
}

func (a *API) projectReviewRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/projects/{id}/submissions/{submissionID}/review-packet", func(w http.ResponseWriter, r *http.Request) {
		projectID, submissionID := r.PathValue("id"), r.PathValue("submissionID")
		project, ok := projects.Find(projectID)
		if !ok || !isManagedProject(projectID) {
			fail(w, http.StatusNotFound, "проект не найден")
			return
		}
		submission, err := a.Store.ProjectSubmission("local", submissionID)
		if errors.Is(err, store.ErrProjectEvidenceNotFound) || err == nil && submission.ProjectID != projectID {
			fail(w, http.StatusNotFound, "доказательство проекта не найдено")
			return
		}
		if err != nil {
			fail(w, http.StatusInternalServerError, "не удалось прочитать доказательство проекта")
			return
		}
		criteriaHash, criteriaOK := projects.CriteriaHash(projectID)
		if submission.ProjectVersion != project.Version || !criteriaOK || submission.CriteriaHash != criteriaHash {
			fail(w, http.StatusConflict, "submission использует устаревшую версию проекта; текущая rubric к нему не применяется")
			return
		}
		evaluationID := r.URL.Query().Get("evaluation_id")
		if evaluationID == "" {
			if err = a.Store.DB().QueryRow(`SELECT id FROM project_evaluations
				WHERE profile_id=? AND project_id=? AND submission_id=? ORDER BY created_at DESC,id DESC LIMIT 1`,
				"local", projectID, submissionID).Scan(&evaluationID); err != nil {
				fail(w, http.StatusNotFound, "оценка проекта не найдена")
				return
			}
		}
		evaluation, err := a.Store.ProjectEvaluation("local", evaluationID)
		if errors.Is(err, store.ErrProjectEvidenceNotFound) || err == nil && (evaluation.ProjectID != projectID || evaluation.SubmissionID != submissionID) {
			fail(w, http.StatusNotFound, "оценка проекта не найдена")
			return
		}
		if err != nil {
			fail(w, http.StatusInternalServerError, "не удалось прочитать оценку проекта")
			return
		}
		if !evaluation.Passed || evaluation.EvaluationStatus != "automatic_pass" || evaluation.ExitCode != 0 {
			fail(w, http.StatusConflict, "для ручного review требуется успешная неизменяемая автоматическая оценка")
			return
		}
		rubric := buildManualReviewRubric(project)
		write(w, http.StatusOK, map[string]any{
			"submission_id": submission.ID, "evaluation_id": evaluation.ID,
			"project_id": project.ID, "project_version": submission.ProjectVersion,
			"criteria_hash": submission.CriteriaHash, "artifact_schema_version": submission.ArtifactSchema,
			"artifact_hash": strings.TrimPrefix(submission.ArtifactHash, "sha256:"), "executed_source_hash": submission.ExecutedSourceHash,
			"evaluation_status": evaluation.EvaluationStatus, "checks_hash": evaluation.ChecksHash,
			"checker_version": evaluation.CheckerVersion, "checker_digest": evaluation.CheckerDigest,
			"rubric": rubric, "files": submission.Files,
			"attestation_template": projectreview.Attestation{
				Version: projectreview.Version, ReviewerID: "replace-with-allowlisted-reviewer-id",
				SubmissionID: submission.ID, EvaluationID: evaluation.ID, ProjectID: project.ID,
				ProjectVersion: submission.ProjectVersion, ArtifactHash: strings.TrimPrefix(submission.ArtifactHash, "sha256:"),
				RubricVersion: rubric.Version, RubricHash: rubric.Hash,
				Decision: "", Comments: "", IssuedAt: time.Now().UTC().Format(time.RFC3339), Signature: "",
			},
		})
	})

	mux.HandleFunc("GET /api/projects/{id}/submissions/{submissionID}/reviews", func(w http.ResponseWriter, r *http.Request) {
		projectID, submissionID := r.PathValue("id"), r.PathValue("submissionID")
		if !isManagedProject(projectID) {
			fail(w, http.StatusNotFound, "проект не найден")
			return
		}
		submission, err := a.Store.ProjectSubmission("local", submissionID)
		if errors.Is(err, store.ErrProjectEvidenceNotFound) || err == nil && submission.ProjectID != projectID {
			fail(w, http.StatusNotFound, "доказательство проекта не найдено")
			return
		}
		if err != nil {
			fail(w, http.StatusInternalServerError, "не удалось прочитать доказательство проекта")
			return
		}
		reviews, err := a.Store.ProjectReviews("local", submissionID)
		if err != nil {
			fail(w, http.StatusInternalServerError, "не удалось прочитать project reviews")
			return
		}
		type reviewResponse struct {
			ID                     string    `json:"id"`
			ReviewerID             string    `json:"reviewer_id"`
			ReviewerKeyFingerprint string    `json:"reviewer_key_fingerprint"`
			Decision               string    `json:"decision"`
			Comments               string    `json:"comments"`
			ArtifactHash           string    `json:"artifact_hash"`
			RubricVersion          string    `json:"rubric_version"`
			RubricHash             string    `json:"rubric_hash"`
			AttestationDigest      string    `json:"attestation_digest"`
			CreatedAt              time.Time `json:"created_at"`
		}
		result := make([]reviewResponse, 0, len(reviews))
		for _, review := range reviews {
			result = append(result, reviewResponse{
				ID: review.ID, ReviewerID: review.Attestation.ReviewerID, ReviewerKeyFingerprint: review.ReviewerKeyFingerprint,
				Decision: review.Attestation.Decision, Comments: review.Attestation.Comments,
				ArtifactHash: review.Attestation.ArtifactHash, RubricVersion: review.Attestation.RubricVersion,
				RubricHash: review.Attestation.RubricHash, AttestationDigest: review.AttestationDigest, CreatedAt: review.CreatedAt,
			})
		}
		write(w, http.StatusOK, map[string]any{"reviews": result})
	})

	mux.HandleFunc("POST /api/projects/{id}/submissions/{submissionID}/reviews", func(w http.ResponseWriter, r *http.Request) {
		projectID, submissionID := r.PathValue("id"), r.PathValue("submissionID")
		if !isManagedProject(projectID) {
			fail(w, http.StatusNotFound, "проект не найден")
			return
		}
		if a.ProjectReviewers == nil {
			fail(w, http.StatusServiceUnavailable, "операторский allowlist project-review keys не настроен")
			return
		}
		var attestation projectreview.Attestation
		if decode(w, r, &attestation) != nil {
			fail(w, http.StatusBadRequest, "подписанное решение reviewer имеет неверный формат")
			return
		}
		if attestation.ProjectID != projectID || attestation.SubmissionID != submissionID {
			fail(w, http.StatusUnprocessableEntity, "review должен быть привязан к этому проекту и submission ID")
			return
		}
		project, ok := projects.Find(projectID)
		if !ok {
			fail(w, http.StatusNotFound, "проект не найден")
			return
		}
		submission, err := a.Store.ProjectSubmission("local", submissionID)
		if errors.Is(err, store.ErrProjectEvidenceNotFound) {
			fail(w, http.StatusNotFound, "доказательство проекта не найдено")
			return
		}
		if err != nil {
			fail(w, http.StatusInternalServerError, "не удалось прочитать доказательство проекта")
			return
		}
		criteriaHash, criteriaOK := projects.CriteriaHash(projectID)
		if !criteriaOK || submission.ProjectVersion != project.Version || submission.CriteriaHash != criteriaHash {
			fail(w, http.StatusConflict, "submission использует устаревшую версию проекта; текущая rubric к нему не применяется")
			return
		}
		rubric := buildManualReviewRubric(project)
		if attestation.ProjectVersion != project.Version || attestation.RubricVersion != rubric.Version || attestation.RubricHash != rubric.Hash {
			fail(w, http.StatusUnprocessableEntity, "review использует неизвестную версию rubric или hash")
			return
		}
		if err := projectreview.Verify(a.ProjectReviewers, attestation); err != nil {
			fail(w, http.StatusUnprocessableEntity, "подпись reviewer не прошла проверку доверенного allowlist")
			return
		}
		// Derive the public-key fingerprint from the exact allowlisted key that
		// Verify used; this is evidence of key possession, not personhood or
		// reviewer independence.
		reviewerKeyFingerprint, err := a.ProjectReviewers.KeyFingerprint(attestation.ReviewerID)
		if err != nil {
			fail(w, http.StatusUnprocessableEntity, "allowlisted reviewer key is unavailable")
			return
		}
		review, err := a.Store.RecordProjectReview("local", attestation, reviewerKeyFingerprint)
		if errors.Is(err, store.ErrProjectEvidenceNotFound) {
			fail(w, http.StatusNotFound, "submission или оценка проекта не найдены")
			return
		}
		if err != nil {
			fail(w, http.StatusConflict, "review не соответствует неизменяемому submission/evaluation")
			return
		}
		write(w, http.StatusCreated, map[string]any{
			"review_id": review.ID, "reviewer_id": review.Attestation.ReviewerID,
			"reviewer_key_fingerprint": review.ReviewerKeyFingerprint,
			"decision":                 review.Attestation.Decision, "artifact_hash": review.Attestation.ArtifactHash,
			"rubric_version": review.Attestation.RubricVersion, "rubric_hash": review.Attestation.RubricHash,
			"attestation_digest": review.AttestationDigest, "created_at": review.CreatedAt,
			"rank_awarded": false,
		})
	})
}
