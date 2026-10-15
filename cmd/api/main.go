package main

import (
	"academy/internal/apilock"
	"academy/internal/content"
	"academy/internal/credentials"
	"academy/internal/httpapi"
	"academy/internal/projectrunner"
	"academy/internal/runner"
	"academy/internal/store"
	"academy/internal/tutor"
	"context"
	"crypto/ed25519"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
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
	addr := env("ACADEMY_ADDR", "127.0.0.1:8380")
	if err := httpapi.ValidateAddress(addr); err != nil {
		slog.Error("address rejected", "error", err)
		os.Exit(1)
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		slog.Error("listen failed", "error", err)
		os.Exit(1)
	}
	defer listener.Close()
	dbPath := env("ACADEMY_DB", ".data/academy.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0700); err != nil {
		panic(err)
	}
	guard, err := apilock.Acquire(dbPath)
	if err != nil {
		slog.Error("database in use", "error", err)
		os.Exit(1)
	}
	defer guard.Close()
	s, err := store.Open(dbPath)
	if err != nil {
		panic(err)
	}
	defer s.Close()
	if err := s.RecoverExamAttempts(); err != nil {
		panic(err)
	}
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
	a.ProjectRunner = projectrunner.NewClient(env("ACADEMY_PROJECT_RUNNER_SOCKET", ".data/project-runner.sock"))
	issuerPath := env("ACADEMY_ISSUER_KEY", filepath.Join(filepath.Dir(dbPath), "issuer.key"))
	var credentialCount int
	if err = s.DB().QueryRow(`SELECT COUNT(*) FROM credentials`).Scan(&credentialCount); err != nil {
		panic(err)
	}
	if _, err = os.Lstat(issuerPath); os.IsNotExist(err) && credentialCount > 0 {
		slog.Error("issuer key missing: restore the original key from private backup before restarting")
		os.Exit(1)
	}
	issuerKey, err := credentials.LoadOrCreateKey(issuerPath)
	if err != nil {
		slog.Error("issuer key rejected: regular private key file with mode 0600 required")
		os.Exit(1)
	}
	var mismatchedKeys int
	if err = s.DB().QueryRow(`SELECT COUNT(*) FROM credentials WHERE key_id<>?`, credentials.KeyID(issuerKey.Public().(ed25519.PublicKey))).Scan(&mismatchedKeys); err != nil {
		panic(err)
	}
	if mismatchedKeys > 0 {
		slog.Error("issuer key differs from stored credentials: restore original key; automatic rotation is unsupported")
		os.Exit(1)
	}
	a.IssuerKey = issuerKey
	// Provider access is opt-in configuration; keys never enter browser settings.
	budget, budgetErr := strconv.Atoi(env("ACADEMY_AI_DAILY_TOKENS", "100000"))
	maxOutput, outputErr := strconv.Atoi(env("ACADEMY_AI_MAX_OUTPUT_TOKENS", "512"))
	if budgetErr != nil || outputErr != nil || budget < 1 || budget > 1000000 || maxOutput < 1 || maxOutput > 1024 {
		slog.Error("invalid AI budget configuration")
		os.Exit(1)
	}
	cfg := tutor.Config{Endpoint: strings.TrimSpace(os.Getenv("ACADEMY_AI_ENDPOINT")), APIKey: os.Getenv("ACADEMY_AI_KEY"), EconomyModel: os.Getenv("ACADEMY_AI_MODEL"), AdvancedModel: os.Getenv("ACADEMY_AI_ADVANCED_MODEL"), FallbackModel: os.Getenv("ACADEMY_AI_FALLBACK_MODEL"), AllowCloud: os.Getenv("ACADEMY_AI_ALLOW_CLOUD") == "true", MaxOutputTokens: maxOutput}
	gateway, err := tutor.New(cfg)
	if err != nil {
		slog.Error("AI configuration rejected; check endpoint, model and cloud consent")
		os.Exit(1)
	}
	a.TutorSettings = httpapi.TutorSettings{DailyBudget: budget, MaxOutputTokens: maxOutput}
	if gateway.Enabled() {
		a.Tutor = gateway
		u, _ := url.Parse(cfg.Endpoint)
		a.TutorSettings.Cloud = u.Scheme == "https"
		a.TutorSettings.ProviderLabel = u.Hostname()
		a.TutorSettings.EconomyModel = cfg.EconomyModel
		a.TutorSettings.AdvancedModel = cfg.AdvancedModel
	}
	srv := &http.Server{Addr: addr, Handler: a.Handler(env("ACADEMY_WEB", "web/dist")), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 230 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()
	slog.Info("academy local API ready", "address", addr)
	if err = srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		slog.Error("server", "error", err)
		os.Exit(1)
	}
}
