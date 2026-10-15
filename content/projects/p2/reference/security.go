package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"net/url"
	"strings"
	"unicode"
)

var errInvalid = errors.New("invalid input")

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func tokenHash(s string) string  { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func equalHash(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }
func normalizeEmail(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) > 254 {
		return "", errInvalid
	}
	a, e := mail.ParseAddress(s)
	if e != nil || a.Address != s || !strings.Contains(a.Address, "@") {
		return "", errInvalid
	}
	return strings.ToLower(s), nil
}
func validPassword(s string) bool { return len(s) >= 12 && len(s) <= 128 }
func validSlug(s string) bool {
	if len(s) < 3 || len(s) > 64 {
		return false
	}
	if s == "api" || s == "healthz" || s == "favicon-ico" {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return s[0] != '-' && s[len(s)-1] != '-'
}
func validTarget(s string) bool {
	if len(s) == 0 || len(s) > 2048 || strings.TrimSpace(s) != s {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	u, e := url.ParseRequestURI(s)
	return e == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.Hostname() != "" && u.User == nil
}
func validTag(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 1 || len(s) > 40 {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
