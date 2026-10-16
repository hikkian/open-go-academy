package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	maxInput     = 32 << 10
	maxBody      = 8 << 10
	requestLimit = 3 * time.Second
	totalLimit   = 35 * time.Second
)

type request struct {
	ProjectID string          `json:"project_id,omitempty"`
	Phase     string          `json:"phase"`
	BaseURL   string          `json:"base_url"`
	State     json.RawMessage `json:"state,omitempty"`
}
type check struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
}
type response struct {
	Checks []check `json:"checks"`
	State  any     `json:"state,omitempty"`
}
type savedState struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Slug     string `json:"slug"`
	LinkID   string `json:"link_id"`
	Target   string `json:"target"`
	Tag      string `json:"tag"`
}

var (
	slugPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$`)
	uuidPattern  = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)
	emailPattern = regexp.MustCompile(`^learner-[0-9a-f]{16}@example\.test$`)
)

type checker struct {
	base     string
	client   *http.Client
	cookies  map[string]string
	checks   []check
	deadline time.Time
	last     result
}

func main() {
	in, err := io.ReadAll(io.LimitReader(os.Stdin, maxInput+1))
	if err != nil || len(in) > maxInput {
		emit(response{Checks: []check{{Name: "protocol.input", Passed: false}}})
		return
	}
	req, err := parseRequest(in)
	if err != nil {
		emit(response{Checks: []check{{Name: "protocol.input", Passed: false}}})
		return
	}
	c, err := newChecker(req.BaseURL)
	if err != nil {
		emit(response{Checks: []check{{Name: "protocol.target", Passed: false}}})
		return
	}
	projectID := req.ProjectID
	if projectID == "" {
		projectID = "p2"
	}
	if projectID != "p2" && projectID != "p3" {
		c.add("protocol.project", false)
		emit(response{Checks: c.checks})
		return
	}
	var state any
	if req.Phase != "health" && req.Phase != "liveness" && req.Phase != "initial" && req.Phase != "persistence" {
		c.add("protocol.phase", false)
		emit(response{Checks: c.checks})
		return
	}
	switch req.Phase {
	case "health":
		if projectID == "p3" {
			c.taskFlowReadiness()
		} else {
			c.add("healthz", c.call("GET", "/healthz", nil, nil, 200, nil) == nil)
		}
	case "liveness":
		if projectID == "p3" {
			c.taskFlowLiveness()
		} else {
			c.add("healthz", c.call("GET", "/healthz", nil, nil, 200, nil) == nil)
		}
	case "initial":
		if projectID == "p3" {
			state = c.taskFlowInitial()
		} else {
			state = c.initial()
		}
	case "persistence":
		if projectID == "p3" {
			state = c.taskFlowPersistence(req.State)
		} else {
			state = c.persistence(req.State)
		}
	default:
		c.add("protocol.phase", false)
	}
	emit(response{Checks: c.checks, State: state})
}
func parseRequest(in []byte) (request, error) {
	var req request
	if len(in) > maxInput {
		return req, errors.New("input limit")
	}
	dec := json.NewDecoder(bytes.NewReader(in))
	dec.DisallowUnknownFields()
	if dec.Decode(&req) != nil {
		return req, errors.New("invalid request")
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return req, errors.New("trailing request data")
	}
	return req, nil
}

func emit(v response) {
	if v.Checks == nil {
		v.Checks = []check{}
	}
	_ = json.NewEncoder(os.Stdout).Encode(v)
}
func newChecker(raw string) (*checker, error) {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "http" || (u.Hostname() != "127.0.0.1" && u.Hostname() != "::1") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, errors.New("invalid target")
	}
	p, e := strconv.Atoi(u.Port())
	if e != nil || p < 1 || p > 65535 {
		return nil, errors.New("invalid port")
	}
	host := net.JoinHostPort(u.Hostname(), strconv.Itoa(p))
	base := "http://" + host
	tr := &http.Transport{Proxy: nil, DisableKeepAlives: true, ResponseHeaderTimeout: requestLimit, TLSHandshakeTimeout: requestLimit, DialContext: (&net.Dialer{Timeout: requestLimit}).DialContext}
	return &checker{base: base, client: &http.Client{Transport: tr, Timeout: requestLimit, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}, cookies: map[string]string{}, deadline: time.Now().Add(totalLimit)}, nil
}
func (c *checker) add(name string, ok bool) {
	c.checks = append(c.checks, check{Name: name, Passed: ok})
}

type result struct {
	status int
	header http.Header
	body   []byte
}

func (c *checker) call(method, path string, payload any, as map[string]string, want int, out any) error {
	if time.Now().After(c.deadline) {
		return errors.New("deadline")
	}
	var body io.Reader
	if payload != nil {
		b, e := json.Marshal(payload)
		if e != nil || len(b) > maxBody {
			return errors.New("payload")
		}
		body = bytes.NewReader(b)
	}
	remaining := time.Until(c.deadline)
	if remaining <= 0 {
		return errors.New("deadline")
	}
	timeout := requestLimit
	if remaining < timeout {
		timeout = remaining
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	req, e := http.NewRequestWithContext(ctx, method, c.base+path, body)
	if e != nil {
		return e
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range c.cookies {
		req.AddCookie(&http.Cookie{Name: k, Value: v})
	}
	for k, v := range as {
		req.Header.Set(k, v)
	}
	resp, e := c.client.Do(req)
	if e != nil {
		return e
	}
	defer resp.Body.Close()
	for _, cookie := range resp.Cookies() {
		if cookie.MaxAge < 0 || cookie.Value == "" {
			delete(c.cookies, cookie.Name)
		} else {
			c.cookies[cookie.Name] = cookie.Value
		}
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, maxBody+1))
	if e != nil || len(b) > maxBody {
		return errors.New("body limit")
	}
	c.last = result{status: resp.StatusCode, header: resp.Header.Clone(), body: b}
	if resp.StatusCode != want {
		return errors.New("status")
	}
	if out != nil && len(b) > 0 {
		d := json.NewDecoder(bytes.NewReader(b))
		d.DisallowUnknownFields()
		if d.Decode(out) != nil {
			return errors.New("json")
		}
		var x any
		if d.Decode(&x) != io.EOF {
			return errors.New("json")
		}
	}
	return nil
}

type sessionPayload struct {
	User struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"user"`
	CSRF string `json:"csrf_token"`
}
type link struct {
	ID      string `json:"id"`
	Slug    string `json:"slug"`
	Target  string `json:"target_url"`
	Enabled bool   `json:"enabled"`
	Tags    []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"tags"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
type tag struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, e := rand.Read(b); e != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
func (c *checker) auth(method, path, email, password string, out *sessionPayload) error {
	return c.call(method, path, map[string]string{"email": email, "password": password}, map[string]string{"Origin": c.base}, 200, out)
}
func (c *checker) csrf() map[string]string {
	return map[string]string{"Origin": c.base, "X-CSRF-Token": c.cookies["linkhub_csrf"]}
}
func (c *checker) initial() any {
	s := savedState{Email: "learner-" + randomHex(8) + "@example.test", Password: "Academy-" + randomHex(20) + "!", Slug: "link-" + randomHex(8), Target: "https://example.test/academy-" + randomHex(6), Tag: "academy-" + randomHex(5)}
	if strings.Contains(s.Email, "@") && s.Password != "" && s.Slug != "" {
	} else {
		c.add("auth.fixture", false)
		return nil
	}
	var a sessionPayload
	e := c.auth("POST", "/api/v1/auth/register", s.Email, s.Password, &a)
	sessionCookie, sessionCookieOK := cookieByName(c.last.header, "linkhub_session")
	csrfCookie, csrfCookieOK := cookieByName(c.last.header, "linkhub_csrf")
	c.add("auth.register", e == nil && a.CSRF != "" && a.User.Email == s.Email && uuidPattern.MatchString(a.User.ID) && a.User.Role == "user" &&
		sessionCookieOK && sessionCookie.Value != "" && sessionCookie.HttpOnly && sessionCookie.Path == "/" && validCookieSameSite(sessionCookie.SameSite) &&
		csrfCookieOK && csrfCookie.Value == a.CSRF && !csrfCookie.HttpOnly && csrfCookie.Path == "/" && validCookieSameSite(csrfCookie.SameSite))
	if e == nil {
		var me sessionPayload
		e = c.call("GET", "/api/v1/me", nil, nil, 200, &me)
		c.add("auth.me", e == nil && me.User.ID == a.User.ID && me.CSRF == a.CSRF)
	} else {
		c.add("auth.me", false)
	}
	bEmail := "other-" + randomHex(8) + "@example.test"
	var b sessionPayload
	e2 := c.auth("POST", "/api/v1/auth/register", bEmail, "Academy-"+randomHex(20)+"!", &b)
	bCookies := cloneCookies(c.cookies)
	c.add("auth.second_user", e2 == nil && b.User.ID != "" && b.User.ID != a.User.ID)
	c.add("auth.reject_wrong_password", c.rejectedLogin(s.Email, "Wrong-"+randomHex(20)+"!"))
	c.add("auth.reject_unknown_user", c.rejectedLogin("unknown-"+randomHex(8)+"@example.test", "Unknown-"+randomHex(20)+"!"))
	var logged sessionPayload
	e3 := c.auth("POST", "/api/v1/auth/login", s.Email, s.Password, &logged)
	aCookies := cloneCookies(c.cookies)
	c.add("auth.login", e3 == nil && logged.User.ID == a.User.ID && logged.CSRF != "")
	var createdTag tag
	e = c.call("POST", "/api/v1/tags", map[string]string{"name": s.Tag}, c.csrf(), 201, &createdTag)
	c.add("tags.create", e == nil && createdTag.ID != "" && createdTag.Name == s.Tag)
	var tags []tag
	e = c.call("GET", "/api/v1/tags", nil, nil, 200, &tags)
	found := false
	for _, v := range tags {
		if v.ID == createdTag.ID && v.Name == s.Tag {
			found = true
		}
	}
	c.add("tags.list", e == nil && found)
	// Exercise tag ownership and CRUD on a disposable tag so the persisted tag remains intact.
	extraTagName := "temporary-" + randomHex(5)
	var extraTag tag
	e = c.call("POST", "/api/v1/tags", map[string]string{"name": extraTagName}, c.csrf(), 201, &extraTag)
	c.add("tags.create_disposable", e == nil && extraTag.ID != "" && extraTag.Name == extraTagName)
	var l link
	e = c.call("POST", "/api/v1/links", map[string]any{"slug": s.Slug, "target_url": s.Target, "enabled": true, "tags": []string{s.Tag}}, c.csrf(), 201, &l)
	s.LinkID = l.ID
	c.add("links.create", e == nil && l.ID != "" && l.Slug == s.Slug && l.Target == s.Target && len(l.Tags) == 1 && l.Tags[0].Name == s.Tag)
	var list []link
	e = c.call("GET", "/api/v1/links", nil, nil, 200, &list)
	found = false
	for _, v := range list {
		if v.ID == s.LinkID && v.Slug == s.Slug {
			found = true
		}
	}
	c.add("links.list", e == nil && found)
	e = c.call("GET", "/api/v1/links/"+url.PathEscape(s.LinkID), nil, nil, 200, &l)
	c.add("links.get", e == nil && l.ID == s.LinkID && len(l.Tags) == 1)
	var bare link
	e = c.call("POST", "/api/v1/links", map[string]any{"slug": "bare-" + randomHex(6), "target_url": "https://example.test/bare"}, c.csrf(), 201, &bare)
	c.add("links.without_tags", e == nil && len(bare.Tags) == 0)
	c.cookies = cloneCookies(bCookies)
	wrong := false
	if e2 == nil && extraTag.ID != "" {
		wrong = c.call("PATCH", "/api/v1/tags/"+url.PathEscape(extraTag.ID), map[string]string{"name": extraTagName + "-intruder"}, c.csrf(), 404, nil) == nil
	} else {
		wrong = false
	}
	c.add("tags.wrong_owner_update", wrong)
	if e2 == nil && extraTag.ID != "" {
		wrong = c.call("DELETE", "/api/v1/tags/"+url.PathEscape(extraTag.ID), nil, c.csrf(), 404, nil) == nil
	} else {
		wrong = false
	}
	c.add("tags.wrong_owner_delete", wrong)
	if e2 == nil {
		wrong = c.call("GET", "/api/v1/links/"+url.PathEscape(s.LinkID), nil, nil, 404, nil) == nil
	}
	c.add("links.wrong_owner_get", wrong)
	if e2 == nil {
		wrong = c.call("PATCH", "/api/v1/links/"+url.PathEscape(s.LinkID), map[string]any{"slug": s.Slug, "target_url": s.Target, "enabled": true, "tags": []string{}}, c.csrf(), 404, nil) == nil
	}
	c.add("links.wrong_owner_update", wrong)
	if e2 == nil {
		wrong = c.call("DELETE", "/api/v1/links/"+url.PathEscape(s.LinkID), nil, c.csrf(), 404, nil) == nil
	}
	c.add("links.wrong_owner_delete", wrong)
	c.cookies = cloneCookies(aCookies)
	e = c.call("POST", "/api/v1/links", map[string]any{"slug": "csrf-" + randomHex(6), "target_url": "https://example.test"}, map[string]string{"Origin": c.base}, 403, nil)
	c.add("security.missing_csrf", e == nil)
	e = c.call("POST", "/api/v1/links", map[string]any{"slug": "csrf-" + randomHex(6), "target_url": "https://example.test"}, map[string]string{"Origin": c.base, "X-CSRF-Token": "wrong-" + c.cookies["linkhub_csrf"]}, 403, nil)
	c.add("security.wrong_csrf", e == nil)
	e = c.call("POST", "/api/v1/links", map[string]any{"slug": "origin-" + randomHex(6), "target_url": "https://example.test"}, map[string]string{"Origin": "https://foreign.example", "X-CSRF-Token": c.cookies["linkhub_csrf"]}, 403, nil)
	c.add("security.foreign_origin", e == nil)
	e = c.call("POST", "/api/v1/links", map[string]any{"slug": "invalid-" + randomHex(6), "target_url": "javascript:alert(1)"}, c.csrf(), 400, nil)
	c.add("links.reject_bad_target", e == nil)
	if bare.ID != "" {
		e = c.call("PATCH", "/api/v1/links/"+url.PathEscape(bare.ID), map[string]any{"slug": bare.Slug, "target_url": "https://example.test/updated", "enabled": true, "tags": []string{}}, c.csrf(), 200, &bare)
	} else {
		e = errors.New("missing")
	}
	c.add("links.update", e == nil && bare.Target == "https://example.test/updated")
	if bare.ID != "" {
		e = c.call("DELETE", "/api/v1/links/"+url.PathEscape(bare.ID), nil, c.csrf(), 204, nil)
	}
	c.add("links.delete", e == nil)
	e = c.call("GET", "/"+url.PathEscape(s.Slug), nil, nil, 302, nil)
	c.add("links.public_redirect", e == nil && c.last.header.Get("Location") == s.Target)
	e = c.call("PATCH", "/api/v1/links/"+url.PathEscape(s.LinkID), map[string]any{"slug": s.Slug, "target_url": s.Target, "enabled": false, "tags": []string{s.Tag}}, c.csrf(), 200, &l)
	disabled := e == nil && !l.Enabled
	redirect404 := c.call("GET", "/"+url.PathEscape(s.Slug), nil, nil, 404, nil) == nil
	c.add("links.disable_redirect", disabled && redirect404)
	e = c.call("PATCH", "/api/v1/links/"+url.PathEscape(s.LinkID), map[string]any{"slug": s.Slug, "target_url": s.Target, "enabled": true, "tags": []string{s.Tag}}, c.csrf(), 200, &l)
	c.add("links.reenable", e == nil && l.Enabled)
	if extraTag.ID != "" {
		e = c.call("PATCH", "/api/v1/tags/"+url.PathEscape(extraTag.ID), map[string]string{"name": extraTagName + "-renamed"}, c.csrf(), 200, &extraTag)
	} else {
		e = errors.New("missing")
	}
	c.add("tags.rename", e == nil && extraTag.Name == extraTagName+"-renamed")
	if extraTag.ID != "" {
		e = c.call("DELETE", "/api/v1/tags/"+url.PathEscape(extraTag.ID), nil, c.csrf(), 204, nil)
	}
	c.add("tags.delete", e == nil)
	var remainingTags []tag
	e = c.call("GET", "/api/v1/tags", nil, nil, 200, &remainingTags)
	found = false
	for _, v := range remainingTags {
		if v.ID == createdTag.ID && v.Name == s.Tag {
			found = true
		}
	}
	c.add("tags.persisted_after_disposable_delete", e == nil && found)
	e = c.call("POST", "/api/v1/auth/logout", nil, c.csrf(), 204, nil)
	c.add("auth.logout", e == nil)
	revoked := c.call("GET", "/api/v1/me", nil, nil, 401, nil) == nil
	c.add("auth.revoked", revoked)
	return s
}
func cloneCookies(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		out[k] = v
	}
	return out
}
func cookieByName(header http.Header, name string) (*http.Cookie, bool) {
	for _, cookie := range (&http.Response{Header: header}).Cookies() {
		if cookie.Name == name {
			return cookie, true
		}
	}
	return nil, false
}
func validCookieSameSite(site http.SameSite) bool {
	return site == http.SameSiteLaxMode || site == http.SameSiteStrictMode
}
func validExpectedTarget(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host == "example.test" && u.User == nil && u.RawQuery == "" && u.Fragment == "" && strings.HasPrefix(u.Path, "/academy-") && len(strings.TrimPrefix(u.Path, "/academy-")) == 12
}

