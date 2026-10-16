package httpapi

import (
	"academy/internal/projectrunner"
	"academy/internal/projects"
	"academy/internal/runner"
	"academy/internal/store"
	"archive/zip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// P1's automatic submit runs the complete isolated CLI vector suite and the
// learner's tests in one request. Keep it below the API server's 230s write
// timeout while leaving enough headroom to return a structured response.
const p1SubmitTimeout = 210 * time.Second

func projectJob(files map[string]string, tests bool, args []string) runner.Request {
	q := runner.Request{Files: map[string]string{}, Assets: map[string]string{}, Args: args, Mode: "run"}
	if tests {
		q.Mode = "test"
	}
	for name, source := range files {
		if strings.HasSuffix(name, ".go") && (tests || !strings.HasSuffix(name, "_test.go")) {
			q.Files[name] = source
		}
		if strings.HasSuffix(name, ".json") {
			q.Assets[name] = source
		}
	}
	return q
}
func hasTests(files map[string]string) bool {
	found := false
	for name, source := range files {
		if !strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
		if err != nil {
			return false
		}
		aliases := map[string]bool{}
		for _, im := range f.Imports {
			if im.Path.Value == `"testing"` {
				alias := "testing"
				if im.Name != nil {
					alias = im.Name.Name
				}
				aliases[alias] = true
			}
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fn.Name.Name == "TestMain" {
				return false
			}
			name := fn.Name.Name
			if fn.Recv != nil || len(name) <= 4 || !strings.HasPrefix(name, "Test") || unicode.IsLower(firstRune(name[4:])) || fn.Type.Results != nil || fn.Type.Params.NumFields() != 1 {
				continue
			}
			star, ok := fn.Type.Params.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			sel, ok := star.X.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "T" {
				continue
			}
			pkg, ok := sel.X.(*ast.Ident)
			if ok && aliases[pkg.Name] {
				found = true
			}
		}
	}
	return found
}
func (a *API) projectRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/projects", func(w http.ResponseWriter, r *http.Request) {
		write(w, 200, projects.AllWithProfiles(a.projectAvailable(r.Context(), "p3"), a.projectAvailable(r.Context(), "p4"), a.projectAvailable(r.Context(), "p5")))
	})
	mux.HandleFunc("GET /api/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		project, ok := projects.Find(r.PathValue("id"))
		if !ok {
			fail(w, 404, "проект не найден")
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		available := a.Runner != nil && a.Runner.Available(ctx)
		if project.ID == "p2" {
			available = a.ProjectRunner != nil && a.ProjectRunner.Available(ctx)
		} else if isManagedProject(project.ID) {
			available = a.projectAvailable(ctx, project.ID)
		}
		write(w, 200, struct {
			projects.Project
			ExecutionAvailable bool `json:"execution_available"`
		}{project, available})
	})
	// This is a deliberately narrow, read-only projection of server-persisted
	// evidence. The learner never supplies hashes or profile identifiers.
	mux.HandleFunc("GET /api/projects/{id}/submissions/{submissionID}", func(w http.ResponseWriter, r *http.Request) {
		projectID, submissionID := r.PathValue("id"), r.PathValue("submissionID")
		if _, ok := projects.Find(projectID); !ok {
			fail(w, 404, "проект не найден")
			return
		}
		submission, err := a.Store.ProjectSubmission("local", submissionID)
		if errors.Is(err, store.ErrProjectEvidenceNotFound) || err == nil && submission.ProjectID != projectID {
			fail(w, 404, "доказательство проекта не найдено")
			return
		}
		if err != nil {
			fail(w, 500, "не удалось прочитать доказательство проекта")
			return
		}
		// Discover the immutable evaluation only within this profile, project,
		// and submission. Then use the store accessor for canonical decoding.
		var evaluationID string
		err = a.Store.DB().QueryRow(`SELECT id FROM project_evaluations
			WHERE profile_id=? AND project_id=? AND submission_id=? ORDER BY created_at DESC,id DESC LIMIT 1`, "local", projectID, submissionID).Scan(&evaluationID)
		if errors.Is(err, sql.ErrNoRows) {
			fail(w, 404, "оценка проекта не найдена")
			return
		}
		if err != nil {
			fail(w, 500, "не удалось прочитать оценку проекта")
			return
		}
		evaluation, err := a.Store.ProjectEvaluation("local", evaluationID)
		if errors.Is(err, store.ErrProjectEvidenceNotFound) {
			fail(w, 404, "оценка проекта не найдена")
			return
		}
		if err != nil {
			fail(w, 500, "не удалось прочитать оценку проекта")
			return
		}
		write(w, 200, map[string]any{
			"submission_id": submission.ID, "evaluation_id": evaluation.ID,
			"project_id": projectID, "project_version": submission.ProjectVersion,
			"criteria_hash": submission.CriteriaHash, "artifact_schema_version": submission.ArtifactSchema,
			"artifact_hash": submission.ArtifactHash, "executed_source_hash": submission.ExecutedSourceHash,
			"evaluation_status": evaluation.EvaluationStatus, "exit_code": evaluation.ExitCode,
			"passed": evaluation.Passed, "checks_hash": evaluation.ChecksHash,
			"checker_version": evaluation.CheckerVersion, "checker_digest": evaluation.CheckerDigest,
			"runtime_policy_hash": evaluation.RuntimePolicyHash,
			"rubric_version":      evaluation.RubricVersion, "rubric_hash": evaluation.RubricHash,
			"submission_created_at": submission.CreatedAt, "evaluation_created_at": evaluation.CreatedAt,
		})
	})
	files := func(project projects.Project) (map[string]string, error) {
		files, err := a.Store.ProjectFiles(project.ID)
		if len(files) == 0 && err == nil {
			files = project.StarterFiles
		}
		return files, err
	}
	mux.HandleFunc("GET /api/projects/{id}/files", func(w http.ResponseWriter, r *http.Request) {
		project, ok := projects.Find(r.PathValue("id"))
		if !ok {
			fail(w, 404, "проект не найден")
			return
		}
		f, err := files(project)
		if err != nil {
			fail(w, 500, "не удалось прочитать файлы")
			return
		}
		write(w, 200, map[string]any{"files": f})
	})
	mux.HandleFunc("PUT /api/projects/{id}/files", func(w http.ResponseWriter, r *http.Request) {
		project, ok := projects.Find(r.PathValue("id"))
		if !ok {
			fail(w, 404, "проект не найден")
			return
		}
		var q struct {
			Files map[string]string `json:"files"`
		}
		if decode(w, r, &q) != nil {
			fail(w, 400, "некорректные файлы")
			return
		}
		if err := projects.ValidateFor(project.ID, q.Files); err != nil {
			fail(w, 422, err.Error())
			return
		}
		if err := a.Store.SaveProjectFiles(project.ID, q.Files); err != nil {
			fail(w, 500, "не удалось сохранить проект")
			return
		}
		write(w, 200, map[string]any{"files": q.Files})
	})
	mux.HandleFunc("GET /api/projects/{id}/snapshots", func(w http.ResponseWriter, r *http.Request) {
		project, ok := projects.Find(r.PathValue("id"))
		if !ok {
			fail(w, 404, "проект не найден")
			return
		}
		s, err := a.Store.ProjectSnapshots(project.ID)
		if err != nil {
			fail(w, 500, "не удалось прочитать версии")
			return
		}
		write(w, 200, s)
	})
	mux.HandleFunc("POST /api/projects/{id}/restore", func(w http.ResponseWriter, r *http.Request) {
		project, ok := projects.Find(r.PathValue("id"))
		if !ok {
			fail(w, 404, "проект не найден")
			return
		}
		var q struct {
			ID int64 `json:"snapshot_id"`
		}
		if decode(w, r, &q) != nil {
			fail(w, 400, "некорректная версия")
			return
		}
		f, err := a.Store.ProjectSnapshot(project.ID, q.ID)
		if err != nil {
			fail(w, 404, "версия не найдена в этом проекте")
			return
		}
		if err = a.Store.SaveProjectFiles(project.ID, f); err != nil {
			fail(w, 500, "не удалось восстановить версию")
			return
		}
		write(w, 200, map[string]any{"files": f})
	})
	mux.HandleFunc("GET /api/projects/{id}/export", func(w http.ResponseWriter, r *http.Request) {
		project, ok := projects.Find(r.PathValue("id"))
		if !ok {
			fail(w, 404, "проект не найден")
			return
		}
		f, err := files(project)
		if err != nil || projects.ValidateFor(project.ID, f) != nil {
			fail(w, 500, "не удалось подготовить архив")
			return
		}
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Disposition", `attachment; filename="`+project.ID+`.zip"`)
		z := zip.NewWriter(w)
		names := make([]string, 0, len(f))
		for name := range f {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			fw, err := z.Create(filepath.ToSlash(name))
			if err != nil {
				break
			}
			if _, err = fw.Write([]byte(f[name])); err != nil {
				break
			}
		}
		z.Close()
	})
	mux.HandleFunc("POST /api/projects/{id}/{action}", func(w http.ResponseWriter, r *http.Request) {
		project, ok := projects.Find(r.PathValue("id"))
		if !ok {
			fail(w, 404, "проект не найден")
			return
		}
		action := r.PathValue("action")
		if action != "run" && action != "test" && action != "submit" {
			fail(w, 404, "операция не найдена")
			return
		}
		var q struct {
			Files map[string]string `json:"files"`
			Args  []string          `json:"args"`
		}
		managed := isManagedProject(project.ID)
		decodeErr := decode(w, r, &q)
		if decodeErr != nil && !(managed && errors.Is(decodeErr, io.EOF)) {
			fail(w, 400, "некорректный проект")
			return
		}
		if managed && len(q.Files) == 0 {
			var err error
			q.Files, err = files(project)
			if err != nil {
				fail(w, 500, "не удалось прочитать сохранённый проект")
				return
			}
		}
		if err := projects.ValidateFor(project.ID, q.Files); err != nil {
			fail(w, 422, err.Error())
			return
		}
		if project.ID == "p2" {
			a.executeP2(w, r, action, q.Files)
			return
		}
		if managed {
			if !a.projectAvailable(r.Context(), project.ID) {
				fail(w, 503, "проект требует доступный pinned runner и соответствующий trusted checker")
				return
			}
			a.executeEvidenceProject(w, r, project.ID, action, q.Files)
			return
		}
		if a.Runner == nil {
			fail(w, 503, "исполнитель недоступен")
			return
		}
		if err := a.Store.SaveProjectFiles(project.ID, q.Files); err != nil {
			fail(w, 500, "не удалось сохранить проект")
			return
		}
		if action != "submit" {
			if action == "test" && !hasTests(q.Files) {
				fail(w, 422, "нужен TestX(*testing.T) в файле _test.go; TestMain для этого учебного проекта не поддерживается")
				return
			}
			result, err := a.Runner.Execute(r.Context(), projectJob(q.Files, action == "test", q.Args))
			if err != nil {
				fail(w, 503, err.Error())
				return
			}
			write(w, 200, result)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), p1SubmitTimeout)
		defer cancel()
		result := runner.Result{Status: "failed", Checks: []runner.Check{}, Passed: true}
		vectors := []struct {
			args   []string
			assets map[string]string
			stdout string
			code   int
		}{
			{[]string{"sum", "-3", "5", "0"}, nil, "2\n", 0},
			{nil, nil, "", 2},
			{[]string{"unknown"}, nil, "", 2},
			{[]string{"sum"}, nil, "", 2},
			{[]string{"sum", "-9223372036854775808", "9223372036854775807"}, nil, "-1\n", 0},
			{[]string{"sum", "-9223372036854775808", "-1"}, nil, "", 2},
			{[]string{"inspect"}, nil, "", 2},
			{[]string{"inspect", "missing.json"}, nil, "", 2},
			{[]string{"inspect", "badtype.json"}, map[string]string{"badtype.json": "[1]"}, "", 2},
			{[]string{"inspect", "null.json"}, map[string]string{"null.json": "null"}, "", 2},

			{[]string{"sum", "9223372036854775807", "1"}, nil, "", 2},
			{[]string{"sum", "not-a-number"}, nil, "", 2},
			{[]string{"inspect", "hidden.json"}, map[string]string{"hidden.json": `["a","b","c","d"]`}, "4\n", 0},
			{[]string{"inspect", "empty.json"}, map[string]string{"empty.json": `[]`}, "0\n", 0},
			{[]string{"inspect", "wrong.json"}, map[string]string{"wrong.json": `{"not":"an array"}`}, "", 2},
		}
		for i, v := range vectors {
			job := projectJob(q.Files, false, v.args)
			job.Assets = map[string]string{}
			for name, source := range v.assets {
				job.Assets[name] = source
			}
			actual, err := a.Runner.Execute(ctx, job)
			if err != nil {
				fail(w, 503, err.Error())
				return
			}
			ok := actual.ExitCode == v.code && actual.Stdout == v.stdout && (v.code == 0 && actual.Status == "ok" && actual.Stderr == "" || v.code != 0 && actual.Status == "error" && controlledCLIError(actual.Stderr))
			result.Checks = append(result.Checks, runner.Check{Name: "Функциональная проверка " + strconv.Itoa(i+1), Passed: ok})
			if !ok {
				result.Passed = false
				result.Stdout = actual.Stdout
				result.Stderr = actual.Stderr
				break
			}
		}
		if result.Passed {
			ok := hasTests(q.Files)
			if ok {
				actual, err := a.Runner.Execute(ctx, projectJob(q.Files, true, nil))
				if err != nil {
					fail(w, 503, err.Error())
					return
				}
				ok = actual.Status == "ok" && actual.ExitCode == 0
				result.Stdout = actual.Stdout
				result.Stderr = actual.Stderr
			}
			result.Checks = append(result.Checks, runner.Check{Name: "Пользовательский go test завершён без ошибок; качество тестов требует ревью", Passed: ok})
			result.Passed = result.Passed && ok
		}
		if result.Passed {
			result.Status = "automatic_pass"
		}
		if _, err := a.Store.RecordAttempt("p1:automatic", q.Files, result.Passed, result.Stdout+result.Stderr); err != nil {
			fail(w, 500, "не удалось сохранить результат")
			return
		}
		write(w, 200, map[string]any{"status": result.Status, "passed": result.Passed, "checks": result.Checks, "stdout": result.Stdout, "stderr": result.Stderr, "manual_review_required": true, "manual_review_labels": []string{"Удобство CLI и качество README", "Сборка и артефакты выпуска", "Читаемость Git-истории и поддерживаемость"}, "milestones": []map[string]any{{"id": "cli", "automatic_passed": result.Passed, "needs_review": false}, {"id": "json", "automatic_passed": result.Passed, "needs_review": false}, {"id": "release", "automatic_passed": result.Passed, "needs_review": true}}})
	})
	a.projectReviewRoutes(mux)
}

