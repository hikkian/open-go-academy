package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTargetAllowlist(t *testing.T) {
	cases := []struct {
		raw string
		ok  bool
	}{
		{"http://127.0.0.1:8097", true}, {"http://[::1]:8097", true},
		{"https://127.0.0.1:8097", false}, {"http://localhost:8097", false},
		{"http://127.0.0.2:8097", false}, {"http://127.0.0.1.evil.test:8097", false},
		{"http://user@127.0.0.1:8097", false}, {"http://127.0.0.1:8097/path", false},
		{"http://127.0.0.1:8097?next=http://example.test", false}, {"http://127.0.0.1", false},
		{"http://127.0.0.1:65536", false},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			_, err := newChecker(tc.raw)
			if (err == nil) != tc.ok {
				t.Fatalf("accepted=%v want=%v", err == nil, tc.ok)
			}
		})
	}
}

func TestRequestStopsAtRedirectAndBoundsBody(t *testing.T) {
	externalHits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/redirect":
			http.Redirect(w, r, "http://example.invalid/private", http.StatusFound)
		case "/oversize":
			w.WriteHeader(200)
			_, _ = w.Write([]byte(strings.Repeat("x", maxBody+1)))
		case "/healthz":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		default:
			externalHits++
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	c, err := newChecker(srv.URL)
	if err != nil {
		t.Fatal("loopback fixture rejected")
	}
	err = c.call("GET", "/redirect", nil, nil, 302, nil)
	if err != nil {
		t.Fatal("redirect response not retained")
	}
	if c.last.header.Get("Location") != "http://example.invalid/private" {
		t.Fatal("redirect location absent")
	}
	if externalHits != 0 {
		t.Fatal("redirect target was fetched")
	}
	if c.call("GET", "/oversize", nil, nil, 200, nil) == nil {
		t.Fatal("oversized response accepted")
	}
	var health struct {
		Status string `json:"status"`
	}
	if c.call("GET", "/healthz", nil, nil, 200, &health) != nil || health.Status != "ok" {
		t.Fatal("valid health response rejected")
	}
}

func TestStateIsStrictAndResponseDoesNotEchoBodies(t *testing.T) {
	c, _ := newChecker("http://127.0.0.1:8097")
	c.persistence(json.RawMessage(`{"email":"a@example.test","password":"private-password","slug":"x-y","link_id":"id","target":"https://example.test","tag":"t","extra":"secret"}`))
	if len(c.checks) != 1 || c.checks[0].Passed {
		t.Fatal("unknown state key accepted")
	}
	body := strings.Repeat("sensitive-body", 40)
	c.add("sample", false)
	out := response{Checks: c.checks, State: nil}
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), body) || strings.Contains(string(raw), "private-password") {
		t.Fatal("private values serialized")
	}
	if strings.Contains(string(raw), "status") || strings.Contains(string(raw), "error") {
		t.Fatal("response includes diagnostics")
	}
}

func TestInputLimitAndStrictRequestJSON(t *testing.T) {
	valid := []byte(`{"phase":"health","base_url":"http://127.0.0.1:8097"}`)
	if _, err := parseRequest(valid); err != nil {
		t.Fatal("valid request rejected")
	}
	invalid := [][]byte{[]byte(`{"phase":"health","base_url":"http://127.0.0.1:8097","extra":true}`), []byte(`{"phase":"health","base_url":"http://127.0.0.1:8097"}{}`), bytes.Repeat([]byte("x"), maxInput+1)}
	for i, raw := range invalid {
		if _, err := parseRequest(raw); err == nil {
			t.Errorf("invalid input case %d accepted", i)
		}
	}
	if requestLimit != 3*time.Second || totalLimit != 35*time.Second {
		t.Fatal("configured request bounds changed")
	}
}

