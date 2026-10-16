package httpapi

import (
	"academy/internal/content"
	"academy/internal/projects"
	"academy/internal/runner"
	"academy/internal/store"
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type p1DeadlineExecutor struct {
	calls    int
	deadline time.Time
}

func (e *p1DeadlineExecutor) Available(context.Context) bool { return true }
func (e *p1DeadlineExecutor) Execute(ctx context.Context, q runner.Request) (runner.Result, error) {
	e.calls++
	if deadline, ok := ctx.Deadline(); ok && e.deadline.IsZero() {
		e.deadline = deadline
	}
	if q.Mode == "test" {
		return runner.Result{Status: "ok", ExitCode: 0}, nil
	}
	results := []runner.Result{
		{Status: "ok", ExitCode: 0, Stdout: "2\n"},
		{Status: "error", ExitCode: 2},
		{Status: "error", ExitCode: 2},
		{Status: "error", ExitCode: 2},
		{Status: "ok", ExitCode: 0, Stdout: "-1\n"},
		{Status: "error", ExitCode: 2},
		{Status: "error", ExitCode: 2},
		{Status: "error", ExitCode: 2},
		{Status: "error", ExitCode: 2},
		{Status: "error", ExitCode: 2},
		{Status: "error", ExitCode: 2},
		{Status: "error", ExitCode: 2},
		{Status: "ok", ExitCode: 0, Stdout: "4\n"},
		{Status: "ok", ExitCode: 0, Stdout: "0\n"},
		{Status: "error", ExitCode: 2},
	}
	index := e.calls - 1
	if index < 0 || index >= len(results) {
		return runner.Result{}, context.Canceled
	}
	result := results[index]
	if result.Status == "error" {
		result.Stderr = "invalid input"
	}
	return result, nil
}

func TestP1SubmitBudgetCoversFullIsolatedEvaluation(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.CreateProfile("Ada", "light"); err != nil {
		t.Fatal(err)
	}
	catalog, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	executor := &p1DeadlineExecutor{}
	a := New(s, catalog, executor, "secret", "127.0.0.1:8080")
	files := projects.Load().StarterFiles
	files["main_test.go"] = "package main\nimport \"testing\"\nfunc TestSmoke(t *testing.T) {}\n"
	requestBody, err := json.Marshal(map[string]any{"files": files})
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8080/api/projects/p1/submit", bytes.NewReader(requestBody))
	r.AddCookie(&http.Cookie{Name: "academy_session", Value: "secret"})
	r.Header.Set("X-CSRF-Token", "secret")
	w := httptest.NewRecorder()
	a.Handler(t.TempDir()).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("submit status=%d body=%s", w.Code, w.Body.String())
	}
	if executor.calls != 16 {
		t.Fatalf("runner executions=%d; want 15 isolated vectors plus test run", executor.calls)
	}
	remaining := time.Until(executor.deadline)
	if remaining < 200*time.Second || remaining > p1SubmitTimeout {
		t.Fatalf("submit context budget=%s; want between 200s and %s", remaining.Round(time.Second), p1SubmitTimeout)
	}
}

func TestProjectAPIStorageZIPAndIsolation(t *testing.T) {
	s, _ := store.Open(":memory:")
	defer s.Close()
	s.CreateProfile("Ada", "light")
	c, _ := content.Load()
	a := New(s, c, nil, "secret", "127.0.0.1:8080")
	h := a.Handler(t.TempDir())
	request := func(method, path string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		r := httptest.NewRequest(method, "http://127.0.0.1:8080/api/projects"+path, bytes.NewReader(b))
		r.AddCookie(&http.Cookie{Name: "academy_session", Value: "secret"})
		r.Header.Set("X-CSRF-Token", "secret")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	files := projects.Load().StarterFiles
	files["fixture.json"] = `["saved"]`
	if w := request("PUT", "/p1/files", map[string]any{"files": files}); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	snapshots, _ := s.ProjectSnapshots("p1")
	if len(snapshots) != 1 {
		t.Fatal(snapshots)
	}
	files["fixture.json"] = `["changed"]`
	request("PUT", "/p1/files", map[string]any{"files": files})
	if w := request("POST", "/p1/restore", map[string]any{"snapshot_id": snapshots[0].ID}); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	w := request("GET", "/p1/export", nil)
	if w.Code != 200 {
		t.Fatal(w)
	}
	z, err := zip.NewReader(bytes.NewReader(w.Body.Bytes()), int64(w.Body.Len()))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range z.File {
		if f.Name == "fixture.json" {
			r, _ := f.Open()
			b, _ := io.ReadAll(r)
			r.Close()
			found = string(b) == `["saved"]`
		}
	}
	if !found {
		t.Fatal("ZIP didn't preserve restored data")
	}
	if request("GET", "/other/export", nil).Code != 404 {
		t.Fatal("other project export admitted")
	}
	if request("POST", "/p1/restore", map[string]any{"snapshot_id": 999}).Code != 404 {
		t.Fatal("unknown snapshot admitted")
	}
	files["go.mod"] = "module cli-toolkit go 1.27.2"
	if request("PUT", "/p1/files", map[string]any{"files": files}).Code != 422 {
		t.Fatal("broken exported module accepted")
	}
}
