package main

import "net/url"

func ValidateTarget(raw string) string {
	if len(raw) > 2048 {
		return "target_too_long"
	}
	u, e := url.Parse(raw)
	if e != nil {
		return "invalid_target"
	}
	if u.Scheme == "" {
		return "invalid_target"
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return "unsupported_scheme"
	}
	if u.Host == "" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return "invalid_target"
	}
	return ""
}
