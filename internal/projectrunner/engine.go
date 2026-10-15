package projectrunner

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"academy/internal/runner"
)

const jobTimeout = 180 * time.Second
const cleanupTimeout = 20 * time.Second
const maxCommandOutput = 32 << 10
const readinessTimeout = 35 * time.Second

type command func(context.Context, []byte, ...string) ([]byte, error)
type engine struct {
	cfg        Config
	slot       chan struct{}
	run        command
	install    string
	poisoned   atomic.Bool
	checkerSeq atomic.Uint32
}

func New(cfg Config) (Engine, error) {
	if cfg.Socket == "" || cfg.GoImage == "" || cfg.CheckerImage == "" || cfg.ExpectedGoMod == "" || cfg.ExpectedGoSum == "" {
		return nil, errors.New("project runner configuration incomplete")
	}
	if cfg.PostgresImage == "" {
		cfg.PostgresImage = PostgresImage
	}
	for _, image := range []string{cfg.GoImage, cfg.CheckerImage, cfg.PostgresImage} {
		if !validImage(image) {
			return nil, errors.New("container images must be immutable digest references or local image IDs")
		}
	}
	canonicalSocket, err := canonicalSocket(cfg.Socket)
	if err != nil {
		return nil, err
	}
	h := sha256.Sum256([]byte(canonicalSocket))
	e := &engine{cfg: cfg, slot: make(chan struct{}, 1), install: hex.EncodeToString(h[:])}
	e.run = func(ctx context.Context, input []byte, args ...string) ([]byte, error) {
		runCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		c := exec.CommandContext(runCtx, "podman", args...)
		if input != nil {
			c.Stdin = bytes.NewReader(input)
		}
		output := &limitedOutput{cancel: cancel}
		c.Stdout, c.Stderr = output, output
		err := c.Run()
		if output.exceeded {
			return output.Bytes(), errors.New("podman output limit exceeded")
		}
		return output.Bytes(), err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := e.preflight(ctx); err != nil {
		return nil, err
	}
	if err := e.reconcile(ctx); err != nil {
		return nil, err
	}
	return e, nil
}

func canonicalSocket(socket string) (string, error) {
	abs, err := filepath.Abs(socket)
	if err != nil {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return "", fmt.Errorf("resolve project runner socket directory: %w", err)
	}
	return filepath.Join(parent, filepath.Base(abs)), nil
}

type limitedOutput struct {
	bytes.Buffer
	cancel   context.CancelFunc
	exceeded bool
	mu       sync.Mutex
}

func (w *limitedOutput) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	remaining := maxCommandOutput - w.Len()
	if remaining < 0 {
		remaining = 0
	}
	if len(p) > remaining {
		if remaining > 0 {
			_, _ = w.Buffer.Write(p[:remaining])
		}
		w.exceeded = true
		w.cancel()
		return len(p), nil
	}
	return w.Buffer.Write(p)
}

