package main

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"academy/internal/projectreview"
)

func TestLoadProjectReviewersRequiresPrivateRegularRegistry(t *testing.T) {
	if registry, err := loadProjectReviewers(""); err != nil || registry != nil {
		t.Fatalf("empty optional registry = %v, %v", registry, err)
	}
	public, _, err := projectreview.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{"reviewers": []map[string]string{{
		"reviewer_id": "external-human-1", "public_key": base64.StdEncoding.EncodeToString(public),
	}}})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "reviewers.json")
	if err := os.WriteFile(path, body, 0600); err != nil {
		t.Fatal(err)
	}
	if registry, err := loadProjectReviewers(path); err != nil || registry == nil {
		t.Fatalf("private registry rejected: %v", err)
	}
	if err := os.Chmod(path, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadProjectReviewers(path); err == nil {
		t.Fatal("world-readable registry accepted")
	}
	if err := os.Chmod(path, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "reviewers-link.json")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := loadProjectReviewers(link); err == nil {
		t.Fatal("symlink reviewer registry accepted")
	}
}
