package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsurePrivateSocketDir(t *testing.T) {
	root := t.TempDir()
	created := filepath.Join(root, "private")
	if err := ensurePrivateSocketDir(created); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(created); err != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("created directory mode: %v %v", info, err)
	}
	public := filepath.Join(root, "public")
	if err := os.Mkdir(public, 0755); err != nil {
		t.Fatal(err)
	}
	if err := ensurePrivateSocketDir(public); err == nil {
		t.Fatal("non-private directory accepted")
	}
	if info, err := os.Stat(public); err != nil || info.Mode().Perm() != 0755 {
		t.Fatalf("rejected directory was changed: %v %v", info, err)
	}
}
