// Package runner executes source only in an independently confined rootless OCI container.
package runner

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const Image = "docker.io/library/golang@sha256:3b15ba438c60ab8fabe5fee3b11025c0049e358b31feff384648f11fc8e335f8"
const MaxSource = 64 << 10
const MaxOutput = 32 << 10

var fileName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*\.go$`)

type Request struct {
	Files  map[string]string `json:"files"`
	Mode   string            `json:"mode"`
	Args   []string          `json:"args,omitempty"`
	Assets map[string]string `json:"assets,omitempty"`
}

// Check is one verdict line. Expected, Actual and Hint are optional (OGAUX2, docs/LESSON_UI_SPEC.md §2.1): the learner
// sees what was expected and what came out, never a hidden input. Criterion is the index in the exercise's criteria, or nil
// when the check maps to none; nil is the spec's -1 (a pointer keeps index 0 distinct from "no criterion"). Empty fields are
// left out of the JSON, so checks that set none (projects, exams) serialize exactly as before.
type Check struct {
	Name      string `json:"name"`
	Passed    bool   `json:"passed"`
	Expected  string `json:"expected,omitempty"`
	Actual    string `json:"actual,omitempty"`
	Hint      string `json:"hint,omitempty"`
	Criterion *int   `json:"criterion,omitempty"`
}
type Result struct {
	Stdout   string            `json:"stdout"`
	Stderr   string            `json:"stderr"`
	ExitCode int               `json:"exit_code"`
	Status   string            `json:"status"`
	Passed   bool              `json:"passed"`
	Checks   []Check           `json:"checks,omitempty"`
	Files    map[string]string `json:"files,omitempty"`
}

func ValidateFiles(files map[string]string) error { return validateFiles(files, false) }
func validateFiles(files map[string]string, tests bool) error {
	if len(files) == 0 || len(files) > 16 {
		return errors.New("требуется от 1 до 16 Go-файлов")
	}
	size := 0
	for name, source := range files {
		if !fileName.MatchString(name) || (!tests && strings.HasSuffix(name, "_test.go")) {
			return errors.New("разрешены только обычные Go-файлы без каталогов и тестов")
		}
		size += len(source)
	}
	if size > MaxSource {
		return errors.New("исходники превышают 64 КиБ")
	}
	return nil
}

type Engine struct {
	slots    chan struct{}
	Timeout  time.Duration
	disabled atomic.Bool
	Scope    string
}

func New() *Engine {
	return &Engine{slots: make(chan struct{}, 1), Timeout: 45 * time.Second, Scope: "default"}
}
func (e *Engine) Ready() bool { return !e.disabled.Load() }
func (e *Engine) policy(name string) []string {
	return append(policy(name), "--label=academy.install="+e.Scope)
}
func policy(name string) []string {
	return []string{"run", "--rm", "--pull=never", "--name", name, "--label", "academy.runner=true", "--network=none", "--http-proxy=false", "--stop-timeout=0", "--read-only", "--cap-drop=ALL", "--security-opt=no-new-privileges", "--user=65532:65532", "--memory=768m", "--memory-swap=768m", "--pids-limit=128", "--cpus=1", "--tmpfs=/tmp:rw,noexec,nosuid,nodev,size=128m", "--tmpfs=/work:rw,nosuid,nodev,size=192m", "--workdir=/work", "--env=CGO_ENABLED=0", "--env=GOTOOLCHAIN=local", "--env=GOPROXY=off", "--env=GOSUMDB=off", "--tmpfs=/cache:rw,nosuid,nodev,size=192m", "--shm-size=16m", "--env=GOCACHE=/cache", "--env=HOME=/tmp", "--env=GOMAXPROCS=2"}
}
func (e *Engine) Preflight(ctx context.Context) (preflightErr error) {
	out, err := exec.CommandContext(ctx, "podman", "info", "--format", "{{.Host.Security.Rootless}} {{.Host.CgroupsVersion}}").Output()
	if err != nil || strings.TrimSpace(string(out)) != "true v2" {
		return fmt.Errorf("требуется rootless Podman с cgroup v2 и seccomp: %s (%v)", out, err)
	}
	// Reconcile only orphaned jobs from this installation, never other Podman containers.
	ids, err := exec.CommandContext(ctx, "podman", "ps", "--all", "--filter", "label=academy.install="+e.Scope, "--format", "{{.ID}}").Output()
	if err != nil {
		return fmt.Errorf("sandbox reconciliation: %w", err)
	}
	for _, id := range strings.Fields(string(ids)) {
		if !regexp.MustCompile(`^[0-9a-f]{12,64}$`).MatchString(id) {
			return errors.New("invalid container ID")
		}
		if err := exec.CommandContext(ctx, "podman", "rm", "--force", "--time=0", id).Run(); err != nil {
			return fmt.Errorf("orphan cleanup: %w", err)
		}
	}

	preflightName := "oga-preflight-" + randomID()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := exec.CommandContext(ctx, "podman", "rm", "--force", "--time=0", "--ignore", preflightName).Run(); err != nil {
			e.disabled.Store(true)
			preflightErr = fmt.Errorf("preflight cleanup: %w", err)
		}
	}()
	args := append(e.policy(preflightName), Image, "sh", "-c", `test "$(cat /sys/fs/cgroup/memory.max)" = 805306368 && test "$(cat /sys/fs/cgroup/memory.swap.max)" = 0 && test "$(cat /sys/fs/cgroup/pids.max)" = 128 && test "$(cat /sys/fs/cgroup/cpu.max)" = '100000 100000' && test "$(id -u)" = 65532 && grep -q '^Seccomp:[[:space:]]*2' /proc/self/status && grep -q '^NoNewPrivs:[[:space:]]*1' /proc/self/status && test "$(go version)" = 'go version go1.27.2 linux/amd64'`)
	if out, err = exec.CommandContext(ctx, "podman", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("sandbox preflight: %s (%v)", out, err)
	}
	if !e.Ready() {
		return errors.New("sandbox cleanup failed")
	}
	return nil
}
func randomID() string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

type outputBudget struct {
	mu        sync.Mutex
	remaining int
	limited   bool
	cancel    context.CancelFunc
}
type boundedWriter struct {
	budget *outputBudget
	buf    bytes.Buffer
}

func (w *boundedWriter) Write(p []byte) (int, error) {
	w.budget.mu.Lock()
	defer w.budget.mu.Unlock()
	n := len(p)
	keep := min(n, w.budget.remaining)
	w.buf.Write(p[:keep])
	w.budget.remaining -= keep
	if keep < n {
		w.budget.limited = true
		w.budget.cancel()
	}
	return n, nil
}
func (e *Engine) Execute(ctx context.Context, req Request) (Result, error) {
	if !e.Ready() {
		return Result{}, errors.New("исполнитель заблокирован после ошибки очистки; перезапустите runner")
	}
	if err := validateFiles(req.Files, req.Mode == "test"); err != nil {
		return Result{}, err
	}
	if req.Mode != "run" && req.Mode != "format" && req.Mode != "test" {
		return Result{}, errors.New("неизвестная операция")
	}
	argSize := 0
	for _, arg := range req.Args {
		argSize += len(arg)
		if strings.IndexByte(arg, 0) >= 0 {
			return Result{}, errors.New("недопустимый аргумент")
		}
	}
	if len(req.Args) > 32 || argSize > 4096 {
		return Result{}, errors.New("слишком много аргументов")
	}
	assetSize := 0
	assetName := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]*\.json$`)
	if len(req.Assets) > 8 {
		return Result{}, errors.New("слишком много файлов данных")
	}
	for name, source := range req.Assets {
		assetSize += len(source)
		if !assetName.MatchString(name) {
			return Result{}, errors.New("разрешены только JSON-файлы данных без каталогов")
		}
	}
	if assetSize > 64<<10 {
		return Result{}, errors.New("данные превышают64КиБ")
	}

	select {
	case e.slots <- struct{}{}:
		defer func() { <-e.slots }()
		if !e.Ready() {
			return Result{}, errors.New("исполнитель заблокирован после ошибки очистки")
		}
	default:
		return Result{}, errors.New("исполнитель занят; повторите после завершения запуска")
	}
	ctx, cancel := context.WithTimeout(ctx, e.Timeout)
	defer cancel()
	var input bytes.Buffer
	tw := tar.NewWriter(&input)
	for name, source := range req.Files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(source))}); err != nil {
			return Result{}, err
		}
		if _, err := tw.Write([]byte(source)); err != nil {
			return Result{}, err
		}
	}
	for name, source := range req.Assets {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0600, Size: int64(len(source))}); err != nil {
			return Result{}, err
		}
		if _, err := tw.Write([]byte(source)); err != nil {
			return Result{}, err
		}
	}

	module := "module academy-workspace\n\ngo 1.27.2\n"
	if err := tw.WriteHeader(&tar.Header{Name: "go.mod", Mode: 0600, Size: int64(len(module))}); err != nil {
		return Result{}, err
	}
	if _, err := tw.Write([]byte(module)); err != nil {
		return Result{}, err
	}

	if err := tw.Close(); err != nil {
		return Result{}, err
	}
	name := "oga-job-" + randomID()
	script := `tar -xf - && go build -p=2 -o /work/program ./*.go && exec /work/program "$@"`
	if req.Mode == "test" {
		script = `tar -xf - && mkdir -p /work/build && export GOTMPDIR=/work/build && go test -p=2 -count=1 ./*.go`
	}
	if req.Mode == "format" {
		script = `tar -xf - && gofmt -w ./*.go && tar -cf - ./*.go`
	}
	args := append(e.policy(name), "-i", Image, "sh", "-c", script, "--")
	args = append(args, req.Args...)
	budget := &outputBudget{remaining: MaxOutput, cancel: cancel}
	if req.Mode == "format" {
		budget.remaining = MaxSource + 16*2048
	}
	stdout, stderr := &boundedWriter{budget: budget}, &boundedWriter{budget: budget}
	cmd := exec.CommandContext(ctx, "podman", args...)
	cmd.Stdin = &input
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.WaitDelay = 2 * time.Second
	err := cmd.Run()
	// Kill the entire container even when the client was cancelled or descendants hold pipes.
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cleanupCancel()
	cleanup := exec.CommandContext(cleanupCtx, "podman", "rm", "--force", "--time=0", "--ignore", name)
	cleanupErr := cleanup.Run()
	if cleanupErr != nil {
		e.disabled.Store(true)
		return Result{}, errors.New("не удалось подтвердить очистку sandbox; исполнитель должен быть перезапущен")
	}
	result := Result{Stdout: stdout.buf.String(), Stderr: stderr.buf.String(), ExitCode: 0, Status: "ok"}
	if err != nil {
		result.ExitCode = -1
		result.Status = "error"
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			result.ExitCode = exit.ExitCode()
		}
	}
	if budget.limited {
		result.Status = "output_limit"
	} else if ctx.Err() != nil {
		result.Status = "timeout"
	}
	if req.Mode == "format" && result.Status == "ok" {
		result.Files = map[string]string{}
		tr := tar.NewReader(bytes.NewReader(stdout.buf.Bytes()))
		for {
			h, err := tr.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return Result{}, errors.New("повреждённый ответ formatter")
			}
			f := strings.TrimPrefix(h.Name, "./")
			if _, ok := req.Files[f]; !ok || h.Typeflag != tar.TypeReg || h.Size > MaxSource {
				return Result{}, errors.New("недопустимый ответ formatter")
			}
			b, err := io.ReadAll(io.LimitReader(tr, MaxSource+1))
			if err != nil || len(b) > MaxSource {
				return Result{}, errors.New("слишком большой ответ formatter")
			}
			result.Files[f] = string(b)
		}
		if len(result.Files) != len(req.Files) {
			return Result{}, errors.New("formatter не вернул все файлы")
		}
		result.Stdout = ""
	}
	return result, nil
}

// Client sends validated requests over a permission-restricted Unix socket; never a network service.
type Client struct{ http *http.Client }

func NewClient(socket string) *Client {
	return &Client{http: &http.Client{Timeout: 60 * time.Second, Transport: &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, "unix", socket)
	}}}}
}
func (c *Client) Available(ctx context.Context) bool {
	req, _ := http.NewRequestWithContext(ctx, "GET", "http://runner/health", nil)
	r, err := c.http.Do(req)
	if err != nil {
		return false
	}
	defer r.Body.Close()
	return r.StatusCode == 200
}
func (c *Client) Execute(ctx context.Context, req Request) (Result, error) {
	b, _ := json.Marshal(req)
	r, err := http.NewRequestWithContext(ctx, "POST", "http://runner/execute", bytes.NewReader(b))
	if err != nil {
		return Result{}, err
	}
	resp, err := c.http.Do(r)
	if err != nil {
		return Result{}, errors.New("исполнитель недоступен; запустите make runner")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return Result{}, errors.New("исполнитель отказал в запуске; повторите позже")
	}
	var result Result
	err = json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result)
	return result, err
}
