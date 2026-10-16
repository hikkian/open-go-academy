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

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		slog.Error("service stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	dsn := os.Getenv("TASKFLOW_DATABASE_URL")
	if dsn == "" {
		slog.Error("TASKFLOW_DATABASE_URL is required")
		return errors.New("TASKFLOW_DATABASE_URL is required")
	}
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return errors.New("database configuration failed")
	}
	defer db.Close()
	if err = db.Ping(ctx); err != nil {
		return errors.New("database unavailable")
	}
	if err = migrate(ctx, db); err != nil {
		return errors.New("database migration failed")
	}
	if len(os.Args) > 1 && os.Args[1] == "migrate-down" {
		if err = rollback(ctx, db); err != nil {
			return errors.New("database rollback failed")
		}
		slog.Info("database migration rolled back")
		return nil
	}
	store := &Store{db: db}
	worker := NewWorker(store, 2, 8, 5)
	workerCtx, stop := context.WithCancel(ctx)
	workerDone := make(chan struct{})
	go func() { defer close(workerDone); worker.Run(workerCtx) }()
	server := &http.Server{Addr: env("TASKFLOW_ADDR", "127.0.0.1:8098"), Handler: NewAPI(store, worker).Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.ListenAndServe() }()
	signals, cancelSignals := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancelSignals()
	var listenFailure bool
	select {
	case err = <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed")
			listenFailure = true
		}
	case <-signals.Done():
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err = server.Shutdown(shutdownCtx); err != nil {
		slog.Error("graceful HTTP shutdown timed out")
	}
	stop()
	joinCtx, joinCancel := context.WithTimeout(context.Background(), 5*time.Second)
	joined := waitForWorker(joinCtx, workerDone)
	joinCancel()
	if !joined {
		slog.Error("worker did not stop before shutdown deadline; skipping drain")
		return errors.New("worker shutdown deadline exceeded")
	}
	drainCtx, drainCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer drainCancel()
	if err = worker.Drain(drainCtx); err != nil {
		slog.Error("worker drain ended with pending work")
	}
	if listenFailure {
		return errors.New("HTTP listener failed")
	}
	return nil
}
func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
