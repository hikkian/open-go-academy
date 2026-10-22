package httpapi

import (
	"academy/internal/content"
	"academy/internal/projectreview"
	"academy/internal/projectrunner"
	"academy/internal/runner"
	"academy/internal/store"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

type Executor interface {
	Available(context.Context) bool
	Execute(context.Context, runner.Request) (runner.Result, error)
}
type ProjectExecutor interface {
	Available(context.Context) bool
	Execute(context.Context, projectrunner.Request) (runner.Result, error)
}
type API struct {
	ProjectRunner    ProjectExecutor
	ProjectReviewers *projectreview.Registry
	Store            *store.Store
	Content          *content.Catalog
	Runner           Executor
	Session          string
	Hosts            map[string]bool
	limits           quota
	Tutor            TutorExecutor
	TutorSettings    TutorSettings
	tutorMu          sync.RWMutex
	tutorRevision    string
	tutorLocal       *tutorLocalRuntime
	tutorGate        chan struct{}
	IssuerKey        ed25519.PrivateKey
	VerifyOrigin     string
}

// TutorSnapshot is the synchronized runtime read used by Tutor handlers.
func (a *API) TutorSnapshot() (TutorExecutor, TutorSettings, string) {
	a.tutorMu.RLock()
	defer a.tutorMu.RUnlock()
	return a.Tutor, a.TutorSettings, a.tutorRevision
}

func newTutorRevision() string {
	var revision [24]byte
	if _, err := rand.Read(revision[:]); err == nil {
		return hex.EncodeToString(revision[:])
	}
	return "fallback-" + strconv.FormatInt(time.Now().UnixNano(), 16)
}

// SetTutorRuntime atomically replaces the active provider and its public settings.
func (a *API) SetTutorRuntime(executor TutorExecutor, settings TutorSettings) {
	a.tutorMu.Lock()
	defer a.tutorMu.Unlock()
	a.Tutor, a.TutorSettings = executor, settings
	a.tutorRevision = newTutorRevision()
}

func New(s *store.Store, c *content.Catalog, r Executor, secret string, hosts ...string) *API {
	a := &API{Store: s, Content: c, Runner: r, Session: secret, Hosts: map[string]bool{}, tutorGate: make(chan struct{}, 1), tutorRevision: newTutorRevision()}
	for _, h := range hosts {
		a.Hosts[h] = true
		if a.VerifyOrigin == "" {
			if host, _, err := net.SplitHostPort(h); err == nil {
				if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
					a.VerifyOrigin = "http://" + h
				}
			}
		}
	}
	return a
}
func Secret(path string) (string, error) {
	if b, err := os.ReadFile(path); err == nil {
		if len(b) != 64 {
			return "", errors.New("invalid local session secret")
		}
		return string(b), nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	secret := hex.EncodeToString(b)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return "", err
	}
	_, err = f.WriteString(secret)
	closeErr := f.Close()
	if err != nil {
		return "", err
	}
	return secret, closeErr
}
func write(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, status int, msg string) {
	write(w, status, map[string]string{"error": msg})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	limit := int64(512 << 10)
	if strings.HasPrefix(r.URL.Path, "/api/projects/") || strings.HasPrefix(r.URL.Path, "/api/exams/") {
		limit = 1 << 20
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if err := rejectDuplicateJSONKeys(raw); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	var x any
	if d.Decode(&x) != io.EOF {
		return errors.New("extra JSON")
	}
	return nil
}

func rejectDuplicateJSONKeys(raw []byte) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	var parseValue func() error
	parseValue = func() error {
		token, err := d.Token()
		if err != nil {
			return err
		}
		delim, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delim {
		case '{':
			seen := make(map[string]struct{})
			for d.More() {
				keyToken, err := d.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return errors.New("invalid JSON object key")
				}
				if _, duplicate := seen[key]; duplicate {
					return errors.New("duplicate JSON object key")
				}
				seen[key] = struct{}{}
				if err := parseValue(); err != nil {
					return err
				}
			}
			end, err := d.Token()
			if err != nil || end != json.Delim('}') {
				return errors.New("invalid JSON object")
			}
		case '[':
			for d.More() {
				if err := parseValue(); err != nil {
					return err
				}
			}
			end, err := d.Token()
			if err != nil || end != json.Delim(']') {
				return errors.New("invalid JSON array")
			}
		default:
			return errors.New("invalid JSON value")
		}
		return nil
	}
	if err := parseValue(); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("extra JSON")
	}
	return nil
}

