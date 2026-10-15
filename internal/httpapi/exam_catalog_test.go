package httpapi

import (
	"academy/internal/assessment"
	"academy/internal/content"
	"academy/internal/credentials"
	"academy/internal/runner"
	"academy/internal/store"
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type catalogFixture struct {
	api     *API
	store   *store.Store
	request func(string, string, any) *httptest.ResponseRecorder
}

func newCatalogFixture(t *testing.T, executor Executor) catalogFixture {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	if _, err = s.CreateProfile("Exam fixture", "light"); err != nil {
		t.Fatal(err)
	}
	c, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	a := New(s, c, executor, "secret", "127.0.0.1:8080")
	_, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	a.IssuerKey = key
	h := a.Handler(t.TempDir())
	return catalogFixture{a, s, func(method, path string, body any) *httptest.ResponseRecorder {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, bytes.NewReader(b))
		r.AddCookie(&http.Cookie{Name: "academy_session", Value: "secret"})
		r.Header.Set("X-CSRF-Token", "secret")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}}
}
func seedExamExercises(t *testing.T, s *store.Store, e *assessment.Exam) {
	t.Helper()
	for _, id := range e.Prerequisites {
		if _, err := s.RecordAttempt(id, map[string]string{"main.go": "trusted prerequisite fixture"}, true, ""); err != nil {
			t.Fatal(err)
		}
	}
}

