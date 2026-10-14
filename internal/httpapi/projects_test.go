package httpapi

import (
	"academy/internal/content"
	"academy/internal/projects"
	"academy/internal/store"
	"archive/zip"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
