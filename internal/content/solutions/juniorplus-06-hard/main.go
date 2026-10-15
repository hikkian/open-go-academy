package main

import (
	"net/http"
	"net/url"
	"strings"
)

func NewSafeRedirectHandler(target string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/go" {
			http.NotFound(w, r)
			return
		}
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method not allowed", 405)
			return
		}
		u, e := url.Parse(target)
		if e != nil || u.Scheme != "" || u.Host != "" || len(u.Path) < 2 || u.Path[0] != '/' || u.Path[1] == '/' || strings.ContainsAny(target, "\\\r\n") {
			http.Error(w, "invalid redirect target", 400)
			return
		}
		http.Redirect(w, r, target, 302)
	})
}
