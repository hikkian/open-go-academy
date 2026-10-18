package projectrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"academy/internal/projects"
	"academy/internal/runner"
)

func testRequest() Request {
	return Request{ProjectID: "p2", Action: "run", Files: map[string]string{"go.mod": "module test\n", "go.sum": "", "main.go": "package main\n"}}
}
func testConfig() Config { return Config{ExpectedGoMod: "module test\n", ExpectedGoSum: ""} }

func TestValidateProjectProtocolAndImmutableLocks(t *testing.T) {
	q := testRequest()
	if err := validate(q, testConfig()); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		change func(*Request)
	}{
		{"project", func(q *Request) { q.ProjectID = "other" }}, {"action", func(q *Request) { q.Action = "format" }},
		{"traversal", func(q *Request) { q.Files["../x.go"] = "x" }}, {"backslash", func(q *Request) { q.Files[`a\\b.go`] = "x" }},
		{"dockerfile", func(q *Request) { q.Files["Dockerfile"] = "FROM evil" }}, {"env", func(q *Request) { q.Files["app.env"] = "SECRET=x" }},
		{"module mutation", func(q *Request) { q.Files["go.mod"] = "module attacker\n" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q := testRequest()
			tc.change(&q)
			if validate(q, testConfig()) == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}

func TestValidateBoundariesAndGenuineTests(t *testing.T) {
	q := testRequest()
	q.Action = "test"
	q.Files["example_test.go"] = "package main\n// func TestFake(t *testing.T) {} is documentation"
	if validate(q, testConfig()) == nil {
		t.Fatal("comment must not satisfy genuine test requirement")
	}
	q.Files["example_test.go"] = "package main\nimport tst \"testing\"\nfunc TestReal(t *tst.T) {}"
	if err := validate(q, testConfig()); err != nil {
		t.Fatal(err)
	}
	q.Files["example_test.go"] += "\nfunc TestMain /* comment between name and paren */ (m *testing.M) {}"
	if validate(q, testConfig()) == nil {
		t.Fatal("TestMain must be rejected")
	}
	q = testRequest()
	q.Files["huge.go"] = strings.Repeat("x", MaxGo+1)
	if validate(q, testConfig()) == nil {
		t.Fatal("oversized Go source must be rejected")
	}
}

func TestP4P5SubmitRejectsGoTestProcessHijacks(t *testing.T) {
	for _, projectID := range []string{"p4", "p5"} {
		project, ok := projects.Find(projectID)
		if !ok {
			t.Fatalf("%s missing", projectID)
		}
		for _, tc := range []struct{ name, file, source string }{
			{"init early exit", "main.go", "package main\nimport \"os\"\nfunc init(){ os.Exit(0) }\n"},
			{"dot-import early exit", "main.go", "package main\nimport . \"os\"\nvar _ = func() int { Exit(0); return 0 }()\n"},
			{"test main", "early_test.go", "package main\nimport \"testing\"\nfunc TestMain(m *testing.M) {}\n"},
			{"test exit", "early_test.go", "package main\nimport (\"os\"; \"testing\")\nfunc TestEarly(t *testing.T){ os.Exit(0) }\n"},
			{"dot-import syscall exit", "early_test.go", "package main\nimport ( . \"syscall\"; \"testing\")\nfunc TestEarly(t *testing.T){ Exit(0) }\n"},
			{"test goexit", "early_test.go", "package main\nimport (\"runtime\"; \"testing\")\nfunc TestEarly(t *testing.T){ runtime.Goexit() }\n"},
		} {
			t.Run(projectID+"/"+tc.name, func(t *testing.T) {
				files := make(map[string]string, len(project.StarterFiles)+1)
				for name, source := range project.StarterFiles {
					files[name] = source
				}
				files[tc.file] = tc.source
				if err := validate(Request{ProjectID: projectID, Action: "submit", Files: files}, testConfig()); err == nil {
					t.Fatal("trusted assessment accepted a user-controlled Go test process exit")
				}
			})
		}
	}
}

func TestP3PolicyAllowsTrustedStarterAndRejectsRuntimeConfiguration(t *testing.T) {
	project, ok := projects.Find("p3")
	if !ok {
		t.Fatal("P3 project not present in catalog")
	}
	q := Request{ProjectID: "p3", Action: "run", Files: map[string]string{}}
	for name, contents := range project.StarterFiles {
		q.Files[name] = contents
	}
	if err := validate(q, testConfig()); err != nil {
		t.Fatalf("P3 starter rejected by worker policy: %v", err)
	}
	for _, name := range []string{"Dockerfile", "compose.yaml", "compose.yml", "docker-compose.yaml", "app.env", ".env"} {
		bad := Request{ProjectID: "p3", Action: "run", Files: map[string]string{}}
		for file, contents := range q.Files {
			bad.Files[file] = contents
		}
		bad.Files[name] = "student-controlled runtime config"
		if err := validate(bad, testConfig()); err == nil {
			t.Errorf("accepted user runtime configuration %q", name)
		}
	}
	for _, name := range []string{"go.mod", "go.sum"} {
		bad := Request{ProjectID: "p3", Action: "run", Files: map[string]string{}}
		for file, contents := range q.Files {
			bad.Files[file] = contents
		}
		bad.Files[name] += "\nreplace example.invalid/x => /host\n"
		if err := validate(bad, testConfig()); err == nil {
			t.Errorf("accepted substituted P3 lock %s", name)
		}
	}
	q.Action = "test"
	if validate(q, testConfig()) == nil {
		t.Fatal("P3 test action passed without a real TestX")
	}
	q.Files["main_test.go"] = "package main\nimport \"testing\"\nfunc TestTaskFlow(t *testing.T) {}\n"
	if err := validate(q, testConfig()); err != nil {
		t.Fatalf("genuine P3 test rejected: %v", err)
	}
}

func TestP3StudentProfileIsPrivateAndRedactsDatabaseSecrets(t *testing.T) {
	project, _ := projects.Find("p3")
	files := map[string]string{}
	for name, source := range project.StarterFiles {
		files[name] = source
	}
	files["main_test.go"] = "package main\nimport \"testing\"\nfunc TestTaskFlow(t *testing.T) {}\n"
	q := Request{ProjectID: "p3", Action: "test", Files: files}
	e := &engine{cfg: Config{ExpectedGoMod: "unused", ExpectedGoSum: "unused"}, slot: make(chan struct{}, 1), install: strings.Repeat("f", 64)}
	e.p3Enabled.Store(true) // fake command represents the immutable taskflow-v1 checker profile
	bootstrap, dsn := "", ""
	var dbArgs, appArgs []string
	psCalls := 0
	e.run = func(_ context.Context, _ []byte, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch args[0] {
		case "ps":
			psCalls++
			if psCalls == 1 {
				return []byte("db\napp\n"), nil
			}
			return nil, nil
		case "rm":
			return nil, nil
		case "run":
			if strings.Contains(joined, "--name academy-") && strings.Contains(joined, "-d") {
				if strings.Contains(joined, "POSTGRES_PASSWORD=") {
					dbArgs = append([]string{}, args...)
					for _, arg := range args {
						if strings.HasPrefix(arg, "POSTGRES_PASSWORD=") {
							bootstrap = strings.TrimPrefix(arg, "POSTGRES_PASSWORD=")
						}
					}
				} else {
					appArgs = append([]string{}, args...)
					for _, arg := range args {
						if strings.HasPrefix(arg, "TASKFLOW_DATABASE_URL=") {
							dsn = strings.TrimPrefix(arg, "TASKFLOW_DATABASE_URL=")
						}
					}
				}
				return []byte("container\n"), nil
			}
		case "exec":
			if strings.Contains(joined, "pg_isready") {
				return nil, nil
			}
			if strings.Contains(joined, "CREATE ROLE") || strings.Contains(joined, "createdb") || strings.Contains(joined, "mkdir") || strings.Contains(joined, "tar") {
				return nil, nil
			}
			if strings.Contains(joined, "go test -mod=readonly ./...") {
				return []byte("POSTGRES_PASSWORD=" + bootstrap + " TASKFLOW_DATABASE_URL=" + dsn), errors.New("student test failed")
			}
			return nil, nil
		}
		return nil, errors.New("unexpected fake command: " + joined)
	}
	out, err := e.Execute(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "error" || out.Passed {
		t.Fatalf("student test result: %+v", out)
	}
	for _, secret := range []string{bootstrap, dsn} {
		if secret == "" || strings.Contains(out.Stderr, secret) {
			t.Fatalf("TaskFlow secret escaped diagnostics: %q", secret)
		}
	}
	if !strings.Contains(dsn, "@127.0.0.1:5432/academy") || !strings.Contains(strings.Join(dbArgs, " "), "--network none") || !strings.Contains(strings.Join(dbArgs, " "), "--user 999:999") || !strings.Contains(strings.Join(appArgs, " "), "--network container:academy-") || !strings.Contains(strings.Join(appArgs, " "), "TASKFLOW_ADDR=0.0.0.0:8098") || !strings.Contains(strings.Join(appArgs, " "), "TASKFLOW_DATABASE_URL=") {
		t.Fatal("TaskFlow did not receive its server-owned private profile")
	}
	for _, args := range [][]string{dbArgs, appArgs} {
		joined := strings.Join(args, " ")
		if strings.Contains(joined, " --network host") || strings.Contains(joined, " -p ") || strings.Contains(joined, " --publish ") {
			t.Fatalf("student profile has host networking/published port: %s", joined)
		}
	}
}

func TestListenLocksAndPreservesNonSocketPath(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "worker.sock")
	first, err := Listen(sock)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Listen(sock); err == nil {
		t.Fatal("second listener acquired live lock")
	}
	if fi, err := os.Stat(sock); err != nil || fi.Mode().Perm() != 0600 {
		t.Fatalf("socket permissions: %v %v", fi, err)
	}
	if err = first.Close(); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(sock, []byte("sentinel"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = Listen(sock); err == nil {
		t.Fatal("regular file replaced")
	}
	b, err := os.ReadFile(sock)
	if err != nil || string(b) != "sentinel" {
		t.Fatal("foreign path changed")
	}
}

func TestClientUsesUnixProtocol(t *testing.T) {
	// The client transport must preserve cancellation and use the configured socket.
	dir := t.TempDir()
	sock := filepath.Join(dir, "x.sock")
	l, err := net.Listen("unix", sock)
	if err != nil {
		t.Skip(err)
	}
	defer l.Close()
	c := NewClient(sock)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if c.Available(ctx) {
		t.Fatal("cancelled client reported available")
	}
}

func TestClientRejectsTrailingAndOversizedResponse(t *testing.T) {
	for _, tc := range []struct{ name, body string }{{"trailing", `{"status":"ok"} {}`}, {"oversized", `{"status":"ok"}` + strings.Repeat(" ", 1<<20)}} {
		t.Run(tc.name, func(t *testing.T) {
			sock := filepath.Join(t.TempDir(), "worker.sock")
			listener, err := net.Listen("unix", sock)
			if err != nil {
				t.Fatal(err)
			}
			server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, tc.body)
			})}
			go func() { _ = server.Serve(listener) }()
			defer server.Close()
			if _, err = NewClient(sock).Execute(context.Background(), Request{}); err == nil {
				t.Fatal("malformed response accepted")
			}
		})
	}
}