func (c *checker) persistence(raw json.RawMessage) any {
	var s savedState
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if len(raw) == 0 || d.Decode(&s) != nil || !emailPattern.MatchString(s.Email) || len(s.Password) < 32 || len(s.Password) > 128 || !slugPattern.MatchString(s.Slug) || !uuidPattern.MatchString(s.LinkID) || s.Tag == "" || !strings.HasPrefix(s.Tag, "academy-") || len(s.Tag) != 18 || !validExpectedTarget(s.Target) {
		c.add("persistence.state", false)
		return nil
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		c.add("persistence.state", false)
		return nil
	}
	c.add("persistence.state", true)
	var sess sessionPayload
	e := c.auth("POST", "/api/v1/auth/login", s.Email, s.Password, &sess)
	c.add("persistence.login", e == nil && sess.User.ID != "")
	var l link
	e = c.call("GET", "/api/v1/links/"+url.PathEscape(s.LinkID), nil, nil, 200, &l)
	c.add("persistence.link", e == nil && l.ID == s.LinkID && l.Slug == s.Slug && l.Target == s.Target && l.Enabled && len(l.Tags) == 1 && l.Tags[0].Name == s.Tag)
	e = c.call("GET", "/"+url.PathEscape(s.Slug), nil, nil, 302, nil)
	c.add("persistence.redirect", e == nil && c.last.header.Get("Location") == s.Target)
	return nil
}

func (c *checker) rejectedLogin(email, password string) bool {
	return c.call("POST", "/api/v1/auth/login", map[string]string{"email": email, "password": password}, map[string]string{"Origin": c.base}, http.StatusUnauthorized, nil) == nil
}
