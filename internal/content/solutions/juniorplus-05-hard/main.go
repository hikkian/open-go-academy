package main

import (
	"encoding/json"
	"net/http"
)

func NewLinksCollectionHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/links" {
			http.NotFound(w, r)
			return
		}
		if r.Method != "GET" {
			w.Header().Set("Allow", "GET")
			http.Error(w, "method not allowed", 405)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode([]map[string]string{{"alias": "go", "target": "https://go.dev/"}})
	})
}
