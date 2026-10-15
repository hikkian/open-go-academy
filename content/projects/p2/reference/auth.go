package main

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	passwordIterations = 600000
	passwordKeyBytes   = 32
	sessionCookie      = "linkhub_session"
)

var dummySalt = []byte("linkhub-dummy-salt-v1")
var kdfSlots = make(chan struct{}, 4)
var dummyHash, _ = pbkdf2.Key(sha256.New, "invalid password", dummySalt, passwordIterations, passwordKeyBytes)
var attemptGuard = struct {
	sync.Mutex
	counts map[string]attemptWindow
}{counts: make(map[string]attemptWindow)}

type attemptWindow struct {
	start time.Time
	count int
}

func allowAttempt(remote string, now time.Time) bool {
	ip, _, e := net.SplitHostPort(remote)
	if e != nil {
		ip = remote
	}
	attemptGuard.Lock()
	defer attemptGuard.Unlock()
	if len(attemptGuard.counts) >= 10000 {
		for k, v := range attemptGuard.counts {
			if now.Sub(v.start) >= time.Minute {
				delete(attemptGuard.counts, k)
			}
		}
	}
	w, ok := attemptGuard.counts[ip]
	if !ok || now.Sub(w.start) >= time.Minute {
		if len(attemptGuard.counts) >= 10000 && !ok {
			return false
		}
		w = attemptWindow{start: now}
	}
	if w.count >= 12 {
		return false
	}
	w.count++
	attemptGuard.counts[ip] = w
	return true
}

func passwordHash(password string) ([]byte, error) {
	salt := make([]byte, 16)
	if _, e := rand.Read(salt); e != nil {
		return nil, e
	}
	key, e := derive(password, salt)
	if e != nil {
		return nil, e
	}
	return append(append([]byte{1}, salt...), key...), nil
}
func derive(password string, salt []byte) ([]byte, error) {
	select {
	case kdfSlots <- struct{}{}:
	default:
		return nil, errors.New("password derivation busy")
	}
	defer func() { <-kdfSlots }()
	return pbkdf2.Key(sha256.New, password, salt, passwordIterations, passwordKeyBytes)
}
func checkPassword(password string, stored []byte) bool {
	if len(stored) != 49 || stored[0] != 1 {
		got, _ := derive(password, dummySalt)
		subtle.ConstantTimeCompare(got, dummyHash)
		return false
	}
	got, e := derive(password, stored[1:17])
	return e == nil && subtle.ConstantTimeCompare(got, stored[17:]) == 1
}

func (a *App) register(w http.ResponseWriter, r *http.Request) {
	if !allowAttempt(r.RemoteAddr, a.now()) {
		writeError(w, 429, "try_later")
		return
	}
	var v struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &v) {
		return
	}
	email, e := normalizeEmail(v.Email)
	if e != nil || !validPassword(v.Password) {
		writeError(w, 400, "invalid_input")
		return
	}
	h, e := passwordHash(v.Password)
	if e != nil {
		writeError(w, 503, "temporarily_unavailable")
		return
	}
	u, e := a.store.CreateUser(email, h)
	if e != nil {
		if errors.Is(e, errConflict) {
			writeError(w, 409, "registration_unavailable")
		} else {
			writeError(w, 500, "internal_error")
		}
		return
	}
	a.newSession(w, r, u)
}
func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if !allowAttempt(r.RemoteAddr, a.now()) {
		writeError(w, 429, "try_later")
		return
	}
	var v struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &v) {
		return
	}
	email, _ := normalizeEmail(v.Email)
	password := v.Password
	validLength := len(password) >= 1 && len(password) <= 128
	if !validLength {
		password = "invalid-password"
	}
	u, h, e := a.store.UserByEmail(email)
	ok := false
	if e != nil && !errors.Is(e, errNotFound) {
		writeError(w, 500, "internal_error")
		return
	}
	if e == nil {
		ok = checkPassword(password, h) && validLength
	} else {
		got, _ := derive(password, dummySalt)
		subtle.ConstantTimeCompare(got, dummyHash)
	}
	if !ok {
		writeError(w, 401, "invalid_credentials")
		return
	}
	a.newSession(w, r, u)
}
func (a *App) newSession(w http.ResponseWriter, r *http.Request, u User) {
	tok, e := randomToken(32)
	if e != nil {
		writeError(w, 500, "internal_error")
		return
	}
	csrf, e := randomToken(32)
	if e != nil {
		writeError(w, 500, "internal_error")
		return
	}
	expires := a.now().Add(sessionTTL)
	if e = a.store.CreateSession(tokenHash(tok), tokenHash(csrf), u.ID, expires); e != nil {
		writeError(w, 500, "internal_error")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: tok, Path: "/", Expires: expires, HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteStrictMode})
	http.SetCookie(w, &http.Cookie{Name: "linkhub_csrf", Value: csrf, Path: "/", Expires: expires, Secure: a.secure, SameSite: http.SameSiteStrictMode})
	writeJSON(w, 200, map[string]any{"user": u, "csrf_token": csrf})
}
func (a *App) me(w http.ResponseWriter, r *http.Request) {
	s := sessionFrom(r.Context())
	if s == nil {
		writeError(w, 401, "unauthorized")
		return
	}
	writeJSON(w, 200, map[string]any{"user": s.user, "csrf_token": s.csrf})
}
func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if s := sessionFrom(r.Context()); s != nil {
		if e := a.store.DeleteSession(s.tokenHash); e != nil {
			writeError(w, 500, "internal_error")
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteStrictMode})
	http.SetCookie(w, &http.Cookie{Name: "linkhub_csrf", Value: "", Path: "/", MaxAge: -1, Secure: a.secure, SameSite: http.SameSiteStrictMode})
	w.WriteHeader(http.StatusNoContent)
}

func (a *App) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, e := r.Cookie(sessionCookie)
		if e == nil && len(c.Value) <= 128 {
			u, csrfHash, exp, e := a.store.Session(tokenHash(c.Value))
			csrf, _ := r.Cookie("linkhub_csrf")
			if e == nil && a.now().Before(exp) && csrf != nil && equalHash(tokenHash(csrf.Value), csrfHash) {
				ctx := withSession(r.Context(), &authSession{user: u, csrf: csrf.Value, tokenHash: tokenHash(c.Value)})
				r = r.WithContext(ctx)
			}
		}
		next.ServeHTTP(w, r)
	})
}
func (a *App) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sessionFrom(r.Context()) == nil {
			writeError(w, 401, "unauthorized")
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (a *App) protectMutation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" || r.Method == "HEAD" || r.Method == "OPTIONS" {
			next.ServeHTTP(w, r)
			return
		}
		origin := r.Header.Get("Origin")
		if origin != "" && origin != a.origin {
			writeError(w, 403, "origin_rejected")
			return
		}
		if strings.EqualFold(r.Header.Get("Sec-Fetch-Site"), "cross-site") {
			writeError(w, 403, "origin_rejected")
			return
		}
		s := sessionFrom(r.Context())
		if s != nil && !equalHash(tokenHash(r.Header.Get("X-CSRF-Token")), tokenHash(s.csrf)) {
			writeError(w, 403, "csrf_rejected")
			return
		}
		next.ServeHTTP(w, r)
	})
}
