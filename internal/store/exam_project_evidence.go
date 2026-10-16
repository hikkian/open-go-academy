package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// examProjectEvidenceSnapshot is copied into the exam attempt and into a
// relational FK-backed table. The JSON copy is convenient to inspect; the
// table makes the evidence durable and links it to exact immutable rows.
type examProjectEvidenceSnapshot struct {
	RequirementID      string `json:"requirement_id"`
	SubmissionID       string `json:"submission_id"`
	EvaluationID       string `json:"evaluation_id"`
	ProjectID          string `json:"project_id"`
	ProjectVersion     string `json:"project_version"`
	CriteriaHash       string `json:"criteria_hash"`
	ArtifactSchema     int    `json:"artifact_schema_version"`
	ArtifactHash       string `json:"artifact_hash"`
	ExecutedSourceHash string `json:"executed_source_hash"`
	CheckerVersion     string `json:"checker_version"`
	CheckerDigest      string `json:"checker_digest"`
	RuntimePolicyHash  string `json:"runtime_policy_hash"`
	RubricVersion      string `json:"rubric_version"`
	RubricHash         string `json:"rubric_hash"`
	EvaluationStatus   string `json:"evaluation_status"`
	ExitCode           int    `json:"exit_code"`
	Passed             bool   `json:"passed"`
	ChecksHash         string `json:"checks_hash"`
}

func resolveProjectEvidenceTx(tx *sql.Tx, profileID string, policy RankPolicy) (map[string]examProjectEvidenceSnapshot, error) {
	resolved := make(map[string]examProjectEvidenceSnapshot, len(policy.ProjectEvidenceRequirements))
	for _, requirement := range policy.ProjectEvidenceRequirements {
		var evidence examProjectEvidenceSnapshot
		var passed int
		err := tx.QueryRow(`SELECT s.id,e.id,s.project_id,s.project_version,s.criteria_hash,s.artifact_schema_version,
			s.artifact_hash,s.executed_source_hash,e.checker_version,e.checker_digest,e.runtime_policy_hash,
			e.rubric_version,e.rubric_hash,e.evaluation_status,e.exit_code,e.passed,e.checks_hash
			FROM project_submissions s JOIN project_evaluations e
			ON e.submission_id=s.id AND e.profile_id=s.profile_id AND e.project_id=s.project_id
			WHERE s.profile_id=? AND s.project_id=? AND s.project_version=? AND s.criteria_hash=? AND s.artifact_schema_version=?
			AND e.checker_version=? AND e.checker_digest=? AND e.runtime_policy_hash=?
			AND e.rubric_version=? AND e.rubric_hash=?
			AND e.evaluation_status='automatic_pass' AND e.exit_code=0 AND e.passed=1
			ORDER BY e.created_at DESC,e.id DESC LIMIT 1`,
			profileID, requirement.ProjectID, requirement.ProjectVersion, requirement.CriteriaHash, requirement.ArtifactSchema,
			requirement.CheckerVersion, requirement.CheckerDigest, requirement.RuntimePolicyHash,
			requirement.RubricVersion, requirement.RubricHash).Scan(
			&evidence.SubmissionID, &evidence.EvaluationID, &evidence.ProjectID, &evidence.ProjectVersion,
			&evidence.CriteriaHash, &evidence.ArtifactSchema, &evidence.ArtifactHash, &evidence.ExecutedSourceHash,
			&evidence.CheckerVersion, &evidence.CheckerDigest, &evidence.RuntimePolicyHash, &evidence.RubricVersion,
			&evidence.RubricHash, &evidence.EvaluationStatus, &evidence.ExitCode, &passed, &evidence.ChecksHash)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: project evidence %s", ErrExamPrerequisites, requirement.RequirementID)
		}
		if err != nil {
			return nil, err
		}
		evidence.RequirementID, evidence.Passed = requirement.RequirementID, passed == 1
		if !evidence.Passed || evidence.EvaluationStatus != "automatic_pass" || evidence.ExitCode != 0 || evidence.ChecksHash == "" {
			return nil, fmt.Errorf("%w: project evidence %s did not pass", ErrExamPrerequisites, requirement.RequirementID)
		}
		resolved[requirement.RequirementID] = evidence
	}
	return resolved, nil
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
		if got != proof {
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
	}
	return nil
}