// This creates only a test fixture for the already independently tested Zero
// assessment. New Junior/Junior+ decisions are exercised through the real API.
func seedPreviousZeroAward(t *testing.T, s *store.Store) {
	t.Helper()
	id := "fixture-previous-zero"
	if err := s.RegisterExam(id, "1", "Zero", "fixture-policy"); err != nil {
		t.Fatal(err)
	}
	a, err := s.StartExam("local", id, "1", tokenHash("fixture-token"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ClaimExamAttempt("local", id, a.ID, tokenHash("fixture-token"), 0, "{}", "{}"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.FinishExamAttempt("local", a.ID, 100, true, "[]"); err != nil {
		t.Fatal(err)
	}
}
func startCatalogExam(t *testing.T, f catalogFixture, id string) struct {
	ID       string `json:"id"`
	Token    string `json:"attempt_token"`
	Revision int64  `json:"revision"`
} {
	t.Helper()
	var created struct {
		ID       string `json:"id"`
		Token    string `json:"attempt_token"`
		Revision int64  `json:"revision"`
	}
	w := f.request("POST", "/api/exams/"+id+"/attempts", map[string]any{})
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	return created
}
func TestNewExamRankPrerequisiteAndCrossExamDraft(t *testing.T) {
	f := newCatalogFixture(t, nil)
	e, _ := assessment.Find("junior-core", "")
	seedExamExercises(t, f.store, e)
	f.store.DB().Exec("UPDATE profiles SET xp=999999")
	if w := f.request("POST", "/api/exams/junior-core/attempts", map[string]any{}); w.Code != 409 {
		t.Fatal("XP/completed exercises bypassed previous rank", w.Code)
	}
	seedPreviousZeroAward(t, f.store)
	created := startCatalogExam(t, f, e.ID)
	other, _ := assessment.Find("juniorplus-api", "")
	payload := map[string]any{"attempt_token": created.Token, "revision": 0, "answers": examAnswers(other), "codes": map[string]string{}}
	if w := f.request("PUT", "/api/exams/attempts/"+created.ID+"/draft", payload); w.Code != 422 {
		t.Fatal("answers from another exam accepted", w.Code, w.Body.String())
	}
	catalogResponse := f.request("GET", "/api/exams", nil)
	var catalog []map[string]any
	if json.Unmarshal(catalogResponse.Body.Bytes(), &catalog) != nil || len(catalog) != 3 {
		t.Fatal("incomplete public exam catalog")
	}
	payload = map[string]any{"attempt_token": strings.Repeat("a", 64), "revision": 0, "answers": map[string]string{}, "codes": map[string]string{}}
	if w := f.request("PUT", "/api/exams/attempts/"+created.ID+"/draft", payload); w.Code != 409 {
		t.Fatal("wrong nonce saved exam", w.Code)
	}
	if _, err := f.store.DB().Exec("UPDATE exam_attempts SET criteria_hash='changed' WHERE id=?", created.ID); err != nil {
		t.Fatal(err)
	}
	payload["attempt_token"] = created.Token
	if w := f.request("POST", "/api/exams/attempts/"+created.ID+"/submit", payload); w.Code != 409 {
		t.Fatal("changed policy accepted", w.Code)
	}
	if w := f.request("GET", "/api/exams/junior-core?version=99", nil); w.Code != 404 {
		t.Fatal("unknown immutable version resolved")
	}
}

type privateExamOutput struct{}

func (privateExamOutput) Available(context.Context) bool { return true }
func (privateExamOutput) Execute(context.Context, runner.Request) (runner.Result, error) {
	return runner.Result{Status: "error", ExitCode: 1, Stdout: "PRIVATE_CASE_MARKER", Stderr: "PRIVATE_CASE_MARKER"}, nil
}
func TestLegacyExamPrivateOutputNeverLeavesGrading(t *testing.T) {
	f := newCatalogFixture(t, privateExamOutput{})
	e := assessment.Load()
	seedExamExercises(t, f.store, e)
	created := startCatalogExam(t, f, e.ID)
	payload := map[string]any{"attempt_token": created.Token, "revision": 0, "answers": examAnswers(e), "codes": map[string]string{"coding": "package main\nfunc sumEven(n int) int{return 0}", "debugging": "package main\nfunc clampPercent(n int) int{return 0}"}}
	w := f.request("POST", "/api/exams/attempts/"+created.ID+"/submit", payload)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	a, err := f.store.ExamAttempt("local", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(w.Body.String(), "PRIVATE_CASE_MARKER") || strings.Contains(a.ChecksJSON, "PRIVATE_CASE_MARKER") || a.Passed {
		t.Fatal("private execution output leaked or failed exam awarded")
	}
}

func TestGenericExamPrivateOutputNeverLeavesGrading(t *testing.T) {
	f := newCatalogFixture(t, privateExamOutput{})
	e, _ := assessment.Find("junior-core", "")
	seedExamExercises(t, f.store, e)
	seedPreviousZeroAward(t, f.store)
	created := startCatalogExam(t, f, e.ID)
	codes := map[string]string{}
	for task, path := range map[string]string{"transitions": "countTransitions", "range": "clampRange", "unicode": "firstUniqueRune"} {
		b, err := os.ReadFile(filepath.Join("../../content/exams/junior-core/reference", path, "main.go"))
		if err != nil {
			t.Fatal(err)
		}
		codes[task] = string(b)
	}
	w := f.request("POST", "/api/exams/attempts/"+created.ID+"/submit", map[string]any{"attempt_token": created.Token, "revision": 0, "answers": examAnswers(e), "codes": codes})
	record, err := f.store.ExamAttempt("local", created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || record.Passed || strings.Contains(w.Body.String(), "PRIVATE_CASE_MARKER") || strings.Contains(record.ChecksJSON, "PRIVATE_CASE_MARKER") {
		t.Fatal("generic private output escaped or failed exam accepted", w.Code)
	}
}

func TestNativeJuniorAndJuniorPlusAssessmentChain(t *testing.T) {
	if os.Getenv("ACADEMY_EXAM_CATALOG_NATIVE") != "1" {
		t.Skip("requires the real separate confined Go runner")
	}
	executor := runner.NewClient("../../.data/runner.sock")
	if !executor.Available(context.Background()) {
		t.Fatal("native runner unavailable")
	}
	f := newCatalogFixture(t, executor)
	seedPreviousZeroAward(t, f.store)
	paths := map[string]map[string]string{"junior-core": {"transitions": "countTransitions", "range": "clampRange", "unicode": "firstUniqueRune"}, "juniorplus-api": {"redirect": "safeRedirectPath", "retry": "retryableStatus", "access": "canAccessResource"}}
	for _, id := range []string{"junior-core", "juniorplus-api"} {
		e, _ := assessment.Find(id, "")
		seedExamExercises(t, f.store, e)
		created := startCatalogExam(t, f, id)
		codes := map[string]string{}
		for task, path := range paths[id] {
			b, err := os.ReadFile(filepath.Join("../../content/exams", id, "reference", path, "main.go"))
			if err != nil {
				t.Fatal(err)
			}
			codes[task] = string(b)
		}
		payload := map[string]any{"attempt_token": created.Token, "revision": 0, "answers": examAnswers(e), "codes": codes}
		w := f.request("POST", "/api/exams/attempts/"+created.ID+"/submit", payload)
		if w.Code != 200 {
			t.Fatal(id, w.Code, w.Body.String())
		}
		var result struct {
			Passed      bool   `json:"passed"`
			RankAwarded bool   `json:"rank_awarded"`
			Rank        string `json:"rank"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if !result.Passed || !result.RankAwarded || result.Rank != e.Rank {
			t.Fatal(id, "native reference failed", w.Body.String())
		}
		if closedDraft := f.request("PUT", "/api/exams/attempts/"+created.ID+"/draft", payload); closedDraft.Code != 409 {
			t.Fatal("completed draft modified", closedDraft.Code)
		}
		if replay := f.request("POST", "/api/exams/attempts/"+created.ID+"/submit", payload); replay.Code != 409 {
			t.Fatal("completed new exam replay admitted", replay.Code)
		}
		if credential := f.request("POST", "/api/credentials", map[string]any{"attempt_id": created.ID, "name": "Проверка экзамена", "consent": true}); credential.Code != 201 || !strings.Contains(credential.Body.String(), `"rank":"`+e.Rank+`"`) {
			t.Fatal("new rank credential failed", credential.Code, credential.Body.String())
		}
		t.Log("PASS actual isolated", id, "grading, atomic rank, replay rejection and credential; prerequisites/previous Zero are explicit test fixtures")
	}
	awards, err := f.store.RankAwards("local")
	if err != nil || len(awards) != 3 {
		t.Fatal("rank chain not persisted", err, len(awards))
	}
}

func TestAllExamCriteriaFitSignedCredentialFormat(t *testing.T) {
	_, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, exam := range assessment.All() {
		id := strings.Repeat("a", 32)
		_, err := credentials.Sign(key, credentials.Payload{ID: id, IssuerName: "Open Go Academy", SubjectName: "Тест", Rank: exam.Rank, ExamID: exam.ID, CriteriaVersion: exam.Version, CriteriaHash: exam.Hash(), AttemptID: strings.Repeat("b", 32), IssuedAt: time.Now().UTC().Format(time.RFC3339Nano), VerifyURL: "http://127.0.0.1:8380/verify/" + id, Competencies: exam.Criteria})
		if err != nil {
			t.Fatal("published criteria cannot be signed", exam.ID, err)
		}
	}
}