func TestSubmitLifecycleScopesCleanupAndMasksPrivateState(t *testing.T) {
	q := testRequest()
	q.Action = "submit"
	var calls [][]string
	phases := 0
	cleanup := false
	psCalls := 0
	bootstrapSecret, studentDSN := "", ""
	fixtureState := `{"email":"learner-fixture@example.test","password":"checker-private-password","slug":"link-fixture","link_id":"123e4567-e89b-12d3-a456-426614174000","target":"https://example.test/academy-fixture","tag":"academy-fixture"}`
	e := &engine{cfg: Config{ExpectedGoMod: "module test\n", ExpectedGoSum: ""}, slot: make(chan struct{}, 1), install: strings.Repeat("a", 64)}
	e.run = func(ctx context.Context, input []byte, args ...string) ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		calls = append(calls, append([]string(nil), args...))
		joined := strings.Join(args, " ")
		if strings.HasPrefix(joined, "ps -aq") {
			cleanup = true
			psCalls++
			if psCalls == 1 {
				return []byte("container-owned\n"), nil
			}
			return nil, nil
		}
		if strings.Contains(joined, "rm -f") {
			return nil, nil
		}
		if strings.Contains(joined, "pg_isready") || strings.Contains(joined, "createdb") || strings.Contains(joined, "CREATE ROLE") || strings.Contains(joined, "SELECT pg_catalog.to_regclass") {
			if strings.Contains(joined, "SELECT pg_catalog.to_regclass") {
				return []byte("t\n"), nil
			}
			return nil, nil
		}
		if strings.Contains(joined, "-f -") {
			return []byte("t\n"), nil
		}
		if strings.Contains(joined, "inspect --format") {
			return []byte("true\n"), nil
		}
		if strings.HasPrefix(joined, "exec ") {
			return nil, nil
		}
		if strings.HasPrefix(joined, "run") && bytes.Contains(input, []byte(`"phase"`)) {
			var in checkerInput
			if json.Unmarshal(input, &in) != nil {
				t.Fatal("invalid checker request")
			}
			if in.Phase != "health" {
				phases++
			}
			if in.Phase == "persistence" && string(in.State) != fixtureState {
				t.Fatal("phase two state not preserved")
			}
			if in.Phase == "health" {
				return []byte(`{"checks":[{"name":"healthz","passed":true}]}`), nil
			}
			return []byte(`{"checks":[{"name":"health","passed":true}],"state":` + fixtureState + `}`), nil
		}
		if strings.HasPrefix(joined, "run") {
			for _, arg := range args {
				if strings.HasPrefix(arg, "POSTGRES_PASSWORD=") {
					bootstrapSecret = strings.TrimPrefix(arg, "POSTGRES_PASSWORD=")
				}
				if strings.HasPrefix(arg, "LINKHUB_DATABASE_URL=") {
					studentDSN = strings.TrimPrefix(arg, "LINKHUB_DATABASE_URL=")
				}
			}
			return []byte("container-id\n"), nil
		}
		return nil, errors.New("unexpected fake CLI invocation: " + joined)
	}
	out, err := e.Execute(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Passed || phases != 2 || !cleanup {
		t.Fatalf("unexpected result/lifecycle: passed=%v phases=%d cleanup=%v", out.Passed, phases, cleanup)
	}
	if bootstrapSecret == "" || studentDSN == "" || strings.Contains(studentDSN, bootstrapSecret) || !strings.Contains(studentDSN, "@127.0.0.1") {
		t.Fatal("bootstrap credentials were exposed or app DSN missing")
	}
	serialized, _ := json.Marshal(out)
	if bytes.Contains(serialized, []byte("checker-private-password")) || bytes.Contains(serialized, []byte("postgres://")) {
		t.Fatal("private state or DSN escaped result")
	}
	for _, call := range calls {
		joined := strings.Join(call, " ")
		if strings.Contains(joined, "--label academy.install=") && !strings.Contains(joined, "--label academy.project-runner=true") {
			t.Fatalf("missing ownership label: %s", joined)
		}
	}
}

