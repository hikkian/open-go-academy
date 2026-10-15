package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr, DatabaseURL, Origin string
	SecureCookies             bool
}

func LoadConfig() (Config, error) {
	c := Config{Addr: env("LINKHUB_ADDR", "127.0.0.1:8097"), Origin: env("LINKHUB_ORIGIN", "http://127.0.0.1:8097")}
	c.DatabaseURL = os.Getenv("LINKHUB_DATABASE_URL")
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("LINKHUB_DATABASE_URL is required")
	}
	origin, secure, parseErr := normalizeOrigin(c.Origin)
	if parseErr != nil {
		return c, fmt.Errorf("LINKHUB_ORIGIN must be an http(s) origin")
	}
	c.Origin = origin
	c.SecureCookies = secure
	return c, nil
}

func normalizeOrigin(raw string) (string, bool, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", false, err
	}
	scheme := strings.ToLower(u.Scheme)
	if (scheme != "http" && scheme != "https") || u.Host == "" || u.Hostname() == "" || u.User != nil || u.Opaque != "" || (u.Path != "" && u.Path != "/") || u.RawQuery != "" || u.Fragment != "" {
		return "", false, fmt.Errorf("invalid origin")
	}
	host := strings.ToLower(u.Host)
	if scheme == "http" && strings.HasSuffix(host, ":80") {
		host = strings.TrimSuffix(host, ":80")
	}
	if scheme == "https" && strings.HasSuffix(host, ":443") {
		host = strings.TrimSuffix(host, ":443")
	}
	return scheme + "://" + host, scheme == "https", nil
}
func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

const sessionTTL = 12 * time.Hour
