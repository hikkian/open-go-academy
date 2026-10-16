package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"academy/internal/projects"
	"academy/internal/runner"
)

func finishFixtureExam(t *testing.T, s *Store, id, rank string, requiredRanks ...string) {
	t.Helper()
	if err := s.RegisterExam(id, "1", rank, id+"-criteria"); err != nil {
		t.Fatal(err)
	}
	attempt, err := s.StartExam("local", id, "1", id+"-token", nil, requiredRanks...)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := s.ClaimExamAttempt("local", id, attempt.ID, id+"-token", attempt.Revision, `{}`, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.FinishExamAttempt("local", claimed.ID, 100, true, `[{"passed":true}]`); err != nil {
		t.Fatal(err)
	}
}

func recordPassingP3Evidence(t *testing.T, s *Store) (ProjectEvidenceRequirement, ProjectSubmission, ProjectEvaluation) {
	t.Helper()
	p, ok := projects.Find("p3")
	if !ok {
		t.Fatal("p3 project is not registered")
	}
	criteriaHash, ok := projects.CriteriaHash("p3")
	if !ok {
		t.Fatal("p3 criteria hash is not registered")
	}
	files := make(map[string]string, len(p.StarterFiles))
	for name, contents := range p.StarterFiles {
		files[name] = contents
	}
	submission, err := s.RecordProjectSubmission("local", "p3", files, files)
	if err != nil {
		t.Fatal(err)
	}
	profile := ProjectEvaluationProfile{
		ProjectVersion: p.Version, CriteriaHash: criteriaHash,
		CheckerVersion: "taskflow-v1", CheckerDigest: "sha256:" + strings.Repeat("a", 64),
		RuntimePolicyHash: "sha256:" + strings.Repeat("b", 64),
		RubricVersion:     "taskflow-automatic-v1", RubricHash: "sha256:" + strings.Repeat("c", 64),
	}
	checks, _ := json.Marshal([]runner.Check{{Name: "taskflow.fixture", Passed: true}})
	evaluation, err := s.RecordProjectEvaluation("local", submission.ID, profile, "automatic_pass", 0, true, string(checks))
	if err != nil {
		t.Fatal(err)
	}
	requirement := ProjectEvidenceRequirement{
		RequirementID: "taskflow", ProjectID: "p3", ProjectVersion: p.Version, CriteriaHash: criteriaHash,
		ArtifactSchema: ProjectArtifactSchemaVersion, CheckerVersion: profile.CheckerVersion,
		CheckerDigest: profile.CheckerDigest, RuntimePolicyHash: profile.RuntimePolicyHash,
		RubricVersion: profile.RubricVersion, RubricHash: profile.RubricHash, RequirePass: true,
	}
	return requirement, submission, evaluation
}

func TestStartExamSnapshotsAndLinksRequiredProjectEvidence(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "rank-evidence.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.CreateProfile("Ada", "light"); err != nil {
		t.Fatal(err)
	}
	finishFixtureExam(t, s, "zero-final", "Zero")
	finishFixtureExam(t, s, "junior-final", "Junior", "Zero")
	finishFixtureExam(t, s, "juniorplus-final", "Junior+", "Junior")
	if _, err = s.RecordAttempt("middle-course-01", map[string]string{"main.go": "package main"}, true, "ok"); err != nil {
		t.Fatal(err)
	}
	requirement, submission, evaluation := recordPassingP3Evidence(t, s)
	if err = s.RegisterExam("middle-final", "4", "Middle", "middle-final-criteria"); err != nil {
		t.Fatal(err)
	}
	policy := RankPolicy{
		ExamID: "middle-final", ExamVersion: "4", PolicyVersion: 1, Rank: "Middle", CriteriaHash: "middle-final-criteria",
		RequiredCourseExerciseIDs: []string{"middle-course-01"}, RequiredPriorRanks: []string{"Junior+"},
		ProjectEvidenceRequirements: []ProjectEvidenceRequirement{requirement},
	}
	registered, err := s.RegisterRankPolicy(policy)
	if err != nil {
		t.Fatal(err)
	}
	attempt, err := s.StartExam("local", "middle-final", "4", "middle-token", nil)
	if err != nil {
		t.Fatal(err)
	}
	var evidence examEvidenceSnapshot
	if err = json.Unmarshal([]byte(attempt.EvidenceJSON), &evidence); err != nil {
		t.Fatal(err)
	}
	if evidence.RankPolicyVersion != registered.PolicyVersion || evidence.RankPolicyHash != registered.PolicyHash {
		t.Fatalf("policy was not snapshotted: %+v", evidence)
	}
	proof, ok := evidence.ProjectEvidence[requirement.RequirementID]
	if !ok || proof.SubmissionID != submission.ID || proof.EvaluationID != evaluation.ID || proof.ArtifactHash != submission.ArtifactHash || !proof.Passed {
		t.Fatalf("wrong linked project proof: %+v", evidence.ProjectEvidence)
	}
	var linkedSubmission, linkedEvaluation string
	if err = s.db.QueryRow(`SELECT submission_id,evaluation_id FROM exam_project_evidence WHERE attempt_id=? AND requirement_id=?`, attempt.ID, requirement.RequirementID).Scan(&linkedSubmission, &linkedEvaluation); err != nil {
		t.Fatal(err)
	}
	if linkedSubmission != submission.ID || linkedEvaluation != evaluation.ID {
		t.Fatalf("relational project link=(%s,%s), want (%s,%s)", linkedSubmission, linkedEvaluation, submission.ID, evaluation.ID)
	}
	if _, err = s.db.Exec(`DELETE FROM exam_project_evidence WHERE attempt_id=?`, attempt.ID); err == nil {
		t.Fatal("exam-linked project evidence was deletable")
	}
	claimed, err := s.ClaimExamAttempt("local", "middle-final", attempt.ID, "middle-token", attempt.Revision, `{}`, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.FinishExamAttempt("local", attempt.ID, 100, true, `[{"passed":true}]`); !errors.Is(err, ErrRankEvidenceRequired) {
		t.Fatalf("project policy unexpectedly opened advanced rank award: %v", err)
	}
	stillGrading, err := s.ExamAttempt("local", attempt.ID)
	if err != nil || stillGrading.Status != "grading" || stillGrading.AnswersJSON != claimed.AnswersJSON {
		t.Fatalf("fail-closed advanced attempt state=%+v err=%v", stillGrading, err)
	}
	awards, err := s.RankAwards("local")
	if err != nil || len(awards) != 3 {
		t.Fatalf("project evidence must not itself award Middle: awards=%+v err=%v", awards, err)
	}
}

func TestAdvancedExamCannotStartWithoutExactPassingProjectEvidence(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "rank-evidence-missing.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.CreateProfile("Ada", "light"); err != nil {
		t.Fatal(err)
	}
	finishFixtureExam(t, s, "zero-final", "Zero")
	finishFixtureExam(t, s, "junior-final", "Junior", "Zero")
	finishFixtureExam(t, s, "juniorplus-final", "Junior+", "Junior")
	if err = s.RegisterExam("middle-final", "4", "Middle", "middle-final-criteria"); err != nil {
		t.Fatal(err)
	}
	policy := RankPolicy{
		ExamID: "middle-final", ExamVersion: "4", PolicyVersion: 1, Rank: "Middle", CriteriaHash: "middle-final-criteria",
		RequiredCourseExerciseIDs: []string{"middle-course-01"}, RequiredPriorRanks: []string{"Junior+"},
		ProjectEvidenceRequirements: []ProjectEvidenceRequirement{{
			RequirementID: "taskflow", ProjectID: "p3", ProjectVersion: "missing-version", CriteriaHash: "sha256:" + strings.Repeat("d", 64),
			ArtifactSchema: ProjectArtifactSchemaVersion, CheckerVersion: "taskflow-v1", CheckerDigest: "sha256:" + strings.Repeat("a", 64),
			RuntimePolicyHash: "sha256:" + strings.Repeat("b", 64), RubricVersion: "taskflow-automatic-v1",
			RubricHash: "sha256:" + strings.Repeat("c", 64), RequirePass: true,
		}},
	}
	if _, err = s.RegisterRankPolicy(policy); err != nil {
		t.Fatal(err)
	}
	if _, err = s.StartExam("local", "middle-final", "4", "middle-token", nil); !errors.Is(err, ErrExamPrerequisites) {
		t.Fatalf("exam started without exact project evidence: %v", err)
	}
}