func TestCancelledJobStillRunsIndependentCleanupAndPoisonsOnFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cleanup := false
	e := &engine{cfg: testConfig(), slot: make(chan struct{}, 1), install: strings.Repeat("b", 64)}
	e.run = func(ctx context.Context, _ []byte, args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "ps" {
			cleanup = true
			return []byte("owned\n"), nil
		}
		if len(args) > 0 && args[0] == "rm" {
			return nil, nil
		}
		return nil, ctx.Err()
	}
	_, _ = e.Execute(ctx, testRequest())
	if !cleanup {
		t.Fatal("cancelled job skipped cleanup")
	}
	// Cleanup failure must latch availability off even after the caller is gone.
	e2 := &engine{cfg: testConfig(), slot: make(chan struct{}, 1), install: strings.Repeat("c", 64)}
	calls := 0
	e2.run = func(ctx context.Context, _ []byte, args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "ps" {
			calls++
			if calls > 0 {
				return nil, errors.New("podman unavailable")
			}
		}
		return nil, ctx.Err()
	}
	poisonedResult, poisonErr := e2.Execute(context.Background(), testRequest())
	if poisonErr == nil || poisonedResult.Passed {
		t.Fatal("cleanup failure must return an error and clear Passed")
	}
	if e2.Available(context.Background()) {
		t.Fatal("cleanup failure did not poison worker")
	}
}

