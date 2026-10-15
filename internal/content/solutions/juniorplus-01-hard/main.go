package main

import "strings"

func ClassifyExchange(status int, contentType string, hasLocation bool) string {
	if status == 201 && strings.HasPrefix(strings.ToLower(contentType), "application/json") {
		return "created"
	}
	if status >= 300 && status < 400 {
		if hasLocation {
			return "redirect"
		}
		return "inconsistent"
	}
	if status == 404 {
		return "resource-missing"
	}
	if status >= 400 && status < 500 {
		return "client-error"
	}
	if status >= 500 && status < 600 {
		return "server-error"
	}
	return "inconsistent"
}
