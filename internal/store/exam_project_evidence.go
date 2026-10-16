package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"academy/internal/projectreview"
)

var (
	ErrExamManualReviewQuorum     = errors.New("exam manual review quorum is not verified")
	ErrExamProjectEvidenceMissing = errors.New("exam project passing evidence is missing")
)

// examProjectEvidenceSnapshot is copied into the exam attempt and into a
// relational FK-backed table. The JSON copy is convenient to inspect; the
// table makes the evidence durable and links it to exact immutable rows.
type examProjectEvidenceSnapshot struct {
	RequirementID      string                      `json:"requirement_id"`
	SubmissionID       string                      `json:"submission_id"`
	EvaluationID       string                      `json:"evaluation_id"`
	ProjectID          string                      `json:"project_id"`
	ProjectVersion     string                      `json:"project_version"`
	CriteriaHash       string                      `json:"criteria_hash"`
	ArtifactSchema     int                         `json:"artifact_schema_version"`
	ArtifactHash       string                      `json:"artifact_hash"`
	ExecutedSourceHash string                      `json:"executed_source_hash"`
	CheckerVersion     string                      `json:"checker_version"`
	CheckerDigest      string                      `json:"checker_digest"`
	RuntimePolicyHash  string                      `json:"runtime_policy_hash"`
	RubricVersion      string                      `json:"rubric_version"`
	RubricHash         string                      `json:"rubric_hash"`
	EvaluationStatus   string                      `json:"evaluation_status"`
	ExitCode           int                         `json:"exit_code"`
	Passed             bool                        `json:"passed"`
	ChecksHash         string                      `json:"checks_hash"`
	ManualReviews      []examProjectReviewSnapshot `json:"manual_reviews,omitempty"`
}

type examProjectReviewSnapshot struct {
	ReviewID               string `json:"review_id"`
	AttestationDigest      string `json:"attestation_digest"`
	ReviewerKeyFingerprint string `json:"reviewer_key_fingerprint"`
}

func resolveProjectEvidenceTx(tx *sql.Tx, profileID string, policy RankPolicy) (map[string]examProjectEvidenceSnapshot, error) {
	resolved := make(map[string]examProjectEvidenceSnapshot, len(policy.ProjectEvidenceRequirements))
	for _, requirement := range policy.ProjectEvidenceRequirements {
		evidence, err := resolveProjectEvidenceRequirementTx(tx, profileID, requirement)
		if err != nil {
			return nil, err
		}
		resolved[requirement.RequirementID] = evidence
	}
	return resolved, nil
}

