package main

import "net/url"

func ValidateLinkInput(a, t string) string {
	if len(a) < 1 || len(a) > 24 {
		return "invalid_alias"
	}
	for _, r := range a {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return "invalid_alias"
		}
	}
	if len(t) > 2048 {
		return "target_too_long"
	}
	u, e := url.Parse(t)
	if e != nil {
		return "invalid_target"
	}
	if u.Scheme == "" {
		return "invalid_target"
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "unsupported_scheme"
	}
	if u.Host == "" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.Opaque != "" {
		return "invalid_target"
	}
	return ""
}
