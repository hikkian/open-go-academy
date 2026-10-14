package httpapi

import (
	"academy/internal/content"
	"academy/internal/runner"
	"academy/internal/store"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

type fakeExecutor struct{ result runner.Result }

func (f fakeExecutor) Available(context.Context) bool { return true }
func (f fakeExecutor) Execute(context.Context, runner.Request) (runner.Result, error) {
	return f.result, nil
}
func TestAPIProfilePersistenceAndBoundaries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "academy.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := content.Load()
	a := New(s, c, nil, "secret", "127.0.0.1:8080")
	h := a.Handler(t.TempDir())
	request := func(method, path, body string, auth bool, host, origin string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, strings.NewReader(body))
		r.Host = host
		r.Header.Set("Origin", origin)
		if auth {
			r.AddCookie(&http.Cookie{Name: "academy_session", Value: "secret"})
			r.Header.Set("X-CSRF-Token", "secret")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if w := request("GET", "/api/profile", "", false, "127.0.0.1:8080", ""); w.Code != 200 || !strings.Contains(w.Body.String(), `"profile":null`) {
		t.Fatal(w)
	}
	for _, tc := range []struct {
		host, origin string
		auth         bool
	}{{"evil.test:8080", "", true}, {"127.0.0.1:8080", "http://evil.test", true}, {"127.0.0.1:8080", "", false}} {
		if w := request("POST", "/api/profile", `{"name":"Ada","theme":"dark"}`, tc.auth, tc.host, tc.origin); w.Code != 403 {
			t.Fatal("boundary bypass", w.Code)
		}
	}
	if w := request("POST", "/api/profile", `{"name":"Ada","theme":"dark","rank":"Senior"}`, true, "127.0.0.1:8080", ""); w.Code != 400 {
		t.Fatal("rank injection accepted")
	}
	if w := request("POST", "/api/profile", `{"name":"Ada","theme":"dark"}`, true, "127.0.0.1:8080", ""); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := request("PUT", "/api/drafts/hello", `{"files":{"../evil.go":"x"}}`, true, "127.0.0.1:8080", ""); w.Code != 422 {
		t.Fatal("unsafe file accepted")
	}
	if w := request("PUT", "/api/drafts/hello", `{"files":{"main.go":"package main"}}`, true, "127.0.0.1:8080", ""); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	s.Close()
	s, err = store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	p, _ := s.Profile()
	d, _ := s.Draft("hello")
	if p.Name != "Ada" || p.Theme != "dark" || p.Rank != "Zero" || d["main.go"] != "package main" {
		t.Fatal(p, d)
	}
	if ValidateAddress("0.0.0.0:8080") == nil || ValidateAddress("localhost:8080") == nil {
		t.Fatal("unsafe bind accepted")
	}
}
func TestGraderRejectsStructureBypassAndReplay(t *testing.T) {
	s, _ := store.Open(":memory:")
	defer s.Close()
	s.CreateProfile("Ada", "light")
	c, _ := content.Load()
	a := New(s, c, fakeExecutor{runner.Result{Status: "ok", Stdout: "42\n", ExitCode: 0}}, "secret", "127.0.0.1:8080")
	h := a.Handler(t.TempDir())
	submit := func(source string) runner.Result {
		q, _ := json.Marshal(runner.Request{Files: map[string]string{"main.go": source}})
		r := httptest.NewRequest("POST", "http://127.0.0.1:8080/api/exercises/repair/submit", strings.NewReader(string(q)))
		r.AddCookie(&http.Cookie{Name: "academy_session", Value: "secret"})
		r.Header.Set("X-CSRF-Token", "secret")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		var result runner.Result
		json.Unmarshal(w.Body.Bytes(), &result)
		return result
	}
	if r := submit(`package main;import "fmt";func main(){fmt.Println(42)}`); r.Passed {
		t.Fatal("structure bypass accepted")
	}
	p, _ := s.Progress()
	if p.XP != 0 {
		t.Fatal("failed awarded XP")
	}
	valid := `package main;import "fmt";func answer()int{return 42};func main(){fmt.Println(answer())}`
	for i := 0; i < 2; i++ {
		if !submit(valid).Passed {
			t.Fatal("valid rejected")
		}
	}
	p, _ = s.Progress()
	if p.XP != 10 || len(p.Completed) != 1 {
		t.Fatal("replay XP", p)
	}
	if structure("two-lines", map[string]string{"main.go": `package main;import "fmt";func main(){fmt.Println("Имя: Ada\nЯзык: Go")}`}).Passed {
		t.Fatal("one call accepted")
	}
}
func TestStructureRejectsDeadBranches(t *testing.T) {
	for id, source := range map[string]string{"repair": `package main;import "fmt";func answer()int{return 7};func main(){if false{fmt.Println(answer())};fmt.Println(42)}`, "two-lines": `package main;import "fmt";func main(){if false{fmt.Println(1);fmt.Println(2)};fmt.Print("Имя: Ada\nЯзык: Go\n")}`} {
		if structure(id, map[string]string{"main.go": source}).Passed {
			t.Fatal("accepted dead branch", id)
		}
	}
}
func TestExecutionRateLimitAndRankMutationDenial(t *testing.T) {
	s, _ := store.Open(":memory:")
	defer s.Close()
	s.CreateProfile("Ada", "light")
	c, _ := content.Load()
	a := New(s, c, fakeExecutor{runner.Result{Status: "ok", Stdout: "Go готов!\n", ExitCode: 0}}, "secret", "127.0.0.1:8080")
	h := a.Handler(t.TempDir())
	request := func(path, body string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://127.0.0.1:8080"+path, strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: "academy_session", Value: "secret"})
		r.Header.Set("X-CSRF-Token", "secret")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if request("/api/ranks/award", `{"rank":"Senior"}`).Code != 404 {
		t.Fatal("client rank route exists")
	}
	for i := 0; i < 12; i++ {
		if w := request("/api/exercises/hello/run", `{"files":{"main.go":"package main;func main(){}"}}`); w.Code != 200 {
			t.Fatal(i, w.Code)
		}
	}
	if w := request("/api/exercises/hello/run", `{"files":{"main.go":"package main;func main(){}"}}`); w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatal("execution flood admitted", w.Code)
	}
	p, _ := s.Profile()
	if p.Rank != "Zero" || p.XP != 0 {
		t.Fatal("run awarded evidence", p)
	}
}
func TestProjectHasTestsRejectsOverridesAndHelpers(t *testing.T) {
	for _, source := range []string{`package main;import "testing";func TestMain(m *testing.M){}`, `package main;import "testing";func Testhelper(t *testing.T){}`, `package main;import "testing";func TestWrong(m *testing.M){}`, `package main;import "testing";func TestMain(m *testing.M){};func TestReal(t *testing.T){}`} {
		if hasTests(map[string]string{"main_test.go": source}) {
			t.Fatal("not a supported test", source)
		}
	}
	if !hasTests(map[string]string{"main_test.go": `package main;import tst "testing";func TestReal(t *tst.T){}`}) {
		t.Fatal("genuine aliased test rejected")
	}
}
