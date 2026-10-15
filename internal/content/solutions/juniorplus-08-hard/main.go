package main

import (
	"bytes"
	"io"
	"net/http"
)

func LimitBody(next http.Handler, maxBytes int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if maxBytes <= 0 || maxBytes > 1<<20 {
			http.Error(w, "invalid body limit", 500)
			return
		}
		original := r.Body
		defer original.Close()
		body, e := io.ReadAll(io.LimitReader(original, maxBytes+1))
		if e != nil {
			http.Error(w, "invalid request body", 400)
			return
		}
		if int64(len(body)) > maxBytes {
			http.Error(w, "request body too large", 413)
			return
		}
		r.Body = http.NoBody
		if len(body) > 0 {
			r.Body = io.NopCloser(bytes.NewReader(body))
		}
		next.ServeHTTP(w, r)
	})
}
