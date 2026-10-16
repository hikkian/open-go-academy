package store

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"academy/internal/projectreview"
	"academy/internal/projects"
	"academy/internal/runner"
)

func storeSignedManualReview(t *testing.T, s *Store, sub ProjectSubmission, ev ProjectEvaluation, reviewer string, fingerprint string, priv []byte, decision, rubricVersion, rubricHash string) ProjectReview {
	t.Helper()
	a, err := projectreview.Sign(projectreview.Attestation{Version: projectreview.Version, ReviewerID: reviewer,
		SubmissionID: sub.ID, EvaluationID: ev.ID, ProjectID: sub.ProjectID, ProjectVersion: sub.ProjectVersion,
		ArtifactHash: strings.TrimPrefix(sub.ArtifactHash, "sha256:"), RubricVersion: rubricVersion, RubricHash: rubricHash,
		Decision: decision, Comments: fmt.Sprintf("fixture review %d", time.Now().UTC().UnixNano()), IssuedAt: "2026-10-16T12:00:00Z"}, priv)
	if err != nil {
		t.Fatal(err)
	}
	pub := priv[len(priv)/2:]
	registryBytes, err := json.Marshal(map[string]any{"reviewers": []map[string]string{{"reviewer_id": reviewer, "public_key": base64.StdEncoding.EncodeToString(pub)}}})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := projectreview.LoadRegistry(strings.NewReader(string(registryBytes)))
	if err != nil {
		t.Fatal(err)
	}
	if err = projectreview.Verify(registry, a); err != nil {
		t.Fatalf("verify fixture before RecordProjectReview: %v", err)
	}
	verifiedFingerprint, err := registry.KeyFingerprint(reviewer)
	if err != nil || verifiedFingerprint != fingerprint {
		t.Fatalf("verified fingerprint=%q want=%q err=%v", verifiedFingerprint, fingerprint, err)
	}
	review, err := s.RecordProjectReview("local", a, fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	return review
}

func testFingerprint(pub []byte) string {
	sum := sha256.Sum256(pub)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func forceProjectReviewCreatedAt(t *testing.T, s *Store, reviewID, createdAt string) {
	t.Helper()
	if _, err := s.db.Exec(`DROP TRIGGER project_reviews_immutable_update`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE project_reviews SET created_at=? WHERE id=?`, createdAt, reviewID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`CREATE TRIGGER project_reviews_immutable_update BEFORE UPDATE ON project_reviews BEGIN SELECT RAISE(ABORT, 'project reviews are immutable'); END`); err != nil {
		t.Fatal(err)
	}
}

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
	pub, priv, err := projectreview.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	fingerprint := testFingerprint(pub)
	requirement.ManualReviewRubricVersion = "project-human-v1"
	requirement.ManualReviewRubricHash = strings.Repeat("d", 64)
	requirement.ReviewTrustPolicyVersion = "operator-allowlist-v1"
	requirement.AllowedReviewerKeyFingerprints = []string{fingerprint}
	requirement.MinimumDistinctReviewerKeyFingerprints = 1
	requirement.ReviewTrustPolicyHash = ComputeReviewTrustPolicyHash(requirement.ReviewTrustPolicyVersion, requirement.AllowedReviewerKeyFingerprints)
	storeSignedManualReview(t, s, submission, evaluation, "fixture-reviewer", fingerprint, priv, "approve", requirement.ManualReviewRubricVersion, requirement.ManualReviewRubricHash)
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
	_, err = s.ClaimExamAttempt("local", "middle-final", attempt.ID, "middle-token", attempt.Revision, `{}`, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	completed, err := s.FinishExamAttempt("local", attempt.ID, 100, true, `[{"passed":true}]`)
	if err != nil || !completed.Passed || completed.Score != 100 || completed.Status != "completed" {
		t.Fatalf("complete evidence did not award advanced exam: %+v err=%v", completed, err)
	}
	if _, err = s.FinishExamAttempt("local", attempt.ID, 100, true, `[{"passed":true}]`); !errors.Is(err, ErrExamAttemptConflict) {
		t.Fatalf("duplicate finish should be rejected: %v", err)
	}
	awards, err := s.RankAwards("local")
	if err != nil || len(awards) != 4 || awards[3].Rank != "Middle" || awards[3].AttemptID != attempt.ID {
		t.Fatalf("complete course, project, review, and objective evidence should award Middle exactly once: awards=%+v err=%v", awards, err)
	}
	profile, err := s.Profile()
	if err != nil || profile.Rank != "Middle" {
		t.Fatalf("profile rank=%q err=%v, want Middle", profile.Rank, err)
	}
}

func TestManualReviewQuorumIsExactSnapshottedAndRevalidated(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "manual-review-quorum.db"))
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
	requirement.ManualReviewRubricVersion = "p3-human-v1"
	requirement.ManualReviewRubricHash = strings.Repeat("f", 64)
	requirement.ReviewTrustPolicyVersion = "review-trust-v1"
	requirement.ReviewTrustPolicyHash = "sha256:" + strings.Repeat("e", 64)
	requirement.MinimumDistinctReviewerKeyFingerprints = 2
	pub1, priv1, err := projectreview.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	pub2, priv2, err := projectreview.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	fp1, fp2 := testFingerprint(pub1), testFingerprint(pub2)
	requirement.AllowedReviewerKeyFingerprints = []string{fp1, fp2}
	if requirement.AllowedReviewerKeyFingerprints[0] > requirement.AllowedReviewerKeyFingerprints[1] {
		requirement.AllowedReviewerKeyFingerprints[0], requirement.AllowedReviewerKeyFingerprints[1] = requirement.AllowedReviewerKeyFingerprints[1], requirement.AllowedReviewerKeyFingerprints[0]
	}
	requirement.ReviewTrustPolicyHash = ComputeReviewTrustPolicyHash(requirement.ReviewTrustPolicyVersion, requirement.AllowedReviewerKeyFingerprints)
	// Wrong rubric, non-approval, and a replay with the same key do not count.
	storeSignedManualReview(t, s, submission, evaluation, "reviewer-a", fp1, priv1, "approve", "other-rubric", strings.Repeat("f", 64))
	storeSignedManualReview(t, s, submission, evaluation, "reviewer-a", fp1, priv1, "approve", "p3-human-v1", strings.Repeat("f", 64))
	storeSignedManualReview(t, s, submission, evaluation, "reviewer-b", fp1, priv1, "approve", "p3-human-v1", strings.Repeat("f", 64))
	storeSignedManualReview(t, s, submission, evaluation, "reviewer-c", fp2, priv2, "changes_requested", "p3-human-v1", strings.Repeat("f", 64))
	if err = s.RegisterExam("middle-final", "4", "Middle", "middle-final-criteria"); err != nil {
		t.Fatal(err)
	}
	policy := RankPolicy{ExamID: "middle-final", ExamVersion: "4", PolicyVersion: 1, Rank: "Middle", CriteriaHash: "middle-final-criteria",
		RequiredCourseExerciseIDs: []string{"middle-course-01"}, RequiredPriorRanks: []string{"Junior+"}, ProjectEvidenceRequirements: []ProjectEvidenceRequirement{requirement}}
	registered, err := s.RegisterRankPolicy(policy)
	if err != nil {
		t.Fatal(err)
	}
	status, err := s.RankPolicyEligibility("local", "middle-final", "4")
	if err != nil || status.Eligible {
		t.Fatalf("under-quorum eligibility=%+v err=%v", status, err)
	}
	if attempt, startErr := s.StartExam("local", "middle-final", "4", "missing-review-token", nil); !errors.Is(startErr, ErrExamPrerequisites) || attempt.ID != "" {
		t.Fatalf("started without quorum: %+v err=%v", attempt, startErr)
	}
	r1 := storeSignedManualReview(t, s, submission, evaluation, "reviewer-a", fp1, priv1, "approve", "p3-human-v1", strings.Repeat("f", 64))
	r2 := storeSignedManualReview(t, s, submission, evaluation, "reviewer-b", fp2, priv2, "approve", "p3-human-v1", strings.Repeat("f", 64))
	// A newer unreviewed passing submission must not shadow the already-reviewed A.
	p3, _ := projects.Find("p3")
	newerFiles := make(map[string]string, len(p3.StarterFiles))
	for name, contents := range p3.StarterFiles {
		newerFiles[name] = contents
	}
	newerSource := map[string]string{}
	for name, contents := range newerFiles {
		newerSource[name] = contents
		break
	}
	newerSubmission, err := s.RecordProjectSubmission("local", "p3", newerFiles, newerSource)
	if err != nil {
		t.Fatal(err)
	}
	newerEvaluation, err := s.RecordProjectEvaluation("local", newerSubmission.ID, ProjectEvaluationProfile{
		ProjectVersion: p3.Version, CriteriaHash: requirement.CriteriaHash, CheckerVersion: requirement.CheckerVersion,
		CheckerDigest: requirement.CheckerDigest, RuntimePolicyHash: requirement.RuntimePolicyHash,
		RubricVersion: requirement.RubricVersion, RubricHash: requirement.RubricHash,
	}, "automatic_pass", 0, true, `[{"name":"taskflow.fixture","passed":true}]`)
	if err != nil {
		t.Fatal(err)
	}
	if newerSubmission.ID == submission.ID {
		t.Fatal("fixture did not create a distinct newer submission")
	}
	if newerEvaluation.ID == evaluation.ID {
		t.Fatal("fixture did not create a distinct newer evaluation")
	}
	startReject := storeSignedManualReview(t, s, submission, evaluation, "reviewer-a", fp1, priv1, "changes_requested", "p3-human-v1", strings.Repeat("f", 64))
	forceProjectReviewCreatedAt(t, s, r1.ID, "2026-10-16T12:00:00.1Z")
	forceProjectReviewCreatedAt(t, s, startReject.ID, "2026-10-16T12:00:00.100001Z")
	status, err = s.RankPolicyEligibility("local", "middle-final", "4")
	if err != nil || status.Eligible {
		t.Fatalf("RFC3339 fractional precision let superseded approval count: %+v err=%v", status, err)
	}
	if _, err = s.StartExam("local", "middle-final", "4", "superseded-before-start", nil); !errors.Is(err, ErrExamPrerequisites) {
		t.Fatalf("Start accepted decision superseded by .100001Z row: %v", err)
	}
	// Equal wall-clock timestamps still follow append order; the later approval restores the quorum.
	r1 = storeSignedManualReview(t, s, submission, evaluation, "reviewer-a", fp1, priv1, "approve", "p3-human-v1", strings.Repeat("f", 64))
	forceProjectReviewCreatedAt(t, s, startReject.ID, "2026-10-16T12:00:01Z")
	forceProjectReviewCreatedAt(t, s, r1.ID, "2026-10-16T12:00:01Z")
	status, err = s.RankPolicyEligibility("local", "middle-final", "4")
	if err != nil || !status.Eligible {
		t.Fatalf("equal timestamp append-order eligibility=%+v err=%v", status, err)
	}
	attempt, err := s.StartExam("local", "middle-final", "4", "middle-review-token", nil)
	if err != nil {
		t.Fatal(err)
	}
	var snapshot examEvidenceSnapshot
	if err = json.Unmarshal([]byte(attempt.EvidenceJSON), &snapshot); err != nil {
		t.Fatal(err)
	}
	proof := snapshot.ProjectEvidence[requirement.RequirementID]
	if snapshot.RankPolicyHash != registered.PolicyHash || len(proof.ManualReviews) != 2 || proof.SubmissionID != submission.ID {
		t.Fatalf("review records not captured in evidence JSON: %+v", proof)
	}
	if len(proof.ManualReviews) != 2 || !((proof.ManualReviews[0].ReviewID == r1.ID && proof.ManualReviews[1].ReviewID == r2.ID) ||
		(proof.ManualReviews[0].ReviewID == r2.ID && proof.ManualReviews[1].ReviewID == r1.ID)) {
		t.Fatalf("unstable review selection: %+v", proof.ManualReviews)
	}
	var linked int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM exam_project_review_evidence WHERE attempt_id=?`, attempt.ID).Scan(&linked); err != nil || linked != 2 {
		t.Fatalf("review links=%d err=%v", linked, err)
	}
	if _, err = s.db.Exec(`DELETE FROM exam_project_review_evidence WHERE attempt_id=?`, attempt.ID); err == nil {
		t.Fatal("review evidence link was deletable")
	}
	postStartReject := storeSignedManualReview(t, s, submission, evaluation, "reviewer-a", fp1, priv1, "changes_requested", "p3-human-v1", strings.Repeat("f", 64))
	forceProjectReviewCreatedAt(t, s, r1.ID, "2026-10-16T12:00:00.1Z")
	forceProjectReviewCreatedAt(t, s, postStartReject.ID, "2026-10-16T12:00:00.100001Z")
	if _, err = s.ClaimExamAttempt("local", "middle-final", attempt.ID, "middle-review-token", attempt.Revision, `{}`, `{}`); err == nil {
		t.Fatal("Claim accepted an approval superseded after Start")
	}
	if err = s.CancelExamAttempt("local", attempt.ID); err != nil {
		t.Fatalf("cancel superseded review-linked attempt: %v", err)
	}
	r1 = storeSignedManualReview(t, s, submission, evaluation, "reviewer-a", fp1, priv1, "approve", "p3-human-v1", strings.Repeat("f", 64))
	attempt, err = s.StartExam("local", "middle-final", "4", "middle-review-retention-token", nil)
	if err != nil {
		t.Fatalf("start restored review quorum: %v", err)
	}
	if err = s.CancelExamAttempt("local", attempt.ID); err != nil {
		t.Fatalf("cancel review-linked retention attempt: %v", err)
	}
	for i := 0; i < 105; i++ {
		other, startErr := s.StartExam("local", "zero-final", "1", fmt.Sprintf("manual-retention-%d", i), nil)
		if startErr != nil {
			t.Fatalf("start retention attempt %d: %v", i, startErr)
		}
		if cancelErr := s.CancelExamAttempt("local", other.ID); cancelErr != nil {
			t.Fatalf("cancel retention attempt %d: %v", i, cancelErr)
		}
	}
	if _, err = s.ExamAttempt("local", attempt.ID); err != nil {
		t.Fatalf("manual-review-linked attempt was pruned: %v", err)
	}
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM exam_project_review_evidence WHERE attempt_id=?`, attempt.ID).Scan(&linked); err != nil || linked != 2 {
		t.Fatalf("manual review evidence links after retention=%d err=%v", linked, err)
	}
	attempt, err = s.StartExam("local", "middle-final", "4", "middle-review-token-2", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.ClaimExamAttempt("local", "middle-final", attempt.ID, "middle-review-token-2", attempt.Revision, `{}`, `{}`)
	if err != nil {
		t.Fatalf("claim did not revalidate review snapshot: %v", err)
	}
	storeSignedManualReview(t, s, submission, evaluation, "reviewer-b", fp2, priv2, "reject", "p3-human-v1", strings.Repeat("f", 64))
	_, validationErr := s.FinishExamAttempt("local", attempt.ID, 100, true, `[{"passed":true}]`)
	if !errors.Is(validationErr, ErrExamPrerequisites) || !strings.Contains(validationErr.Error(), "superseded") {
		t.Fatalf("transactional snapshot validation did not reject post-Claim supersession: %v", validationErr)
	}
	stillGrading, err := s.ExamAttempt("local", attempt.ID)
	if err != nil || stillGrading.Status != "grading" || stillGrading.Passed || stillGrading.Score != 0 {
		t.Fatalf("failed finish mutated attempt: %+v err=%v", stillGrading, err)
	}
	awards, err := s.RankAwards("local")
	if err != nil || len(awards) != 3 {
		t.Fatalf("superseded review awarded advanced rank: awards=%+v err=%v", awards, err)
	}
	profile, err := s.Profile()
	if err != nil || profile.Rank != "Junior+" {
		t.Fatalf("superseded review mutated profile rank=%q err=%v", profile.Rank, err)
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
			ManualReviewRubricVersion: "p3-human-v1", ManualReviewRubricHash: strings.Repeat("d", 64),
			ReviewTrustPolicyVersion: "operator-v1", AllowedReviewerKeyFingerprints: []string{"sha256:" + strings.Repeat("1", 64)},
			MinimumDistinctReviewerKeyFingerprints: 1,
		}},
	}
	policy.ProjectEvidenceRequirements[0].ReviewTrustPolicyHash = ComputeReviewTrustPolicyHash(policy.ProjectEvidenceRequirements[0].ReviewTrustPolicyVersion, policy.ProjectEvidenceRequirements[0].AllowedReviewerKeyFingerprints)
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
	if status.Eligible || len(status.Requirements) != 4 || status.Requirements[0].Status != "missing" || status.Requirements[1].Status != "missing" || status.Requirements[2].Status != "verified" || status.Requirements[3].Status != "verified" {
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
