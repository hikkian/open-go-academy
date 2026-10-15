package httpapi

import (
	"academy/internal/content"
	"academy/internal/projectrunner"
	"academy/internal/projects"
	"academy/internal/runner"
	"academy/internal/store"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type p2Executor struct{ calls []projectrunner.Request }

func (p *p2Executor) Available(context.Context) bool { return true }
func (p *p2Executor) Execute(_ context.Context, request projectrunner.Request) (runner.Result, error) {
	p.calls = append(p.calls, request)
	return runner.Result{Status: "automatic_pass", Passed: request.Action == "submit", Checks: []runner.Check{{Name: "fixture", Passed: true}}}, nil
}

func TestP2RoutesSeparateWorkerPolicyAndEvidence(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.CreateProfile("Ada", "light")
	c, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	a := New(s, c, nil, "secret", "127.0.0.1:8080")
	worker := &p2Executor{}
	a.ProjectRunner = worker
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
	p, _ := projects.Find("p2")
	files := p.StarterFiles
	files["internal/helper.go"] = "package internal\n"
	files["Dockerfile"] = "export-only configuration"
	if w := request("PUT", "/p2/files", map[string]any{"files": files}); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	snapshots, _ := s.ProjectSnapshots("p2")
	if len(snapshots) != 1 {
		t.Fatal("P2 snapshot missing")
	}
	if w := request("POST", "/p1/restore", map[string]any{"snapshot_id": snapshots[0].ID}); w.Code != 404 {
		t.Fatal("cross-project snapshot restored")
	}
	if w := request("POST", "/p2/submit", map[string]any{"files": files}); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if len(worker.calls) != 1 || worker.calls[0].ProjectID != "p2" {
		t.Fatal("wrong execution boundary")
	}
	for _, name := range []string{"Dockerfile", "compose.yaml"} {
		if _, ok := worker.calls[0].Files[name]; ok {
			t.Fatal("export configuration reached worker", name)
		}
	}
	if worker.calls[0].Files["go.mod"] != files["go.mod"] || worker.calls[0].Files["internal/helper.go"] == "" {
		t.Fatal("policy/source lost")
	}
	files["go.mod"] += "\nreplace example.test/invalid => /host\n"
	if w := request("POST", "/p2/submit", map[string]any{"files": files}); w.Code != 422 {
		t.Fatal("modified dependency policy accepted")
	}
	if len(worker.calls) != 1 {
		t.Fatal("invalid policy reached worker")
	}
	var awards int
	if err := s.DB().QueryRow("SELECT COUNT(*) FROM rank_awards").Scan(&awards); err != nil || awards != 0 {
		t.Fatal("project assigned rank", err)
	}
	a.ProjectRunner = nil
	if w := request("GET", "/p2", nil); w.Code != 200 || !bytes.Contains(w.Body.Bytes(), []byte(`"execution_available":false`)) {
		t.Fatal("unavailable worker not represented", w.Body.String())
	}
}