func isManagedProject(id string) bool {
	switch id {
	case "p3", "p4", "p5":
		return true
	default:
		return false
	}
}

func (a *API) projectAvailable(parent context.Context, projectID string) bool {
	ctx, cancel := context.WithTimeout(parent, 2*time.Second)
	defer cancel()
	if a.ProjectRunner == nil || !a.ProjectRunner.Available(ctx) {
		return false
	}
	capability, ok := a.ProjectRunner.(interface {
		SupportsProject(context.Context, string) bool
	})
	return ok && capability.SupportsProject(ctx, projectID)
}

func (a *API) taskFlowAvailable(parent context.Context) bool {
	return a.projectAvailable(parent, "p3")
}

func firstRune(s string) rune { r, _ := utf8.DecodeRuneInString(s); return r }
func controlledCLIError(s string) bool {
	trimmed := strings.TrimSpace(s)
	return trimmed != "" && len(s) < 2048 && !strings.HasPrefix(trimmed, "panic:") && !strings.HasPrefix(trimmed, "fatal error:") && !strings.Contains(s, "\ngoroutine ")
}

// executeP2 uses a different isolated database worker; export-only infrastructure
// files are never interpreted by the worker or used to configure its containers.
func (a *API) executeP2(w http.ResponseWriter, r *http.Request, action string, files map[string]string) {
	if a.ProjectRunner == nil {
		fail(w, 503, "исполнитель P2 недоступен; настройте отдельный project-runner")
		return
	}
	if err := a.Store.SaveProjectFiles("p2", files); err != nil {
		fail(w, 500, "не удалось сохранить проект")
		return
	}
	source := map[string]string{}
	for name, data := range files {
		base := filepath.Base(name)
		if base == "Dockerfile" || base == "compose.yaml" || base == "docker-compose.yml" {
			continue
		}
		source[name] = data
	}
	ctx, cancel := context.WithTimeout(r.Context(), 210*time.Second)
	defer cancel()
	result, err := a.ProjectRunner.Execute(ctx, projectrunner.Request{ProjectID: "p2", Action: action, Files: source})
	if err != nil {
		fail(w, 503, "изолированная проверка P2 недоступна или завершилась ошибкой")
		return
	}
	if action == "submit" {
		if _, err := a.Store.RecordAttempt("p2:automatic", files, result.Passed, result.Stdout+result.Stderr); err != nil {
			fail(w, 500, "не удалось сохранить результат")
			return
		}
	}
	write(w, 200, struct {
		runner.Result
		ManualReviewRequired bool     `json:"manual_review_required"`
		ManualReviewLabels   []string `json:"manual_review_labels"`
	}{result, true, []string{"Архитектура и обоснование компромиссов", "Безопасность сессий, ownership и границ данных", "PostgreSQL-интеграция, миграции и эксплуатация"}})
}