func TestRemoveOwnedRetriesDBAfterNetworkNamespaceDependents(t *testing.T) {
	removed := map[string]bool{}
	dbAttempts := 0
	e := &engine{}
	e.run = func(_ context.Context, _ []byte, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		if strings.HasPrefix(joined, "ps -aq") {
			if removed["db"] && removed["app"] {
				return nil, nil
			}
			return []byte("db\napp\n"), nil
		}
		if len(args) > 0 && args[0] == "rm" {
			id := args[len(args)-1]
			if id == "db" {
				dbAttempts++
				if !removed["app"] {
					return nil, errors.New("network namespace in use")
				}
			}
			removed[id] = true
			return nil, nil
		}
		return nil, errors.New("unexpected fake CLI invocation: " + joined)
	}
	if err := e.removeOwned(context.Background(), "label=academy.project-runner=true"); err != nil {
		t.Fatal(err)
	}
	if !removed["db"] || !removed["app"] || dbAttempts < 2 {
		t.Fatalf("dependency retry failed: removed=%v db attempts=%d", removed, dbAttempts)
	}
}

func TestRemoveOwnedReturnsFinalRemovalFailure(t *testing.T) {
	e := &engine{}
	listed := 0
	attempted := map[string]int{}
	e.run = func(_ context.Context, _ []byte, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		if strings.HasPrefix(joined, "ps -aq") {
			listed++
			if listed == 1 {
				return []byte("db\napp\n"), nil
			}
			return []byte("db\n"), nil
		}
		if len(args) > 0 && args[0] == "rm" {
			id := args[len(args)-1]
			attempted[id]++
			if id == "app" {
				return nil, nil
			}
			return nil, errors.New("persistent removal failure")
		}
		return nil, errors.New("unexpected fake CLI invocation: " + joined)
	}
	if err := e.removeOwned(context.Background(), "label=academy.project-runner=true"); err == nil {
		t.Fatal("remaining DB container did not fail cleanup")
	}
	if attempted["app"] != 1 || attempted["db"] < 2 {
		t.Fatalf("cleanup did not process/retry all targets: %v", attempted)
	}
}