// resolveProjectEvidenceRequirementTx walks passing immutable evaluations in
// newest-first order and selects the newest candidate with a current review quorum.
func resolveProjectEvidenceRequirementTx(tx *sql.Tx, profileID string, requirement ProjectEvidenceRequirement) (examProjectEvidenceSnapshot, error) {
	rows, err := tx.Query(`SELECT s.id,e.id,s.project_id,s.project_version,s.criteria_hash,s.artifact_schema_version,
			s.artifact_hash,s.executed_source_hash,e.checker_version,e.checker_digest,e.runtime_policy_hash,
			e.rubric_version,e.rubric_hash,e.evaluation_status,e.exit_code,e.passed,e.checks_hash
			FROM project_submissions s JOIN project_evaluations e
			ON e.submission_id=s.id AND e.profile_id=s.profile_id AND e.project_id=s.project_id
			WHERE s.profile_id=? AND s.project_id=? AND s.project_version=? AND s.criteria_hash=? AND s.artifact_schema_version=?
			AND e.checker_version=? AND e.checker_digest=? AND e.runtime_policy_hash=?
			AND e.rubric_version=? AND e.rubric_hash=?
			AND e.evaluation_status='automatic_pass' AND e.exit_code=0 AND e.passed=1
			ORDER BY e.created_at DESC,e.id DESC`,
		profileID, requirement.ProjectID, requirement.ProjectVersion, requirement.CriteriaHash, requirement.ArtifactSchema,
		requirement.CheckerVersion, requirement.CheckerDigest, requirement.RuntimePolicyHash,
		requirement.RubricVersion, requirement.RubricHash)
	if err != nil {
		return examProjectEvidenceSnapshot{}, err
	}
	candidates := make([]examProjectEvidenceSnapshot, 0)
	for rows.Next() {
		var evidence examProjectEvidenceSnapshot
		var passed int
		if err = rows.Scan(&evidence.SubmissionID, &evidence.EvaluationID, &evidence.ProjectID, &evidence.ProjectVersion,
			&evidence.CriteriaHash, &evidence.ArtifactSchema, &evidence.ArtifactHash, &evidence.ExecutedSourceHash,
			&evidence.CheckerVersion, &evidence.CheckerDigest, &evidence.RuntimePolicyHash, &evidence.RubricVersion,
			&evidence.RubricHash, &evidence.EvaluationStatus, &evidence.ExitCode, &passed, &evidence.ChecksHash); err != nil {
			rows.Close()
			return examProjectEvidenceSnapshot{}, err
		}
		evidence.RequirementID, evidence.Passed = requirement.RequirementID, passed == 1
		candidates = append(candidates, evidence)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return examProjectEvidenceSnapshot{}, err
	}
	if err = rows.Close(); err != nil {
		return examProjectEvidenceSnapshot{}, err
	}
	if len(candidates) == 0 {
		return examProjectEvidenceSnapshot{}, fmt.Errorf("%w: project evidence %s: %w", ErrExamPrerequisites, requirement.RequirementID, ErrExamProjectEvidenceMissing)
	}
	quorumMissing := false
	for _, candidate := range candidates {
		if requirement.MinimumDistinctReviewerKeyFingerprints > 0 {
			reviews, reviewErr := resolveManualReviewsTx(tx, profileID, requirement, candidate)
			if errors.Is(reviewErr, ErrExamManualReviewQuorum) {
				quorumMissing = true
				continue
			}
			if reviewErr != nil {
				return examProjectEvidenceSnapshot{}, reviewErr
			}
			candidate.ManualReviews = reviews
		}
		return candidate, nil
	}
	if quorumMissing {
		return examProjectEvidenceSnapshot{}, fmt.Errorf("%w: manual review quorum %s: %w", ErrExamPrerequisites, requirement.RequirementID, ErrExamManualReviewQuorum)
	}
	return examProjectEvidenceSnapshot{}, fmt.Errorf("%w: project evidence %s: %w", ErrExamPrerequisites, requirement.RequirementID, ErrExamProjectEvidenceMissing)
}

func containsSortedFingerprint(fingerprints []string, target string) bool {
	for _, fingerprint := range fingerprints {
		if fingerprint == target {
			return true
		}
		if fingerprint > target {
			return false
		}
	}
	return false
}

