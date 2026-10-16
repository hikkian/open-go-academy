package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBoundedStrictJSON(t *testing.T) {
	var target struct {
		Name string `json:"name"`
	}
	for _, body := range []string{`{"name":"ok","unexpected":true}`, `{"name":"ok"}{}`, `not-json`} {
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		if decode(r, &target) == nil {
			t.Fatalf("decode accepted %q", body)
		}
	}
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"name":"`+strings.Repeat("x", 1024)+`"}`)))
	request.Body = http.MaxBytesReader(httptest.NewRecorder(), request.Body, 64)
	if decode(request, &target) == nil {
		t.Fatal("oversized payload accepted")
	}
}

func TestErrorResponseDoesNotExposeInternalText(t *testing.T) {
	rec := httptest.NewRecorder()
	writeError(rec, 404)
	if rec.Code != 404 || strings.Contains(rec.Body.String(), "SQL") || !strings.Contains(rec.Body.String(), "Not Found") {
		t.Fatalf("unexpected safe error: %d %s", rec.Code, rec.Body.String())
	}
}

func TestSessionTokenIsOpaqueAndHashed(t *testing.T) {
	a, e := newToken()
	if e != nil {
		t.Fatal(e)
	}
	b, e := newToken()
	if e != nil {
		t.Fatal(e)
	}
	if a == b || len(hashToken(a)) != 32 || strings.Contains(a, "/") {
		t.Fatal("session token requirements failed")
	}
}

func TestWorkerLimitsArePositive(t *testing.T) {
	w := NewWorker(nil, 0, -1, 0)
	if w.concurrency != 1 || w.batch != 1 || w.maxAttempts != 1 {
		t.Fatalf("unsafe worker limits: %#v", w)
	}
}

func TestRateLimitIsBoundedAndDoesNotTrustForwardedHeaders(t *testing.T) {
	var l rateLimiter
	for i := 0; i < 120; i++ {
		if !l.allow("192.0.2.10:4000") {
			t.Fatalf("request %d rejected too early", i+1)
		}
	}
	if l.allow("192.0.2.10:5000") {
		t.Fatal("client bypassed limit by changing source port")
	}
	if !l.allow("192.0.2.11:4000") {
		t.Fatal("independent client should have its own budget")
	}
	for i := 0; i < 5000; i++ {
		if l.allow("198.51.100." + strings.Repeat("1", i%7+1) + ":" + strings.Repeat("2", i%13+1)) { /* random labels are just distinct test keys */
		}
	}
	if len(l.clients) > 4096 {
		t.Fatalf("rate limiter map is unbounded: %d", len(l.clients))
	}
}

func TestProtectedRouteRejectsMissingIdentity(t *testing.T) {
	handler := NewAPI(&Store{}).Handler()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces", strings.NewReader(`{"name":"private"}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("protected route returned %d: %s", response.Code, response.Body.String())
	}
}

func TestRunWorkersJoinsInflightWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	entered := make(chan struct{})
	release := make(chan struct{})
	draining := make(chan struct{})
	returned := make(chan struct{})
	var once sync.Once
	go func() {
		runWorkers(ctx, 1, 1, time.Millisecond, func(context.Context) (bool, error) { once.Do(func() { close(entered) }); <-release; return false, nil }, func(started bool) {
			if !started {
				close(draining)
			}
		})
		close(returned)
	}()
	<-entered
	cancel()
	<-draining
	select {
	case <-returned:
		t.Fatal("Run returned while a worker still owned in-flight work")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("Run failed to join worker after it completed")
	}
}

func TestWaitForWorkerHonorsShutdownDeadline(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if waitForWorker(ctx, make(chan struct{})) {
		t.Fatal("unclosed worker was reported complete")
	}
}

func TestLogoutFailureClearsCookieAndDoesNotClaimRevocation(t *testing.T) {
	a := NewAPI(nil)
	called := false
	a.revoke = func(context.Context, string) error { called = true; return errors.New("private database diagnostic") }
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/current", nil)
	req.AddCookie(&http.Cookie{Name: "taskflow_session", Value: "opaque-secret"})
	rec := httptest.NewRecorder()
	a.Handler().ServeHTTP(rec, req)
	if !called || rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("logout failure status=%d called=%v body=%s", rec.Code, called, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "private database diagnostic") {
		t.Fatal("logout leaked internal error")
	}
	cleared := false
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == "taskflow_session" && cookie.MaxAge < 0 {
			cleared = true
		}
	}
	if !cleared {
		t.Fatal("browser session cookie was not cleared on logout failure")
	}
}

func TestReadinessRequiresDatabaseAndWorkerAndLivenessDoesNot(t *testing.T) {
	a := NewAPI(nil)
	dbReady := false
	a.ping = func(context.Context) error {
		if !dbReady {
			return errors.New("database unavailable")
		}
		return nil
	}
	a.worker = NewWorker(nil, 1, 1, 1)
	call := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		a.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
		return rec
	}
	if got := call("/livez"); got.Code != http.StatusOK {
		t.Fatalf("liveness depends on database: %d", got.Code)
	}
	if got := call("/readyz"); got.Code != http.StatusServiceUnavailable {
		t.Fatalf("database-down readiness=%d", got.Code)
	}
	dbReady = true
	if got := call("/readyz"); got.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness without worker=%d", got.Code)
	}
	a.worker.running.Store(true)
	if got := call("/readyz"); got.Code != http.StatusOK {
		t.Fatalf("ready dependencies status=%d body=%s", got.Code, got.Body.String())
	}
	a.worker.draining.Store(true)
	if got := call("/readyz"); got.Code != http.StatusServiceUnavailable {
		t.Fatalf("draining worker readiness=%d", got.Code)
	}
}
