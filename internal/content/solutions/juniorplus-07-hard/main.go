package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"
	"strings"
	"unicode/utf8"
)

func ValidateAlias(s string) string {
	if s == "" || strings.TrimSpace(s) != s {
		return "invalid_alias"
	}
	if utf8.RuneCountInString(s) > 24 {
		return "alias_too_long"
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-') {
			return "invalid_alias"
		}
	}
	return ""
}
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
func ValidateCreateJSON(raw string) string {
	if len(raw) > 1024 {
		return "body_too_large"
	}
	d := json.NewDecoder(bytes.NewReader([]byte(raw)))
	d.DisallowUnknownFields()
	var v struct {
		Alias  string
		Target string
	}
	if d.Decode(&v) != nil {
		return "malformed_json"
	}
	var extra any
	if e := d.Decode(&extra); e != io.EOF {
		return "malformed_json"
	}
	if e := ValidateAlias(v.Alias); e != "" {
		return e
	}
	return ValidateTarget(v.Target)
}
