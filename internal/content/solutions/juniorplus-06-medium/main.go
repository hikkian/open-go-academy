package main

import "net/http"

func NewShortRedirectHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/go/go" && r.URL.Path != "/go/http" {
			http.NotFound(w, r)
			return
		}
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method not allowed", 405)
			return
		}
		target := "/articles/go"
		if r.URL.Path == "/go/http" {
			target = "/guide/http"
		}
		http.Redirect(w, r, target, 302)
	})
}
