package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeStore struct {
	sessions map[string]struct {
		u    User
		csrf string
		exp  time.Time
	}
	links          map[string]Link
	gotOwner       string
	deleted        string
	redirect       string
	createCalled   bool
	next           User
	passwordHash   []byte
	conflicts      int
	conflictAlways bool
	createCalls    int
}

func newFake() *fakeStore {
	return &fakeStore{sessions: map[string]struct {
		u    User
		csrf string
		exp  time.Time
	}{}, links: map[string]Link{}}
}
func (f *fakeStore) CreateUser(e string, hash []byte) (User, error) {
	f.passwordHash = append([]byte(nil), hash...)
	f.next = User{ID: "u1", Email: e, Role: "user"}
	return f.next, nil
}
func (f *fakeStore) UserByEmail(e string) (User, []byte, error) { return User{}, nil, errNotFound }
func (f *fakeStore) CreateSession(t, c, u string, x time.Time) error {
	f.sessions[t] = struct {
		u    User
		csrf string
		exp  time.Time
	}{User{ID: u, Role: "user"}, c, x}
	return nil
}
func (f *fakeStore) Session(t string) (User, string, time.Time, error) {
	s, ok := f.sessions[t]
	if !ok {
		return User{}, "", time.Time{}, errNotFound
	}
	return s.u, s.csrf, s.exp, nil
}
func (f *fakeStore) DeleteSession(t string) error { delete(f.sessions, t); return nil }
func (f *fakeStore) CreateLink(o string, in LinkInput) (Link, error) {
	f.gotOwner = o
	f.createCalls++
	if f.conflictAlways || f.conflicts > 0 {
		f.conflicts--
		return Link{}, errConflict
	}
	f.createCalled = true
	l := Link{ID: "new", Slug: in.Slug, TargetURL: in.TargetURL, Enabled: *in.Enabled, Tags: []Tag{}}
	f.links[l.ID] = l
	return l, nil
}
func (f *fakeStore) ListLinks(o string, _, _ int) ([]Link, error) {
	f.gotOwner = o
	return []Link{}, nil
}
func (f *fakeStore) GetLink(o, id string) (Link, error) {
	f.gotOwner = o
	l, ok := f.links[id]
	if !ok || o != "u1" {
		return Link{}, errNotFound
	}
	return l, nil
}
func (f *fakeStore) UpdateLink(o, id string, _ LinkInput) (Link, error) {
	f.gotOwner = o
	return Link{}, errNotFound
}
func (f *fakeStore) DeleteLink(o, id string) error {
	f.gotOwner = o
	f.deleted = id
	return errNotFound
}
func (f *fakeStore) CreateTag(string, TagInput) (Tag, error) {
	return Tag{ID: "tag-1", Name: "example"}, nil
}
func (f *fakeStore) ListTags(string) ([]Tag, error)                  { return []Tag{}, nil }
func (f *fakeStore) UpdateTag(string, string, TagInput) (Tag, error) { return Tag{}, errNotFound }
func (f *fakeStore) DeleteTag(string, string) error                  { return errNotFound }
func (f *fakeStore) RedirectTarget(slug string) (string, error) {
	if slug != "short" || f.redirect == "" {
		return "", errNotFound
	}
	return f.redirect, nil
}

