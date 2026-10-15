package httpapi

import (
	"academy/internal/assessment"
	"academy/internal/content"
	"academy/internal/runner"
	"academy/internal/store"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type examExecutor struct{ calls int }

func (*examExecutor) Available(context.Context) bool { return true }
func (e *examExecutor) Execute(_ context.Context, q runner.Request) (runner.Result, error) {
	e.calls++
	n, _ := strconv.Atoi(q.Args[0])
	value := 0
	if strings.Contains(q.Files["solution.go"], "sumEven") {
		for i := 0; i <= n; i++ {
			if i%2 == 0 {
				value += i
			}
		}
	} else {
		value = n
		if value < 0 {
			value = 0
		}
		if value > 100 {
			value = 100
		}
	}
	return runner.Result{Status: "ok", ExitCode: 0, Stdout: strconv.Itoa(value) + "\n"}, nil
}
func examAnswers(e *assessment.Exam) map[string]string {
	answers := map[string]string{}
	for _, q := range e.Questions {
		for _, o := range q.Options {
			candidate := map[string]string{q.ID: o.ID}
			for _, check := range e.QuizChecks(candidate) {
				if check.ID == q.ID && check.Passed {
					answers[q.ID] = o.ID
				}
			}
		}
	}
	return answers
}
func TestExamBoundariesDraftCASPassFailAndReplay(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	s.CreateProfile("Ada", "light")
	c, _ := content.Load()
	executor := &examExecutor{}
	a := New(s, c, executor, "secret", "127.0.0.1:8080")
	h := a.Handler(t.TempDir())
	request := func(method, path string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		r := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, bytes.NewReader(b))
		r.AddCookie(&http.Cookie{Name: "academy_session", Value: "secret"})
		r.Header.Set("X-CSRF-Token", "secret")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	public := request("GET", "/api/exams/zero-core", nil)
	if public.Code != 200 {
		t.Fatal(public.Body.String())
	}
	for _, key := range []string{`"correctIDs"`, `"correct_ids"`, `"cases"`, `"drivers"`, `"Expected"`} {
		if strings.Contains(public.Body.String(), key) {
			t.Fatal("private exam leaked", key)
		}
	}
	s.DB().Exec(`UPDATE profiles SET xp=999999`)
	if w := request("POST", "/api/exams/zero-core/attempts", map[string]any{}); w.Code != 409 {
		t.Fatal("XP bypassed prerequisite", w.Code)
	}
	for _, id := range []string{"hello", "repair", "two-lines"} {
		s.RecordAttempt(id, map[string]string{"main.go": "trusted fixture"}, true, "ok")
	}
	start := func() (string, string, int64) {
		w := request("POST", "/api/exams/zero-core/attempts", map[string]any{})
		if w.Code != 201 {
			t.Fatal(w.Body.String())
		}
		var result struct {
			ID       string `json:"id"`
			Token    string `json:"attempt_token"`
			Revision int64  `json:"revision"`
		}
		json.Unmarshal(w.Body.Bytes(), &result)
		return result.ID, result.Token, result.Revision
	}
	id, token, revision := start()
	path := "/api/exams/attempts/" + id
	forged := map[string]any{"attempt_token": token, "revision": revision, "answers": map[string]string{}, "codes": map[string]string{}, "passed": true, "score": 100, "rank": "Senior"}
	if request("POST", path+"/submit", forged).Code != 400 {
		t.Fatal("client outcome accepted")
	}
	input := map[string]any{"attempt_token": token, "revision": revision, "answers": map[string]string{}, "codes": map[string]string{}}
	save := request("PUT", path+"/draft", input)
	if save.Code != 200 {
		t.Fatal(save.Body.String())
	}
	if request("PUT", path+"/draft", input).Code != 409 {
		t.Fatal("stale draft accepted")
	}
	if request("POST", path+"/submit", input).Code != 409 {
		t.Fatal("stale submission accepted")
	}
	input["revision"] = revision + 1
	failed := request("POST", path+"/submit", input)
	if failed.Code != 200 || strings.Contains(failed.Body.String(), `"passed":true`) {
		t.Fatal("incomplete exam passed", failed.Body.String())
	}
	awards, _ := s.RankAwards("local")
	if len(awards) != 0 || executor.calls != 0 {
		t.Fatal("incomplete quiz executed/awarded", awards, executor.calls)
	}
	id, token, revision = start()
	path = "/api/exams/attempts/" + id
	e := assessment.Load()
	input = map[string]any{"attempt_token": token, "revision": revision, "answers": examAnswers(e), "codes": map[string]string{"coding": "package main;func sumEven(limit int)int{v:=0;for i:=0;i<=limit;i++{if i%2==0{v+=i}};return v}", "debugging": "package main;func clampPercent(value int)int{if value<0{return 0};if value>100{return 100};return value}"}}
	good := request("POST", path+"/submit", input)
	if good.Code != 200 {
		t.Fatal(good.Body.String())
	}
	var result struct {
		Passed      bool `json:"passed"`
		Score       int  `json:"score"`
		RankAwarded bool `json:"rank_awarded"`
	}
	json.Unmarshal(good.Body.Bytes(), &result)
	if !result.Passed || result.Score != 100 || !result.RankAwarded {
		t.Fatal(good.Body.String())
	}
	calls := executor.calls
	if request("POST", path+"/submit", input).Code != 409 || executor.calls != calls {
		t.Fatal("replay ran again")
	}
	awards, _ = s.RankAwards("local")
	if len(awards) != 1 || awards[0].Rank != "Zero" {
		t.Fatal(awards)
	}
	learning := request("GET", "/api/learning", nil)
	if !strings.Contains(learning.Body.String(), `"verified":true`) {
		t.Fatal("award absent from learning", learning.Body.String())
	}
	stored, _ := s.ExamAttempt("local", id)
	input["revision"] = stored.Revision
	if request("PUT", path+"/draft", input).Code != 409 {
		t.Fatal("completed exam mutable")
	}
	if request("GET", "/api/exams/attempts/does-not-exist", nil).Code != 404 {
		t.Fatal("unknown attempt exposed")
	}
}
