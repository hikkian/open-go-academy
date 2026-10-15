package main

import (
	"encoding/json"
	"net/http"
)

func NewJSONEchoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/echo" {
			http.NotFound(w, r)
			return
		}
		if r.Method != "POST" {
			w.Header().Set("Allow", "POST")
			http.Error(w, "method not allowed", 405)
			return
		}
		var v map[string]any
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		var x struct{ Message string }
		if d.Decode(&x) != nil || x.Message == "" {
			http.Error(w, "invalid request", 400)
			return
		}
		_ = v
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(200)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": x.Message})
	})
}
