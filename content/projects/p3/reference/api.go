package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

type API struct {
	store  *Store
	rates  rateLimiter
	worker *Worker
	ping   func(context.Context) error
	revoke func(context.Context, string) error
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func NewAPI(s *Store, workers ...*Worker) *API {
	a := &API{store: s}
	if s != nil && s.db != nil {
		a.ping = s.db.Ping
		a.revoke = s.RevokeSession
	}
	if len(workers) > 0 {
		a.worker = workers[0]
	}
	return a
}
func (a *API) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc("GET /livez", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "alive"}) })
	m.HandleFunc("GET /readyz", a.ready)
	m.HandleFunc("POST /api/v1/users", a.register)
	m.HandleFunc("POST /api/v1/sessions", a.login)
	m.HandleFunc("DELETE /api/v1/sessions/current", a.logout)
	m.HandleFunc("POST /api/v1/workspaces", a.auth(a.workspace))
	m.HandleFunc("GET /api/v1/workspaces", a.auth(a.workspaces))
	m.HandleFunc("POST /api/v1/workspaces/{workspace}/projects", a.auth(a.project))
	m.HandleFunc("GET /api/v1/workspaces/{workspace}/projects", a.auth(a.projects))
	m.HandleFunc("PUT /api/v1/workspaces/{workspace}/members/{email}", a.auth(a.memberRole))
	m.HandleFunc("POST /api/v1/projects/{project}/tasks", a.auth(a.task))
	m.HandleFunc("GET /api/v1/projects/{project}/tasks", a.auth(a.tasks))
	m.HandleFunc("PATCH /api/v1/tasks/{task}", a.auth(a.updateTask))
	m.HandleFunc("DELETE /api/v1/tasks/{task}", a.auth(a.deleteTask))
	m.HandleFunc("POST /api/v1/tasks/{task}/comments", a.auth(a.comment))
	m.HandleFunc("GET /api/v1/tasks/{task}/comments", a.auth(a.comments))
	return limitBody(requestLog(a.rates.wrap(m)), 1<<20)
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}
func (w *statusWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(p)
}
func requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		status := sw.status
		if status == 0 {
			status = http.StatusOK
		}
		route := r.Pattern
		if route == "" {
			route = "unmatched"
		}
		slog.Info("http request", "method", r.Method, "route", route, "status", status, "duration_ms", time.Since(started).Milliseconds())
	})
}

type bucket struct {
	started time.Time
	count   int
}
type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]bucket
}