func TestInitialAndPersistenceAgainstMockLinkHub(t *testing.T) {
	type user struct{ email, password, id string }
	users := map[string]user{}
	sessions := map[string]user{}
	links := map[string]map[string]any{}
	idCounter := 0
	slugs := map[string]string{}
	tags := map[string]map[string]any{}
	tagCounter := 0
	userCounter := 0
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		write := func(status int, v any) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			if v != nil {
				_ = json.NewEncoder(w).Encode(v)
			}
		}
		cookie, _ := r.Cookie("linkhub_session")
		u, authed := sessions[func() string {
			if cookie != nil {
				return cookie.Value
			}
			return ""
		}()]
		if r.URL.Path == "/healthz" {
			write(200, map[string]string{"status": "ok"})
			return
		}
		if r.URL.Path == "/api/v1/auth/register" || r.URL.Path == "/api/v1/auth/login" {
			var in struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			_ = json.NewDecoder(r.Body).Decode(&in)
			if r.URL.Path == "/api/v1/auth/register" {
				if _, ok := users[in.Email]; ok {
					write(409, nil)
					return
				}
				userCounter++
				u = user{email: in.Email, password: in.Password, id: fmt.Sprintf("123e4567-e89b-42d3-a456-%012x", userCounter)}
				users[in.Email] = u
			} else {
				var ok bool
				u, ok = users[in.Email]
				if !ok || u.password != in.Password {
					write(401, nil)
					return
				}
			}
			token, csrf := "s"+randomHex(8), "c"+randomHex(8)
			sessions[token] = u
			http.SetCookie(w, &http.Cookie{Name: "linkhub_session", Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode})
			http.SetCookie(w, &http.Cookie{Name: "linkhub_csrf", Value: csrf, Path: "/", SameSite: http.SameSiteStrictMode})
			write(200, map[string]any{"user": map[string]string{"id": u.id, "email": u.email, "role": "user"}, "csrf_token": csrf})
			return
		}
		if r.URL.Path == "/api/v1/me" {
			if !authed {
				write(401, nil)
				return
			}
			csrf, _ := r.Cookie("linkhub_csrf")
			cv := ""
			if csrf != nil {
				cv = csrf.Value
			}
			write(200, map[string]any{"user": map[string]string{"id": u.id, "email": u.email, "role": "user"}, "csrf_token": cv})
			return
		}
		if r.URL.Path == "/api/v1/auth/logout" {
			if !authed {
				write(401, nil)
				return
			}
			delete(sessions, cookie.Value)
			write(204, nil)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/") && !authed {
			write(401, nil)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/") && r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != "http://"+r.Host {
			write(403, nil)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/") && r.Method != "GET" && r.URL.Path != "/api/v1/auth/logout" {
			csrf, _ := r.Cookie("linkhub_csrf")
			if csrf == nil || r.Header.Get("X-CSRF-Token") != csrf.Value {
				write(403, nil)
				return
			}
		}
		if r.URL.Path == "/api/v1/tags" && r.Method == "POST" {
			var in map[string]string
			_ = json.NewDecoder(r.Body).Decode(&in)
			tagCounter++
			v := map[string]any{"id": fmt.Sprintf("tag%d", tagCounter), "name": in["name"], "owner": u.id}
			tags[v["id"].(string)] = v
			write(201, map[string]any{"id": v["id"], "name": v["name"]})
			return
		}
		if r.URL.Path == "/api/v1/tags" && r.Method == "GET" {
			out := []any{}
			for _, v := range tags {
				if v["owner"] == u.id {
					out = append(out, map[string]any{"id": v["id"], "name": v["name"]})
				}
			}
			write(200, out)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/tags/") {
			id := strings.TrimPrefix(r.URL.Path, "/api/v1/tags/")
			v, ok := tags[id]
			if !ok || v["owner"] != u.id {
				write(404, nil)
				return
			}
			if r.Method == "PATCH" {
				var in map[string]string
				_ = json.NewDecoder(r.Body).Decode(&in)
				v["name"] = in["name"]
				write(200, map[string]any{"id": v["id"], "name": v["name"]})
				return
			}
			if r.Method == "DELETE" {
				delete(tags, id)
				write(204, nil)
				return
			}
		}
		if r.URL.Path == "/api/v1/links" && r.Method == "POST" {
			var in map[string]any
			_ = json.NewDecoder(r.Body).Decode(&in)
			if strings.HasPrefix(fmt.Sprint(in["target_url"]), "javascript:") {
				write(400, nil)
				return
			}
			idCounter++
			id := fmt.Sprintf("123e4567-e89b-42d3-a456-%012x", idCounter)
			slug := in["slug"].(string)
			if slug == "" {
				slug = "auto-" + id
			}
			included := []any{}
			if names, ok := in["tags"].([]any); ok {
				for _, n := range names {
					for _, v := range tags {
						if v["name"] == n && v["owner"] == u.id {
							included = append(included, map[string]any{"id": v["id"], "name": v["name"]})
						}
					}
				}
			}
			v := map[string]any{"id": id, "slug": slug, "target_url": in["target_url"], "enabled": true, "tags": included, "created_at": "2026-01-01T00:00:00Z", "updated_at": "2026-01-01T00:00:00Z", "owner": u.id}
			links[id] = v
			slugs[slug] = id
			copy := map[string]any{}
			for k, x := range v {
				if k != "owner" {
					copy[k] = x
				}
			}
			write(201, copy)
			return
		}
		if r.URL.Path == "/api/v1/links" && r.Method == "GET" {
			out := []any{}
			for _, v := range links {
				if v["owner"] == u.id {
					copy := map[string]any{}
					for k, x := range v {
						if k != "owner" {
							copy[k] = x
						}
					}
					out = append(out, copy)
				}
			}
			write(200, out)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/v1/links/") {
			id := strings.TrimPrefix(r.URL.Path, "/api/v1/links/")
			v, ok := links[id]
			if !ok || v["owner"] != u.id {
				write(404, nil)
				return
			}
			if r.Method == "GET" {
				copy := map[string]any{}
				for k, x := range v {
					if k != "owner" {
						copy[k] = x
					}
				}
				write(200, copy)
				return
			}
			if r.Method == "PATCH" {
				var in map[string]any
				_ = json.NewDecoder(r.Body).Decode(&in)
				v["slug"] = in["slug"]
				v["target_url"] = in["target_url"]
				v["enabled"] = in["enabled"]
				if names, ok := in["tags"].([]any); ok {
					included := []any{}
					for _, n := range names {
						for _, tv := range tags {
							if tv["name"] == n && tv["owner"] == u.id {
								included = append(included, map[string]any{"id": tv["id"], "name": tv["name"]})
							}
						}
					}
					v["tags"] = included
				}
				copy := map[string]any{}
				for k, x := range v {
					if k != "owner" {
						copy[k] = x
					}
				}
				write(200, copy)
				return
			}
			if r.Method == "DELETE" {
				delete(links, id)
				write(204, nil)
				return
			}
		}
		if r.Method == "GET" && r.URL.Path != "/" {
			id, ok := slugs[strings.TrimPrefix(r.URL.Path, "/")]
			if !ok || links[id]["enabled"] != true {
				write(404, nil)
				return
			}
			http.Redirect(w, r, links[id]["target_url"].(string), 302)
			return
		}
		write(404, nil)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c, err := newChecker(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	state := c.initial()
	for _, v := range c.checks {
		if !v.Passed {
			t.Errorf("initial check %s failed", v.Name)
		}
	}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	c2, err := newChecker(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	c2.persistence(raw)
	for _, v := range c2.checks {
		if !v.Passed {
			t.Errorf("persistence check %s failed", v.Name)
		}
	}
	if len(c.checks) < 27 || len(c2.checks) != 4 {
		t.Fatalf("unexpected check counts initial=%d persistence=%d", len(c.checks), len(c2.checks))
	}
}

func TestInvalidLoginMustBeRejected(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusOK, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(status) }))
			defer server.Close()
			c, err := newChecker(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			if accepted := c.rejectedLogin("unknown@example.test", "Wrong-password-value"); accepted != (status == http.StatusUnauthorized) {
				t.Fatal("invalid-login verdict accepted wrong status", status)
			}
		})
	}
}