func (a *API) executeP3(w http.ResponseWriter, r *http.Request, action string, files map[string]string) {
	a.executeEvidenceProject(w, r, "p3", action, files)
}

func (a *API) executeEvidenceProject(w http.ResponseWriter, r *http.Request, projectID, action string, files map[string]string) {
	if a.ProjectRunner == nil {
		fail(w, 503, "изолированный исполнитель проекта недоступен")
		return
	}
	if projectID == "p3" && (action == "test" || action == "submit") && !hasTests(files) {
		fail(w, 422, "для тестирования и отправки нужен настоящий TestX(*testing.T); TestMain запрещён")
		return
	}
	if (projectID == "p4" || projectID == "p5") && (action == "test" || action == "submit") {
		if err := projectrunner.ValidateAssessmentProcessHooks(files); err != nil {
			fail(w, 422, err.Error())
			return
		}
	}
	if err := a.Store.SaveProjectFiles(projectID, files); err != nil {
		fail(w, 500, "не удалось сохранить проект")
		return
	}
	source := make(map[string]string, len(files))
	for name, value := range files {
		if projectID == "p3" {
			base := filepath.Base(name)
			if base == "Dockerfile" || base == "Containerfile" || base == "compose.yaml" || base == "compose.yml" || base == "docker-compose.yml" || base == "docker-compose.yaml" || base == ".env.example" || strings.HasSuffix(base, ".env") {
				continue
			}
		}
		source[name] = value
	}
	ctx, cancel := context.WithTimeout(r.Context(), 210*time.Second)
	defer cancel()
	request := projectrunner.Request{ProjectID: projectID, Action: action, Files: source}
	var result runner.Result
	var evaluationProfile projectrunner.EvaluationProfile
	var err error
	if action == "submit" {
		evaluator, ok := a.ProjectRunner.(interface {
			ExecuteWithProfile(context.Context, projectrunner.Request) (runner.Result, projectrunner.EvaluationProfile, error)
		})
		if !ok {
			fail(w, 503, "runner не передал доверенный профиль проверки")
			return
		}
		result, evaluationProfile, err = evaluator.ExecuteWithProfile(ctx, request)
	} else {
		result, err = a.ProjectRunner.Execute(ctx, request)
	}
	if err != nil {
		fail(w, 503, "изолированная проверка проекта недоступна или завершилась ошибкой")
		return
	}
	var projectEvidence map[string]any
	if action == "submit" {
		submission, recordErr := a.Store.RecordProjectSubmission("local", projectID, files, source)
		if recordErr != nil {
			fail(w, 500, "не удалось сохранить неизменяемый пакет проекта")
			return
		}
		checksJSON, marshalErr := json.Marshal(result.Checks)
		if marshalErr != nil {
			fail(w, 500, "не удалось сохранить критерии проверки проекта")
			return
		}
		evaluation, evaluationErr := a.Store.RecordProjectEvaluation("local", submission.ID, store.ProjectEvaluationProfile{
			ProjectVersion: evaluationProfile.ProjectVersion, CriteriaHash: evaluationProfile.CriteriaHash,
			CheckerVersion: evaluationProfile.CheckerVersion, CheckerDigest: evaluationProfile.CheckerDigest,
			RuntimePolicyHash: evaluationProfile.RuntimePolicyHash,
			RubricVersion:     evaluationProfile.RubricVersion, RubricHash: evaluationProfile.RubricHash,
		}, projectEvaluationStatus(result), result.ExitCode, result.Passed, string(checksJSON))
		if evaluationErr != nil {
			fail(w, 500, "не удалось закрепить серверный результат проверки проекта")
			return
		}
		if _, err = a.Store.RecordAttempt(projectID+":automatic", files, result.Passed, result.Stdout+result.Stderr); err != nil {
			fail(w, 500, "не удалось сохранить результат")
			return
		}
		projectEvidence = map[string]any{
			"submission_id": submission.ID, "evaluation_id": evaluation.ID,
			"project_version": submission.ProjectVersion, "criteria_hash": submission.CriteriaHash,
			"artifact_hash": submission.ArtifactHash, "executed_source_hash": submission.ExecutedSourceHash,
			"checker_version": evaluation.CheckerVersion, "checker_digest": evaluation.CheckerDigest,
			"runtime_policy_hash": evaluation.RuntimePolicyHash, "rubric_version": evaluation.RubricVersion,
			"rubric_hash": evaluation.RubricHash, "checks_hash": evaluation.ChecksHash,
			"evaluation_status": evaluation.EvaluationStatus, "exit_code": evaluation.ExitCode,
		}
	}
	manualReviewLabels := map[string][]string{
		"p3": {"Архитектура и обоснование компромиссов", "Качество кода и педагогическая ясность", "Эксплуатационный разбор failure и security limits"},
		"p4": {"Интерпретация CPU/heap профилей", "Воспроизводимость baseline/after измерений", "Capacity/SLO расчёт и эксплуатационные выводы"},
		"p5": {"Архитектура, альтернативы и ADR", "Риск-модель, эксплуатация и инцидент", "Независимая аутентифицированная защита точного артефакта"},
	}
	write(w, 200, struct {
		runner.Result
		ManualReviewRequired bool     `json:"manual_review_required"`
		ManualReviewLabels   []string `json:"manual_review_labels"`
		ProjectEvidence      any      `json:"project_evidence,omitempty"`
		RankAwarded          bool     `json:"rank_awarded"`
	}{result, true, manualReviewLabels[projectID], projectEvidence, false})
}

// runner.Result.Status describes execution transport (for example, "ok"),
// while project_evaluations stores a narrower assessment state. Keep those
// vocabularies separate so a successfully executed deterministic checker is
// persisted as machine evidence rather than rejected as an invalid status.
func projectEvaluationStatus(result runner.Result) string {
	if result.Status == "automatic_pass" {
		return "automatic_pass"
	}
	if result.Status == "ok" {
		if result.Passed && result.ExitCode == 0 {
			return "automatic_pass"
		}
		return "failed"
	}
	return "error"
}