func (l *rateLimiter) wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(r.RemoteAddr) {
			w.Header().Set("Retry-After", "60")
			writeError(w, 429)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (l *rateLimiter) allow(remote string) bool {
	if host, _, e := net.SplitHostPort(remote); e == nil {
		remote = host
	}
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.clients == nil {
		l.clients = make(map[string]bucket)
	}
	b, ok := l.clients[remote]
	if !ok {
		if len(l.clients) >= 4096 {
			for k, v := range l.clients {
				if now.Sub(v.started) > time.Minute {
					delete(l.clients, k)
				}
			}
			if len(l.clients) >= 4096 {
				return false
			}
		}
		b = bucket{started: now}
	}
	if now.Sub(b.started) >= time.Minute {
		b = bucket{started: now}
	}
	b.count++
	l.clients[remote] = b
	return b.count <= 120
}
func (a *API) auth(next func(http.ResponseWriter, *http.Request, identity)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, e := r.Cookie("taskflow_session")
		if e != nil {
			writeError(w, 401)
			return
		}
		id, e := a.store.identity(r.Context(), c.Value)
		if e != nil {
			writeError(w, 401)
			return
		}
		next(w, r, id)
	}
}
func (a *API) register(w http.ResponseWriter, r *http.Request) {
	var v struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if decode(r, &v) != nil {
		writeError(w, 400)
		return
	}
	token, e := a.store.Register(r.Context(), v.Email, v.Password)
	if e != nil {
		writeError(w, 400)
		return
	}
	setSession(w, token)
	writeJSON(w, 201, map[string]string{"status": "created"})
}
func (a *API) login(w http.ResponseWriter, r *http.Request) {
	var v struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if decode(r, &v) != nil {
		writeError(w, 400)
		return
	}
	token, e := a.store.Login(r.Context(), v.Email, v.Password)
	if e != nil {
		writeError(w, 401)
		return
	}
	setSession(w, token)
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	c, e := r.Cookie("taskflow_session")
	if e == nil {
		if a.revoke == nil {
			clearSession(w)
			writeError(w, http.StatusServiceUnavailable)
			return
		}
		if err := a.revoke(r.Context(), c.Value); err != nil {
			clearSession(w)
			writeError(w, http.StatusServiceUnavailable)
			return
		}
	}
	clearSession(w)
	w.WriteHeader(204)
}
func clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: "taskflow_session", Value: "", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, MaxAge: -1})
}
func (a *API) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
	defer cancel()
	if a.ping == nil || a.worker == nil || a.ping(ctx) != nil || !a.worker.Ready() {
		writeError(w, http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
func (a *API) workspace(w http.ResponseWriter, r *http.Request, u identity) {
	var v struct {
		Name string `json:"name"`
	}
	if decode(r, &v) != nil || len(strings.TrimSpace(v.Name)) == 0 || len(v.Name) > 100 {
		writeError(w, 400)
		return
	}
	id, e := a.store.CreateWorkspace(r.Context(), u, v.Name)
	if e != nil {
		writeError(w, 400)
		return
	}
	writeJSON(w, 201, map[string]string{"id": id})
}
func (a *API) workspaces(w http.ResponseWriter, r *http.Request, u identity) {
	after := r.URL.Query().Get("after")
	if after != "" && !uuidPattern.MatchString(after) {
		writeError(w, 400)
		return
	}
	items, e := a.store.ListWorkspaces(r.Context(), u, after)
	if e != nil {
		writeError(w, 500)
		return
	}
	writeJSON(w, 200, items)
}
func (a *API) projects(w http.ResponseWriter, r *http.Request, u identity) {
	workspace := r.PathValue("workspace")
	after := r.URL.Query().Get("after")
	if !uuidPattern.MatchString(workspace) || (after != "" && !uuidPattern.MatchString(after)) {
		writeError(w, 400)
		return
	}
	items, e := a.store.ListProjects(r.Context(), u, workspace, after)
	if e != nil {
		writeError(w, 500)
		return
	}
	writeJSON(w, 200, items)
}
func (a *API) memberRole(w http.ResponseWriter, r *http.Request, u identity) {
	email := r.PathValue("email")
	if len(email) < 3 || len(email) > 254 || !strings.Contains(email, "@") {
		writeError(w, 400)
		return
	}
	var v struct {
		Role string `json:"role"`
	}
	if decode(r, &v) != nil || v.Role != "editor" && v.Role != "viewer" {
		writeError(w, 400)
		return
	}
	if e := a.store.SetMemberRole(r.Context(), u, r.PathValue("workspace"), email, v.Role); e != nil {
		if errors.Is(e, ErrDenied) {
			writeError(w, 404)
		} else {
			writeError(w, 400)
		}
		return
	}
	writeJSON(w, 200, map[string]string{"status": "updated"})
}
func (a *API) project(w http.ResponseWriter, r *http.Request, u identity) {
	var v struct {
		Name string `json:"name"`
	}
	if decode(r, &v) != nil || len(strings.TrimSpace(v.Name)) == 0 || len(v.Name) > 120 {
		writeError(w, 400)
		return
	}
	id, e := a.store.CreateProject(r.Context(), u, r.PathValue("workspace"), v.Name)
	if e != nil {
		if errors.Is(e, ErrDenied) {
			writeError(w, 404)
		} else {
			writeError(w, 400)
		}
		return
	}
	writeJSON(w, 201, map[string]string{"id": id})
}
func (a *API) task(w http.ResponseWriter, r *http.Request, u identity) {
	var v struct {
		Title          string `json:"title"`
		Description    string `json:"description"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if decode(r, &v) != nil || len(strings.TrimSpace(v.Title)) == 0 || len(v.Title) > 200 || len(v.Description) > 4000 || len(v.IdempotencyKey) == 0 || len(v.IdempotencyKey) > 128 {
		writeError(w, 400)
		return
	}
	id, created, e := a.store.CreateTask(r.Context(), u, r.PathValue("project"), v.Title, v.Description, v.IdempotencyKey)
	if e != nil {
		if errors.Is(e, ErrDenied) {
			writeError(w, 404)
		} else {
			writeError(w, 400)
		}
		return
	}
	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	writeJSON(w, status, map[string]string{"id": id})
}
func (a *API) comment(w http.ResponseWriter, r *http.Request, u identity) {
	var v struct {
		Body string `json:"body"`
	}
	if decode(r, &v) != nil || len(strings.TrimSpace(v.Body)) == 0 || len(v.Body) > 2000 {
		writeError(w, 400)
		return
	}
	id, e := a.store.AddComment(r.Context(), u, r.PathValue("task"), v.Body)
	if e != nil {
		if errors.Is(e, ErrDenied) {
			writeError(w, 404)
		} else {
			writeError(w, 400)
		}
		return
	}
	writeJSON(w, 201, map[string]string{"id": id})
}

func (a *API) tasks(w http.ResponseWriter, r *http.Request, u identity) {
	after := r.URL.Query().Get("after")
	if !uuidPattern.MatchString(r.PathValue("project")) || (after != "" && !uuidPattern.MatchString(after)) {
		writeError(w, 400)
		return
	}
	items, e := a.store.ListTasks(r.Context(), u, r.PathValue("project"), after)
	if e != nil {
		writeError(w, 500)
		return
	}
	writeJSON(w, 200, items)
}
func (a *API) updateTask(w http.ResponseWriter, r *http.Request, u identity) {
	var v struct {
		State   string `json:"state"`
		Version int    `json:"version"`
	}
	if decode(r, &v) != nil || v.Version < 1 || len(v.State) > 16 {
		writeError(w, 400)
		return
	}
	item, e := a.store.UpdateTaskState(r.Context(), u, r.PathValue("task"), v.State, v.Version)
	if e != nil {
		if errors.Is(e, ErrDenied) {
			writeError(w, 404)
		} else {
			writeError(w, 400)
		}
		return
	}
	writeJSON(w, 200, item)
}

func (a *API) deleteTask(w http.ResponseWriter, r *http.Request, u identity) {
	if e := a.store.DeleteTask(r.Context(), u, r.PathValue("task")); e != nil {
		if errors.Is(e, ErrDenied) {
			writeError(w, 404)
		} else {
			writeError(w, 400)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (a *API) comments(w http.ResponseWriter, r *http.Request, u identity) {
	after := r.URL.Query().Get("after")
	if !uuidPattern.MatchString(r.PathValue("task")) || (after != "" && !uuidPattern.MatchString(after)) {
		writeError(w, 400)
		return
	}
	items, e := a.store.ListComments(r.Context(), u, r.PathValue("task"), after)
	if e != nil {
		writeError(w, 500)
		return
	}
	writeJSON(w, 200, items)
}
func decode(r *http.Request, v any) error {
	d := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return e
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		return errors.New("one JSON document required")
	}
	return nil
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int) {
	writeJSON(w, status, map[string]string{"error": http.StatusText(status)})
}
func setSession(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{Name: "taskflow_session", Value: token, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteStrictMode, Expires: time.Now().Add(24 * time.Hour)})
}
func limitBody(next http.Handler, limit int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		next.ServeHTTP(w, r)
	})
}
