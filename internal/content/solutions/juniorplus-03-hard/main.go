package main

import "net/url"

func RequireHTTPS(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "invalid-url"
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		if u.Scheme == "" {
			return "invalid-url"
		}
		return "unsupported-scheme"
	}
	if u.Host == "" || u.Hostname() == "" || u.User != nil {
		return "invalid-url"
	}
	if u.Scheme == "http" {
		return "https-required"
	}
	return "https-selected"
}
