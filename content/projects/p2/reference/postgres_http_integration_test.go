package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// This end-to-end test uses only the explicitly supplied disposable test DB.
// It creates uniquely named users and its cleanup deletes only those users.
func TestPostgresHTTPFlow(t *testing.T) {
	dsn := os.Getenv("LINKHUB_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set LINKHUB_TEST_DATABASE_URL to an isolated disposable PostgreSQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	store, err := OpenPG(ctx, dsn)
	if err != nil {
		t.Fatal("could not open the opt-in test database")
	}
	defer store.Close()

	random := make([]byte, 12)
	if _, err = rand.Read(random); err != nil {
		t.Fatal("could not create fixture IDs")
	}
	suffix := hex.EncodeToString(random)
	emailA, emailB := "flow-a-"+suffix+"@example.test", "flow-b-"+suffix+"@example.test"
	defer func() {
		cleanup, stop := context.WithTimeout(context.Background(), 6*time.Second)
		defer stop()
		_, _ = store.pool.Exec(cleanup, `DELETE FROM users WHERE email_normalized IN ($1,$2)`, emailA, emailB)
	}()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("could not open local test listener")
	}
	origin := "http://" + listener.Addr().String()
	server := &http.Server{Handler: NewHandler(store, Config{Origin: origin}, slog.New(slog.NewTextHandler(io.Discard, nil))), ReadHeaderTimeout: 2 * time.Second}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		shutdown, stop := context.WithTimeout(context.Background(), 3*time.Second)
		defer stop()
		_ = server.Shutdown(shutdown)
	})

	clientA := integrationHTTPClient(t, nil)
	registerA := integrationJSON(t, clientA, origin, http.MethodPost, "/api/v1/auth/register", map[string]any{"email": emailA, "password": "integration-password-a"}, "", http.StatusOK)
	csrfA := integrationString(t, registerA, "csrf_token")
	meA := integrationJSON(t, clientA, origin, http.MethodGet, "/api/v1/me", nil, "", http.StatusOK)
	userA := meA["user"].(map[string]any)
	assertKeys(t, userA, "id", "email", "role")
	saveResponseFixture(t, "user.json", userA)

	tag := integrationJSON(t, clientA, origin, http.MethodPost, "/api/v1/tags", map[string]any{"name": "topic"}, csrfA, http.StatusCreated)
	tagID := integrationString(t, tag, "id")
	saveResponseFixture(t, "tag.json", tag)
	linkA := integrationJSON(t, clientA, origin, http.MethodPost, "/api/v1/links", map[string]any{"slug": "flow-" + suffix, "target_url": "https://example.test/one", "enabled": true, "tags": []string{"topic"}}, csrfA, http.StatusCreated)
	linkID := integrationString(t, linkA, "id")
	assertKeys(t, linkA, "id", "slug", "target_url", "enabled", "tags", "created_at", "updated_at")
	if tags := linkA["tags"].([]any); len(tags) != 1 || tags[0].(map[string]any)["id"] != tagID {
		t.Fatal("tag association was not returned")
	}
	saveResponseFixture(t, "link.json", linkA)
	linkB := integrationJSON(t, clientA, origin, http.MethodPost, "/api/v1/links", map[string]any{"slug": "plain-" + suffix, "target_url": "https://example.test/two", "enabled": true}, csrfA, http.StatusCreated)
	if tags := linkB["tags"].([]any); len(tags) != 0 {
		t.Fatal("link without tags returned a tag association")
	}
	saveResponseFixture(t, "untagged-link.json", linkB)

	listBody := integrationRawJSON(t, clientA, origin, http.MethodGet, "/api/v1/links?limit=10&offset=0", nil, "", http.StatusOK)
	listRows, ok := listBody.([]any)
	if !ok || len(listRows) != 2 {
		t.Fatal("list did not return both owned links")
	}
	saveResponseFixture(t, "links.json", listRows)
	got := integrationJSON(t, clientA, origin, http.MethodGet, "/api/v1/links/"+linkID, nil, "", http.StatusOK)
	if got["id"] != linkID {
		t.Fatal("get returned the wrong link")
	}

	clientB := integrationHTTPClient(t, nil)
	registerB := integrationJSON(t, clientB, origin, http.MethodPost, "/api/v1/auth/register", map[string]any{"email": emailB, "password": "integration-password-b"}, "", http.StatusOK)
	csrfB := integrationString(t, registerB, "csrf_token")
	ownerError := integrationJSON(t, clientB, origin, http.MethodGet, "/api/v1/links/"+linkID, nil, "", http.StatusNotFound)
	if ownerError["error"] != "not_found" {
		t.Fatal("cross-owner response was not generic not_found")
	}
	saveResponseFixture(t, "not-found.json", ownerError)
	intruderUpdate := map[string]any{"slug": "changed-" + suffix, "target_url": "https://example.test/changed", "enabled": true, "tags": []string{}}
	integrationJSON(t, clientB, origin, http.MethodPatch, "/api/v1/links/"+linkID, intruderUpdate, csrfB, http.StatusNotFound)
	integrationRequest(t, clientB, origin, http.MethodDelete, "/api/v1/links/"+linkID, nil, csrfB, http.StatusNotFound)

	missingCSRF := integrationRequest(t, clientA, origin, http.MethodPost, "/api/v1/tags", map[string]any{"name": "blocked"}, "", http.StatusForbidden)
	if missingCSRF["error"] != "csrf_rejected" {
		t.Fatal("missing CSRF token was not rejected")
	}

	publicClient := integrationHTTPClient(t, func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse })
	response := integrationResponse(t, publicClient, origin, http.MethodGet, "/"+linkA["slug"].(string), nil, "", http.StatusFound)
	if response.Header.Get("Location") != "https://example.test/one" {
		response.Body.Close()
		t.Fatal("redirect returned an unexpected destination")
	}
	response.Body.Close()

	updated := map[string]any{"slug": linkA["slug"], "target_url": "https://example.test/one", "enabled": false, "tags": []string{"topic"}}
	integrationJSON(t, clientA, origin, http.MethodPatch, "/api/v1/links/"+linkID, updated, csrfA, http.StatusOK)
	disabledResponse := integrationResponse(t, publicClient, origin, http.MethodGet, "/"+linkA["slug"].(string), nil, "", http.StatusNotFound)
	disabledResponse.Body.Close()

	integrationRequest(t, clientA, origin, http.MethodPost, "/api/v1/auth/logout", nil, csrfA, http.StatusNoContent)
	integrationJSON(t, clientA, origin, http.MethodGet, "/api/v1/me", nil, "", http.StatusUnauthorized)

	login := integrationJSON(t, clientA, origin, http.MethodPost, "/api/v1/auth/login", map[string]any{"email": emailA, "password": "integration-password-a"}, "", http.StatusOK)
	if integrationString(t, login, "csrf_token") == "" {
		t.Fatal("login did not return the CSRF token")
	}
	integrationJSON(t, clientA, origin, http.MethodGet, "/api/v1/me", nil, "", http.StatusOK)
}

