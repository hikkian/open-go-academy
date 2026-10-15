package main

import (
	"fmt"
	"io"
	"net/http"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (w *statusRecorder) WriteHeader(n int) {
	if w.status != 0 {
		return
	}
	w.status = n
	w.ResponseWriter.WriteHeader(n)
}
func (w *statusRecorder) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}
func RedactedAccessLog(next http.Handler, dst io.Writer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		if sw.status == 0 {
			sw.status = http.StatusOK
		}
		_, _ = fmt.Fprintf(dst, "method=%s status=%d\n", r.Method, sw.status)
	})
}
