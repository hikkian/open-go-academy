package main

import (
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
