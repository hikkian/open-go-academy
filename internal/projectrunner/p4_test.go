package projectrunner

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"academy/internal/projects"
)

func TestP4SubmitUsesIsolatedNoDatabaseJobAndTrustedOverlay(t *testing.T) {
	var runArgs []string
	var tarBody []byte
	e := &engine{install: "test", cfg: Config{GoImage: "sha256:" + strings.Repeat("a", 64)}}
	e.run = func(_ context.Context, input []byte, args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "run" {
			runArgs = append([]string(nil), args...)
			tarBody = append([]byte(nil), input...)
			return []byte("ok\n"), nil
		}
		if len(args) > 0 && args[0] == "ps" {
			return nil, nil
		}
		t.Fatalf("unexpected Podman command: %v", args)
		return nil, nil
	}
	module, _, ok := projects.ModulePolicy("p4")
	if !ok {
		t.Fatal("P4 module policy missing")
	}
	result, err := e.runP4(context.Background(), Request{ProjectID: "p4", Action: "submit", Files: map[string]string{"go.mod": module, "main.go": "package main\n"}})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" || !result.Passed || len(result.Checks) != 4 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if strings.Contains(result.Stdout+result.Stderr, "ok") {
		t.Fatalf("Submit returned learner/test output: %+v", result)
	}
	joined := strings.Join(runArgs, " ")
	for _, required := range []string{"--network none", "--read-only", "--user 65532:65532", "--cap-drop=all", "--security-opt=no-new-privileges", "--cpus=1", "--memory=1536m", "--pids-limit=128", "--tmpfs /tmp:rw,noexec,nosuid,nodev,size=256m,mode=1777", "--tmpfs /work:rw,nosuid,nodev,size=512m,mode=1777"} {
		if !strings.Contains(joined, required) {
			t.Errorf("sandbox missing %q in %s", required, joined)
		}
	}
	if strings.Contains(joined, "postgres") || strings.Contains(joined, "--network container:") {
		t.Fatalf("P4 unexpectedly starts or depends on PostgreSQL: %s", joined)
	}
	tr := tar.NewReader(bytes.NewReader(tarBody))
	found := false
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(tr)
		if h.Name == "academy_scalelab_acceptance_test.go" {
			found = strings.Contains(string(body), "AggregateBaseline") && strings.Contains(string(body), "math.MaxUint64")
		}
	}
	if !found {
		t.Fatal("submit did not add the trusted deterministic acceptance tests")
	}
}

func TestP4SubmitFailureNeverReturnsLearnerOrTrustedTestOutput(t *testing.T) {
	const secret = "student-output-secret-marker"
	e := &engine{install: "test"}
	e.run = func(_ context.Context, _ []byte, args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "run" {
			return []byte(secret), errors.New("go test failed")
		}
		if len(args) > 0 && args[0] == "ps" {
			return nil, nil
		}
		t.Fatalf("unexpected Podman command: %v", args)
		return nil, nil
	}
	result, err := e.runP4(context.Background(), Request{ProjectID: "p4", Action: "submit", Files: map[string]string{"go.mod": "module scalelab\n\ngo 1.27.2\n"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Stdout+result.Stderr, secret) || !strings.Contains(result.Stderr, "ScaleLab") {
		t.Fatalf("Submit leaked output or omitted generic diagnostic: %+v", result)
	}
}

func TestP4WorkerRequiresServerPolicyFiles(t *testing.T) {
	project, ok := projects.Find("p4")
	if !ok {
		t.Fatal("P4 catalog entry missing")
	}
	files := make(map[string]string, len(project.StarterFiles))
	for name, source := range project.StarterFiles {
		files[name] = source
	}
	if err := validate(Request{ProjectID: "p4", Action: "run", Files: files}, testConfig()); err != nil {
		t.Fatalf("bundled P4 starter rejected: %v", err)
	}
	for _, missing := range []string{"README.md", "docs/project.md", "docs/rubric.md", "go.mod", "main.go", "aggregate.go"} {
		bad := make(map[string]string, len(files))
		for name, source := range files {
			bad[name] = source
		}
		delete(bad, missing)
		if err := validate(Request{ProjectID: "p4", Action: "run", Files: bad}, testConfig()); err == nil {
			t.Errorf("worker accepted P4 without %s", missing)
		}
	}
}

func TestP4ProfileRequiresTheLabeledTrustedImage(t *testing.T) {
	digest := "sha256:" + strings.Repeat("c", 64)
	e := &engine{cfg: Config{GoImage: "sha256:" + strings.Repeat("a", 64), CheckerImage: digest}}
	if e.SupportsProject("p4") {
		t.Fatal("P4 advertised before image label verification")
	}
	e.p4Enabled.Store(true)
	if !e.SupportsProject("p4") {
		t.Fatal("verified P4 profile not advertised")
	}
	profile, ok := e.ProjectProfile("p4")
	project, exists := projects.Find("p4")
	criteria, criteriaOK := projects.CriteriaHash("p4")
	oracleHash := sha256.Sum256([]byte(p4AcceptanceTests))
	oracleDigest := "sha256:" + hex.EncodeToString(oracleHash[:])
	if !ok || !exists || !criteriaOK || profile.ProjectVersion != project.Version || profile.CriteriaHash != criteria || profile.CheckerVersion != p4CheckerVersion || profile.CheckerDigest != oracleDigest || profile.CheckerDigest == digest || !strings.HasPrefix(profile.RuntimePolicyHash, "sha256:") || !strings.HasPrefix(profile.RubricHash, "sha256:") {
		t.Fatalf("unexpected immutable P4 profile: %+v", profile)
	}
}

func TestP4StarterValidatesWithoutGoSumAndRequiresRealTestsForTestAction(t *testing.T) {
	project, ok := projects.Find("p4")
	if !ok {
		t.Fatal("P4 project missing from catalog")
	}
	files := make(map[string]string, len(project.StarterFiles))
	for name, source := range project.StarterFiles {
		files[name] = source
	}
	request := Request{ProjectID: "p4", Action: "run", Files: files}
	if err := validate(request, testConfig()); err != nil {
		t.Fatalf("bundled P4 starter rejected: %v", err)
	}
	if _, ok := files["go.sum"]; ok {
		t.Fatal("P4 standard-library-only starter unexpectedly requires go.sum")
	}
	request.Action = "test"
	if err := validate(request, testConfig()); err != nil {
		t.Fatalf("starter has genuine Go test files: %v", err)
	}
	delete(files, "aggregate_test.go")
	delete(files, "academy_test.go")
	for name := range files {
		if strings.HasSuffix(name, "_test.go") {
			delete(files, name)
		}
	}
	if err := validate(request, testConfig()); err == nil {
		t.Fatal("test action accepted without a genuine TestX function")
	}
}

func TestP4CleanupFailurePoisonsWorker(t *testing.T) {
	e := &engine{install: "test"}
	e.run = func(_ context.Context, _ []byte, args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "run" {
			return []byte("ok"), nil
		}
		if len(args) > 0 && args[0] == "ps" {
			return []byte("orphan\n"), nil
		}
		return nil, context.DeadlineExceeded
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	module, _, ok := projects.ModulePolicy("p4")
	if !ok {
		t.Fatal("P4 module policy missing")
	}
	_, err := e.runP4(ctx, Request{Action: "run", Files: map[string]string{"go.mod": module}})
	if err == nil || !e.poisoned.Load() {
		t.Fatalf("cleanup failure must poison worker, err=%v", err)
	}
}
