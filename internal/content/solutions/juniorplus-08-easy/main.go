package main

import "net/http"

func WithRequestID(next http.Handler, newID func() string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Request-ID", newID())
		next.ServeHTTP(w, r)
	})
}
