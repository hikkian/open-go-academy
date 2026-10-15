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
	"time"
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

func TestPrivatePassportUsesVerifiedEvidenceDatesActivityAndProjectCatalog(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.CreateProfile("Ada", "light"); err != nil {
		t.Fatal(err)
	}
	catalog, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	h := New(s, catalog, nil, "secret", "127.0.0.1:8080").Handler(t.TempDir())
	request := func(path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8080"+path, nil)
		r.AddCookie(&http.Cookie{Name: "academy_session", Value: "secret"})
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	firstAt := "2026-02-10T09:30:00Z"
	first, err := s.RecordAttempt("hello", map[string]string{"main.go": "package main"}, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB().Exec(`UPDATE attempts SET created_at=? WHERE id=?`, firstAt, first.ID); err != nil {
		t.Fatal(err)
	}
	type achievement struct {
		ID       string    `json:"id"`
		EarnedAt time.Time `json:"earned_at"`
	}
	type project struct {
		ID                   string `json:"id"`
		Title                string `json:"title"`
		AutomaticPassed      bool   `json:"automatic_passed"`
		ManualReviewRequired bool   `json:"manual_review_required"`
	}
	type activityDay struct {
		Date            string `json:"date"`
		VerifiedCount   int    `json:"verified_count"`
		CumulativeCount int    `json:"cumulative_count"`
	}
	type privatePassport struct {
		Achievements []achievement `json:"achievements"`
		Projects     []project     `json:"projects"`
		Activity     []activityDay `json:"activity"`
	}
	decode := func() privatePassport {
		w := request("/api/passport")
		if w.Code != 200 {
			t.Fatalf("GET private passport: %d %s", w.Code, w.Body)
		}
		var got privatePassport
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		return got
	}
	before := decode()
	if len(before.Projects) != 1 || before.Projects[0].ID != "p1" || before.Projects[0].Title != "Go CLI Toolkit" || before.Projects[0].AutomaticPassed || !before.Projects[0].ManualReviewRequired {
		t.Fatalf("project before completion=%+v", before.Projects)
	}
	if len(before.Activity) != 1 || before.Activity[0] != (activityDay{Date: "2026-02-10", VerifiedCount: 1, CumulativeCount: 1}) {
		t.Fatalf("activity before P1=%+v", before.Activity)
	}
	if len(before.Achievements) != 1 || before.Achievements[0].ID != "hello" || !before.Achievements[0].EarnedAt.Equal(time.Date(2026, 2, 10, 9, 30, 0, 0, time.UTC)) {
		t.Fatalf("badge date=%+v", before.Achievements)
	}
	secondAt := "2026-02-12T17:45:00Z"
	second, err := s.RecordAttempt("p1:automatic", map[string]string{"main.go": "package main"}, true, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.DB().Exec(`UPDATE attempts SET created_at=? WHERE id=?`, secondAt, second.ID); err != nil {
		t.Fatal(err)
	}
	after := decode()
	if len(after.Projects) != 1 || !after.Projects[0].AutomaticPassed || !after.Projects[0].ManualReviewRequired {
		t.Fatalf("project after P1=%+v", after.Projects)
	}
	if len(after.Activity) != 2 || after.Activity[0] != (activityDay{Date: "2026-02-10", VerifiedCount: 1, CumulativeCount: 1}) || after.Activity[1] != (activityDay{Date: "2026-02-12", VerifiedCount: 1, CumulativeCount: 2}) {
		t.Fatalf("activity after P1=%+v", after.Activity)
	}
	if len(after.Achievements) != 2 || after.Achievements[1].ID != "p1:automatic" || !after.Achievements[1].EarnedAt.Equal(time.Date(2026, 2, 12, 17, 45, 0, 0, time.UTC)) {
		t.Fatalf("P1 badge date=%+v", after.Achievements)
	}
	if err = s.SetPassportPublic("local", true); err != nil {
		t.Fatal(err)
	}
	public := request("/api/passport/public")
	if public.Code != 200 {
		t.Fatalf("GET public passport: %d %s", public.Code, public.Body)
	}
	for _, privateOnly := range []string{`"achievements"`, `"projects"`, `"activity"`} {
		if strings.Contains(public.Body.String(), privateOnly) {
			t.Fatalf("private field %s leaked into public passport: %s", privateOnly, public.Body)
		}
	}
}
