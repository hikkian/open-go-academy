package main

import (
	"net/url"
	"strings"
	"unicode/utf8"
)

func SafeRedirectPath(path string) bool {
	if !utf8.ValidString(path) || path == "" || path[0] != '/' || strings.ContainsAny(path, "?#") {
		return false
	}
	decoded, err := url.PathUnescape(path)
	if err != nil || !utf8.ValidString(decoded) || len(decoded) == 0 || decoded[0] != '/' || strings.HasPrefix(decoded, "//") || strings.ContainsAny(decoded, "%?#") {
		return false
	}
	for _, r := range decoded {
		if r <= 0x20 || r == 0x7f || r == '\\' {
			return false
		}
	}
	for _, segment := range strings.Split(decoded, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	return true
}
