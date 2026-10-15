package httpapi

import (
	"academy/internal/content"
	"academy/internal/runner"
	"academy/internal/store"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type functionExecutor struct {
	request runner.Request
	result  runner.Result
}

func (f *functionExecutor) Available(context.Context) bool { return true }
func (f *functionExecutor) Execute(_ context.Context, q runner.Request) (runner.Result, error) {
	f.request = q
	return f.result, nil
}
func TestFunctionPublicDiagnosticsPrivateOutputAndXP(t *testing.T) {
	catalog, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	id := "zero-02-easy"
	source, ok := catalog.Reference(id)
	if !ok {
		t.Fatal("published function reference missing")
	}
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.CreateProfile("Ученик", "light")
	executor := &functionExecutor{result: runner.Result{Status: "error", ExitCode: 1, Stdout: "PRIVATE_MARKER", Stderr: "PRIVATE_MARKER", Files: map[string]string{"hidden.go": "PRIVATE_MARKER"}}}
	a := New(s, catalog, executor, "secret", "127.0.0.1:8080")
	handler := a.Handler(t.TempDir())
	request := func(action string) *httptest.ResponseRecorder {
		raw, _ := json.Marshal(runner.Request{Files: map[string]string{"main.go": source}, Args: []string{"CLIENT_ARGUMENT"}, Assets: map[string]string{"input.json": "CLIENT_ASSET"}})
		r := httptest.NewRequest("POST", "http://127.0.0.1:8080/api/exercises/"+id+"/"+action, bytes.NewReader(raw))
		r.AddCookie(&http.Cookie{Name: "academy_session", Value: "secret"})
		r.Header.Set("X-CSRF-Token", "secret")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	w := request("submit")
	if w.Code != 200 || strings.Contains(w.Body.String(), "PRIVATE_MARKER") {
		t.Fatal(w.Code, w.Body)
	}
	if len(executor.request.Args) != 1 || strings.Contains(executor.request.Args[0], "CLIENT_ARGUMENT") || len(executor.request.Assets) != 0 || executor.request.Files["academy_driver.go"] == "" {
		t.Fatal("untrusted request parameters survived", executor.request)
	}
	p, _ := s.Progress()
	if p.XP != 0 || len(p.Attempts) != 1 || strings.Contains(p.Attempts[0].Output, "PRIVATE_MARKER") {
		t.Fatal("private output persisted or failure earned XP", p)
	}
	executor.result = runner.Result{Status: "error", ExitCode: 1, Stdout: "PUBLIC_OUTPUT", Stderr: "PUBLIC_COMPILER_DIAGNOSTIC"}
	w = request("run")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "PUBLIC_OUTPUT") || !strings.Contains(w.Body.String(), "PUBLIC_COMPILER_DIAGNOSTIC") {
		t.Fatal(w.Code, w.Body)
	}
	var vectors struct {
		Public  []json.RawMessage `json:"public_expected"`
		Private []json.RawMessage `json:"private_expected"`
	}
	b, err := os.ReadFile("../grading/specs/" + id + ".json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &vectors); err != nil {
		t.Fatal(err)
	}
	var output strings.Builder
	for _, v := range append(vectors.Public, vectors.Private...) {
		var compact bytes.Buffer
		if err = json.Compact(&compact, v); err != nil {
			t.Fatal(err)
		}
		output.Write(compact.Bytes())
		output.WriteByte('\n')
	}
	executor.result = runner.Result{Status: "ok", ExitCode: 0, Stdout: output.String()}
	for range 2 {
		w = request("submit")
		if w.Code != 200 || !strings.Contains(w.Body.String(), `"passed":true`) {
			t.Fatal(w.Code, w.Body)
		}
	}
	p, _ = s.Progress()
	if p.XP != 10 || p.Evidence[id] == 0 {
		t.Fatal("function XP/evidence not server-derived or replay-safe", p)
	}
}
