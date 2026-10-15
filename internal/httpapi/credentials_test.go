package httpapi

import (
	"academy/internal/assessment"
	"academy/internal/content"
	"academy/internal/credentials"
	"academy/internal/store"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCredentialIssueForgeryPrivacyRevokeAndImport(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.CreateProfile("PRIVATE_NAME", "light")
	catalog, _ := content.Load()
	a := New(s, catalog, nil, "secret", "127.0.0.1:8080")
	_, key, _ := ed25519.GenerateKey(rand.Reader)
	a.IssuerKey = key
	h := a.Handler(t.TempDir())
	request := func(method, path string, body any, auth bool) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		r := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, bytes.NewReader(b))
		if auth {
			r.AddCookie(&http.Cookie{Name: "academy_session", Value: "secret"})
			r.Header.Set("X-CSRF-Token", "secret")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	exam := assessment.Load()
	draft := map[string]any{"attempt_id": "missing", "name": "Псевдоним", "consent": true}
	if w := request("POST", "/api/credentials", draft, false); w.Code != 401 && w.Code != 403 {
		t.Fatal(w.Code, w.Body)
	}
	if w := request("POST", "/api/credentials", draft, true); w.Code != 404 {
		t.Fatal(w.Code, w.Body)
	}
	for _, id := range exam.Prerequisites {
		if _, err = s.RecordAttempt(id, map[string]string{"main.go": "trusted test fixture"}, true, ""); err != nil {
			t.Fatal(err)
		}
	}
	attempt, err := s.StartExam("local", exam.ID, exam.Version, tokenHash(strings.Repeat("a", 64)), exam.Prerequisites)
	if err != nil {
		t.Fatal(err)
	}
	draft["attempt_id"] = attempt.ID
	if w := request("POST", "/api/credentials", draft, true); w.Code != 409 {
		t.Fatal(w.Code, w.Body)
	}
	_, err = s.ClaimExamAttempt("local", exam.ID, attempt.ID, tokenHash(strings.Repeat("a", 64)), 0, `{}`, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.FinishExamAttempt("local", attempt.ID, 100, true, `[]`)
	if err != nil {
		t.Fatal(err)
	}
	draft["consent"] = false
	if w := request("POST", "/api/credentials", draft, true); w.Code != 422 {
		t.Fatal(w.Code, w.Body)
	}
	draft["consent"] = true
	w := request("POST", "/api/credentials", draft, true)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body)
	}
	var summary struct {
		ID        string `json:"id"`
		Published bool   `json:"published"`
	}
	json.Unmarshal(w.Body.Bytes(), &summary)
	if summary.Published {
		t.Fatal("private by default")
	}
	id := summary.ID
	if w := request("POST", "/api/credentials", draft, true); w.Code != 409 {
		t.Fatal(w.Code, w.Body)
	}
	if w := request("GET", "/api/verify/"+id, nil, false); w.Code != 404 || strings.Contains(w.Body.String(), "Псевдоним") {
		t.Fatal(w.Code, w.Body)
	}
	if w := request("PUT", "/api/credentials/"+id+"/privacy", map[string]any{"published": true, "consent": false}, true); w.Code != 422 {
		t.Fatal(w.Code, w.Body)
	}
	if w := request("PUT", "/api/credentials/"+id+"/privacy", map[string]any{"published": true, "consent": true}, true); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	if w := request("GET", "/api/verify/"+id, nil, false); w.Code != 200 || !strings.Contains(w.Body.String(), `"signature_valid":true`) || !strings.Contains(w.Body.String(), `"status":"active"`) {
		t.Fatal(w.Code, w.Body)
	}
	w = request("GET", "/api/credentials/"+id+"/json", nil, true)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	var envelope credentials.Envelope
	json.Unmarshal(w.Body.Bytes(), &envelope)
	if err := credentials.Verify(key.Public().(ed25519.PublicKey), envelope); err != nil {
		t.Fatal(err)
	}
	envelope.Payload.Rank = "Senior"
	if w := request("POST", "/api/credentials/check", envelope, true); w.Code != 200 || !strings.Contains(w.Body.String(), `"signature_valid":false`) {
		t.Fatal(w.Code, w.Body)
	}
	if w := request("POST", "/api/credentials/"+id+"/revoke", map[string]any{}, true); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	if w := request("GET", "/api/verify/"+id, nil, false); w.Code != 200 || !strings.Contains(w.Body.String(), `"status":"revoked"`) {
		t.Fatal(w.Code, w.Body)
	}
	if w := request("GET", "/api/passport/public", nil, false); w.Code != 404 {
		t.Fatal(w.Code, w.Body)
	}
	if w := request("PUT", "/api/passport/privacy", map[string]any{"published": true, "consent": true}, true); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	if w := request("GET", "/api/passport/public", nil, false); w.Code != 200 || !strings.Contains(w.Body.String(), "PRIVATE_NAME") {
		t.Fatal(w.Code, w.Body)
	}
	if w := request("PUT", "/api/credentials/"+id+"/privacy", map[string]any{"published": false}, true); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	if w := request("GET", "/api/verify/"+id, nil, false); w.Code != 404 {
		t.Fatal(w.Code, w.Body)
	}
}
