package main

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"academy/internal/projectreview"
)

func TestRunWritesPrivateKeyAndUsableStrictRegistry(t *testing.T) {
	dir := t.TempDir()
	privatePath := filepath.Join(dir, "reviewer.key")
	registryPath := filepath.Join(dir, "reviewers.json")
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--reviewer-id", "operator:r1", "--private-out", privatePath, "--registry-out", registryPath}, &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("keygen must be silent: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	private, err := os.ReadFile(privatePath)
	if err != nil {
		t.Fatal(err)
	}
	defer clear(private)
	if len(private) != ed25519.PrivateKeySize {
		t.Fatalf("private key length=%d, want %d raw bytes", len(private), ed25519.PrivateKeySize)
	}
	privateInfo, err := os.Stat(privatePath)
	if err != nil {
		t.Fatal(err)
	}
	if privateInfo.Mode().Perm() != 0600 {
		t.Fatalf("private mode=%#o, want 0600", privateInfo.Mode().Perm())
	}
	registryBytes, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	registryInfo, err := os.Stat(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	if registryInfo.Mode().Perm() != 0600 {
		t.Fatalf("registry mode=%#o, want 0600", registryInfo.Mode().Perm())
	}
	if bytes.Contains(registryBytes, private) || bytes.Contains(stdout.Bytes(), private) || bytes.Contains(stderr.Bytes(), private) {
		t.Fatal("private key bytes leaked outside private key file")
	}
	var shape struct {
		Reviewers []struct {
			ID  string `json:"reviewer_id"`
			Key string `json:"public_key"`
		} `json:"reviewers"`
	}
	if err := json.Unmarshal(registryBytes, &shape); err != nil {
		t.Fatal(err)
	}
	if len(shape.Reviewers) != 1 || shape.Reviewers[0].ID != "operator:r1" {
		t.Fatalf("unexpected registry shape: %+v", shape)
	}
	public, err := base64.StdEncoding.DecodeString(shape.Reviewers[0].Key)
	if err != nil || len(public) != ed25519.PublicKeySize {
		t.Fatalf("invalid public key encoding/size: len=%d err=%v", len(public), err)
	}
	loaded, err := projectreview.LoadRegistry(bytes.NewReader(registryBytes))
	if err != nil {
		t.Fatalf("generated registry is not accepted by the strict loader: %v", err)
	}
	attestation := projectreview.Attestation{
		Version: projectreview.Version, ReviewerID: "operator:r1", SubmissionID: "submission1", EvaluationID: "evaluation1",
		ProjectID: "p5", ProjectVersion: "v1", ArtifactHash: projectreview.Hash([]byte("artifact")),
		RubricVersion: "v1", RubricHash: projectreview.Hash([]byte("rubric")), Decision: "approve", IssuedAt: "2026-10-16T12:00:00Z",
	}
	signed, err := projectreview.Sign(attestation, ed25519.PrivateKey(private))
	if err != nil {
		t.Fatal(err)
	}
	if err := projectreview.Verify(loaded, signed); err != nil {
		t.Fatalf("generated key and registry do not round-trip: %v", err)
	}
}

func TestRunRejectsInvalidReviewerIDWithoutCreatingFiles(t *testing.T) {
	for _, id := range []string{"", "has space", "../reviewer", "-starts-with-symbol", "r\n1", string(bytes.Repeat([]byte{'a'}, 129))} {
		t.Run(id, func(t *testing.T) {
			dir := t.TempDir()
			privatePath, registryPath := filepath.Join(dir, "key"), filepath.Join(dir, "registry.json")
			var stdout, stderr bytes.Buffer
			err := run([]string{"--reviewer-id", id, "--private-out", privatePath, "--registry-out", registryPath}, &stdout, &stderr)
			if err == nil {
				t.Fatal("invalid reviewer ID accepted")
			}
			if stdout.Len() != 0 || fileExists(privatePath) || fileExists(registryPath) {
				t.Fatalf("invalid ID produced output: stdout=%q private=%t registry=%t", stdout.String(), fileExists(privatePath), fileExists(registryPath))
			}
		})
	}
}

func TestRunDoesNotOverwriteExistingFilesOrSymlinks(t *testing.T) {
	t.Run("existing private output", func(t *testing.T) {
		dir := t.TempDir()
		privatePath, registryPath := filepath.Join(dir, "key"), filepath.Join(dir, "registry.json")
		if err := os.WriteFile(privatePath, []byte("preserve-private"), 0600); err != nil {
			t.Fatal(err)
		}
		mustFail(t, privatePath, registryPath)
		assertFile(t, privatePath, "preserve-private")
		if fileExists(registryPath) {
			t.Fatal("registry was created after private O_EXCL failure")
		}
	})
	t.Run("existing registry cleans partial private output", func(t *testing.T) {
		dir := t.TempDir()
		privatePath, registryPath := filepath.Join(dir, "key"), filepath.Join(dir, "registry.json")
		if err := os.WriteFile(registryPath, []byte("preserve-registry"), 0600); err != nil {
			t.Fatal(err)
		}
		mustFail(t, privatePath, registryPath)
		assertFile(t, registryPath, "preserve-registry")
		if fileExists(privatePath) {
			t.Fatal("partially created private key was not removed")
		}
	})
	t.Run("private symlink is preserved", func(t *testing.T) {
		dir := t.TempDir()
		privatePath, registryPath, target := filepath.Join(dir, "key-link"), filepath.Join(dir, "registry.json"), filepath.Join(dir, "target")
		if err := os.WriteFile(target, []byte("do-not-replace"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, privatePath); err != nil {
			t.Fatal(err)
		}
		mustFail(t, privatePath, registryPath)
		assertFile(t, target, "do-not-replace")
		if fileExists(registryPath) {
			t.Fatal("registry was created after symlink rejection")
		}
	})
	t.Run("registry symlink cleans private key and preserves target", func(t *testing.T) {
		dir := t.TempDir()
		privatePath, registryPath, target := filepath.Join(dir, "key"), filepath.Join(dir, "registry-link"), filepath.Join(dir, "target")
		if err := os.WriteFile(target, []byte("do-not-replace"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, registryPath); err != nil {
			t.Fatal(err)
		}
		mustFail(t, privatePath, registryPath)
		assertFile(t, target, "do-not-replace")
		if fileExists(privatePath) {
			t.Fatal("partially created private key was not removed")
		}
	})
}

func TestRunRejectsMissingOrSameOutputPaths(t *testing.T) {
	dir := t.TempDir()
	var stdout, stderr bytes.Buffer
	if err := run([]string{"--reviewer-id", "r1", "--private-out", filepath.Join(dir, "same"), "--registry-out", filepath.Join(dir, ".", "same")}, &stdout, &stderr); err == nil {
		t.Fatal("same private and registry path accepted")
	}
	if fileExists(filepath.Join(dir, "same")) || stdout.Len() != 0 {
		t.Fatal("same output path created a file or wrote stdout")
	}
}

func mustFail(t *testing.T, privatePath, registryPath string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	err := run([]string{"--reviewer-id", "r1", "--private-out", privatePath, "--registry-out", registryPath}, &stdout, &stderr)
	if err == nil {
		t.Fatal("existing output path/symlink was overwritten")
	}
	if stdout.Len() != 0 {
		t.Fatalf("unsafe output on error: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("%s content=%q, want %q", path, got, want)
	}
}

func fileExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
