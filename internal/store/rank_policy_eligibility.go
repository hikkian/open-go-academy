package store

import "fmt"

type RankEvidenceRequirementStatus struct {
	RequirementID string `json:"requirement_id"`
	Kind          string `json:"kind"`
	Status        string `json:"status"`
	Reason        string `json:"reason,omitempty"`
}

type RankPolicyEligibilityStatus struct {
	ExamID        string                          `json:"exam_id"`
	ExamVersion   string                          `json:"exam_version"`
	Rank          string                          `json:"rank"`
	PolicyVersion int                             `json:"policy_version"`
	PolicyHash    string                          `json:"policy_hash"`
	Eligible      bool                            `json:"eligible"`
	Requirements  []RankEvidenceRequirementStatus `json:"requirements"`
}

// RankPolicyEligibility reports server-derived evidence state for display.
// StartExam repeats every check transactionally; this response is advisory UI.
func (s *Store) RankPolicyEligibility(profileID, examID, examVersion string) (RankPolicyEligibilityStatus, error) {
	if profileID != "local" {
		return RankPolicyEligibilityStatus{}, ErrProfileNotFound
	}
	policy, err := s.RankPolicyForExam(examID, examVersion)
	if err != nil {
		return RankPolicyEligibilityStatus{}, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return RankPolicyEligibilityStatus{}, err
	}
	defer tx.Rollback()
	result := RankPolicyEligibilityStatus{ExamID: examID, ExamVersion: examVersion, Rank: policy.Rank,
		PolicyVersion: policy.PolicyVersion, PolicyHash: policy.PolicyHash, Eligible: true,
		Requirements: make([]RankEvidenceRequirementStatus, 0, len(policy.RequiredCourseExerciseIDs)+len(policy.RequiredPriorRanks)+len(policy.ProjectEvidenceRequirements))}
	add := func(id, kind string, found bool, reason string) {
		status := "verified"
		if !found {
			status, result.Eligible = "missing", false
		}
		result.Requirements = append(result.Requirements, RankEvidenceRequirementStatus{RequirementID: id, Kind: kind, Status: status, Reason: reason})
	}
	for _, exerciseID := range policy.RequiredCourseExerciseIDs {
		var count int
		if err = tx.QueryRow(`SELECT COUNT(*) FROM verified_progress WHERE exercise_id=?`, exerciseID).Scan(&count); err != nil {
			return RankPolicyEligibilityStatus{}, err
		}
		reason := ""
		if count == 0 {
			reason = "course_evidence_missing"
		}
		add(exerciseID, "course", count > 0, reason)
	}
	for _, requiredRank := range policy.RequiredPriorRanks {
		var count int
		if err = tx.QueryRow(`SELECT COUNT(*) FROM rank_awards WHERE profile_id=? AND rank=?`, profileID, requiredRank).Scan(&count); err != nil {
			return RankPolicyEligibilityStatus{}, err
		}
		reason := ""
		if count == 0 {
			reason = "prior_rank_missing"
		}
		add("rank:"+requiredRank, "prior_rank", count > 0, reason)
	}
	for _, requirement := range policy.ProjectEvidenceRequirements {
		var count int
		err = tx.QueryRow(`SELECT COUNT(*) FROM project_submissions s JOIN project_evaluations e
			ON e.submission_id=s.id AND e.profile_id=s.profile_id AND e.project_id=s.project_id
			WHERE s.profile_id=? AND s.project_id=? AND s.project_version=? AND s.criteria_hash=? AND s.artifact_schema_version=?
			AND e.checker_version=? AND e.checker_digest=? AND e.runtime_policy_hash=?
			AND e.rubric_version=? AND e.rubric_hash=?
			AND e.evaluation_status='automatic_pass' AND e.exit_code=0 AND e.passed=1`,
			profileID, requirement.ProjectID, requirement.ProjectVersion, requirement.CriteriaHash, requirement.ArtifactSchema,
			requirement.CheckerVersion, requirement.CheckerDigest, requirement.RuntimePolicyHash,
			requirement.RubricVersion, requirement.RubricHash).Scan(&count)
		if err != nil {
			return RankPolicyEligibilityStatus{}, err
		}
		reason := ""
		if count == 0 {
			reason = fmt.Sprintf("project_evidence_missing:%s", requirement.ProjectID)
		}
		add("project:"+requirement.RequirementID, "project", count > 0, reason)
	}
	if err = tx.Commit(); err != nil {
		return RankPolicyEligibilityStatus{}, err
	}
	return result, nil
}
