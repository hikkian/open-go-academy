package store

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestFreshReopenAndIdempotentVerifiedXP(t *testing.T) {
	p := filepath.Join(t.TempDir(), "academy.db")
	s, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Profile(); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("profile err=%v", err)
	}
	profile, err := s.CreateProfile("Ada", "dark")
	if err != nil {
		t.Fatal(err)
	}
	if profile.Rank != "Zero" || profile.XP != 0 {
		t.Fatalf("profile=%+v", profile)
	}
	files := map[string]string{"main.go": "package main"}
	if err = s.SaveDraft("hello", files); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RecordAttempt("hello", files, false, "compile error"); err != nil {
		t.Fatal(err)
	}
	progress, err := s.Progress()
	if err != nil {
		t.Fatal(err)
	}
	if len(progress.Completed) != 0 || progress.XP != 0 {
		t.Fatalf("failed attempt granted progress: %+v", progress)
	}
	for i := 0; i < 2; i++ {
		if _, err = s.RecordAttempt("hello", files, true, "ok\n"); err != nil {
			t.Fatal(err)
		}
	}
	progress, err = s.Progress()
	if err != nil {
		t.Fatal(err)
	}
	if len(progress.Completed) != 1 || progress.XP != 10 || len(progress.Attempts) != 3 {
		t.Fatalf("progress=%+v", progress)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	profile, err = s.Profile()
	if err != nil {
		t.Fatal(err)
	}
	if profile.XP != 10 || profile.Name != "Ada" {
		t.Fatalf("reopened profile=%+v", profile)
	}
	draft, err := s.Draft("hello")
	if err != nil {
		t.Fatal(err)
	}
	if draft["main.go"] != files["main.go"] {
		t.Fatalf("draft=%v", draft)
	}
}

