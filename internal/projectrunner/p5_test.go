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

	"academy/internal/projects"
)

func TestP5ProfileUsesExactCatalogAndOracleIdentity(t *testing.T) {
	e := &engine{cfg: Config{GoImage: "sha256:" + strings.Repeat("a", 64), CheckerImage: "sha256:" + strings.Repeat("b", 64)}}
	if e.SupportsProject("p5") {
		t.Fatal("P5 advertised without verified image label")
	}
	e.p5Enabled.Store(true)
	profile, ok := e.ProjectProfile("p5")
	project, exists := projects.Find("p5")
	criteria, criteriaOK := projects.CriteriaHash("p5")
	oracle := sha256.Sum256([]byte(p5AcceptanceTests + "\x00" + p5ArtifactPolicy))
	if !ok || !exists || !criteriaOK || profile.ProjectVersion != project.Version || profile.CriteriaHash != criteria || profile.CheckerVersion != p5CheckerVersion || profile.RubricVersion != "capstone-deterministic-v1" || profile.CheckerDigest != "sha256:"+hex.EncodeToString(oracle[:]) || !strings.HasPrefix(profile.RuntimePolicyHash, "sha256:") || !strings.HasPrefix(profile.RubricHash, "sha256:") {
		t.Fatalf("unexpected P5 evaluation profile: %+v", profile)
	}
	if e.SupportsProject("p5") != e.p5Enabled.Load() {
		t.Fatal("P5 capability does not follow trusted image label")
	}
}

func TestP5StarterValidationAndUserTestRequirement(t *testing.T) {
	project, ok := projects.Find("p5")
	if !ok {
		t.Fatal("P5 catalog entry missing")
	}
	files := make(map[string]string, len(project.StarterFiles))
	for name, source := range project.StarterFiles {
		files[name] = source
	}
	q := Request{ProjectID: "p5", Action: "run", Files: files}
	if err := validate(q, testConfig()); err != nil {
		t.Fatalf("bundled P5 starter rejected: %v", err)
	}
	for _, missing := range []string{"docs/project.md", "docs/threat-model.md", "design/requirements.md", "design/adr-template.md", "queue.go", "main.go"} {
		bad := make(map[string]string, len(files))
		for name, source := range files {
			bad[name] = source
		}
		delete(bad, missing)
		if err := validate(Request{ProjectID: "p5", Action: "run", Files: bad}, testConfig()); err == nil {
			t.Errorf("worker accepted Capstone without %s", missing)
		}
	}
	q.Action = "test"
	if err := validate(q, testConfig()); err == nil {
		t.Fatal("test action passed without genuine test")
	}
	files["queue_test.go"] = "package main\nimport \"testing\"\nfunc TestPrototype(t *testing.T) {}\n"
	if err := validate(q, testConfig()); err != nil {
		t.Fatalf("genuine test rejected: %v", err)
	}
}

func TestP5SubmitUsesNoDatabaseSandboxAndTrustedPrototypeOracle(t *testing.T) {
	var argsSeen []string
	var payload []byte
	e := &engine{install: "p5-test", cfg: Config{GoImage: "sha256:" + strings.Repeat("a", 64)}}
	e.run = func(_ context.Context, input []byte, args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "run" {
			argsSeen = append([]string(nil), args...)
			payload = append([]byte(nil), input...)
			return []byte("ok\n"), nil
		}
		if len(args) > 0 && args[0] == "ps" {
			return nil, nil
		}
		t.Fatalf("unexpected Podman command %v", args)
		return nil, nil
	}
	files := map[string]string{"go.mod": "module p5-capstone-prototype\n\ngo 1.27.2\n", "queue.go": "package main\n"}
	for _, name := range []string{"docs/project.md", "docs/rubric.md", "docs/threat-model.md", "design/requirements.md", "design/architecture-template.md", "design/operations-template.md", "design/postmortem-template.md", "design/threat-template.md"} {
		files[name] = "submitted artifact"
	}
	for _, name := range []string{"design/adr/ADR-001-delivery.md", "design/adr/ADR-002-storage.md", "design/adr/ADR-003-tenancy.md"} {
		files[name] = "decision"
	}
	result, err := e.runP5(context.Background(), Request{ProjectID: "p5", Action: "submit", Files: files})
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != "ok" || !result.Passed || len(result.Checks) != 6 {
		t.Fatalf("machine pass or review scope misreported: %+v", result)
	}
	if result.Stdout != "" {
		t.Fatalf("Submit returned compiler/test stdout: %q", result.Stdout)
	}
	if result.Checks[5].Passed != true || !strings.Contains(result.Checks[5].Name, "review") {
		t.Fatalf("human review disclaimer missing: %+v", result.Checks[5])
	}
	joined := strings.Join(argsSeen, " ")
	for _, want := range []string{"--network none", "--read-only", "--user 65532:65532", "--cap-drop=all", "--security-opt=no-new-privileges", "--cpus=1", "--memory=1536m", "--pids-limit=128"} {
		if !strings.Contains(joined, want) {
			t.Errorf("sandbox missing %q", want)
		}
	}
	if strings.Contains(joined, "postgres") || strings.Contains(joined, "--network container:") {
		t.Fatalf("P5 unexpectedly depends on PostgreSQL: %s", joined)
	}
	tr := tar.NewReader(bytes.NewReader(payload))
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
		if h.Name == "academy_capstone_acceptance_test.go" {
			found = strings.Contains(string(body), "FenceToken") && strings.Contains(string(body), "TenantID")
		}
	}
	if !found {
		t.Fatal("trusted prototype tests missing from Submit payload")
	}
}

