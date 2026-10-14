package main

import (
	"academy/internal/runner"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	socket := os.Getenv("ACADEMY_RUNNER_SOCKET")
	if socket == "" {
		socket = ".data/runner.sock"
	}
	if err := os.MkdirAll(filepath.Dir(socket), 0700); err != nil {
		slog.Error("socket directory", "error", err)
		os.Exit(1)
	}
	// Never unlink a live server or another file.
	if info, err := os.Lstat(socket); err == nil {
		if info.Mode()&os.ModeSocket != 0 {
			if old, err := net.DialTimeout("unix", socket, time.Second); err != nil {
				if err := os.Remove(socket); err != nil {
					panic(err)
				}
			} else {
				old.Close()
				slog.Error("runner already running")
				os.Exit(1)
			}
		} else {
			slog.Error("socket already exists; stop old runner/remove stale socket", "socket", socket)
			os.Exit(1)
		}
	}
	e := runner.New()
	absolute, _ := filepath.Abs(socket)
	e.Scope = fmt.Sprintf("%x", sha256.Sum256([]byte(absolute)))[:16]
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	err := e.Preflight(ctx)
	cancel()
	if err != nil {
		slog.Error("runner disabled", "error", err)
		os.Exit(1)
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		slog.Error("listen", "error", err)
		os.Exit(1)
	}
	defer os.Remove(socket)
	if err = os.Chmod(socket, 0600); err != nil {
		panic(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if !e.Ready() {
			http.Error(w, "cleanup failure", 503)
			return
		}
		w.Write([]byte("ok"))
	})
	mux.HandleFunc("POST /execute", func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 128<<10)
		var q runner.Request
		d := json.NewDecoder(r.Body)
		d.DisallowUnknownFields()
		if err := d.Decode(&q); err != nil {
			http.Error(w, "invalid job", 400)
			return
		}
		result, err := e.Execute(r.Context(), q)
		if err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})
	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 5 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 30 * time.Second}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()
	slog.Info("local runner ready", "socket", socket, "image", runner.Image)
	if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		slog.Error("serve", "error", err)
	}
}