func testHandler(f *fakeStore) http.Handler {
	return NewHandler(f, Config{Origin: "http://127.0.0.1:8097"}, slog.New(slog.NewTextHandler(io.Discard, nil)))
}
func authRequest(method, path, body string, csrf bool) *http.Request {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "raw-session"})
	r.AddCookie(&http.Cookie{Name: "linkhub_csrf", Value: "raw-csrf"})
	r.Header.Set("Origin", "http://127.0.0.1:8097")
	if csrf {
		r.Header.Set("X-CSRF-Token", "raw-csrf")
	}
	return r
}
func seedAuth(f *fakeStore) {
	f.sessions[tokenHash("raw-session")] = struct {
		u    User
		csrf string
		exp  time.Time
	}{User{ID: "u1", Email: "a@example.test", Role: "user"}, tokenHash("raw-csrf"), time.Now().Add(time.Hour)}
}
func TestMutationRequiresOriginAndCSRF(t *testing.T) {
	for _, tc := range []struct {
		name, origin, csrf string
		want               int
	}{{"missing csrf", "http://127.0.0.1:8097", "", 403}, {"cross origin", "https://evil.example", "raw-csrf", 403}, {"cross-site fetch", "http://127.0.0.1:8097", "raw-csrf", 403}, {"valid", "http://127.0.0.1:8097", "raw-csrf", 201}} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake()
			seedAuth(f)
			r := authRequest("POST", "/api/v1/links", `{"slug":"hello","target_url":"https://example.com","enabled":true}`, tc.csrf != "")
			r.Header.Set("Origin", tc.origin)
			if tc.name == "cross-site fetch" {
				r.Header.Set("Sec-Fetch-Site", "cross-site")
			}
			w := httptest.NewRecorder()
			testHandler(f).ServeHTTP(w, r)
			if w.Code != tc.want {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
			if (tc.want == 201) != f.createCalled {
				t.Fatalf("create called=%v", f.createCalled)
			}
			if tc.want == 201 && f.gotOwner != "u1" {
				t.Fatalf("owner=%q", f.gotOwner)
			}
		})
	}
}
func TestCreateLinkRejectsInvalidAndIgnoresOwnerField(t *testing.T) {
	f := newFake()
	seedAuth(f)
	for _, body := range []string{`{"slug":"okay","target_url":"javascript:alert(1)","enabled":true}`, `{"slug":"okay","target_url":"https://example.com\r\nLocation: evil","enabled":true}`, `{"slug":"okay","target_url":"https://example.com","enabled":true,"owner_id":"victim"}`, `{"slug":"okay","target_url":"https://example.com","enabled":true} {}`} {
		r := authRequest("POST", "/api/v1/links", body, true)
		w := httptest.NewRecorder()
		testHandler(f).ServeHTTP(w, r)
		if w.Code != 400 {
			t.Errorf("body %s: got %d, want 400 (%s)", body, w.Code, w.Body.String())
		}
	}
	if f.createCalled {
		t.Fatal("invalid payload reached store")
	}
}
func TestOwnerPredicateAndLogoutRevocation(t *testing.T) {
	f := newFake()
	seedAuth(f)
	r := authRequest("GET", "/api/v1/links/other", "", false)
	w := httptest.NewRecorder()
	testHandler(f).ServeHTTP(w, r)
	if w.Code != 404 || f.gotOwner != "u1" {
		t.Fatalf("status=%d owner=%q", w.Code, f.gotOwner)
	}
	r = authRequest("POST", "/api/v1/auth/logout", "", true)
	w = httptest.NewRecorder()
	testHandler(f).ServeHTTP(w, r)
	if w.Code != 204 || len(f.sessions) != 0 {
		t.Fatalf("logout status=%d sessions=%d", w.Code, len(f.sessions))
	}
	r = authRequest("GET", "/api/v1/me", "", false)
	w = httptest.NewRecorder()
	testHandler(f).ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("after logout status=%d", w.Code)
	}
}
func TestRedirectOnlyEnabledLookupAndNoOutboundFetch(t *testing.T) {
	f := newFake()
	f.redirect = "https://example.org/path"
	w := httptest.NewRecorder()
	testHandler(f).ServeHTTP(w, httptest.NewRequest("GET", "/short", nil))
	if w.Code != 302 || w.Header().Get("Location") != f.redirect {
		t.Fatalf("status=%d Location=%q", w.Code, w.Header().Get("Location"))
	}
	f.redirect = ""
	w = httptest.NewRecorder()
	testHandler(f).ServeHTTP(w, httptest.NewRequest("GET", "/short", nil))
	if w.Code != 404 {
		t.Fatalf("missing redirect status=%d", w.Code)
	}
}
func TestHelpersAndJSONBounds(t *testing.T) {
	cases := map[string]bool{"https://example.com": true, "http://127.0.0.1:8080/a": true, "https://[::1]:443/path": true, "https://:443/path": false, "http:///path": false, "javascript:alert(1)": false, "https://user:pass@example.com": false, "https://a.example/\r\nX: y": false, "data:text/plain,x": false}
	for s, want := range cases {
		if got := validTarget(s); got != want {
			t.Errorf("validTarget(%q)=%v want %v", s, got, want)
		}
	}
	f := newFake()
	seedAuth(f)
	r := authRequest("POST", "/api/v1/links", `{"slug":"okay","target_url":"https://example.com","enabled":true}`, true)
	r.Body = io.NopCloser(bytes.NewReader(bytes.Repeat([]byte("x"), 17<<10)))
	w := httptest.NewRecorder()
	testHandler(f).ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("oversize status=%d", w.Code)
	}
	var v map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &v); e != nil {
		t.Fatal(e)
	}
	if _, ok := v["error"]; !ok {
		t.Fatal("missing generic error")
	}
	if errors.Is(errInvalid, errNotFound) {
		t.Fatal("error sentinels overlap")
	}
}