func TestFailedStudentTestsRedactBothDatabaseCredentials(t *testing.T) {
	q := testRequest()
	q.Action = "test"
	q.Files["main_test.go"] = "package main\nimport \"testing\"\nfunc TestReal(t *testing.T){}"
	e := &engine{cfg: testConfig(), slot: make(chan struct{}, 1), install: strings.Repeat("e", 64)}
	bootstrap, dsn := "", ""
	psCalls := 0
	e.run = func(_ context.Context, _ []byte, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		if strings.HasPrefix(joined, "ps -aq") {
			psCalls++
			if psCalls == 1 {
				return []byte("owned\n"), nil
			}
			return nil, nil
		}
		if strings.Contains(joined, "rm -fv") {
			return nil, nil
		}
		if strings.Contains(joined, "POSTGRES_PASSWORD=") {
			for _, a := range args {
				if strings.HasPrefix(a, "POSTGRES_PASSWORD=") {
					bootstrap = strings.TrimPrefix(a, "POSTGRES_PASSWORD=")
				}
			}
			return []byte("db\n"), nil
		}
		if strings.Contains(joined, "pg_isready") || strings.Contains(joined, "CREATE ROLE") || strings.Contains(joined, "createdb") {
			return nil, nil
		}
		if strings.HasPrefix(joined, "run ") {
			for _, a := range args {
				if strings.HasPrefix(a, "LINKHUB_DATABASE_URL=") {
					dsn = strings.TrimPrefix(a, "LINKHUB_DATABASE_URL=")
				}
			}
			return []byte("app\n"), nil
		}
		if strings.Contains(joined, "go test -mod=readonly ./...") {
			return []byte("bootstrap=" + bootstrap + " app=" + dsn + " dsn=" + dsn), errors.New("test failed")
		}
		if strings.HasPrefix(joined, "exec ") {
			return nil, nil
		}
		return nil, errors.New("unexpected fake command: " + joined)
	}
	out, err := e.Execute(context.Background(), q)
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "error" || out.Passed {
		t.Fatalf("test result: %+v", out)
	}
	for _, secret := range []string{bootstrap, dsn} {
		if secret == "" || strings.Contains(out.Stderr, secret) {
			t.Fatal("student diagnostics exposed ephemeral database credentials")
		}
	}
}

