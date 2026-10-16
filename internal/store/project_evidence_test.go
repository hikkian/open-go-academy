package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"academy/internal/projects"
	"academy/internal/runner"
)

func TestProjectArtifactHashIsCanonicalAndVersioned(t *testing.T) {
	first := map[string]string{"z/雪.go": "package p\n", "README.md": "line\r\n"}
	second := map[string]string{"README.md": "line\r\n", "z/雪.go": "package p\n"}
	base := ProjectArtifactHash("p3", "p3-v1", first)
	if got := ProjectArtifactHash("p3", "p3-v1", second); got != base {
		t.Fatal("hash depends on Go map iteration order")
	}
	if ProjectArtifactHash("p3", "p3-v2", first) == base {
		t.Fatal("project version was omitted from artifact hash")
	}
	mutated := cloneFiles(first)
	mutated["z/雪.go"] += "// changed\n"
	if ProjectArtifactHash("p3", "p3-v1", mutated) == base {
		t.Fatal("content change was omitted from artifact hash")
	}
	mutated = cloneFiles(first)
	mutated["a/雪.go"] = mutated["z/雪.go"]
	delete(mutated, "z/雪.go")
	if ProjectArtifactHash("p3", "p3-v1", mutated) == base {
		t.Fatal("path change was omitted from artifact hash")
	}
}

func TestProjectArtifactHashIgnoresGeneratedMapInsertionOrder(t *testing.T) {
	rng := rand.New(rand.NewSource(0x0a11ce))
	for sample := 0; sample < 300; sample++ {
		count := rng.Intn(40) + 1
		files := make(map[string]string, count)
		names := make([]string, 0, count)
		for i := 0; i < count; i++ {
			name := fmt.Sprintf("nested/%02d/файл-%02d.go", sample, i)
			value := make([]byte, rng.Intn(128)+1)
			_, _ = rng.Read(value)
			files[name] = string(value)
			names = append(names, name)
		}
		permuted := append([]string(nil), names...)
		rng.Shuffle(len(permuted), func(i, j int) { permuted[i], permuted[j] = permuted[j], permuted[i] })
		rebuilt := make(map[string]string, count)
		for _, name := range permuted {
			rebuilt[name] = files[name]
		}
		if got, want := ProjectArtifactHash("p3", "p3-taskflow-v1", rebuilt), ProjectArtifactHash("p3", "p3-taskflow-v1", files); got != want {
			t.Fatalf("sample %d hash depends on insertion order: %s != %s", sample, got, want)
		}
		mutated := cloneFiles(rebuilt)
		first := permuted[0]
		mutated[first] += "\x00mutated"
		if ProjectArtifactHash("p3", "p3-taskflow-v1", mutated) == ProjectArtifactHash("p3", "p3-taskflow-v1", files) {
			t.Fatalf("sample %d content mutation did not alter artifact hash", sample)
		}
	}
}