func TestRegisterStoresOnlyPasswordHashAndSetsCookieFlags(t *testing.T) {
	f := newFake()
	h := NewHandler(f, Config{Origin: "https://academy.test", SecureCookies: true}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	r := httptest.NewRequest("POST", "/api/v1/auth/register", strings.NewReader(`{"email":"Learner@example.test","password":"correct horse battery staple"}`))
	r.Header.Set("Origin", "https://academy.test")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if len(f.passwordHash) != 49 || string(f.passwordHash) == "correct horse battery staple" {
		t.Fatalf("password hash was not stored in versioned salted form (%d bytes)", len(f.passwordHash))
	}
	var sessionPayload map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &sessionPayload); e != nil {
		t.Fatal(e)
	}
	assertKeys(t, sessionPayload, "user", "csrf_token")
	assertKeys(t, sessionPayload["user"].(map[string]any), "id", "email", "role")
	var session, csrf *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			session = c
		}
		if c.Name == "linkhub_csrf" {
			csrf = c
		}
	}
	if session == nil || !session.HttpOnly || !session.Secure || session.SameSite != http.SameSiteStrictMode {
		t.Fatalf("session cookie flags: %#v", session)
	}
	if csrf == nil || csrf.HttpOnly || !csrf.Secure || csrf.SameSite != http.SameSiteStrictMode {
		t.Fatalf("csrf cookie flags: %#v", csrf)
	}
	if f.sessions[tokenHash(session.Value)].csrf != tokenHash(csrf.Value) {
		t.Fatal("raw session or CSRF value was stored")
	}
	if !checkPassword("correct horse battery staple", f.passwordHash) || checkPassword("correct horse battery staple!", f.passwordHash) {
		t.Fatal("password KDF accepted the wrong password or rejected the stored password")
	}
}

func TestGeneratedSlugRetriesUniqueConstraint(t *testing.T) {
	f := newFake()
	f.conflicts = 2
	seedAuth(f)
	r := authRequest("POST", "/api/v1/links", `{"target_url":"https://example.test"}`, true)
	w := httptest.NewRecorder()
	testHandler(f).ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if f.createCalls != 3 || !validSlug(f.links["new"].Slug) {
		t.Fatalf("calls=%d slug=%q", f.createCalls, f.links["new"].Slug)
	}
	var link map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &link); e != nil {
		t.Fatal(e)
	}
	assertKeys(t, link, "id", "slug", "target_url", "enabled", "tags", "created_at", "updated_at")
}

func assertKeys(t *testing.T, v map[string]any, keys ...string) {
	t.Helper()
	for _, key := range keys {
		if _, ok := v[key]; !ok {
			t.Errorf("response missing required schema field %q in %#v", key, v)
		}
	}
}

func TestGeneratedSlugExhaustionReturnsConflict(t *testing.T) {
	f := newFake()
	f.conflictAlways = true
	seedAuth(f)
	r := authRequest("POST", "/api/v1/links", `{"target_url":"https://example.test"}`, true)
	w := httptest.NewRecorder()
	testHandler(f).ServeHTTP(w, r)
	if w.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if f.createCalls != 4 || f.createCalled || len(f.links) != 0 {
		t.Fatalf("attempts=%d created=%v records=%d", f.createCalls, f.createCalled, len(f.links))
	}
}

func TestTagResponseHasOpenAPISchemaFields(t *testing.T) {
	f := newFake()
	seedAuth(f)
	r := authRequest("POST", "/api/v1/tags", `{"name":"example"}`, true)
	w := httptest.NewRecorder()
	testHandler(f).ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var tag map[string]any
	if e := json.Unmarshal(w.Body.Bytes(), &tag); e != nil {
		t.Fatal(e)
	}
	assertKeys(t, tag, "id", "name")
}