func (a *API) Handler(staticDir string) http.Handler {
	mux := http.NewServeMux()
	a.projectRoutes(mux)
	a.examRoutes(mux)
	a.tutorRoutes(mux)
	a.credentialRoutes(mux)
	mux.HandleFunc("GET /api/curriculum", func(w http.ResponseWriter, r *http.Request) {
		m, err := content.Manifest()
		if err != nil {
			fail(w, 500, "manifest недоступен")
			return
		}
		write(w, 200, m)
	})
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		write(w, 200, map[string]any{"status": "ok", "runner_available": a.Runner != nil && a.Runner.Available(ctx)})
	})
	mux.HandleFunc("GET /api/profile", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "academy_session", Value: a.Session, Path: "/", HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 365 * 86400})
		p, err := a.Store.Profile()
		if errors.Is(err, store.ErrProfileNotFound) {
			write(w, 200, map[string]any{"profile": nil, "csrf_token": a.Session})
			return
		}
		if err != nil {
			fail(w, 500, "не удалось прочитать профиль")
			return
		}
		write(w, 200, map[string]any{"profile": p, "csrf_token": a.Session})
	})
	saveProfile := func(w http.ResponseWriter, r *http.Request) {
		var q struct {
			Name  string `json:"name"`
			Theme string `json:"theme"`
		}
		if decode(w, r, &q) != nil {
			fail(w, 400, "некорректный профиль")
			return
		}
		q.Name = strings.TrimSpace(q.Name)
		if utf8.RuneCountInString(q.Name) < 1 || utf8.RuneCountInString(q.Name) > 64 || (q.Theme != "light" && q.Theme != "dark") {
			fail(w, 422, "имя: 1–64 символа; тема: light или dark")
			return
		}
		var p store.Profile
		var err error
		if r.Method == "POST" {
			p, err = a.Store.CreateProfile(q.Name, q.Theme)
		} else {
			p, err = a.Store.UpdateProfile(q.Name, q.Theme)
		}
		if err != nil {
			fail(w, 409, "профиль уже существует или недоступен")
			return
		}
		write(w, 200, p)
	}
	mux.HandleFunc("POST /api/profile", saveProfile)
	mux.HandleFunc("PUT /api/profile", saveProfile)
	mux.HandleFunc("GET /api/lessons", func(w http.ResponseWriter, r *http.Request) {
		lessons := a.Content.Search(r.URL.Query().Get("q"))
		write(w, 200, lessons)
	})
	mux.HandleFunc("GET /api/lessons/{id}", func(w http.ResponseWriter, r *http.Request) {
		l, ok := a.Content.FindLesson(r.PathValue("id"))
		if !ok {
			fail(w, 404, "урок не найден")
			return
		}
		write(w, 200, l)
	})
	mux.HandleFunc("GET /api/exercises/{id}/solution", func(w http.ResponseWriter, r *http.Request) {
		source, ok := a.Content.Reference(r.PathValue("id"))
		if !ok {
			fail(w, 404, "решение не найдено")
			return
		}
		write(w, 200, map[string]string{"source": source})
	})

	mux.HandleFunc("GET /api/learning", func(w http.ResponseWriter, r *http.Request) {
		p, err := a.Store.Progress()
		if err != nil {
			fail(w, 500, "не удалось прочитать свидетельства прогресса")
			return
		}
		snapshot, _, err := a.learningSnapshot(p)
		if err != nil {
			fail(w, 500, "не удалось собрать проверенные навыки и ранги")
			return
		}
		write(w, 200, snapshot)
	})

	mux.HandleFunc("GET /api/progress", func(w http.ResponseWriter, r *http.Request) {
		p, err := a.Store.Progress()
		if err != nil {
			fail(w, 500, "не удалось прочитать прогресс")
			return
		}
		for i := range p.Attempts {
			p.Attempts[i].Files = nil
			if len(p.Attempts[i].Output) > 4096 {
				p.Attempts[i].Output = p.Attempts[i].Output[:4096] + "… (вывод сокращён в истории)"
			}
		}
		write(w, 200, p)
	})
	mux.HandleFunc("GET /api/drafts/{id}", func(w http.ResponseWriter, r *http.Request) {
		ex, ok := a.Content.FindExercise(r.PathValue("id"))
		if !ok {
			fail(w, 404, "задание не найдено")
			return
		}
		files, err := a.Store.Draft(ex.ID)
		if err != nil {
			fail(w, 500, "не удалось прочитать черновик")
			return
		}
		if len(files) == 0 {
			files = map[string]string{ex.StarterFile: ex.StarterCode}
		}
		write(w, 200, map[string]any{"files": files})
	})
	mux.HandleFunc("PUT /api/drafts/{id}", func(w http.ResponseWriter, r *http.Request) {
		if _, ok := a.Content.FindExercise(r.PathValue("id")); !ok {
			fail(w, 404, "задание не найдено")
			return
		}
		var q runner.Request
		if decode(w, r, &q) != nil || runner.ValidateFiles(q.Files) != nil {
			fail(w, 422, "некорректные Go-файлы (максимум 64 КиБ)")
			return
		}
		if a.Store.SaveDraft(r.PathValue("id"), q.Files) != nil {
			fail(w, 500, "не удалось сохранить черновик")
			return
		}
		write(w, 200, map[string]bool{"saved": true})
	})
	mux.HandleFunc("POST /api/exercises/{id}/{action}", func(w http.ResponseWriter, r *http.Request) {
		ex, ok := a.Content.FindExercise(r.PathValue("id"))
		if !ok {
			fail(w, 404, "задание не найдено")
			return
		}
		action := r.PathValue("action")
		if action != "run" && action != "submit" && action != "format" {
			fail(w, 404, "операция не найдена")
			return
		}
		var q runner.Request
		if decode(w, r, &q) != nil || runner.ValidateFiles(q.Files) != nil {
			fail(w, 422, "некорректные Go-файлы (максимум 64 КиБ)")
			return
		}
		if a.Runner == nil {
			fail(w, 503, "исполнитель недоступен; запустите make runner")
			return
		}
		if ex.Kind == "function" && action != "format" {
			a.functionExercise(w, r, ex.ID, action, q.Files)
			return
		}
		q.Mode = "run"
		if action == "format" {
			q.Mode = "format"
		}
		result, err := a.Runner.Execute(r.Context(), q)
		if err != nil {
			fail(w, 503, err.Error())
			return
		}
		if action == "submit" {
			// The verdict is made here, outside the student's container. Printed PASS is never trusted.
			result.Checks = lessonChecks(ex, result, q.Files)
			result.Passed = true
			for _, c := range result.Checks {
				result.Passed = result.Passed && c.Passed
			}
			if _, err := a.Store.RecordAttempt(ex.ID, q.Files, result.Passed, result.Stdout+result.Stderr); err != nil {
				fail(w, 500, "не удалось сохранить проверку; повторите отправку")
				return
			}
		}
		write(w, 200, result)
	})
	mux.Handle("/api/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fail(w, 404, "API route not found") }))
	fs := http.FileServer(http.Dir(staticDir))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			http.NotFound(w, r)
			return
		}
		clean := filepath.Clean(r.URL.Path)
		if strings.Contains(clean, "..") {
			http.NotFound(w, r)
			return
		}
		if _, err := os.Stat(filepath.Join(staticDir, clean)); err != nil {
			http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; connect-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'")
		if !a.Hosts[r.Host] {
			fail(w, 403, "недопустимый Host")
			return
		}
		origin := r.Header.Get("Origin")
		if origin != "" && origin != "http://"+r.Host {
			fail(w, 403, "недопустимый Origin")
			return
		}
		if site := r.Header.Get("Sec-Fetch-Site"); site == "cross-site" {
			fail(w, 403, "cross-site запрос запрещён")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/profile" && r.URL.Path != "/api/health" && !strings.HasPrefix(r.URL.Path, "/api/verify/") && r.URL.Path != "/api/passport/public" {
			if !a.sessionOK(r) {
				fail(w, 401, "откройте локальный профиль")
				return
			}
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" {
			if !a.sessionOK(r) || subtle.ConstantTimeCompare([]byte(r.Header.Get("X-CSRF-Token")), []byte(a.Session)) != 1 {
				fail(w, 403, "недействительная сессия или CSRF token")
				return
			}
			if r.URL.Path != "/api/profile" {
				if _, err := a.Store.Profile(); err != nil {
					fail(w, 401, "создайте профиль")
					return
				}
			}
			if !a.limits.allow(strings.HasPrefix(r.URL.Path, "/api/exams/") && strings.HasSuffix(r.URL.Path, "/submit") || strings.HasPrefix(r.URL.Path, "/api/exercises/") || strings.HasPrefix(r.URL.Path, "/api/projects/") && (strings.HasSuffix(r.URL.Path, "/run") || strings.HasSuffix(r.URL.Path, "/test") || strings.HasSuffix(r.URL.Path, "/submit") || strings.HasSuffix(r.URL.Path, "/reviews"))) {
				w.Header().Set("Retry-After", "60")
				fail(w, 429, "слишком много запросов; повторите через минуту")
				return
			}

		}
		mux.ServeHTTP(w, r)
	})
}
func (a *API) sessionOK(r *http.Request) bool {
	c, err := r.Cookie("academy_session")
	return err == nil && subtle.ConstantTimeCompare([]byte(c.Value), []byte(a.Session)) == 1
}
func ValidateAddress(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return errors.New("API допускает только числовой loopback адрес; public mode не реализован")
	}
	return nil
}
