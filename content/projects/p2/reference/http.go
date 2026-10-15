package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type authSession struct {
	user            User
	csrf, tokenHash string
}
type contextKey int

const sessionKey contextKey = 1

func withSession(c context.Context, s *authSession) context.Context {
	return context.WithValue(c, sessionKey, s)
}
func sessionFrom(c context.Context) *authSession {
	s, _ := c.Value(sessionKey).(*authSession)
	return s
}

type App struct {
	store  Store
	origin string
	secure bool
	log    *slog.Logger
	now    func() time.Time
}

func NewHandler(store Store, c Config, log *slog.Logger) http.Handler {
	a := &App{store: store, origin: c.Origin, secure: c.SecureCookies, log: log, now: time.Now}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.Handle("POST /api/v1/auth/register", a.protectMutation(http.HandlerFunc(a.register)))
	mux.Handle("POST /api/v1/auth/login", a.protectMutation(http.HandlerFunc(a.login)))
	mux.Handle("POST /api/v1/auth/logout", a.authenticate(a.protectMutation(http.HandlerFunc(a.logout))))
	mux.Handle("GET /api/v1/me", a.authenticate(http.HandlerFunc(a.me)))
	mutate := func(h http.Handler) http.Handler { return a.authenticate(a.protectMutation(a.requireAuth(h))) }
	mux.Handle("GET /api/v1/links", a.authenticate(a.requireAuth(http.HandlerFunc(a.listLinks))))
	mux.Handle("POST /api/v1/links", mutate(http.HandlerFunc(a.createLink)))
	mux.Handle("GET /api/v1/links/{id}", a.authenticate(a.requireAuth(http.HandlerFunc(a.getLink))))
	mux.Handle("PATCH /api/v1/links/{id}", mutate(http.HandlerFunc(a.updateLink)))
	mux.Handle("DELETE /api/v1/links/{id}", mutate(http.HandlerFunc(a.deleteLink)))
	mux.Handle("GET /api/v1/tags", a.authenticate(a.requireAuth(http.HandlerFunc(a.listTags))))
	mux.Handle("POST /api/v1/tags", mutate(http.HandlerFunc(a.createTag)))
	mux.Handle("PATCH /api/v1/tags/{id}", mutate(http.HandlerFunc(a.updateTag)))
	mux.Handle("DELETE /api/v1/tags/{id}", mutate(http.HandlerFunc(a.deleteTag)))
	mux.HandleFunc("GET /{slug}", a.redirect)
	return requestLog(a, securityHeaders(mux))
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
		w.ResponseWriter.WriteHeader(status)
	}
}
func (w *statusRecorder) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}
func requestLog(a *App, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := a.now()
		id, _ := randomToken(9)
		w.Header().Set("X-Request-ID", id)
		rw := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rw, r)
		status := rw.status
		if status == 0 {
			status = 200
		}
		if a.log != nil {
			a.log.Info("http request", "request_id", id, "method", r.Method, "status", status, "duration_ms", a.now().Sub(start).Milliseconds())
		}
	})
}
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if e := dec.Decode(dst); e != nil {
		writeError(w, 400, "invalid_json")
		return false
	}
	var extra any
	if e := dec.Decode(&extra); e != io.EOF {
		writeError(w, 400, "invalid_json")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}