func TestP5SubmitFailureNeverReturnsLearnerOrTrustedTestOutput(t *testing.T) {
	const secret = "capstone-output-secret-marker"
	e := &engine{install: "p5-test"}
	e.run = func(_ context.Context, _ []byte, args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "run" {
			return []byte(secret), errors.New("go test failed")
		}
		if len(args) > 0 && args[0] == "ps" {
			return nil, nil
		}
		t.Fatalf("unexpected Podman command %v", args)
		return nil, nil
	}
	result, err := e.runP5(context.Background(), Request{Action: "submit", Files: map[string]string{"go.mod": "module p5-capstone-prototype\n\ngo 1.27.2\n"}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Stdout+result.Stderr, secret) || !strings.Contains(result.Stderr, "Capstone") {
		t.Fatalf("Submit leaked output or omitted generic diagnostic: %+v", result)
	}
}

func TestP5ArtifactChecksRequireThreeAdrRecords(t *testing.T) {
	files := map[string]string{}
	for _, name := range []string{"docs/project.md", "docs/rubric.md", "docs/threat-model.md", "design/requirements.md", "design/architecture-template.md", "design/operations-template.md", "design/postmortem-template.md", "design/threat-template.md"} {
		files[name] = "content"
	}
	if checks := p5ArtifactChecks(map[string]string{}); checks[0].Passed || checks[1].Passed {
		t.Fatalf("missing artifacts passed: %+v", checks)
	}
	for _, name := range []string{"docs/project.md", "docs/rubric.md", "docs/threat-model.md", "design/requirements.md", "design/architecture-template.md", "design/operations-template.md", "design/postmortem-template.md", "design/threat-template.md"} {
		files[name] = "content"
	}
	files["design/adr/ADR-001-delivery.md"] = "delivery"
	files["design/adr/ADR-002-storage.md"] = "storage"
	if checks := p5ArtifactChecks(files); !checks[0].Passed || checks[1].Passed {
		t.Fatalf("two ADR files should fail the ADR count: %+v", checks)
	}
	files["design/adr/ADR-003-tenancy.md"] = "tenancy"
	if checks := p5ArtifactChecks(files); !checks[0].Passed || !checks[1].Passed {
		t.Fatalf("complete scaffold failed: %+v", checks)
	}
}

func TestP5CleanupFailurePoisonsWorker(t *testing.T) {
	e := &engine{install: "p5-test"}
	e.run = func(_ context.Context, _ []byte, args ...string) ([]byte, error) {
		if len(args) > 0 && args[0] == "run" {
			return nil, nil
		}
		if len(args) > 0 && args[0] == "ps" {
			return []byte("orphan\n"), nil
		}
		return nil, context.DeadlineExceeded
	}
	_, err := e.runP5(context.Background(), Request{Action: "run", Files: map[string]string{"go.mod": "module p5-capstone-prototype\n\ngo 1.27.2\n"}})
	if err == nil || !e.poisoned.Load() {
		t.Fatalf("cleanup failure did not poison P5 worker: %v", err)
	}
}
