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
	"os"
	"path/filepath"
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
	a.TutorSettings = TutorSettings{DailyBudget: 1000000, MaxOutputTokens: 512}
	handler := a.Handler(t.TempDir())
	request := func(method string, payload any) *httptest.ResponseRecorder {
		if method == "POST" {
			fields, _ := payload.(map[string]any)
			if fields != nil {
				if _, exists := fields["provider_revision"]; !exists {
					_, _, revision := a.TutorSnapshot()
					fields["provider_revision"] = revision
				}
			}
		}
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
	lesson := content.Lesson{ID: "long-public", Title: "Тема", Body: strings.Repeat("PRIVATE_BODY", 5000), LearningOutcomes: []string{"Понять публичную цель"}, Glossary: []content.GlossaryEntry{{Term: "word", Translation: "слово", Definition: "описание"}}}
	context := tutorLessonContext(lesson)
	if len(context) > 24<<10 || !utf8.ValidString(context) || !strings.Contains(context, "PRIVATE_BODY") || !strings.Contains(context, "Целевая версия Go") {
		t.Fatal("unbounded/private/invalid overview")
	}
}

func TestTutorLessonContextContainsBoundedPublicCourseText(t *testing.T) {
	lesson := content.Lesson{ID: "topic", Title: "Тема", Rank: "Junior", Module: "HTTP", GoVersion: "1.27.2", Body: "Полный текст курса", LearningOutcomes: []string{"Понять API"}, Exercises: []content.Exercise{{Skills: []string{"http"}}}, Glossary: []content.GlossaryEntry{{Term: "API", Translation: "интерфейс", Definition: "Публичное описание"}}}
	got := tutorLessonContext(lesson)
	for _, want := range []string{"Полный текст курса", "1.27.2", "Junior / HTTP", "http", "Публичное описание", "Понять API", "источник истины"} {
		if !strings.Contains(got, want) {
			t.Fatalf("lesson context lacks %q", want)
		}
	}
	if strings.Contains(got, "self_check") || len(got) > 24<<10 {
		t.Fatal("private or oversized lesson context")
	}
}

func TestTutorCourseRetrievalIsPublishedBoundedAndCurrentFirst(t *testing.T) {
	catalog, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	current := "juniorplus-09"
	ctx, selected := tutorCourseContext(catalog, "Как выбрать SQL индекс и проверить план запроса", current)
	if len(ctx) > 24<<10 || len(selected) < 2 || selected[0]["id"] != current {
		t.Fatalf("bad current-first retrieval: %d bytes %+v", len(ctx), selected)
	}
	foundSQL := false
	for _, item := range selected {
		if strings.Contains(strings.ToLower(item["title"]), "sql") || strings.Contains(strings.ToLower(item["title"]), "индекс") {
			foundSQL = true
		}
		if _, ok := catalog.FindLesson(item["id"]); !ok {
			t.Fatalf("retrieval included unpublished lesson %q", item["id"])
		}
	}
	if !foundSQL {
		t.Fatalf("query did not retrieve SQL lesson: %+v", selected)
	}
	for _, item := range selected {
		lesson, _ := catalog.FindLesson(item["id"])
		for _, exercise := range lesson.Exercises {
			if source, ok := catalog.Reference(exercise.ID); ok && source != "" && strings.Contains(ctx, source) {
				t.Fatalf("reference solution leaked for %s", exercise.ID)
			}
		}
		for _, answer := range lesson.SelfCheck {
			if answer.Answer != "" && strings.Contains(ctx, answer.Answer) {
				t.Fatal("self-check answer leaked into tutor context")
			}
		}
	}
	generic, items := tutorCourseContext(catalog, "Объясни SQL joins, transaction и индекс", "")
	if len(generic) > 24<<10 || len(items) == 0 || len(items) > 3 {
		t.Fatalf("bad generic retrieval: %d bytes %+v", len(generic), items)
	}
	concurrency, items := tutorCourseContext(catalog, "параллельные транзакции и конкурентные запросы", "")
	if len(items) == 0 || (!strings.Contains(strings.ToLower(concurrency), "транзакц") && !strings.Contains(strings.ToLower(concurrency), "конкурент")) {
		t.Fatalf("concurrency query did not retrieve relevant public material: %+v", items)
	}
}