var digestImage = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/:+-]*@sha256:[0-9a-f]{64}$`)
var localImage = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func validImage(s string) bool { return digestImage.MatchString(s) || localImage.MatchString(s) }
func (e *engine) pod(ctx context.Context, args ...string) ([]byte, error) {
	return e.run(ctx, nil, args...)
}
func (e *engine) preflight(ctx context.Context) error {
	out, err := e.pod(ctx, "info", "--format", "{{.Host.Security.Rootless}} {{.Host.CgroupsVersion}}")
	if err != nil || strings.TrimSpace(string(out)) != "true v2" {
		return errors.New("rootless Podman with cgroup v2 is required")
	}
	for _, img := range []string{e.cfg.GoImage, e.cfg.CheckerImage, e.cfg.PostgresImage} {
		if _, err = e.pod(ctx, "image", "exists", img); err != nil {
			return errors.New("required immutable project image is not cached")
		}
	}
	return nil
}
func (e *engine) Available(ctx context.Context) bool {
	return !e.poisoned.Load() && e.preflight(ctx) == nil
}
func (e *engine) labels(job string) []string {
	return []string{"--label", "academy.project-runner=true", "--label", "academy.install=" + e.install, "--label", "academy.job=" + job}
}
func (e *engine) reconcile(ctx context.Context) error {
	return e.removeOwned(ctx, "label=academy.project-runner=true", "label=academy.install="+e.install)
}
func (e *engine) removeOwned(ctx context.Context, filters ...string) error {
	list := append([]string{"ps", "-aq"}, flattenFilters(filters)...)
	out, err := e.pod(ctx, list...)
	if err != nil {
		return fmt.Errorf("cannot inspect owned project containers: %w", err)
	}
	pending := strings.Fields(string(out))
	maxAttempts := len(pending) + 1
	var finalFailures []error
	for attempt := 0; len(pending) > 0 && attempt < maxAttempts; attempt++ {
		next := make([]string, 0, len(pending))
		failures := make([]error, 0, len(pending))
		for _, id := range pending {
			if _, x := e.pod(ctx, "rm", "-fv", id); x != nil {
				next = append(next, id)
				failures = append(failures, fmt.Errorf("remove owned container %s: %w", id, x))
			}
		}
		pending, finalFailures = next, failures
	}
	var failures []error
	if len(pending) > 0 {
		failures = append(failures, finalFailures...)
	}
	remaining, err := e.pod(ctx, list...)
	if err != nil {
		failures = append(failures, fmt.Errorf("verify owned container cleanup: %w", err))
	} else if ids := strings.Fields(string(remaining)); len(ids) > 0 {
		failures = append(failures, fmt.Errorf("owned containers remain after cleanup: %s", strings.Join(ids, ",")))
	}
	return errors.Join(failures...)
}
func flattenFilters(filters []string) []string {
	out := make([]string, 0, len(filters)*2)
	for _, f := range filters {
		out = append(out, "--filter", f)
	}
	return out
}
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return hex.EncodeToString(b), nil
}
func tarFiles(files map[string]string) ([]byte, error) {
	var b bytes.Buffer
	tw := tar.NewWriter(&b)
	for name, data := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(data)), Typeflag: tar.TypeReg}); err != nil {
			return nil, err
		}
		if _, err := io.WriteString(tw, data); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
func (e *engine) Execute(parent context.Context, req Request) (result runner.Result, err error) {
	if err := validate(req, e.cfg); err != nil {
		return runner.Result{}, err
	}
	select {
	case e.slot <- struct{}{}:
		defer func() { <-e.slot }()
	default:
		return runner.Result{}, errors.New("project worker busy")
	}
	if e.poisoned.Load() {
		return runner.Result{}, errors.New("project runner unavailable")
	}
	ctx, cancel := context.WithTimeout(parent, jobTimeout)
	defer cancel()
	job, err := randomHex(12)
	if err != nil {
		return runner.Result{}, err
	}
	bootstrapPass, err := randomHex(24)
	if err != nil {
		return runner.Result{}, err
	}
	appPass, err := randomHex(24)
	if err != nil {
		return runner.Result{}, err
	}
	files, err := tarFiles(req.Files)
	if err != nil {
		return runner.Result{}, err
	}
	name := func(s string) string { return "academy-" + job + "-" + s }
	db := name("db")
	labels := e.labels(job)
	dbArgs := append([]string{"run", "--pull=never", "-d", "--name", db}, labels...)
	dbArgs = append(dbArgs, "--stop-timeout=0", "--http-proxy=false", "--log-driver=k8s-file", "--log-opt=max-size=32kb", "--network", "none", "--user", "999:999", "--read-only", "--cap-drop=all", "--security-opt=no-new-privileges", "--cpus=1", "--memory=512m", "--memory-swap=512m", "--pids-limit=128", "--shm-size=64m", "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=64m,mode=1777", "--tmpfs", "/var/lib/postgresql:rw,noexec,nosuid,nodev,size=384m,mode=1777", "--tmpfs", "/var/run/postgresql:rw,noexec,nosuid,nodev,size=16m,mode=1777", "-e", "POSTGRES_PASSWORD="+bootstrapPass, "-e", "POSTGRES_INITDB_ARGS=--auth-host=scram-sha-256", e.cfg.PostgresImage)
	cleanup := func() error {
		cc, stop := context.WithTimeout(context.Background(), cleanupTimeout)
		defer stop()
		return e.removeOwned(cc, "label=academy.project-runner=true", "label=academy.install="+e.install, "label=academy.job="+job)
	}
	defer func() {
		if cleanupErr := cleanup(); cleanupErr != nil {
			e.poisoned.Store(true)
			result.Passed = false
			result.Status = "error"
			result.ExitCode = 1
			result.Stdout, result.Stderr = "", ""
			err = errors.Join(err, fmt.Errorf("project job cleanup failed: %w", cleanupErr))
		}
	}()
	if _, err = e.pod(ctx, dbArgs...); err != nil {
		return runner.Result{Status: "error", ExitCode: 1}, errors.New("database sandbox failed to start")
	}
	if err = e.waitDB(ctx, db); err != nil {
		return runner.Result{Status: "error", ExitCode: 1}, err
	}
	// Bootstrap is fixed SQL; generated credentials contain only lowercase hex.
	role := "student_" + job
	sql := fmt.Sprintf("CREATE ROLE %s LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOREPLICATION PASSWORD '%s'", role, appPass)
	if _, err = e.pod(ctx, "exec", db, "psql", "-U", "postgres", "-d", "postgres", "-v", "ON_ERROR_STOP=1", "-c", sql); err != nil {
		return runner.Result{Status: "error", ExitCode: 1}, errors.New("database bootstrap failed")
	}
	if _, err = e.pod(ctx, "exec", db, "createdb", "-U", "postgres", "-O", role, "academy"); err != nil {
		return runner.Result{Status: "error", ExitCode: 1}, errors.New("database bootstrap failed")
	}
	dsn := "postgres://" + role + ":" + appPass + "@127.0.0.1:5432/academy?sslmode=disable"
	app := name("app")
	var build []byte
	if req.Action == "test" {
		build, err = e.runStudent(ctx, app, db, files, dsn, true, labels)
	} else {
		build, err = e.runStudent(ctx, app, db, files, dsn, false, labels)
	}
	if err != nil {
		result := runner.Result{Status: "error", ExitCode: 1}
		if req.Action != "submit" {
			result.Stderr = bounded(redact(string(build), bootstrapPass, appPass, dsn))
		}
		return result, nil
	}
	if req.Action == "run" {
		checked, diagnostics, readyErr := e.waitReady(ctx, job, db, app, labels)
		if readyErr != nil {
			msg := "Сервер не прошёл проверку готовности."
			if len(diagnostics) > 0 {
				msg = bounded(redact(string(diagnostics), bootstrapPass, appPass, dsn))
			}
			return runner.Result{Status: "error", ExitCode: 1, Passed: false, Checks: checked.Checks, Stderr: msg}, nil
		}
		return runner.Result{Status: "ok", ExitCode: 0, Stdout: "Сервер запущен в изолированной среде.", Passed: false, Checks: checked.Checks}, nil
	}
	if req.Action == "test" {
		return runner.Result{Status: "ok", ExitCode: 0, Stdout: bounded(redact(string(build), bootstrapPass, appPass, dsn)), Passed: false, Checks: []runner.Check{{Name: "Пользовательские Go-тесты выполнены (результат требует ревью)", Passed: true}}}, nil
	}
	result, err = e.submit(ctx, job, db, app, dsn, labels, files)
	if err != nil {
		return runner.Result{}, err
	}
	return result, nil
}

func (e *engine) waitReady(ctx context.Context, job, db, app string, labels []string) (checkerOutput, []byte, error) {
	deadline := time.Now().Add(readinessTimeout)
	var latest checkerOutput
	for {
		state, inspectErr := e.pod(ctx, "inspect", "--format", "{{.State.Running}}", app)
		if inspectErr != nil || strings.TrimSpace(string(state)) != "true" {
			logs, _ := e.pod(ctx, "logs", app)
			return latest, logs, errors.New("student server exited before readiness")
		}
		out, err := e.runChecker(ctx, job, db, labels, checkerInput{Phase: "health", BaseURL: "http://127.0.0.1:8097"})
		if err != nil {
			return latest, nil, err
		}
		latest = out
		ready, hasHealth := false, false
		for _, check := range out.Checks {
			if strings.TrimSpace(check.Name) == "" {
				return latest, nil, errors.New("invalid checker check")
			}
			if check.Name == "healthz" {
				hasHealth, ready = true, check.Passed
			}
		}
		if ready && hasHealth {
			return latest, nil, nil
		}
		if time.Now().After(deadline) {
			return latest, nil, errors.New("server readiness timed out")
		}
		select {
		case <-ctx.Done():
			return latest, nil, ctx.Err()
		case <-time.After(250 * time.Millisecond):
		}
	}
}

func (e *engine) waitDB(ctx context.Context, db string) error {
	for {
		// The image initializes with a temporary Unix-only server. Wait for
		// the final TCP listener before creating the application role/database.
		if _, err := e.pod(ctx, "exec", db, "pg_isready", "-h", "127.0.0.1", "-U", "postgres"); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.New("database startup timed out")
		case <-time.After(250 * time.Millisecond):
		}
	}
}
func (e *engine) runStudent(ctx context.Context, app, db string, files []byte, dsn string, tests bool, labels []string) ([]byte, error) {
	waiter := "while [ ! -f /work/READY ]; do sleep 0.1; done; cd /work/src && go build -mod=readonly -o /work/build/server . && exec /work/build/server"
	args := []string{"run", "--pull=never", "-d", "--name", app}
	args = append(args, labels...)
	args = append(args, "--stop-timeout=0", "--log-driver=k8s-file", "--log-opt=max-size=32kb", "--network", "container:"+db, "--user", "65532:65532", "--read-only", "--cap-drop=all", "--security-opt=no-new-privileges", "--http-proxy=false", "--cpus=1", "--memory=1536m", "--memory-swap=1536m", "--pids-limit=128", "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=128m", "--tmpfs", "/work:rw,nosuid,nodev,size=512m", "--tmpfs", "/cache:rw,nosuid,nodev,size=512m", "-e", "HOME=/tmp", "-e", "GOPROXY=off", "-e", "GOSUMDB=off", "-e", "GOTOOLCHAIN=local", "-e", "CGO_ENABLED=0", "-e", "GOFLAGS=", "-e", "GOMODCACHE=/opt/academy/gomod", "-e", "GOCACHE=/cache", "-e", "GOTMPDIR=/work/build", "-e", "LINKHUB_ADDR=0.0.0.0:8097", "-e", "LINKHUB_ORIGIN=http://127.0.0.1:8097", "-e", "LINKHUB_DATABASE_URL="+dsn, e.cfg.GoImage, "/bin/sh", "-c", waiter)
	if _, err := e.pod(ctx, args...); err != nil {
		return nil, err
	}
	if _, err := e.pod(ctx, "exec", app, "/bin/sh", "-c", "mkdir -p /work/src /work/build /cache && if [ -d /opt/academy/gocache ]; then cp -a /opt/academy/gocache/. /cache/; fi"); err != nil {
		return nil, err
	}
	if _, err := e.run(ctx, files, "exec", "-i", app, "tar", "-xf", "-", "-C", "/work/src"); err != nil {
		return nil, err
	}
	if tests {
		return e.pod(ctx, "exec", app, "/bin/sh", "-c", "cd /work/src && go test -mod=readonly ./...")
	}
	if _, err := e.pod(ctx, "exec", app, "touch", "/work/READY"); err != nil {
		return nil, err
	}
	out, err := e.pod(ctx, "inspect", "--format", "{{.State.Running}}", app)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(out)) != "true" {
		logs, _ := e.pod(ctx, "logs", app)
		return logs, errors.New("student server exited during build")
	}
	return nil, nil
}
func bounded(s string) string {
	if len(s) > 32<<10 {
		return s[:32<<10]
	}
	return s
}
func redact(s string, secrets ...string) string {
	for _, secret := range secrets {
		if secret != "" {
			s = strings.ReplaceAll(s, secret, "[REDACTED]")
		}
	}
	return s
}

type checkerInput struct {
	Phase   string          `json:"phase"`
	BaseURL string          `json:"base_url"`
	State   json.RawMessage `json:"state,omitempty"`
}
type checkerOutput struct {
	Checks []runner.Check  `json:"checks"`
	State  json.RawMessage `json:"state,omitempty"`
}

type checkerFixture struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Slug     string `json:"slug"`
	LinkID   string `json:"link_id"`
	Target   string `json:"target"`
	Tag      string `json:"tag"`
}

const persistedFixtureSQL = `SET search_path = pg_catalog;
SELECT EXISTS (
 SELECT 1 FROM public.users u
 JOIN public.links l ON l.owner_id = u.id
 JOIN public.link_tags lt ON lt.link_id = l.id
 JOIN public.tags t ON t.id = lt.tag_id AND t.owner_id = u.id
 WHERE u.email_normalized = :'email'
   AND l.slug = :'slug'
   AND l.id::text = :'link_id'
   AND l.target_url = :'target'
   AND l.enabled = true
   AND t.name = :'tag'
) AND EXISTS (
 SELECT 1 FROM public.users u
 JOIN public.sessions s ON s.user_id = u.id
 WHERE u.email_normalized = :'email'
   AND s.expires_at > now()
);`

func (e *engine) persistedFixtureCheck(ctx context.Context, db string, appRole string, raw json.RawMessage) runner.Check {
	check := runner.Check{Name: "Ссылка, тег-связь и активная сессия сохранены в PostgreSQL", Passed: false}
	var fixture checkerFixture
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&fixture) != nil {
		return check
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return check
	}
	if fixture.Email == "" || fixture.Password == "" || fixture.Slug == "" || fixture.LinkID == "" || fixture.Target == "" || fixture.Tag == "" || len(fixture.Email) > 254 || len(fixture.Slug) > 64 || len(fixture.LinkID) > 64 || len(fixture.Target) > 2048 || len(fixture.Tag) > 40 {
		return check
	}
	args := []string{"exec", "-i", db, "psql", "-X", "-U", appRole, "-d", "academy", "-v", "ON_ERROR_STOP=1", "-qtA", "-v", "email=" + fixture.Email, "-v", "slug=" + fixture.Slug, "-v", "link_id=" + fixture.LinkID, "-v", "target=" + fixture.Target, "-v", "tag=" + fixture.Tag, "-f", "-"}
	// The checker password is deliberately not forwarded to psql or interpolated into SQL.
	out, err := e.run(ctx, []byte(persistedFixtureSQL), args...)
	check.Passed = err == nil && strings.TrimSpace(string(out)) == "t"
	return check
}

func (e *engine) submit(ctx context.Context, job, db, app, dsn string, labels []string, files []byte) (runner.Result, error) {
	ready, _, err := e.waitReady(ctx, job, db, app, labels)
	if err != nil {
		return runner.Result{Status: "error", ExitCode: 1, Passed: false, Checks: ready.Checks, Stderr: "Сервер не прошёл проверку готовности."}, nil
	}
	// Trusted checker executes in its own image and receives only its JSON protocol on stdin.
	first, err := e.runChecker(ctx, job, db, labels, checkerInput{Phase: "initial", BaseURL: "http://127.0.0.1:8097"})
	if err != nil {
		return runner.Result{}, err
	}
	// Restart user code from a fresh container while preserving only the database.
	if _, err = e.pod(ctx, "rm", "-f", app); err != nil {
		return runner.Result{}, errors.New("cannot stop phase-one server")
	}
	app2 := "academy-" + job + "-app2"
	if _, err = e.runStudent(ctx, app2, db, files, dsn, false, labels); err != nil {
		return runner.Result{Status: "error", ExitCode: 1}, nil
	}
	ready2, _, err := e.waitReady(ctx, job, db, app2, labels)
	if err != nil {
		return runner.Result{Status: "error", ExitCode: 1, Passed: false, Checks: append(ready.Checks, ready2.Checks...), Stderr: "Сервер после перезапуска не прошёл проверку готовности."}, nil
	}
	second, err := e.runChecker(ctx, job, db, labels, checkerInput{Phase: "persistence", BaseURL: "http://127.0.0.1:8097", State: first.State})
	if err != nil {
		return runner.Result{}, err
	}
	checks := append(append(append(ready.Checks, first.Checks...), ready2.Checks...), second.Checks...)
	checks = append(checks, e.persistedFixtureCheck(ctx, db, "student_"+job, first.State))
	for _, table := range []string{"users", "sessions", "links", "tags", "link_tags"} {
		out, x := e.pod(ctx, "exec", db, "psql", "-X", "-U", "student_"+job, "-d", "academy", "-tAc", "SELECT pg_catalog.to_regclass('public."+table+"') IS NOT NULL")
		if x != nil || strings.TrimSpace(string(out)) != "t" {
			checks = append(checks, runner.Check{Name: "Обязательная таблица " + table, Passed: false})
		} else {
			checks = append(checks, runner.Check{Name: "Обязательная таблица " + table, Passed: true})
		}
	}
	passed := len(checks) > 0
	for _, c := range checks {
		if strings.TrimSpace(c.Name) == "" {
			return runner.Result{}, errors.New("invalid checker check")
		}
		passed = passed && c.Passed
	}
	return runner.Result{Status: "ok", ExitCode: 0, Passed: passed, Checks: checks}, nil
}

func (e *engine) runChecker(ctx context.Context, job, db string, labels []string, input checkerInput) (checkerOutput, error) {
	checker := "academy-" + job + "-checker-" + input.Phase + "-" + fmt.Sprint(e.checkerSeq.Add(1))
	b, _ := json.Marshal(input)
	args := []string{"run", "--pull=never", "--rm", "--name", checker}
	args = append(args, labels...)
	args = append(args, "--stop-timeout=0", "--http-proxy=false", "--network", "container:"+db, "--user", "65532:65532", "--read-only", "--cap-drop=all", "--security-opt=no-new-privileges", "--cpus=1", "--memory=768m", "--memory-swap=768m", "--pids-limit=128", "--tmpfs", "/tmp:rw,noexec,nosuid,nodev,size=32m", "--tmpfs", "/work:rw,noexec,nosuid,nodev,size=16m", "-i", e.cfg.CheckerImage, "/opt/academy/checker")
	out, err := e.run(ctx, b, args...)
	if err != nil {
		return checkerOutput{}, errors.New("trusted checker failed")
	}
	if len(out) > 32<<10 {
		return checkerOutput{}, errors.New("checker response too large")
	}
	var result checkerOutput
	d := json.NewDecoder(bytes.NewReader(out))
	d.DisallowUnknownFields()
	if d.Decode(&result) != nil || len(result.Checks) == 0 {
		return checkerOutput{}, errors.New("invalid checker response")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return checkerOutput{}, errors.New("invalid checker response")
	}
	return result, nil
}
