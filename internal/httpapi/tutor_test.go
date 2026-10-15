package httpapi

import (
	"academy/internal/content"
	"academy/internal/store"
	"academy/internal/tutor"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestTutorConsentBudgetContextAndNoRankAuthority(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.CreateProfile("PRIVATE_PROFILE_NAME", "light")
	catalog, _ := content.Load()
	a := New(s, catalog, nil, "test-secret", "127.0.0.1:8080")
	a.TutorSettings = TutorSettings{DailyBudget: 100000, MaxOutputTokens: 512}
	handler := a.Handler(t.TempDir())
	request := func(method string, payload any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(payload)
		r := httptest.NewRequest(method, "http://127.0.0.1:8080/api/tutor", bytes.NewReader(b))
		r.AddCookie(&http.Cookie{Name: "academy_session", Value: "test-secret"})
		r.Header.Set("X-CSRF-Token", "test-secret")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := request("GET", nil); w.Code != 200 || !strings.Contains(w.Body.String(), `"enabled":false`) {
		t.Fatal(w.Code, w.Body)
	}
	valid := map[string]any{"mode": "hint", "message": "Как работает функция? password=SECRET_PASSWORD", "consent": true, "share_code": false, "lesson_id": "zero-01"}
	if w := request("POST", valid); w.Code != 503 {
		t.Fatal(w.Code, w.Body)
	}
	calls := 0
	var sent string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		b, _ := io.ReadAll(r.Body)
		sent = string(b)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"content":"Проверь условие цикла."}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`)
	}))
	defer provider.Close()
	g, err := tutor.New(tutor.Config{Endpoint: provider.URL, EconomyModel: "test-model", MaxOutputTokens: 512})
	if err != nil {
		t.Fatal(err)
	}
	a.Tutor = g
	valid["consent"] = false
	if w := request("POST", valid); w.Code != 422 {
		t.Fatal(w.Code, w.Body)
	}
	if calls != 0 {
		t.Fatal("request without consent reached provider")
	}
	valid["consent"] = true
	valid["source"] = "PRIVATE_SOURCE"
	if w := request("POST", valid); w.Code != 422 {
		t.Fatal(w.Code, w.Body)
	}
	delete(valid, "source")
	if w := request("POST", valid); w.Code != 200 || !strings.Contains(w.Body.String(), "Проверь условие цикла.") {
		t.Fatal(w.Code, w.Body)
	}
	for _, forbidden := range []string{"PRIVATE_PROFILE_NAME", "PRIVATE_SOURCE", "SECRET_PASSWORD", "TaskCases", "criteria_hash", "attempt_token"} {
		if strings.Contains(sent, forbidden) {
			t.Fatalf("context leaked %s", forbidden)
		}
	}
	if !strings.Contains(sent, "zero-01") {
		t.Fatal("public lesson context missing")
	}
	if calls != 1 {
		t.Fatal(calls)
	}
	u, err := s.AIUsage(time.Now().UTC().Format("2006-01-02"))
	if err != nil || u.UsedTokens != 15 || u.ReservedTokens != 0 {
		t.Fatal(u, err)
	}
	p, _ := s.Profile()
	awards, _ := s.RankAwards("local")
	if p.XP != 0 || len(awards) != 0 {
		t.Fatal("tutor changed learning authority", p, awards)
	}
	a.TutorSettings.DailyBudget = 1
	if w := request("POST", valid); w.Code != 429 {
		t.Fatal(w.Code, w.Body)
	}
	if calls != 1 {
		t.Fatal("quota bypass reached provider")
	}
	valid["rank"] = "Senior"
	if w := request("POST", valid); w.Code != 400 {
		t.Fatal(w.Code, w.Body)
	}
}

func TestTutorLongLessonUsesBoundedPublicOverview(t *testing.T) {
	lesson := content.Lesson{ID: "long-public", Title: "Тема", Body: strings.Repeat("PRIVATE_BODY", 5000), LearningOutcomes: []string{"Понять публичную цель"}, Glossary: []content.GlossaryEntry{{Term: "word", Translation: "слово", Definition: strings.Repeat("я", 20000)}}}
	context := tutorLessonContext(lesson)
	if len(context) > 16384 || !utf8.ValidString(context) || strings.Contains(context, "PRIVATE_BODY") || !strings.Contains(context, "Понять публичную цель") {
		t.Fatal("unbounded/private/invalid overview")
	}
}