func TestBoundedCommandOutputCancelsAtLimit(t *testing.T) {
	cancelled := false
	w := &limitedOutput{cancel: func() { cancelled = true }}
	chunk := bytes.Repeat([]byte("x"), maxCommandOutput+4096)
	if n, err := w.Write(chunk); err != nil || n != len(chunk) {
		t.Fatalf("writer must drain caller output: %d %v", n, err)
	}
	if w.Len() != maxCommandOutput || !w.exceeded || !cancelled {
		t.Fatalf("output cap not enforced: size=%d exceeded=%v cancelled=%v", w.Len(), w.exceeded, cancelled)
	}
}

func TestPreflightRejectsNonV2Cgroups(t *testing.T) {
	e := &engine{run: func(context.Context, []byte, ...string) ([]byte, error) { return []byte("true v1"), nil }}
	if e.preflight(context.Background()) == nil {
		t.Fatal("cgroup v1 accepted")
	}
}

func TestTaskFlowWorkflowRequiresPinnedCheckerCapability(t *testing.T) {
	mod, sum, _ := projects.ModulePolicy("p3")
	checkerDigest := "sha256:" + strings.Repeat("b", 64)
	e := &engine{cfg: Config{GoImage: "sha256:" + strings.Repeat("a", 64), CheckerImage: "registry.example/academy/taskflow@" + checkerDigest, PostgresImage: "registry.example/postgres@sha256:" + strings.Repeat("c", 64), ExpectedGoMod: mod, ExpectedGoSum: sum}}
	if !e.SupportsProject("p2") {
		t.Fatal("P2 workflow unexpectedly disabled")
	}
	if e.SupportsProject("p3") {
		t.Fatal("P3 exposed before TaskFlow-capable checker profile is verified")
	}
	e.p3Enabled.Store(true)
	if !e.SupportsProject("p3") {
		t.Fatal("verified TaskFlow checker capability not exposed")
	}
	profile, ok := e.ProjectProfile("p3")
	if !ok || profile.ProjectVersion != "p3-taskflow-v1" || profile.CheckerVersion != p3CheckerVersion || profile.CheckerDigest != checkerDigest || !strings.HasPrefix(profile.RuntimePolicyHash, "sha256:") || !strings.HasPrefix(profile.RubricHash, "sha256:") {
		t.Fatalf("verified TaskFlow evidence profile is incomplete: %+v", profile)
	}
}

func TestTaskFlowExecutionFailsClosedWithoutPinnedChecker(t *testing.T) {
	project, _ := projects.Find("p3")
	files := map[string]string{}
	for name, source := range project.StarterFiles {
		files[name] = source
	}
	e := &engine{cfg: testConfig(), slot: make(chan struct{}, 1)}
	calls := 0
	e.run = func(context.Context, []byte, ...string) ([]byte, error) { calls++; return nil, nil }
	_, err := e.Execute(context.Background(), Request{ProjectID: "p3", Action: "run", Files: files})
	if err == nil || !strings.Contains(err.Error(), "checker profile") {
		t.Fatalf("unverified P3 checker was executed: %v", err)
	}
	if calls != 0 {
		t.Fatalf("disabled P3 profile invoked runner %d times", calls)
	}
}

func TestReadinessWaitsForHealthAndCheckerJSONMustEnd(t *testing.T) {
	e := &engine{cfg: Config{CheckerImage: "sha256:" + strings.Repeat("a", 64)}, install: strings.Repeat("d", 64)}
	checks := 0
	e.run = func(_ context.Context, input []byte, args ...string) ([]byte, error) {
		if strings.HasPrefix(strings.Join(args, " "), "inspect") {
			return []byte("true"), nil
		}
		var in checkerInput
		if json.Unmarshal(input, &in) != nil {
			t.Fatal("bad checker request")
		}
		checks++
		if in.Phase != "health" {
			t.Fatalf("unexpected phase %q", in.Phase)
		}
		passed := checks >= 3
		b, _ := json.Marshal(checkerOutput{Checks: []runner.Check{{Name: "healthz", Passed: passed}}})
		return b, nil
	}
	ready, _, err := e.waitReady(context.Background(), "job", "db", "app", "p2", nil)
	if err != nil || checks != 3 || len(ready.Checks) != 1 || !ready.Checks[0].Passed {
		t.Fatalf("readiness result: %+v calls=%d err=%v", ready, checks, err)
	}
	e.run = func(context.Context, []byte, ...string) ([]byte, error) {
		return []byte(`{"checks":[{"name":"ok","passed":true}]} {}`), nil
	}
	if _, err = e.runChecker(context.Background(), "job", "db", nil, checkerInput{Phase: "health", BaseURL: "http://127.0.0.1:8097"}); err == nil {
		t.Fatal("trailing checker JSON accepted")
	}
}