func TestTutorExplicitContextTogglesAndLimits(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.CreateProfile("private", "light"); err != nil {
		t.Fatal(err)
	}
	catalog, _ := content.Load()
	a := New(s, catalog, nil, "csrf", "127.0.0.1:8080")
	a.TutorSettings = TutorSettings{DailyBudget: 1000000, MaxOutputTokens: 512}
	var sent string
	calls := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		b, _ := io.ReadAll(r.Body)
		sent = string(b)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`)
	}))
	defer provider.Close()
	g, err := tutor.New(tutor.Config{Endpoint: provider.URL, EconomyModel: "test", MaxOutputTokens: 512})
	if err != nil {
		t.Fatal(err)
	}
	a.Tutor = g
	h := a.Handler(t.TempDir())
	request := func(raw string) *httptest.ResponseRecorder {
		var payload map[string]any
		if json.Unmarshal([]byte(raw), &payload) == nil && payload != nil {
			if _, exists := payload["provider_revision"]; !exists {
				_, _, revision := a.TutorSnapshot()
				payload["provider_revision"] = revision
				encoded, _ := json.Marshal(payload)
				raw = string(encoded)
			}
		}
		r := httptest.NewRequest("POST", "http://127.0.0.1:8080/api/tutor", strings.NewReader(raw))
		r.AddCookie(&http.Cookie{Name: "academy_session", Value: "csrf"})
		r.Header.Set("X-CSRF-Token", "csrf")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	base := `{"mode":"debug","message":"помоги","consent":true,"share_code":false,"source":"SOURCE_SECRET","share_diagnostics":false,"diagnostics":{"status":"failed","exit_code":1,"stdout":"","stderr":"DIAGNOSTIC_SECRET","checks":{"total":1,"passed":0,"failed":1}},"share_progress":false,"client_progress":"FORGED"}`
	if w := request(base); w.Code != 400 {
		t.Fatalf("client progress must be rejected: %d %s", w.Code, w.Body)
	}
	base = `{"mode":"debug","message":"помоги","consent":true,"share_code":false,"source":"","share_diagnostics":false,"diagnostics":{"status":"failed","exit_code":1,"stdout":"","stderr":"DIAGNOSTIC_SECRET","checks":{"total":1,"passed":0,"failed":1}},"share_progress":false}`
	if w := request(base); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	for _, secret := range []string{"DIAGNOSTIC_SECRET", "SOURCE_SECRET", "current_rank", "rank_completion"} {
		if strings.Contains(sent, secret) {
			t.Fatalf("unchecked context leaked %s", secret)
		}
	}
	base = `{"mode":"debug","message":"помоги","consent":true,"share_code":true,"source":"password=SOURCE_SECRET","share_diagnostics":true,"diagnostics":{"status":"failed","exit_code":1,"stdout":"","stderr":"token=DIAGNOSTIC_SECRET","checks":{"total":1,"passed":0,"failed":1}},"share_progress":true}`
	if w := request(base); w.Code != 200 {
		t.Fatal(w.Code, w.Body)
	}
	for _, fragment := range []string{"SOURCE_SECRET", "DIAGNOSTIC_SECRET", `"id":"program","level":0`} {
		if strings.Contains(sent, fragment) {
			t.Fatalf("expected redacted or safe progress only, found %q", fragment)
		}
	}
	for _, fragment := range []string{"[скрыто]", "недоверенные данные", "не доказательство запуска", "rank_completion", "available_exercises"} {
		if !strings.Contains(sent, fragment) {
			t.Fatalf("selected context missing %q", fragment)
		}
	}
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
	tooLong := `{"mode":"debug","message":"x","consent":true,"share_diagnostics":true,"diagnostics":{"status":"failed","stdout":"` + strings.Repeat("x", 4097) + `","stderr":"","checks":{"total":0,"passed":0,"failed":0}}}`
	if w := request(tooLong); w.Code != 422 {
		t.Fatalf("oversized diagnostics accepted: %d", w.Code)
	}
	badChecks := `{"mode":"debug","message":"x","consent":true,"share_diagnostics":true,"diagnostics":{"status":"complete","exit_code":0,"stdout":"","stderr":"","checks":{"total":2,"passed":1,"failed":0}}}`
	if w := request(badChecks); w.Code != 422 {
		t.Fatalf("inconsistent checks accepted: %d", w.Code)
	}
	invalidUTF8 := []byte(`{"mode":"debug","message":"x","consent":true,"diagnostics":{"status":"failed","stdout":"","stderr":"`)
	invalidUTF8 = append(invalidUTF8, 0xff)
	invalidUTF8 = append(invalidUTF8, []byte(`","checks":{"total":0,"passed":0,"failed":0}}}`)...)
	r := httptest.NewRequest("POST", "http://127.0.0.1:8080/api/tutor", bytes.NewReader(invalidUTF8))
	r.AddCookie(&http.Cookie{Name: "academy_session", Value: "csrf"})
	r.Header.Set("X-CSRF-Token", "csrf")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("invalid UTF-8 accepted: %d", w.Code)
	}
	if calls != 2 {
		t.Fatal("invalid diagnostics reached provider")
	}
}

func TestTutorLocalProviderConfigRoutesAuthPersistenceAndRestore(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.CreateProfile("local", "light")
	catalog, _ := content.Load()
	a := New(s, catalog, nil, "secret", "127.0.0.1:8080")
	base := tutor.Config{Endpoint: "http://127.0.0.1:11434/v1/chat/completions", EconomyModel: "env-model", MaxOutputTokens: 512}
	g, err := tutor.New(base)
	if err != nil {
		t.Fatal(err)
	}
	a.Tutor = g
	a.TutorSettings = TutorSettings{DailyBudget: 10000, MaxOutputTokens: 512, ProviderLabel: "127.0.0.1", ProviderLocation: "unknown", EconomyModel: "env-model"}
	path := t.TempDir() + "/tutor-provider.json"
	t.Setenv("ACADEMY_TUTOR_CLOUD_PROVIDER_CONFIG", filepath.Join(filepath.Dir(path), "missing-parent", "tutor-cloud-provider.json"))
	if err = a.ConfigureLocalTutor(base, path); err != nil {
		t.Fatal(err)
	}
	h := a.Handler(t.TempDir())
	request := func(method, path, body, csrf string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: "academy_session", Value: "secret"})
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := request("PUT", "/api/tutor/local-provider", `{"endpoint":"http://127.0.0.1:11435/v1/chat/completions","model":"ui-model","advanced_model":"ui-advanced"}`, ""); w.Code != 403 {
		t.Fatalf("CSRF=%d %s", w.Code, w.Body)
	}
	if w := request("PUT", "/api/tutor/local-provider", `{"endpoint":"https://api.example/v1/chat/completions","model":"ui-model"}`, "secret"); w.Code != 422 {
		t.Fatalf("remote provider accepted: %d %s", w.Code, w.Body)
	}
	if w := request("PUT", "/api/tutor/local-provider", `{"endpoint":"http://127.0.0.1:11435/v1/chat/completions","model":"ui-model","api_key":"SECRET"}`, "secret"); w.Code != 400 {
		t.Fatalf("key field accepted: %d %s", w.Code, w.Body)
	}
	put := request("PUT", "/api/tutor/local-provider", `{"endpoint":"http://127.0.0.1:11435/v1","model":"ui-model","advanced_model":"ui-advanced"}`, "secret")
	if put.Code != 200 {
		t.Fatalf("PUT=%d %s", put.Code, put.Body)
	}
	if !strings.Contains(put.Body.String(), "/v1/chat/completions") {
		t.Fatal("/v1 UI endpoint was not normalized")
	}
	active, settings, _ := a.TutorSnapshot()
	if active == nil || settings.ProviderLocation != "local" || settings.EconomyModel != "ui-model" {
		t.Fatalf("active provider/settings not updated: %+v", settings)
	}
	b, err := os.ReadFile(path)
	if err != nil || strings.Contains(string(b), "key") {
		t.Fatalf("persisted config unsafe: %s %v", b, err)
	}
	if w := request("GET", "/api/tutor", "", ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"local_provider_configurable":true`) || !strings.Contains(w.Body.String(), `"model":"ui-model"`) || strings.Contains(w.Body.String(), "SECRET") {
		t.Fatalf("GET config=%d %s", w.Code, w.Body)
	}
	if w := request("PUT", "/api/tutor/local-provider", `{"endpoint":"http://127.0.0.1:11435/v1/chat/completions","model":"`+strings.Repeat("x", 9000)+`"}`, "secret"); w.Code != 400 {
		t.Fatalf("oversized body=%d", w.Code)
	}
	if w := request("DELETE", "/api/tutor/local-provider", "", "secret"); w.Code != 200 {
		t.Fatalf("DELETE=%d %s", w.Code, w.Body)
	}
	_, settings, _ = a.TutorSnapshot()
	if settings.EconomyModel != "env-model" || settings.ProviderLocation != "unknown" {
		t.Fatalf("ENV provider not restored: %+v", settings)
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("persisted config remains: %v", err)
	}
}

func TestTutorCloudProviderConfigurationIsExplicitSecretAndNoProbe(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.CreateProfile("cloud", "light"); err != nil {
		t.Fatal(err)
	}
	catalog, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	a := New(s, catalog, nil, "csrf-secret", "127.0.0.1:8080")
	base := tutor.Config{Endpoint: "http://127.0.0.1:11434/v1/chat/completions", EconomyModel: "env-model", MaxOutputTokens: 512}
	g, err := tutor.New(base)
	if err != nil {
		t.Fatal(err)
	}
	a.Tutor = g
	a.TutorSettings = TutorSettings{DailyBudget: 100000, MaxOutputTokens: 512, ProviderLabel: "127.0.0.1", ProviderLocation: "unknown", EconomyModel: "env-model"}
	path := filepath.Join(t.TempDir(), "tutor-provider.json")
	if err = a.ConfigureLocalTutor(base, path); err != nil {
		t.Fatal(err)
	}
	h := a.Handler(t.TempDir())
	request := func(method, route, body, csrf string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://127.0.0.1:8080"+route, strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: "academy_session", Value: "csrf-secret"})
		if csrf != "" {
			r.Header.Set("X-CSRF-Token", csrf)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for _, bad := range []string{
		`{"endpoint":"http://8.8.8.8/v1","api_key":"secret","model":"m"}`,
		`{"endpoint":"https://127.0.0.1/v1","api_key":"secret","model":"m"}`,
		`{"endpoint":"https://10.0.0.1/v1","api_key":"secret","model":"m"}`,
		`{"endpoint":"https://does-not-exist.invalid/v1","api_key":"secret","model":"m"}`,
		`{"endpoint":"https://8.8.8.8/v1","api_key":"secret"}`,
		`{"endpoint":"https://8.8.8.8/v1","model":"m"}`,
		`{"endpoint":"https://8.8.8.8/v1","api_key":"secret","model":"m","extra":"x"}`,
	} {
		if w := request("PUT", "/api/tutor/cloud-provider", bad, "csrf-secret"); w.Code != 422 && w.Code != 400 {
			t.Errorf("unsafe config status=%d body=%s", w.Code, w.Body)
		}
	}
	if w := request("PUT", "/api/tutor/cloud-provider", `{"endpoint":"https://8.8.8.8/v1","api_key":"secret","model":"cloud-model"}`, ""); w.Code != 403 {
		t.Fatalf("missing CSRF accepted: %d %s", w.Code, w.Body)
	}
	calls := 0 // Saving config must never probe or call the provider.
	_ = calls
	before := request("GET", "/api/tutor", "", "")
	var beforeConfig map[string]any
	if err = json.Unmarshal(before.Body.Bytes(), &beforeConfig); err != nil {
		t.Fatal(err)
	}
	beforeRevision, _ := beforeConfig["provider_revision"].(string)
	if beforeRevision == "" {
		t.Fatal("GET omitted provider revision")
	}
	put := request("PUT", "/api/tutor/cloud-provider", `{"endpoint":"https://8.8.8.8/v1","api_key":"DO_NOT_EXPOSE_KEY","model":"cloud-model","advanced_model":"cloud-reasoner"}`, "csrf-secret")
	if put.Code != 200 {
		t.Fatalf("PUT=%d %s", put.Code, put.Body)
	}
	if strings.Contains(put.Body.String(), "DO_NOT_EXPOSE_KEY") {
		t.Fatal("PUT returned API key")
	}
	cloudPath := filepath.Join(filepath.Dir(path), "tutor-cloud-provider.json")
	b, err := os.ReadFile(cloudPath)
	if err != nil || !strings.Contains(string(b), "DO_NOT_EXPOSE_KEY") {
		t.Fatalf("cloud secret not persisted server-side: err=%v", err)
	}
	info, err := os.Stat(cloudPath)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("cloud config mode=%v err=%v", info, err)
	}
	active, settings, _ := a.TutorSnapshot()
	if active == nil || !settings.Cloud || settings.ProviderLocation != "cloud" || settings.EconomyModel != "cloud-model" {
		t.Fatalf("cloud runtime not active: %+v", settings)
	}
	get := request("GET", "/api/tutor", "", "")
	if get.Code != 200 || !strings.Contains(get.Body.String(), `"cloud_provider_configured":true`) || !strings.Contains(get.Body.String(), "cloud-model") || strings.Contains(get.Body.String(), "DO_NOT_EXPOSE_KEY") {
		t.Fatalf("unsafe GET=%d %s", get.Code, get.Body)
	}
	var activeConfig map[string]any
	if err = json.Unmarshal(get.Body.Bytes(), &activeConfig); err != nil {
		t.Fatal(err)
	}
	if activeConfig["provider_revision"] == beforeRevision {
		t.Fatal("provider switch did not rotate the server revision")
	}
	if w := request("DELETE", "/api/tutor/cloud-provider", "", "csrf-secret"); w.Code != 200 {
		t.Fatalf("DELETE=%d %s", w.Code, w.Body)
	}
	active, settings, _ = a.TutorSnapshot()
	if active == nil || settings.EconomyModel != "env-model" || settings.ProviderLocation != "unknown" {
		t.Fatalf("ENV provider not restored: %+v", settings)
	}
	if _, err = os.Stat(cloudPath); !os.IsNotExist(err) {
		t.Fatalf("cloud config not deleted: %v", err)
	}
}

func TestTutorStaleProviderConsentRejectedBeforeBudgetAndCall(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.CreateProfile("local", "light"); err != nil {
		t.Fatal(err)
	}
	catalog, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	a := New(s, catalog, nil, "csrf", "127.0.0.1:8080")
	a.TutorSettings = TutorSettings{DailyBudget: 1000000, MaxOutputTokens: 128, ProviderLocation: "local"}
	calls := 0
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`)
	}))
	defer provider.Close()
	g, err := tutor.New(tutor.Config{Endpoint: provider.URL, EconomyModel: "test", MaxOutputTokens: 128})
	if err != nil {
		t.Fatal(err)
	}
	a.SetTutorRuntime(g, a.TutorSettings)
	h := a.Handler(t.TempDir())
	request := func(method, raw string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://127.0.0.1:8080/api/tutor", strings.NewReader(raw))
		r.AddCookie(&http.Cookie{Name: "academy_session", Value: "csrf"})
		if method != "GET" {
			r.Header.Set("X-CSRF-Token", "csrf")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	var oldConfig map[string]any
	if w := request("GET", ""); w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &oldConfig) != nil {
		t.Fatalf("initial GET=%d %s", w.Code, w.Body)
	}
	oldRevision := oldConfig["provider_revision"].(string)
	a.SetTutorRuntime(g, a.TutorSettings)
	var currentConfig map[string]any
	if w := request("GET", ""); w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &currentConfig) != nil {
		t.Fatalf("current GET=%d %s", w.Code, w.Body)
	}
	currentRevision := currentConfig["provider_revision"].(string)
	if currentRevision == oldRevision {
		t.Fatal("provider runtime revision did not change")
	}
	stale := map[string]any{"mode": "explain", "message": "объясни Go", "consent": true, "provider_revision": oldRevision}
	b, _ := json.Marshal(stale)
	if w := request("POST", string(b)); w.Code != 409 {
		t.Fatalf("stale provider consent=%d %s", w.Code, w.Body)
	}
	usage, err := s.AIUsage(time.Now().UTC().Format("2006-01-02"))
	if err != nil || usage.Requests != 0 || usage.ReservedTokens != 0 || calls != 0 {
		t.Fatalf("stale provider request consumed budget or called provider: usage=%+v calls=%d err=%v", usage, calls, err)
	}
	stale["provider_revision"] = currentRevision
	b, _ = json.Marshal(stale)
	if w := request("POST", string(b)); w.Code != 200 {
		t.Fatalf("current provider consent=%d %s", w.Code, w.Body)
	}
	if calls != 1 {
		t.Fatalf("current revision call count=%d", calls)
	}
}

