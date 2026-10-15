package httpapi

import (
	"academy/internal/projectrunner"
	"academy/internal/projects"
	"academy/internal/runner"
	"archive/zip"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

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
	mux.HandleFunc("GET /api/projects", func(w http.ResponseWriter, r *http.Request) { write(w, 200, projects.All()) })
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
		}
		write(w, 200, struct {
			projects.Project
			ExecutionAvailable bool `json:"execution_available"`
		}{project, available})
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
		if decode(w, r, &q) != nil {
			fail(w, 400, "некорректный проект")
			return
		}
		if err := projects.ValidateFor(project.ID, q.Files); err != nil {
			fail(w, 422, err.Error())
			return
		}
		if project.ID == "p2" {
			a.executeP2(w, r, action, q.Files)
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
		ctx, cancel := context.WithTimeout(r.Context(), 180*time.Second)
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
		write(w, 200, map[string]any{"status": result.Status, "passed": result.Passed, "checks": result.Checks, "stdout": result.Stdout, "stderr": result.Stderr, "manual_review_required": true, "milestones": []map[string]any{{"id": "cli", "automatic_passed": result.Passed, "needs_review": false}, {"id": "json", "automatic_passed": result.Passed, "needs_review": false}, {"id": "release", "automatic_passed": result.Passed, "needs_review": true}}})
	})
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
		ManualReviewRequired bool `json:"manual_review_required"`
	}{result, true})
}