func integrationHTTPClient(t *testing.T, redirect func(*http.Request, []*http.Request) error) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal("could not create test cookie jar")
	}
	return &http.Client{Jar: jar, CheckRedirect: redirect}
}

func integrationJSON(t *testing.T, client *http.Client, origin, method, path string, body any, csrf string, status int) map[string]any {
	t.Helper()
	v := integrationRawJSON(t, client, origin, method, path, body, csrf, status)
	if m, ok := v.(map[string]any); ok {
		return m
	}
	t.Fatalf("%s %s returned a non-object JSON response", method, path)
	return nil
}

func integrationRawJSON(t *testing.T, client *http.Client, origin, method, path string, body any, csrf string, status int) any {
	t.Helper()
	resp := integrationResponse(t, client, origin, method, path, body, csrf, status)
	defer resp.Body.Close()
	var result any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("%s %s returned invalid JSON", method, path)
	}
	return result
}

func integrationRequest(t *testing.T, client *http.Client, origin, method, path string, body any, csrf string, status int) map[string]any {
	t.Helper()
	resp := integrationResponse(t, client, origin, method, path, body, csrf, status)
	defer resp.Body.Close()
	if status == http.StatusNoContent {
		return nil
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("%s %s returned invalid error JSON", method, path)
	}
	return result
}

func integrationResponse(t *testing.T, client *http.Client, origin, method, path string, body any, csrf string, status int) *http.Response {
	t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatal("could not encode request fixture")
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, origin+path, reader)
	if err != nil {
		t.Fatal("could not build local HTTP request")
	}
	req.Header.Set("Origin", origin)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s request failed", method, path)
	}
	if resp.StatusCode != status {
		resp.Body.Close()
		t.Fatalf("%s %s returned status %d, want %d", method, path, resp.StatusCode, status)
	}
	return resp
}

func integrationString(t *testing.T, value map[string]any, field string) string {
	t.Helper()
	s, ok := value[field].(string)
	if !ok || s == "" {
		t.Fatalf("response omitted required string field %q", field)
	}
	return s
}

func saveResponseFixture(t *testing.T, name string, value any) {
	t.Helper()
	dir := os.Getenv("LINKHUB_RESPONSE_FIXTURE_DIR")
	if dir == "" {
		return
	}
	encoded, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal("could not encode response fixture")
	}
	if fixtureHasCredential(json.RawMessage(encoded)) {
		t.Fatal("refusing to write a credential-bearing response fixture")
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		t.Fatal("could not create response fixture directory")
	}
	if err = os.WriteFile(filepath.Join(dir, name), append(encoded, '\n'), 0600); err != nil {
		t.Fatal("could not save response fixture")
	}
}

func fixtureHasCredential(raw json.RawMessage) bool {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return true
	}
	return hasCredentialKey(value)
}
func hasCredentialKey(value any) bool {
	switch v := value.(type) {
	case map[string]any:
		for k, child := range v {
			switch strings.ToLower(k) {
			case "password", "csrf_token", "session", "session_token", "cookie", "set-cookie", "token":
				return true
			}
			if hasCredentialKey(child) {
				return true
			}
		}
	case []any:
		for _, child := range v {
			if hasCredentialKey(child) {
				return true
			}
		}
	}
	return false
}