func TestAttemptRetentionPreservesVerifiedEvidence(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "retention.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.CreateProfile("Ada", "light"); err != nil {
		t.Fatal(err)
	}
	verified, err := s.RecordAttempt("hello", map[string]string{"main.go": "package main"}, true, "ok")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 110; i++ {
		if _, err := s.RecordAttempt("hello", map[string]string{"main.go": "broken"}, false, "compile error"); err != nil {
			t.Fatalf("failed attempt %d: %v", i, err)
		}
	}
	p, err := s.Progress()
	if err != nil {
		t.Fatal(err)
	}
	if p.XP != 10 || len(p.Completed) != 1 || p.Completed[0] != "hello" {
		t.Fatalf("progress lost verified completion: %+v", p)
	}
	if p.Evidence["hello"] != verified.ID {
		t.Fatalf("evidence=%v, want original attempt %d", p.Evidence, verified.ID)
	}
	if len(p.Attempts) != 100 {
		t.Fatalf("recent attempt window has %d rows, want 100", len(p.Attempts))
	}
	for i, attempt := range p.Attempts {
		if attempt.ID == verified.ID {
			t.Fatal("old verified evidence should be returned by Evidence, not recent history")
		}
		if i > 0 && p.Attempts[i-1].ID >= attempt.ID {
			t.Fatal("recent attempts are not chronological")
		}
	}
	var retained int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM attempts`).Scan(&retained); err != nil {
		t.Fatal(err)
	}
	if retained != 101 {
		t.Fatalf("stored %d attempts, want 100 recent plus protected evidence", retained)
	}
	var passed int
	if err := s.db.QueryRow(`SELECT passed FROM attempts WHERE id=?`, p.Evidence["hello"]).Scan(&passed); err != nil {
		t.Fatalf("protected evidence row missing: %v", err)
	}
	if passed != 1 {
		t.Fatal("protected evidence is not a passed attempt")
	}
}

func TestConcurrentVerifiedReplaysAwardXPOnce(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "concurrent.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.CreateProfile("Ada", "dark"); err != nil {
		t.Fatal(err)
	}
	const workers = 10
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.RecordAttempt("hello", map[string]string{"main.go": "package main"}, true, "ok")
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	p, err := s.Progress()
	if err != nil {
		t.Fatal(err)
	}
	if p.XP != 10 || len(p.Completed) != 1 || p.Evidence["hello"] == 0 {
		t.Fatalf("concurrent retries produced invalid progress: %+v", p)
	}
	if len(p.Attempts) != workers {
		t.Fatalf("got %d attempts, want %d", len(p.Attempts), workers)
	}
}

func TestProjectFilesMigrationReplaceSnapshotsAndProjectIsolation(t *testing.T) {
	p := filepath.Join(t.TempDir(), "upgrade.db")
	db, err := sql.Open("sqlite", p)
	if err != nil {
		t.Fatal(err)
	}
	migration1, err := migrations.ReadFile("migrations/0001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY,applied_at TEXT NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(string(migration1)); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO schema_migrations(version,applied_at) VALUES(1,'2026-10-15T00:00:00Z'); INSERT INTO profiles(id,name,theme,rank,xp) VALUES('local','Ada','dark','Zero',20)`); err != nil {
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := s.Profile()
	if err != nil || profile.Name != "Ada" || profile.XP != 20 {
		t.Fatalf("migration lost existing profile: %+v, %v", profile, err)
	}
	initial := map[string]string{"main.go": "package main\n", "go.mod": "module example\n"}
	if err := s.SaveProjectFiles("project-a", initial); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveProjectFiles("project-a", initial); err != nil {
		t.Fatal(err)
	}
	snapshots, err := s.ProjectSnapshots("project-a")
	if err != nil || len(snapshots) != 1 {
		t.Fatalf("identical save made a snapshot: %+v, %v", snapshots, err)
	}
	updated := map[string]string{"main.go": "package main\nfunc main() {}\n", "helper.go": "package main\n"}
	if err := s.SaveProjectFiles("project-a", updated); err != nil {
		t.Fatal(err)
	}
	got, err := s.ProjectFiles("project-a")
	if err != nil || !sameFiles(got, updated) {
		t.Fatalf("current files=%v, err=%v", got, err)
	}
	snapshots, err = s.ProjectSnapshots("project-a")
	if err != nil || len(snapshots) != 2 {
		t.Fatalf("snapshots=%+v, err=%v", snapshots, err)
	}
	old, err := s.ProjectSnapshot("project-a", snapshots[1].ID)
	if err != nil || !sameFiles(old, initial) {
		t.Fatalf("first version=%v, err=%v", old, err)
	}
	if _, err := s.ProjectSnapshot("project-b", snapshots[1].ID); !errors.Is(err, ErrProjectSnapshotNotFound) {
		t.Fatalf("cross-project lookup err=%v", err)
	}
	if _, err := s.ProjectSnapshot("project-a", 99999); !errors.Is(err, ErrProjectSnapshotNotFound) {
		t.Fatalf("missing lookup err=%v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	reopened, err := s.ProjectFiles("project-a")
	if err != nil || !sameFiles(reopened, updated) {
		t.Fatalf("reopened files=%v, err=%v", reopened, err)
	}
	reopenedSnapshots, err := s.ProjectSnapshots("project-a")
	if err != nil || len(reopenedSnapshots) != 2 {
		t.Fatalf("reopened snapshots=%+v, err=%v", reopenedSnapshots, err)
	}
}

func TestProjectFileQuotasAtomicityAndSnapshotRetention(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "projects.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	base := map[string]string{"main.go": "version-0"}
	if err := s.SaveProjectFiles("project", base); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`CREATE TRIGGER reject_project_file BEFORE INSERT ON project_files WHEN NEW.path='b.go' BEGIN SELECT RAISE(ABORT,'injected failure'); END`); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveProjectFiles("project", map[string]string{"a.go": "partial", "b.go": "failure"}); err == nil {
		t.Fatal("expected injected write failure")
	}
	atomic, err := s.ProjectFiles("project")
	if err != nil || !sameFiles(atomic, base) {
		t.Fatalf("failed replacement was not atomic: %v, %v", atomic, err)
	}
	if _, err := s.db.Exec(`DROP TRIGGER reject_project_file`); err != nil {
		t.Fatal(err)
	}
	tooMany := map[string]string{}
	for i := 0; i <= MaxProjectFiles; i++ {
		tooMany[fmt.Sprintf("f%d.go", i)] = "x"
	}
	if err := s.SaveProjectFiles("project", tooMany); err == nil {
		t.Fatal("expected file count quota error")
	}
	tooLarge := map[string]string{"main.go": strings.Repeat("x", MaxProjectSourceBytes+1)}
	if err := s.SaveProjectFiles("project", tooLarge); err == nil {
		t.Fatal("expected source size quota error")
	}
	got, err := s.ProjectFiles("project")
	if err != nil || !sameFiles(got, base) {
		t.Fatalf("quota error changed current files: %v, %v", got, err)
	}
	if err := s.SaveProjectFiles("project", map[string]string{"../escape.go": "bad"}); err == nil {
		t.Fatal("expected unsafe path error")
	}
	for i := 1; i <= 12; i++ {
		files := map[string]string{"main.go": fmt.Sprintf("version-%d", i)}
		if err := s.SaveProjectFiles("project", files); err != nil {
			t.Fatalf("save version %d: %v", i, err)
		}
	}
	snapshots, err := s.ProjectSnapshots("project")
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != MaxProjectSnapshots {
		t.Fatalf("got %d snapshots, want %d", len(snapshots), MaxProjectSnapshots)
	}
	newest, err := s.ProjectSnapshot("project", snapshots[0].ID)
	if err != nil || newest["main.go"] != "version-12" {
		t.Fatalf("newest snapshot=%v, err=%v", newest, err)
	}
	oldestRetained, err := s.ProjectSnapshot("project", snapshots[len(snapshots)-1].ID)
	if err != nil || oldestRetained["main.go"] != "version-3" {
		t.Fatalf("oldest retained snapshot=%v, err=%v", oldestRetained, err)
	}
}