func TestP3SubmissionAndEvaluationAreImmutableAndIdempotent(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "project-evidence.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.CreateProfile("Ada", "light"); err != nil {
		t.Fatal(err)
	}
	project, ok := projects.Find("p3")
	if !ok {
		t.Fatal("P3 project missing")
	}
	files := cloneFiles(project.StarterFiles)
	executed := cloneFiles(files)
	submission, err := s.RecordProjectSubmission("local", "p3", files, executed)
	if err != nil {
		t.Fatal(err)
	}
	replayedSubmission, err := s.RecordProjectSubmission("local", "p3", cloneFiles(files), cloneFiles(executed))
	if err != nil || replayedSubmission.ID != submission.ID || replayedSubmission.ArtifactHash != submission.ArtifactHash {
		t.Fatalf("same artifact replay = %+v, err=%v", replayedSubmission, err)
	}
	alteredExecuted := cloneFiles(executed)
	alteredExecuted["main.go"] += "// not submitted\n"
	if _, err = s.RecordProjectSubmission("local", "p3", files, alteredExecuted); err == nil {
		t.Fatal("runner source that differs from the submitted artifact was accepted")
	}
	if _, err = s.RecordProjectSubmission("other", "p3", files, executed); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("foreign profile submission err=%v", err)
	}
	criteriaHash, ok := projects.CriteriaHash("p3")
	if !ok {
		t.Fatal("P3 criteria hash missing")
	}
	digest := "sha256:" + strings.Repeat("a", 64)
	profile := ProjectEvaluationProfile{ProjectVersion: project.Version, CriteriaHash: criteriaHash,
		CheckerVersion: "taskflow-v1", CheckerDigest: digest,
		RuntimePolicyHash: "sha256:" + strings.Repeat("b", 64), RubricVersion: "taskflow-automatic-v1",
		RubricHash: "sha256:" + strings.Repeat("c", 64)}
	checks, _ := json.Marshal([]runner.Check{{Name: "p3.health", Passed: true}})
	evaluation, err := s.RecordProjectEvaluation("local", submission.ID, profile, "automatic_pass", 0, true, string(checks))
	if err != nil {
		t.Fatal(err)
	}
	replayedEvaluation, err := s.RecordProjectEvaluation("local", submission.ID, profile, "automatic_pass", 0, true, string(checks))
	if err != nil || replayedEvaluation.ID != evaluation.ID {
		t.Fatalf("same evaluation replay = %+v, err=%v", replayedEvaluation, err)
	}
	conflict, _ := json.Marshal([]runner.Check{{Name: "p3.health", Passed: false}})
	if _, err = s.RecordProjectEvaluation("local", submission.ID, profile, "failed", 1, false, string(conflict)); err == nil {
		t.Fatal("conflicting replay changed immutable evaluation")
	}
	badProfile := profile
	badProfile.ProjectVersion = "another-project-version"
	if _, err = s.RecordProjectEvaluation("local", submission.ID, badProfile, "automatic_pass", 0, true, string(checks)); err == nil {
		t.Fatal("evaluation with mismatched project version accepted")
	}

	failedFiles := cloneFiles(files)
	failedFiles["README.md"] += "\nFailed candidate revision.\n"
	failedSubmission, err := s.RecordProjectSubmission("local", "p3", failedFiles, failedFiles)
	if err != nil {
		t.Fatal(err)
	}
	failedChecks, _ := json.Marshal([]runner.Check{{Name: "p3.acceptance", Passed: false}})
	failedEvaluation, err := s.RecordProjectEvaluation("local", failedSubmission.ID, profile, "failed", 1, false, string(failedChecks))
	if err != nil || failedEvaluation.Passed || failedEvaluation.EvaluationStatus != "failed" {
		t.Fatalf("failed project evaluation not retained: %+v err=%v", failedEvaluation, err)
	}
	infraFiles := cloneFiles(files)
	infraFiles["README.md"] += "\nInfrastructure-error candidate revision.\n"
	infraSubmission, err := s.RecordProjectSubmission("local", "p3", infraFiles, infraFiles)
	if err != nil {
		t.Fatal(err)
	}
	infraEvaluation, err := s.RecordProjectEvaluation("local", infraSubmission.ID, profile, "error", 1, false, `null`)
	if err != nil || infraEvaluation.Passed || infraEvaluation.EvaluationStatus != "error" {
		t.Fatalf("infrastructure result not retained distinctly: %+v err=%v", infraEvaluation, err)
	}
	stored, err := s.ProjectEvaluation("local", evaluation.ID)
	if err != nil || !stored.Passed || stored.SubmissionID != submission.ID || stored.CheckerDigest != profile.CheckerDigest {
		t.Fatalf("stored evaluation lost trusted provenance: %+v err=%v", stored, err)
	}
	for _, table := range []string{"project_submissions", "project_evaluations"} {
		if _, err = s.DB().Exec("UPDATE " + table + " SET created_at='changed'"); err == nil {
			t.Fatalf("%s accepted UPDATE", table)
		}
		if _, err = s.DB().Exec("DELETE FROM " + table); err == nil {
			t.Fatalf("%s accepted DELETE", table)
		}
	}
}
