package store

import (
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"academy/internal/projectreview"
	"academy/internal/projects"
)

func TestVerifiedSeniorCommunicationEvidenceRequiresPersistedValidAward(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "senior-review-evidence.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.CreateProfile("Ada", "light"); err != nil {
		t.Fatal(err)
	}
	if id, err := s.VerifiedSeniorCommunicationEvidence("local"); err != nil || id != "" {
		t.Fatalf("without Senior award: id=%q err=%v", id, err)
	}
	if err = s.RegisterExam("senior-final", "1", "Senior", "criteria"); err != nil {
		t.Fatal(err)
	}
	_, err = s.db.Exec(`INSERT INTO exam_attempts(id,profile_id,exam_id,version,rank,criteria_hash,token_hash,status,score,passed,created_at,finished_at,evidence_json)
		VALUES('senior-attempt','local','senior-final','1','Senior','criteria','token','completed',100,1,'2026-10-18T00:00:00Z','2026-10-18T00:00:00Z','{')`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`INSERT INTO rank_awards(profile_id,exam_id,version,rank,attempt_id,awarded_at) VALUES('local','senior-final','1','Senior','senior-attempt','2026-10-18T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if id, err := s.VerifiedSeniorCommunicationEvidence("local"); err == nil || id != "" {
		t.Fatalf("corrupt serialized evidence: id=%q err=%v", id, err)
	}
}

func TestVerifiedSeniorCommunicationEvidenceForAcceptedMultiKeyAward(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "senior-review-positive.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.CreateProfile("Ada", "light"); err != nil {
		t.Fatal(err)
	}
	if err = s.RegisterExam("senior-final", "1", "Senior", "senior-criteria"); err != nil {
		t.Fatal(err)
	}
	project, ok := projects.Find("p5")
	if !ok {
		t.Fatal("p5 project missing")
	}
	files := make(map[string]string, len(project.StarterFiles))
	for name, contents := range project.StarterFiles {
		files[name] = contents
	}
	submission, err := s.RecordProjectSubmission("local", "p5", files, files)
	if err != nil {
		t.Fatal(err)
	}
	criteria, ok := projects.CriteriaHash("p5")
	if !ok {
		t.Fatal("p5 criteria hash missing")
	}
	profile := ProjectEvaluationProfile{ProjectVersion: project.Version, CriteriaHash: criteria,
		CheckerVersion: "capstone-deterministic-v1", CheckerDigest: "sha256:" + strings.Repeat("a", 64), RuntimePolicyHash: "sha256:" + strings.Repeat("b", 64),
		RubricVersion: "capstone-deterministic-v1", RubricHash: "sha256:" + strings.Repeat("c", 64)}
	evaluation, err := s.RecordProjectEvaluation("local", submission.ID, profile, "automatic_pass", 0, true, `[{"name":"fixture","passed":true}]`)
	if err != nil {
		t.Fatal(err)
	}
	pub1, priv1, err := projectreview.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	pub2, priv2, err := projectreview.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	pub3, priv3, err := projectreview.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	keys := []struct {
		fingerprint string
		private     []byte
	}{{testFingerprint(pub1), priv1}, {testFingerprint(pub2), priv2}, {testFingerprint(pub3), priv3}}
	sort.Slice(keys, func(i, j int) bool { return keys[i].fingerprint < keys[j].fingerprint })
	fps := make([]string, len(keys))
	for i, key := range keys {
		fps[i] = key.fingerprint
		storeSignedManualReview(t, s, submission, evaluation, []string{"senior-a", "senior-b", "senior-c"}[i], key.fingerprint, key.private, "approve", "p5-human-v1", strings.Repeat("d", 64))
	}
	requirement := ProjectEvidenceRequirement{RequirementID: "senior-project", ProjectID: "p5", ProjectVersion: project.Version, CriteriaHash: criteria,
		ArtifactSchema: ProjectArtifactSchemaVersion, CheckerVersion: profile.CheckerVersion, CheckerDigest: profile.CheckerDigest,
		RuntimePolicyHash: profile.RuntimePolicyHash, RubricVersion: profile.RubricVersion, RubricHash: profile.RubricHash, RequirePass: true,
		ManualReviewRubricVersion: "p5-human-v1", ManualReviewRubricHash: strings.Repeat("d", 64), ReviewTrustPolicyVersion: "review-trust-v1",
		AllowedReviewerKeyFingerprints: fps, MinimumDistinctReviewerKeyFingerprints: 3}
	requirement.ReviewTrustPolicyHash = ComputeReviewTrustPolicyHash(requirement.ReviewTrustPolicyVersion, fps)
	policy := RankPolicy{ExamID: "senior-final", ExamVersion: "1", PolicyVersion: 1, Rank: "Senior", CriteriaHash: "senior-criteria",
		RequiredCourseExerciseIDs: []string{"senior-course-fixture"}, RequiredPriorRanks: []string{"Middle+"}, ProjectEvidenceRequirements: []ProjectEvidenceRequirement{requirement}}
	if _, err = s.RegisterRankPolicy(policy); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RecordAttempt("senior-course-fixture", map[string]string{"main.go": "package main"}, true, "ok"); err != nil {
		t.Fatal(err)
	}
	// Prior rank is an input prerequisite fixture; the tested award below is produced by FinishExamAttempt.
	if err = s.RegisterExam("middleplus-fixture", "1", "Middle+", "middleplus-criteria"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`INSERT INTO exam_attempts(id,profile_id,exam_id,version,rank,criteria_hash,token_hash,status,score,passed,created_at,finished_at,evidence_json) VALUES('middleplus-attempt','local','middleplus-fixture','1','Middle+','middleplus-criteria','token','completed',100,1,'2026-10-18T00:00:00Z','2026-10-18T00:00:00Z','{}')`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`INSERT INTO rank_awards(profile_id,exam_id,version,rank,attempt_id,awarded_at) VALUES('local','middleplus-fixture','1','Middle+','middleplus-attempt','2026-10-18T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	attempt, err := s.StartExam("local", "senior-final", "1", "senior-token", nil)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := s.ClaimExamAttempt("local", "senior-final", attempt.ID, "senior-token", attempt.Revision, `{}`, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.FinishExamAttempt("local", claimed.ID, 100, true, `[{"passed":true}]`); err != nil {
		t.Fatal(err)
	}
	id, err := s.VerifiedSeniorCommunicationEvidence("local")
	if err != nil || !strings.HasPrefix(id, "sha256:") || len(id) != 71 {
		t.Fatalf("valid award evidence id=%q err=%v", id, err)
	}
	replay, err := s.VerifiedSeniorCommunicationEvidence("local")
	if err != nil || replay != id {
		t.Fatalf("evidence ID was not stable: first=%q replay=%q err=%v", id, replay, err)
	}
	storeSignedManualReview(t, s, submission, evaluation, "senior-a", keys[0].fingerprint, keys[0].private, "changes_requested", "p5-human-v1", strings.Repeat("d", 64))
	afterSupersedingReview, err := s.VerifiedSeniorCommunicationEvidence("local")
	if err != nil || afterSupersedingReview != id {
		t.Fatalf("new review rewrote awarded evidence: first=%q after=%q err=%v", id, afterSupersedingReview, err)
	}
}
