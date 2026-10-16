package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"academy/internal/projectreview"
)

func TestRunSignsWithoutPrintingPrivateKey(t *testing.T) {
	pub, priv, err := projectreview.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "reviewer.key")
	if err := os.WriteFile(keyPath, priv, 0600); err != nil {
		t.Fatal(err)
	}
	a := projectreview.Attestation{Version: 1, ReviewerID: "r1", SubmissionID: "s1", EvaluationID: "e1", ProjectID: "p5", ProjectVersion: "1", ArtifactHash: projectreview.Hash([]byte("artifact")), RubricVersion: "v1", RubricHash: projectreview.Hash([]byte("rubric")), Decision: "approve", IssuedAt: "2026-10-16T12:00:00Z"}
	input, _ := json.Marshal(a)
	var out, stderr bytes.Buffer
	if err := run([]string{"--key", keyPath}, bytes.NewReader(input), &out, &stderr); err != nil {
		t.Fatal(err)
	}
	var signed projectreview.Attestation
	if err := json.Unmarshal(out.Bytes(), &signed); err != nil {
		t.Fatal(err)
	}
	reg, err := projectreview.LoadRegistry(bytes.NewReader([]byte(`{"reviewers":[{"reviewer_id":"r1","public_key":"` + encode(pub) + `"}]}`)))
	if err != nil {
		t.Fatal(err)
	}
	if err := projectreview.Verify(reg, signed); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(out.Bytes(), priv) || bytes.Contains(stderr.Bytes(), priv) {
		t.Fatal("private key bytes leaked")
	}
}

func TestRunRequiresMode0600Key(t *testing.T) {
	_, priv, err := projectreview.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(path, priv, 0644); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if err := run([]string{"--key", path}, bytes.NewReader([]byte(`{}`)), &out, &stderr); err == nil {
		t.Fatal("permissive key mode accepted")
	}
	if out.Len() != 0 {
		t.Fatal("unexpected stdout")
	}
}

func encode(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
