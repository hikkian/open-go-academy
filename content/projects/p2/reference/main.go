package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if e := run(); e != nil {
		slog.Error("server stopped", "error", e)
		os.Exit(1)
	}
}
func run() error {
	cfg, e := LoadConfig()
	if e != nil {
		return e
	}
	if len(os.Args) == 2 && os.Args[1] == "migrate-down" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return RollbackMigrations(ctx, cfg.DatabaseURL)
	}
	if len(os.Args) > 1 {
		return errors.New("unknown command")
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	store, e := OpenPG(ctx, cfg.DatabaseURL)
	if e != nil {
		return errors.New("database unavailable")
	}
	defer store.Close()
	srv := &http.Server{Addr: cfg.Addr, Handler: NewHandler(store, cfg, logger), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	stop, stopCancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopCancel()
	select {
	case e = <-errCh:
		if errors.Is(e, http.ErrServerClosed) {
			return nil
		}
		return e
	case <-stop.Done():
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	return srv.Shutdown(shutdownCtx)
}