func TestTutorCloudSwitchFailureRestoresPreviousSecretAndRuntime(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.CreateProfile("learner", "light"); err != nil {
		t.Fatal(err)
	}
	catalog, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	a := New(s, catalog, nil, "csrf", "127.0.0.1:8080")
	a.TutorSettings = TutorSettings{DailyBudget: 100000, MaxOutputTokens: 128}
	configPath := filepath.Join(t.TempDir(), "tutor-provider.json")
	if err = a.ConfigureLocalTutor(tutor.Config{}, configPath); err != nil {
		t.Fatal(err)
	}
	h := a.Handler(t.TempDir())
	request := func(body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("PUT", "http://127.0.0.1:8080/api/tutor/cloud-provider", strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: "academy_session", Value: "csrf"})
		r.Header.Set("X-CSRF-Token", "csrf")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := request(`{"endpoint":"https://8.8.8.8/v1","api_key":"KEEP_PREVIOUS_KEY","model":"cloud-a"}`); w.Code != 200 {
		t.Fatalf("first cloud config=%d %s", w.Code, w.Body)
	}
	rt := a.localTutorRuntime()
	rt.mu.Lock()
	badLocalPath := filepath.Join(t.TempDir(), "nonempty-directory")
	if err = os.Mkdir(badLocalPath, 0700); err != nil {
		rt.mu.Unlock()
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(badLocalPath, "keep"), []byte("test"), 0600); err != nil {
		rt.mu.Unlock()
		t.Fatal(err)
	}
	rt.path = badLocalPath
	rt.mu.Unlock()
	if w := request(`{"endpoint":"https://8.8.8.8/v1","api_key":"REPLACEMENT_KEY","model":"cloud-b"}`); w.Code != 500 {
		t.Fatalf("failed transition status=%d %s", w.Code, w.Body)
	}
	_, settings, _ := a.TutorSnapshot()
	if settings.EconomyModel != "cloud-a" {
		t.Fatalf("failed transition changed runtime: %+v", settings)
	}
	cloudPath := filepath.Join(filepath.Dir(configPath), "tutor-cloud-provider.json")
	persisted, err := tutor.LoadCloudProvider(cloudPath)
	if err != nil || persisted.Model != "cloud-a" || persisted.APIKey != "KEEP_PREVIOUS_KEY" {
		t.Fatalf("failed transition lost old persisted provider: %+v err=%v", persisted, err)
	}
}