func TestTaskFlowRunProbeUsesLivenessWhileSubmitUsesReadiness(t *testing.T) {
	e := &engine{cfg: Config{CheckerImage: "sha256:" + strings.Repeat("a", 64)}, install: strings.Repeat("d", 64)}
	var phases []string
	e.run = func(_ context.Context, input []byte, args ...string) ([]byte, error) {
		if strings.HasPrefix(strings.Join(args, " "), "inspect") {
			return []byte("true"), nil
		}
		var request checkerInput
		if json.Unmarshal(input, &request) != nil {
			t.Fatal("invalid trusted checker request")
		}
		phases = append(phases, request.Phase)
		name := "readyz"
		if request.Phase == "liveness" {
			name = "livez"
		}
		return []byte(`{"checks":[{"name":"` + name + `","passed":true}]}`), nil
	}
	if _, _, err := e.waitLiveness(context.Background(), "job", "db", "app", "p3", nil); err != nil {
		t.Fatalf("liveness probe failed: %v", err)
	}
	if _, _, err := e.waitReady(context.Background(), "job", "db", "app", "p3", nil); err != nil {
		t.Fatalf("readiness probe failed: %v", err)
	}
	if len(phases) != 2 || phases[0] != "liveness" || phases[1] != "health" {
		t.Fatalf("unexpected P3 probes: %v", phases)
	}
}

func TestPersistedFixtureQueryRequiresStoredRowsAndDoesNotSendPassword(t *testing.T) {
	state := json.RawMessage(`{"email":"learner@example.test","password":"checker-password","slug":"link-one","link_id":"123e4567-e89b-12d3-a456-426614174000","target":"https://example.test/academy-abcdef123456","tag":"academy"}`)
	called := 0
	var captured []string
	var query []byte
	answer := []byte("f\n")
	e := &engine{run: func(_ context.Context, input []byte, args ...string) ([]byte, error) {
		called++
		captured = append([]string(nil), args...)
		query = append([]byte(nil), input...)
		return answer, nil
	}}
	check := e.persistedFixtureCheck(context.Background(), "db", "student_fixture", state)
	if check.Passed || called != 1 {
		t.Fatalf("empty fixture storage passed: %+v", check)
	}
	joined := strings.Join(captured, " ")
	if !strings.Contains(joined, "-U student_fixture") || strings.Contains(joined, "-U postgres") || !bytes.Contains(query, []byte("SET search_path = pg_catalog")) || !bytes.Contains(query, []byte("FROM public.users")) {
		t.Fatal("SQL verification acquired privileged role or untrusted search path")
	}
	if !bytes.Contains(query, []byte(":'email'")) || strings.Contains(string(query), "learner@example.test") || strings.Contains(joined, "checker-password") {
		t.Fatal("fixture values were interpolated or checker password was forwarded")
	}
	answer = []byte("t\n")
	check = e.persistedFixtureCheck(context.Background(), "db", "student_fixture", state)
	if !check.Passed {
		t.Fatalf("stored fixture rejected: %+v", check)
	}
	called = 0
	if e.persistedFixtureCheck(context.Background(), "db", "student_fixture", json.RawMessage(`{"email":"x","unknown":true}`)).Passed || called != 0 {
		t.Fatal("invalid checker state passed or reached psql")
	}
}

func TestSlotRejectsConcurrentJob(t *testing.T) {
	e := &engine{cfg: testConfig(), slot: make(chan struct{}, 1)}
	e.slot <- struct{}{}
	_, err := e.Execute(context.Background(), testRequest())
	if err == nil || err.Error() != "project worker busy" {
		t.Fatalf("busy result: %v", err)
	}
}