func (a *App) createLink(w http.ResponseWriter, r *http.Request) {
	var in LinkInput
	if !decode(w, r, &in) {
		return
	}
	generated := in.Slug == ""
	if generated {
		b := make([]byte, 8)
		if _, e := rand.Read(b); e != nil {
			writeError(w, 500, "internal_error")
			return
		}
		in.Slug = hex.EncodeToString(b)
	}
	if in.Enabled == nil {
		v := true
		in.Enabled = &v
	}
	if !validSlug(in.Slug) || !validTarget(in.TargetURL) || !validTags(in.Tags) {
		writeError(w, 400, "invalid_input")
		return
	}
	var v Link
	var e error
	for attempt := 0; attempt < 4; attempt++ {
		v, e = a.store.CreateLink(sessionFrom(r.Context()).user.ID, in)
		if !generated || !errors.Is(e, errConflict) {
			break
		}
		if attempt == 3 {
			break
		}
		b := make([]byte, 8)
		if _, randomErr := rand.Read(b); randomErr != nil {
			writeError(w, 500, "internal_error")
			return
		}
		in.Slug = hex.EncodeToString(b)
	}
	if e != nil {
		mapStoreError(w, e)
		return
	}
	writeJSON(w, 201, v)
}
func (a *App) listLinks(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, e := strconv.Atoi(raw)
		if e != nil {
			writeError(w, 400, "invalid_pagination")
			return
		}
		limit = parsed
	}
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 100 {
		writeError(w, 400, "invalid_pagination")
		return
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		parsed, e := strconv.Atoi(raw)
		if e != nil {
			writeError(w, 400, "invalid_pagination")
			return
		}
		offset = parsed
	}
	if offset < 0 || offset > 100000 {
		writeError(w, 400, "invalid_pagination")
		return
	}
	v, e := a.store.ListLinks(sessionFrom(r.Context()).user.ID, limit, offset)
	if e != nil {
		mapStoreError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (a *App) getLink(w http.ResponseWriter, r *http.Request) {
	v, e := a.store.GetLink(sessionFrom(r.Context()).user.ID, r.PathValue("id"))
	if e != nil {
		mapStoreError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (a *App) updateLink(w http.ResponseWriter, r *http.Request) {
	var in LinkInput
	if !decode(w, r, &in) {
		return
	}
	if in.Enabled == nil || !validSlug(in.Slug) || !validTarget(in.TargetURL) || !validTags(in.Tags) {
		writeError(w, 400, "invalid_input")
		return
	}
	v, e := a.store.UpdateLink(sessionFrom(r.Context()).user.ID, r.PathValue("id"), in)
	if e != nil {
		mapStoreError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (a *App) deleteLink(w http.ResponseWriter, r *http.Request) {
	if e := a.store.DeleteLink(sessionFrom(r.Context()).user.ID, r.PathValue("id")); e != nil {
		mapStoreError(w, e)
		return
	}
	w.WriteHeader(204)
}
func (a *App) listTags(w http.ResponseWriter, r *http.Request) {
	v, e := a.store.ListTags(sessionFrom(r.Context()).user.ID)
	if e != nil {
		mapStoreError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (a *App) createTag(w http.ResponseWriter, r *http.Request) {
	var in TagInput
	if !decode(w, r, &in) {
		return
	}
	if !validTag(in.Name) {
		writeError(w, 400, "invalid_input")
		return
	}
	v, e := a.store.CreateTag(sessionFrom(r.Context()).user.ID, in)
	if e != nil {
		mapStoreError(w, e)
		return
	}
	writeJSON(w, 201, v)
}
func (a *App) updateTag(w http.ResponseWriter, r *http.Request) {
	var in TagInput
	if !decode(w, r, &in) {
		return
	}
	if !validTag(in.Name) {
		writeError(w, 400, "invalid_input")
		return
	}
	v, e := a.store.UpdateTag(sessionFrom(r.Context()).user.ID, r.PathValue("id"), in)
	if e != nil {
		mapStoreError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (a *App) deleteTag(w http.ResponseWriter, r *http.Request) {
	if e := a.store.DeleteTag(sessionFrom(r.Context()).user.ID, r.PathValue("id")); e != nil {
		mapStoreError(w, e)
		return
	}
	w.WriteHeader(204)
}
func validTags(tags []string) bool {
	if len(tags) > 20 {
		return false
	}
	seen := map[string]bool{}
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if !validTag(t) || seen[strings.ToLower(t)] {
			return false
		}
		seen[strings.ToLower(t)] = true
	}
	return true
}
func mapStoreError(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, errNotFound):
		writeError(w, 404, "not_found")
	case errors.Is(e, errConflict):
		writeError(w, 409, "conflict")
	case errors.Is(e, errInvalid):
		writeError(w, 400, "invalid_input")
	default:
		writeError(w, 500, "internal_error")
	}
}
func (a *App) redirect(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if !validSlug(slug) {
		http.NotFound(w, r)
		return
	}
	target, e := a.store.RedirectTarget(slug)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}