func TestExamAttemptRetentionPreservesProjectEvidenceLinks(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "project-evidence-retention.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.CreateProfile("Ada", "light"); err != nil {
		t.Fatal(err)
	}
	finishFixtureExam(t, s, "zero-final", "Zero")
	finishFixtureExam(t, s, "junior-final", "Junior", "Zero")
	finishFixtureExam(t, s, "juniorplus-final", "Junior+", "Junior")
	if _, err = s.RecordAttempt("middle-course-01", map[string]string{"main.go": "package main"}, true, "ok"); err != nil {
		t.Fatal(err)
	}
	requirement, _, _ := recordPassingP3Evidence(t, s)
	if err = s.RegisterExam("middle-final", "4", "Middle", "middle-final-criteria"); err != nil {
		t.Fatal(err)
	}
	policy := RankPolicy{
		ExamID: "middle-final", ExamVersion: "4", PolicyVersion: 1, Rank: "Middle", CriteriaHash: "middle-final-criteria",
		RequiredCourseExerciseIDs: []string{"middle-course-01"}, RequiredPriorRanks: []string{"Junior+"},
		ProjectEvidenceRequirements: []ProjectEvidenceRequirement{requirement},
	}
	if _, err = s.RegisterRankPolicy(policy); err != nil {
		t.Fatal(err)
	}
	protected, err := s.StartExam("local", "middle-final", "4", "middle-token", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.CancelExamAttempt("local", protected.ID); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 110; i++ {
		attempt, startErr := s.StartExam("local", "zero-final", "1", fmt.Sprintf("retention-%d", i), nil)
		if startErr != nil {
			t.Fatalf("start retention attempt %d: %v", i, startErr)
		}
		if err = s.CancelExamAttempt("local", attempt.ID); err != nil {
			t.Fatalf("cancel retention attempt %d: %v", i, err)
		}
	}
	if _, err = s.ExamAttempt("local", protected.ID); err != nil {
		t.Fatalf("project-evidence-linked exam attempt was pruned: %v", err)
	}
	var linked int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM exam_project_evidence WHERE attempt_id=?`, protected.ID).Scan(&linked); err != nil || linked != 1 {
		t.Fatalf("project evidence link count=%d err=%v, want 1", linked, err)
	}
}

func TestRankPolicyEligibilityExplainsMissingEvidenceAndUpdatesFromServerRows(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "rank-policy-eligibility.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.CreateProfile("Ada", "light"); err != nil {
		t.Fatal(err)
	}
	requirement, _, _ := recordPassingP3Evidence(t, s)
	if err = s.RegisterExam("middle-final", "4", "Middle", "middle-final-criteria"); err != nil {
		t.Fatal(err)
	}
	policy := RankPolicy{
		ExamID: "middle-final", ExamVersion: "4", PolicyVersion: 1, Rank: "Middle", CriteriaHash: "middle-final-criteria",
		RequiredCourseExerciseIDs: []string{"middle-course-01"}, RequiredPriorRanks: []string{"Junior+"},
		ProjectEvidenceRequirements: []ProjectEvidenceRequirement{requirement},
	}
	if _, err = s.RegisterRankPolicy(policy); err != nil {
		t.Fatal(err)
	}
	status, err := s.RankPolicyEligibility("local", "middle-final", "4")
	if err != nil {
		t.Fatal(err)
	}
	if status.Eligible || len(status.Requirements) != 3 || status.Requirements[0].Status != "missing" || status.Requirements[1].Status != "missing" || status.Requirements[2].Status != "verified" {
		t.Fatalf("missing evidence status=%+v", status)
	}
	if _, err = s.RecordAttempt("middle-course-01", map[string]string{"main.go": "package main"}, true, "ok"); err != nil {
		t.Fatal(err)
	}
	finishFixtureExam(t, s, "zero-final", "Zero")
	finishFixtureExam(t, s, "junior-final", "Junior", "Zero")
	finishFixtureExam(t, s, "juniorplus-final", "Junior+", "Junior")
	status, err = s.RankPolicyEligibility("local", "middle-final", "4")
	if err != nil || !status.Eligible {
		t.Fatalf("complete evidence status=%+v err=%v", status, err)
	}
	for _, requirementStatus := range status.Requirements {
		if requirementStatus.Status != "verified" || requirementStatus.Reason != "" {
			t.Fatalf("unexpected requirement status: %+v", requirementStatus)
		}
	}
}
