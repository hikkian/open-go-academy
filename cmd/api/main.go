package main

import (
	"academy/internal/content"
	"academy/internal/httpapi"
	"academy/internal/runner"
	"academy/internal/store"
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
func main() {
	addr := env("ACADEMY_ADDR", "127.0.0.1:8080")
	if err := httpapi.ValidateAddress(addr); err != nil {
		slog.Error("address rejected", "error", err)
		os.Exit(1)
	}
	dbPath := env("ACADEMY_DB", ".data/academy.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		panic(err)
	}
	s, err := store.Open(dbPath)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	if err = os.Chmod(dbPath, 0600); err != nil {
		panic(err)
	}
	catalog, err := content.Load()
	if err != nil {
		panic(err)
	}
	secret, err := httpapi.Secret(filepath.Join(filepath.Dir(dbPath), "session.secret"))
	if err != nil {
		panic(err)
	}
	_, port, _ := net.SplitHostPort(addr)
	a := httpapi.New(s, catalog, runner.NewClient(env("ACADEMY_RUNNER_SOCKET", ".data/runner.sock")), secret, addr, "localhost:"+port, "127.0.0.1:5173", "localhost:5173")
	srv := &http.Server{Addr: addr, Handler: a.Handler(env("ACADEMY_WEB", "web/dist")), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 200 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()
	slog.Info("academy local API ready", "address", addr)
	if err = srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "error", err)
		os.Exit(1)
	}
}