func resolveManualReviewsTx(tx *sql.Tx, profileID string, requirement ProjectEvidenceRequirement, proof examProjectEvidenceSnapshot) ([]examProjectReviewSnapshot, error) {
	// project_reviews is append-only, so rowid provides a true server acceptance order;
	// created_at is informational and RFC3339Nano strings have variable precision.
	rows, err := tx.Query(`SELECT id,attestation_digest,reviewer_key_fingerprint,attestation_json,decision
		FROM project_reviews WHERE profile_id=? AND submission_id=? AND evaluation_id=? AND project_id=?
		AND project_version=? AND artifact_hash=replace(?,'sha256:','') AND rubric_version=? AND rubric_hash=?
		AND reviewer_key_fingerprint<>'' ORDER BY rowid DESC`, profileID, proof.SubmissionID, proof.EvaluationID,
		proof.ProjectID, proof.ProjectVersion, proof.ArtifactHash, requirement.ManualReviewRubricVersion, requirement.ManualReviewRubricHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	latest := make(map[string]examProjectReviewSnapshot)
	for rows.Next() {
		var item examProjectReviewSnapshot
		var raw, decision string
		if err = rows.Scan(&item.ReviewID, &item.AttestationDigest, &item.ReviewerKeyFingerprint, &raw, &decision); err != nil {
			return nil, err
		}
		if !projectreview.ValidKeyFingerprint(item.ReviewerKeyFingerprint) || !containsSortedFingerprint(requirement.AllowedReviewerKeyFingerprints, item.ReviewerKeyFingerprint) {
			continue
		}
		if _, exists := latest[item.ReviewerKeyFingerprint]; exists {
			continue
		}
		latest[item.ReviewerKeyFingerprint] = examProjectReviewSnapshot{}
		var a projectreview.Attestation
		if err = json.Unmarshal([]byte(raw), &a); err != nil {
			return nil, err
		}
		if err = validateProjectReviewDigest(a, item.AttestationDigest); err != nil {
			return nil, err
		}
		if decision != a.Decision || a.SubmissionID != proof.SubmissionID || a.EvaluationID != proof.EvaluationID ||
			a.ProjectID != proof.ProjectID || a.ProjectVersion != proof.ProjectVersion || a.ArtifactHash != strings.TrimPrefix(proof.ArtifactHash, "sha256:") ||
			a.RubricVersion != requirement.ManualReviewRubricVersion || a.RubricHash != requirement.ManualReviewRubricHash {
			return nil, errors.New("manual review row does not match its attestation")
		}
		if decision == "approve" {
			latest[item.ReviewerKeyFingerprint] = item
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	selected := make([]examProjectReviewSnapshot, 0, requirement.MinimumDistinctReviewerKeyFingerprints)
	for _, fingerprint := range requirement.AllowedReviewerKeyFingerprints {
		if review, ok := latest[fingerprint]; ok && review.ReviewID != "" {
			selected = append(selected, review)
			if len(selected) == requirement.MinimumDistinctReviewerKeyFingerprints {
				break
			}
		}
	}
	if len(selected) < requirement.MinimumDistinctReviewerKeyFingerprints {
		return nil, fmt.Errorf("%w: manual review quorum %s", ErrExamManualReviewQuorum, requirement.RequirementID)
	}
	return selected, nil
}

func insertProjectEvidenceTx(tx *sql.Tx, attemptID, profileID string, policy RankPolicy, evidence map[string]examProjectEvidenceSnapshot, createdAt time.Time) error {
	for _, requirement := range policy.ProjectEvidenceRequirements {
		proof, ok := evidence[requirement.RequirementID]
		if !ok {
			return fmt.Errorf("project evidence snapshot missing requirement %s", requirement.RequirementID)
		}
		_, err := tx.Exec(`INSERT INTO exam_project_evidence(
			attempt_id,profile_id,exam_id,exam_version,policy_version,requirement_id,submission_id,evaluation_id,
			project_id,project_version,criteria_hash,artifact_schema_version,artifact_hash,executed_source_hash,
			checker_version,checker_digest,runtime_policy_hash,rubric_version,rubric_hash,evaluation_status,exit_code,
			passed,checks_hash,created_at
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			attemptID, profileID, policy.ExamID, policy.ExamVersion, policy.PolicyVersion, proof.RequirementID,
			proof.SubmissionID, proof.EvaluationID, proof.ProjectID, proof.ProjectVersion, proof.CriteriaHash,
			proof.ArtifactSchema, proof.ArtifactHash, proof.ExecutedSourceHash, proof.CheckerVersion,
			proof.CheckerDigest, proof.RuntimePolicyHash, proof.RubricVersion, proof.RubricHash,
			proof.EvaluationStatus, proof.ExitCode, boolInt(proof.Passed), proof.ChecksHash, createdAt.Format(time.RFC3339Nano))
		if err != nil {
			return err
		}
		for _, review := range proof.ManualReviews {
			var submissionID, evaluationID, projectID, projectVersion, artifactHash, rubricVersion, rubricHash, decision string
			err = tx.QueryRow(`SELECT submission_id,evaluation_id,project_id,project_version,artifact_hash,rubric_version,rubric_hash,decision
				FROM project_reviews WHERE id=? AND profile_id=? AND attestation_digest=? AND reviewer_key_fingerprint=?`,
				review.ReviewID, profileID, review.AttestationDigest, review.ReviewerKeyFingerprint).Scan(&submissionID, &evaluationID, &projectID, &projectVersion, &artifactHash, &rubricVersion, &rubricHash, &decision)
			if err != nil {
				return err
			}
			_, err = tx.Exec(`INSERT INTO exam_project_review_evidence(attempt_id,profile_id,exam_id,exam_version,policy_version,requirement_id,
				review_id,submission_id,evaluation_id,project_id,project_version,artifact_hash,rubric_version,rubric_hash,decision,
				attestation_digest,reviewer_key_fingerprint,created_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				attemptID, profileID, policy.ExamID, policy.ExamVersion, policy.PolicyVersion, proof.RequirementID, review.ReviewID,
				submissionID, evaluationID, projectID, projectVersion, artifactHash, rubricVersion, rubricHash, decision,
				review.AttestationDigest, review.ReviewerKeyFingerprint, createdAt.Format(time.RFC3339Nano))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func rankPolicyForAttemptTx(tx *sql.Tx, examID, examVersion string, version int) (RankPolicy, error) {
	var rank, criteriaHash, storedHash, raw, createdAt string
	err := tx.QueryRow(`SELECT rank,criteria_hash,policy_hash,definition_json,created_at FROM rank_policies
		WHERE exam_id=? AND exam_version=? AND policy_version=?`, examID, examVersion, version).
		Scan(&rank, &criteriaHash, &storedHash, &raw, &createdAt)
	if errors.Is(err, sql.ErrNoRows) {
		return RankPolicy{}, ErrRankPolicyNotFound
	}
	if err != nil {
		return RankPolicy{}, err
	}
	var definition rankPolicyDefinition
	if err = json.Unmarshal([]byte(raw), &definition); err != nil {
		return RankPolicy{}, err
	}
	if err = validateRankPolicy(RankPolicy{ExamID: definition.ExamID, ExamVersion: definition.ExamVersion, PolicyVersion: definition.PolicyVersion,
		Rank: definition.Rank, CriteriaHash: definition.CriteriaHash, RequiredCourseExerciseIDs: definition.RequiredCourseExerciseIDs,
		RequiredPriorRanks: definition.RequiredPriorRanks, ProjectEvidenceRequirements: definition.ProjectEvidenceRequirements}); err != nil {
		return RankPolicy{}, fmt.Errorf("stored rank policy validation failed: %w", err)
	}
	canonical, err := json.Marshal(definition)
	if err != nil {
		return RankPolicy{}, err
	}
	if string(canonical) != raw || rankPolicyHash(canonical) != storedHash || definition.ExamID != examID ||
		definition.ExamVersion != examVersion || definition.PolicyVersion != version || definition.Rank != rank ||
		definition.CriteriaHash != criteriaHash {
		return RankPolicy{}, errors.New("stored rank policy integrity check failed")
	}
	created, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return RankPolicy{}, err
	}
	return RankPolicy{ExamID: definition.ExamID, ExamVersion: definition.ExamVersion, PolicyVersion: version,
		Rank: definition.Rank, CriteriaHash: definition.CriteriaHash, RequiredCourseExerciseIDs: definition.RequiredCourseExerciseIDs,
		RequiredPriorRanks: definition.RequiredPriorRanks, ProjectEvidenceRequirements: definition.ProjectEvidenceRequirements,
		PolicyHash: storedHash, CreatedAt: created}, nil
}

func validateProjectEvidenceSnapshotTx(tx *sql.Tx, profileID, attemptID, examID, examVersion, rank, criteriaHash string, evidence examEvidenceSnapshot) error {
	if evidence.RankPolicyVersion == 0 {
		if evidence.RankPolicyHash != "" || len(evidence.ProjectEvidence) != 0 {
			return errors.New("exam project evidence has no registered rank policy")
		}
		var linked int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM exam_project_evidence WHERE attempt_id=?`, attemptID).Scan(&linked); err != nil {
			return err
		}
		if linked != 0 {
			return errors.New("unexpected linked project evidence")
		}
		if err := tx.QueryRow(`SELECT COUNT(*) FROM exam_project_review_evidence WHERE attempt_id=?`, attemptID).Scan(&linked); err != nil {
			return err
		}
		if linked != 0 {
			return errors.New("unexpected linked project review evidence")
		}
		return nil
	}
	policy, err := rankPolicyForAttemptTx(tx, examID, examVersion, evidence.RankPolicyVersion)
	if err != nil {
		return fmt.Errorf("exam rank policy is unavailable: %w", err)
	}
	if policy.Rank != rank || policy.CriteriaHash != criteriaHash || policy.PolicyHash != evidence.RankPolicyHash {
		return errors.New("exam rank policy snapshot changed")
	}
	for _, exerciseID := range policy.RequiredCourseExerciseIDs {
		if _, exists := evidence.VerifiedProgress[exerciseID]; !exists {
			return fmt.Errorf("%w: course evidence %s", ErrExamPrerequisites, exerciseID)
		}
	}
	for _, requiredRank := range policy.RequiredPriorRanks {
		if _, exists := evidence.RankAwards[requiredRank]; !exists {
			return fmt.Errorf("%w: rank evidence %s", ErrExamPrerequisites, requiredRank)
		}
	}
	if len(evidence.ProjectEvidence) != len(policy.ProjectEvidenceRequirements) {
		return errors.New("exam project evidence set is incomplete")
	}
	var linked int
	if err = tx.QueryRow(`SELECT COUNT(*) FROM exam_project_evidence WHERE attempt_id=?`, attemptID).Scan(&linked); err != nil {
		return err
	}
	if linked != len(policy.ProjectEvidenceRequirements) {
		return errors.New("exam project evidence links are incomplete")
	}
	for _, requirement := range policy.ProjectEvidenceRequirements {
		proof, ok := evidence.ProjectEvidence[requirement.RequirementID]
		if !ok || proof.RequirementID != requirement.RequirementID || proof.ProjectID != requirement.ProjectID ||
			proof.ProjectVersion != requirement.ProjectVersion || proof.CriteriaHash != requirement.CriteriaHash ||
			proof.ArtifactSchema != requirement.ArtifactSchema || proof.CheckerVersion != requirement.CheckerVersion ||
			proof.CheckerDigest != requirement.CheckerDigest || proof.RuntimePolicyHash != requirement.RuntimePolicyHash ||
			proof.RubricVersion != requirement.RubricVersion || proof.RubricHash != requirement.RubricHash ||
			!proof.Passed || proof.EvaluationStatus != "automatic_pass" || proof.ExitCode != 0 || proof.ChecksHash == "" {
			return fmt.Errorf("%w: project evidence %s does not match its policy", ErrExamPrerequisites, requirement.RequirementID)
		}
		var got examProjectEvidenceSnapshot
		var passed int
		var created string
		err = tx.QueryRow(`SELECT requirement_id,submission_id,evaluation_id,project_id,project_version,criteria_hash,
			artifact_schema_version,artifact_hash,executed_source_hash,checker_version,checker_digest,runtime_policy_hash,
			rubric_version,rubric_hash,evaluation_status,exit_code,passed,checks_hash,created_at
			FROM exam_project_evidence WHERE attempt_id=? AND profile_id=? AND exam_id=? AND exam_version=?
			AND policy_version=? AND requirement_id=?`, attemptID, profileID, examID, examVersion,
			policy.PolicyVersion, requirement.RequirementID).Scan(&got.RequirementID, &got.SubmissionID, &got.EvaluationID,
			&got.ProjectID, &got.ProjectVersion, &got.CriteriaHash, &got.ArtifactSchema, &got.ArtifactHash,
			&got.ExecutedSourceHash, &got.CheckerVersion, &got.CheckerDigest, &got.RuntimePolicyHash,
			&got.RubricVersion, &got.RubricHash, &got.EvaluationStatus, &got.ExitCode, &passed, &got.ChecksHash, &created)
		if err != nil {
			return fmt.Errorf("exam project evidence link is missing: %w", err)
		}
		got.Passed = passed == 1
		if !reflect.DeepEqual(got, examProjectEvidenceSnapshot{RequirementID: proof.RequirementID, SubmissionID: proof.SubmissionID,
			EvaluationID: proof.EvaluationID, ProjectID: proof.ProjectID, ProjectVersion: proof.ProjectVersion, CriteriaHash: proof.CriteriaHash,
			ArtifactSchema: proof.ArtifactSchema, ArtifactHash: proof.ArtifactHash, ExecutedSourceHash: proof.ExecutedSourceHash,
			CheckerVersion: proof.CheckerVersion, CheckerDigest: proof.CheckerDigest, RuntimePolicyHash: proof.RuntimePolicyHash,
			RubricVersion: proof.RubricVersion, RubricHash: proof.RubricHash, EvaluationStatus: proof.EvaluationStatus,
			ExitCode: proof.ExitCode, Passed: proof.Passed, ChecksHash: proof.ChecksHash}) {
			return errors.New("exam project evidence snapshot does not match its immutable link")
		}
		var currentProjectVersion, currentCriteria, currentArtifact, currentSource, currentChecks, currentCheckerDigest, currentRuntime, currentRubric string
		var currentStatus, currentRubricVersion, currentCheckerVersion string
		var currentPassed, currentExitCode, currentArtifactSchema int
		err = tx.QueryRow(`SELECT s.project_version,s.criteria_hash,s.artifact_schema_version,s.artifact_hash,s.executed_source_hash,e.checks_hash,e.checker_version,e.checker_digest,
			e.runtime_policy_hash,e.rubric_version,e.rubric_hash,e.evaluation_status,e.exit_code,e.passed
			FROM project_submissions s JOIN project_evaluations e
			ON e.submission_id=s.id AND e.profile_id=s.profile_id AND e.project_id=s.project_id
			WHERE s.id=? AND s.profile_id=? AND s.project_id=? AND e.id=?`,
			proof.SubmissionID, profileID, proof.ProjectID, proof.EvaluationID).Scan(&currentProjectVersion, &currentCriteria,
			&currentArtifactSchema, &currentArtifact, &currentSource,
			&currentChecks, &currentCheckerVersion, &currentCheckerDigest, &currentRuntime, &currentRubricVersion,
			&currentRubric, &currentStatus, &currentExitCode, &currentPassed)
		if err != nil {
			return fmt.Errorf("exam project evidence backing row is missing: %w", err)
		}
		if currentProjectVersion != proof.ProjectVersion || currentCriteria != proof.CriteriaHash || currentArtifactSchema != proof.ArtifactSchema ||
			currentArtifact != proof.ArtifactHash || currentSource != proof.ExecutedSourceHash || currentChecks != proof.ChecksHash ||
			currentCheckerVersion != proof.CheckerVersion || currentCheckerDigest != proof.CheckerDigest ||
			currentRuntime != proof.RuntimePolicyHash || currentRubricVersion != proof.RubricVersion ||
			currentRubric != proof.RubricHash || currentStatus != proof.EvaluationStatus || currentExitCode != proof.ExitCode ||
			(currentPassed == 1) != proof.Passed {
			return errors.New("exam project evidence backing row changed")
		}
		if requirement.MinimumDistinctReviewerKeyFingerprints == 0 {
			if len(proof.ManualReviews) != 0 {
				return errors.New("unexpected manual review snapshot")
			}
		} else if err = validateManualReviewSnapshotTx(tx, profileID, attemptID, examID, examVersion, policy, requirement, proof); err != nil {
			return err
		}
	}
	var linkedReviews int
	if err = tx.QueryRow(`SELECT COUNT(*) FROM exam_project_review_evidence WHERE attempt_id=?`, attemptID).Scan(&linkedReviews); err != nil {
		return err
	}
	expectedReviews := 0
	for _, proof := range evidence.ProjectEvidence {
		expectedReviews += len(proof.ManualReviews)
	}
	if linkedReviews != expectedReviews {
		return errors.New("exam project review evidence links are incomplete")
	}
	return nil
}

func validateManualReviewSnapshotTx(tx *sql.Tx, profileID, attemptID, examID, examVersion string, policy RankPolicy, requirement ProjectEvidenceRequirement, proof examProjectEvidenceSnapshot) error {
	if len(proof.ManualReviews) != requirement.MinimumDistinctReviewerKeyFingerprints {
		return fmt.Errorf("%w: manual review quorum %s is incomplete", ErrExamPrerequisites, requirement.RequirementID)
	}
	seen := map[string]bool{}
	for _, review := range proof.ManualReviews {
		if !projectreview.ValidKeyFingerprint(review.ReviewerKeyFingerprint) || !containsSortedFingerprint(requirement.AllowedReviewerKeyFingerprints, review.ReviewerKeyFingerprint) || seen[review.ReviewerKeyFingerprint] {
			return errors.New("manual review key quorum is not distinct")
		}
		seen[review.ReviewerKeyFingerprint] = true
		var got examProjectReviewSnapshot
		var linkedDigest, linkedFingerprint, gotSubmission, gotEvaluation, gotProject, gotProjectVersion, gotArtifact, gotRubricVersion, gotRubricHash, decision, raw string
		err := tx.QueryRow(`SELECT x.review_id,x.attestation_digest,x.reviewer_key_fingerprint,r.attestation_digest,r.reviewer_key_fingerprint,r.submission_id,r.evaluation_id,r.project_id,r.project_version,r.artifact_hash,r.rubric_version,r.rubric_hash,r.decision,r.attestation_json
			FROM exam_project_review_evidence x JOIN project_reviews r ON r.id=x.review_id
			WHERE x.attempt_id=? AND x.profile_id=? AND x.exam_id=? AND x.exam_version=? AND x.policy_version=? AND x.requirement_id=? AND x.review_id=?`,
			attemptID, profileID, examID, examVersion, policy.PolicyVersion, requirement.RequirementID, review.ReviewID).Scan(
			&got.ReviewID, &got.AttestationDigest, &got.ReviewerKeyFingerprint, &linkedDigest, &linkedFingerprint, &gotSubmission, &gotEvaluation, &gotProject, &gotProjectVersion, &gotArtifact, &gotRubricVersion, &gotRubricHash, &decision, &raw)
		if err != nil {
			return fmt.Errorf("manual review immutable link missing: %w", err)
		}
		if got != review || linkedDigest != got.AttestationDigest || linkedFingerprint != got.ReviewerKeyFingerprint || gotSubmission != proof.SubmissionID || gotEvaluation != proof.EvaluationID || gotProject != proof.ProjectID ||
			gotProjectVersion != proof.ProjectVersion || gotArtifact != strings.TrimPrefix(proof.ArtifactHash, "sha256:") || gotRubricVersion != requirement.ManualReviewRubricVersion ||
			gotRubricHash != requirement.ManualReviewRubricHash || decision != "approve" {
			return errors.New("manual review evidence does not match exact project submission or policy")
		}
		var a projectreview.Attestation
		if err = json.Unmarshal([]byte(raw), &a); err != nil {
			return err
		}
		if err = validateProjectReviewDigest(a, got.AttestationDigest); err != nil {
			return err
		}
		if a.Decision != "approve" || a.SubmissionID != proof.SubmissionID || a.EvaluationID != proof.EvaluationID || a.ProjectID != proof.ProjectID ||
			a.ProjectVersion != proof.ProjectVersion || a.ArtifactHash != strings.TrimPrefix(proof.ArtifactHash, "sha256:") || a.RubricVersion != requirement.ManualReviewRubricVersion || a.RubricHash != requirement.ManualReviewRubricHash {
			return errors.New("manual review attestation binding changed")
		}
		var latestID, latestDecision string
		// project_reviews is append-only; SQLite rowid is its server acceptance sequence.
		err = tx.QueryRow(`SELECT id,decision FROM project_reviews WHERE profile_id=? AND submission_id=? AND evaluation_id=? AND project_id=?
			AND project_version=? AND artifact_hash=? AND rubric_version=? AND rubric_hash=? AND reviewer_key_fingerprint=?
			ORDER BY rowid DESC LIMIT 1`, profileID, proof.SubmissionID, proof.EvaluationID, proof.ProjectID,
			proof.ProjectVersion, strings.TrimPrefix(proof.ArtifactHash, "sha256:"), requirement.ManualReviewRubricVersion,
			requirement.ManualReviewRubricHash, review.ReviewerKeyFingerprint).Scan(&latestID, &latestDecision)
		if err != nil {
			return fmt.Errorf("latest manual review decision missing: %w", err)
		}
		if latestID != review.ReviewID || latestDecision != "approve" {
			return fmt.Errorf("%w: manual review approval was superseded", ErrExamPrerequisites)
		}
	}
	return nil
}

func validateProjectReviewDigest(a projectreview.Attestation, digest string) error {
	canonical, err := json.Marshal(a)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(append([]byte("open-go-academy/project-review-attestation/v1\x00"), canonical...))
	if "sha256:"+hex.EncodeToString(sum[:]) != digest {
		return errors.New("manual review attestation digest mismatch")
	}
	return nil
}
