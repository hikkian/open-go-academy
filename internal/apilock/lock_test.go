//go:build linux || darwin || freebsd || openbsd || netbsd

package apilock

import (
	"path/filepath"
	"testing"
)

func TestDatabaseLockExcludesSecondOwnerAndReleases(t *testing.T) {
	path := filepath.Join(t.TempDir(), "academy.db")
	first, err := Acquire(path)
	if err != nil {
		t.Fatal(err)
	}
	if other, err := Acquire(path); err == nil {
		other.Close()
		t.Fatal("second API could acquire database")
	}
	if err = first.Close(); err != nil {
		t.Fatal(err)
	}
	next, err := Acquire(path)
	if err != nil {
		t.Fatal("lock not released", err)
	}
	next.Close()
}
