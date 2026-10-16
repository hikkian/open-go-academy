package store

import (
	"path/filepath"
	"strings"
	"testing"

	"academy/internal/projects"
)

func TestP2MoreThanBaselineFilesAndP1SameSetRejected(t *testing.T) {
	db := filepath.Join(t.TempDir(), "quota.db")
	s, err := Open(db)
	if err != nil {
		t.Fatal(err)
	}
	p2, _ := projects.Find("p2")
	files := p2.StarterFiles
	for i := 0; len(files) < 17; i++ {
		files["extra"+strings.Repeat("x", i)+".md"] = "x"
	}
	files["assets/part-a.md"] = strings.Repeat("a", 80*1024)
	files["assets/part-b.md"] = strings.Repeat("b", 80*1024)
	if err = s.SaveProjectFiles("p2", files); err != nil {
		t.Fatalf("P2 >16 files rejected: %v", err)
	}
	if err = s.SaveProjectFiles("p1", files); err == nil {
		t.Fatal("P1 accepted same >16-file set")
	}
	got, err := s.ProjectFiles("p2")
	if err != nil || len(got) != len(files) {
		t.Fatalf("stored files=%d want=%d err=%v", len(got), len(files), err)
	}
	snaps, err := s.ProjectSnapshots("p2")
	if err != nil || len(snaps) != 1 {
		t.Fatalf("snapshots=%v err=%v", snaps, err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(db)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err = s.ProjectFiles("p2")
	if err != nil || len(got) != len(files) {
		t.Fatalf("reopened files=%d want=%d err=%v", len(got), len(files), err)
	}
	snaps, err = s.ProjectSnapshots("p2")
	if err != nil || len(snaps) != 1 {
		t.Fatalf("reopened snapshots=%v err=%v", snaps, err)
	}
}

func TestP5WorkspaceUsesManagedProjectQuotaAboveLegacyP1Limit(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	project, ok := projects.Find("p5")
	if !ok {
		t.Fatal("P5 catalog entry missing")
	}
	files := cloneFiles(project.StarterFiles)
	for i := 1; len(files) <= 16; i++ {
		files["design/extra/evidence-"+strings.Repeat("x", i)+".md"] = "# Evidence\n"
	}
	if len(files) <= MaxProjectFiles {
		t.Fatalf("test fixture must exceed legacy P1 cap %d", MaxProjectFiles)
	}
	if err := s.SaveProjectFiles("p5", files); err != nil {
		t.Fatalf("P5 managed workspace quota rejected a valid artifact: %v", err)
	}
	stored, err := s.ProjectFiles("p5")
	if err != nil || len(stored) != len(files) {
		t.Fatalf("stored P5 files=%d want=%d err=%v", len(stored), len(files), err)
	}
}
