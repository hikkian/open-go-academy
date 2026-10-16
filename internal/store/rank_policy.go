package store

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"academy/internal/projectreview"
)

const rankPolicyHashDomain = "open-go-academy/rank-policy/v1\x00"

var ErrRankPolicyNotFound = errors.New("rank policy not found")

// ProjectEvidenceRequirement pins a policy requirement to the exact project
// and evaluator identity whose immutable evidence may satisfy it.
type ProjectEvidenceRequirement struct {
	RequirementID                          string   `json:"requirement_id"`
	ProjectID                              string   `json:"project_id"`
	ProjectVersion                         string   `json:"project_version"`
	CriteriaHash                           string   `json:"criteria_hash"`
	ArtifactSchema                         int      `json:"artifact_schema"`
	CheckerVersion                         string   `json:"checker_version"`
	CheckerDigest                          string   `json:"checker_digest"`
	RuntimePolicyHash                      string   `json:"runtime_policy_hash"`
	RubricVersion                          string   `json:"rubric_version"`
	RubricHash                             string   `json:"rubric_hash"`
	RequirePass                            bool     `json:"require_pass"`
	ManualReviewRubricVersion              string   `json:"manual_review_rubric_version,omitempty"`
	ManualReviewRubricHash                 string   `json:"manual_review_rubric_hash,omitempty"`
	ReviewTrustPolicyVersion               string   `json:"review_trust_policy_version,omitempty"`
	ReviewTrustPolicyHash                  string   `json:"review_trust_policy_hash,omitempty"`
	AllowedReviewerKeyFingerprints         []string `json:"allowed_reviewer_key_fingerprints,omitempty"`
	MinimumDistinctReviewerKeyFingerprints int      `json:"minimum_distinct_reviewer_key_fingerprints,omitempty"`
}

// RankPolicy is a server-authored, versioned definition. Slices are ordered
// deliberately; their order is part of the canonical serialized definition
// and therefore part of PolicyHash.
type RankPolicy struct {
	ExamID                      string                       `json:"exam_id"`
	ExamVersion                 string                       `json:"exam_version"`
	PolicyVersion               int                          `json:"policy_version"`
	Rank                        string                       `json:"rank"`
	CriteriaHash                string                       `json:"criteria_hash"`
	RequiredCourseExerciseIDs   []string                     `json:"required_course_exercise_ids"`
	RequiredPriorRanks          []string                     `json:"required_prior_ranks"`
	ProjectEvidenceRequirements []ProjectEvidenceRequirement `json:"project_evidence_requirements"`
	PolicyHash                  string                       `json:"policy_hash"`
	CreatedAt                   time.Time                    `json:"created_at"`
}

type rankPolicyDefinition struct {
	ExamID                      string                       `json:"exam_id"`
	ExamVersion                 string                       `json:"exam_version"`
	PolicyVersion               int                          `json:"policy_version"`
	Rank                        string                       `json:"rank"`
	CriteriaHash                string                       `json:"criteria_hash"`
	RequiredCourseExerciseIDs   []string                     `json:"required_course_exercise_ids"`
	RequiredPriorRanks          []string                     `json:"required_prior_ranks"`
	ProjectEvidenceRequirements []ProjectEvidenceRequirement `json:"project_evidence_requirements"`
}

func rankPolicyID(s string) bool {
	if !validExamText(s) || strings.TrimSpace(s) != s {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' || r == '+') {
			return false
		}
	}
	return true
}

func validateRankPolicy(p RankPolicy) error {
	if !rankPolicyID(p.ExamID) || !rankPolicyID(p.ExamVersion) || p.PolicyVersion < 1 || !validExamText(p.CriteriaHash) {
		return errors.New("invalid rank policy binding or version")
	}
	switch p.Rank {
	case "Middle", "Middle+", "Senior":
	default:
		return errors.New("rank policy must target an advanced rank")
	}
	if len(p.RequiredCourseExerciseIDs) == 0 || len(p.RequiredPriorRanks) == 0 || len(p.ProjectEvidenceRequirements) == 0 {
		return errors.New("advanced rank policy requires course, prior-rank, and project evidence")
	}
	previousRank := map[string]string{"Middle": "Junior+", "Middle+": "Middle", "Senior": "Middle+"}[p.Rank]
	priorRankIncluded := false
	seenExercises := make(map[string]bool, len(p.RequiredCourseExerciseIDs))
	for _, id := range p.RequiredCourseExerciseIDs {
		if !rankPolicyID(id) || seenExercises[id] {
			return fmt.Errorf("invalid or duplicate required course exercise %q", id)
		}
		seenExercises[id] = true
	}
	seenRanks := make(map[string]bool, len(p.RequiredPriorRanks))
	for _, rank := range p.RequiredPriorRanks {
		if !validRank(rank) || rank == p.Rank || seenRanks[rank] {
			return fmt.Errorf("invalid or duplicate required prior rank %q", rank)
		}
		seenRanks[rank] = true
		priorRankIncluded = priorRankIncluded || rank == previousRank
	}
	if !priorRankIncluded {
		return fmt.Errorf("rank policy for %s must require its previous rank %s", p.Rank, previousRank)
	}
	seenRequirements := make(map[string]bool, len(p.ProjectEvidenceRequirements))
	requiredProject, minimumReviewers := map[string]string{"Middle": "p3", "Middle+": "p4", "Senior": "p5"}[p.Rank], map[string]int{"Middle": 1, "Middle+": 1, "Senior": 2}[p.Rank]
	requiredReviewGate := false
	for _, r := range p.ProjectEvidenceRequirements {
		if !rankPolicyID(r.RequirementID) || seenRequirements[r.RequirementID] {
			return fmt.Errorf("invalid or duplicate project requirement %q", r.RequirementID)
		}
		seenRequirements[r.RequirementID] = true
		if !rankPolicyID(r.ProjectID) || !rankPolicyID(r.ProjectVersion) || !validExamText(r.CriteriaHash) ||
			r.ArtifactSchema != ProjectArtifactSchemaVersion || !validExamText(r.CheckerVersion) ||
			!validExamText(r.CheckerDigest) || !validExamText(r.RuntimePolicyHash) ||
			!validExamText(r.RubricVersion) || !validExamText(r.RubricHash) {
			return fmt.Errorf("invalid project evidence requirement %q", r.RequirementID)
		}
		if !r.RequirePass {
			return fmt.Errorf("project evidence requirement %q must require a passing evaluation", r.RequirementID)
		}
		if r.ProjectID == requiredProject && r.MinimumDistinctReviewerKeyFingerprints >= minimumReviewers &&
			r.ManualReviewRubricVersion != "" && r.ManualReviewRubricHash != "" && r.ReviewTrustPolicyVersion != "" && r.ReviewTrustPolicyHash != "" &&
			len(r.AllowedReviewerKeyFingerprints) >= minimumReviewers {
			requiredReviewGate = true
		}
		reviewFields := []string{r.ManualReviewRubricVersion, r.ManualReviewRubricHash, r.ReviewTrustPolicyVersion, r.ReviewTrustPolicyHash}
		reviewConfigured := r.MinimumDistinctReviewerKeyFingerprints != 0
		for _, field := range reviewFields {
			reviewConfigured = reviewConfigured || field != ""
		}
		if reviewConfigured {
			if !validExamText(r.ManualReviewRubricVersion) || !validUnprefixedSHA256(r.ManualReviewRubricHash) ||
				!validExamText(r.ReviewTrustPolicyVersion) || !validSHA256(r.ReviewTrustPolicyHash) ||
				r.MinimumDistinctReviewerKeyFingerprints < 1 || r.MinimumDistinctReviewerKeyFingerprints > 16 ||
				len(r.AllowedReviewerKeyFingerprints) < r.MinimumDistinctReviewerKeyFingerprints || len(r.AllowedReviewerKeyFingerprints) > 256 ||
				r.ReviewTrustPolicyHash != ComputeReviewTrustPolicyHash(r.ReviewTrustPolicyVersion, r.AllowedReviewerKeyFingerprints) {
				return fmt.Errorf("invalid manual review policy for project requirement %q", r.RequirementID)
			}
			previous := ""
			for _, fingerprint := range r.AllowedReviewerKeyFingerprints {
				if !projectreview.ValidKeyFingerprint(fingerprint) || fingerprint <= previous {
					return fmt.Errorf("allowed reviewer fingerprints must be canonical, unique, and sorted for requirement %q", r.RequirementID)
				}
				previous = fingerprint
			}
		} else if len(r.AllowedReviewerKeyFingerprints) != 0 {
			return fmt.Errorf("allowed reviewer fingerprints require manual review policy for %q", r.RequirementID)
		}
	}
	if !requiredReviewGate {
		return fmt.Errorf("rank policy for %s requires project %s with at least %d distinct allowed-key human reviews", p.Rank, requiredProject, minimumReviewers)
	}
	return nil
}

type reviewTrustPolicyDefinition struct {
	Version                        string   `json:"version"`
	AllowedReviewerKeyFingerprints []string `json:"allowed_reviewer_key_fingerprints"`
}

// ComputeReviewTrustPolicyHash binds a version to an exact, pre-sorted set of
// canonical public-key fingerprints. It establishes key bytes, not identity.
func ComputeReviewTrustPolicyHash(version string, fingerprints []string) string {
	body, _ := json.Marshal(reviewTrustPolicyDefinition{Version: version, AllowedReviewerKeyFingerprints: append([]string{}, fingerprints...)})
	sum := sha256.Sum256(append([]byte("open-go-academy/review-trust-policy/v1\x00"), body...))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validUnprefixedSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && strings.ToLower(value) == value
}

func rankPolicyCanonical(p RankPolicy) ([]byte, error) {
	definition := rankPolicyDefinition{
		ExamID: p.ExamID, ExamVersion: p.ExamVersion, PolicyVersion: p.PolicyVersion,
		Rank: p.Rank, CriteriaHash: p.CriteriaHash,
		RequiredCourseExerciseIDs:   append([]string{}, p.RequiredCourseExerciseIDs...),
		RequiredPriorRanks:          append([]string{}, p.RequiredPriorRanks...),
		ProjectEvidenceRequirements: append([]ProjectEvidenceRequirement{}, p.ProjectEvidenceRequirements...),
	}
	return json.Marshal(definition)
}

func rankPolicyHash(canonical []byte) string {
	sum := sha256.Sum256(append([]byte(rankPolicyHashDomain), canonical...))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// RegisterRankPolicy registers an immutable policy bound to one registered
// exam version and its rank/criteria hash. Re-registering identical content is
// idempotent; any content change for that key is rejected.
func (s *Store) RegisterRankPolicy(p RankPolicy) (RankPolicy, error) {
	if err := validateRankPolicy(p); err != nil {
		return RankPolicy{}, err
	}
	var examRank, examHash string
	err := s.db.QueryRow(`SELECT rank,criteria_hash FROM exams WHERE id=? AND version=?`, p.ExamID, p.ExamVersion).Scan(&examRank, &examHash)
	if err != nil {
		return RankPolicy{}, err
	}
	if examRank != p.Rank || examHash != p.CriteriaHash {
		return RankPolicy{}, errors.New("rank policy does not match registered exam")
	}
	canonical, err := rankPolicyCanonical(p)
	if err != nil {
		return RankPolicy{}, err
	}
	hash := rankPolicyHash(canonical)
	_, err = s.db.Exec(`INSERT INTO rank_policies(exam_id,exam_version,policy_version,rank,criteria_hash,policy_hash,definition_json,created_at)
		VALUES(?,?,?,?,?,?,?,?) ON CONFLICT(exam_id,exam_version,policy_version) DO NOTHING`,
		p.ExamID, p.ExamVersion, p.PolicyVersion, p.Rank, p.CriteriaHash, hash, string(canonical), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return RankPolicy{}, err
	}
	registered, err := s.RankPolicy(p.ExamID, p.ExamVersion, p.PolicyVersion)
	if err != nil {
		return RankPolicy{}, err
	}
	if registered.PolicyHash != hash {
		return RankPolicy{}, errors.New("rank policy version is immutable")
	}
	return registered, nil
}

// RankPolicy reads a previously registered immutable policy and verifies its
// stored canonical JSON and domain-separated hash before returning it.
func (s *Store) RankPolicy(examID, examVersion string, policyVersion int) (RankPolicy, error) {
	var rank, criteriaHash, storedHash, raw, createdAt string
	err := s.db.QueryRow(`SELECT rank,criteria_hash,policy_hash,definition_json,created_at FROM rank_policies
		WHERE exam_id=? AND exam_version=? AND policy_version=?`, examID, examVersion, policyVersion).
		Scan(&rank, &criteriaHash, &storedHash, &raw, &createdAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RankPolicy{}, ErrRankPolicyNotFound
		}
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
	if string(canonical) != raw || rankPolicyHash(canonical) != storedHash || definition.Rank != rank || definition.CriteriaHash != criteriaHash {
		return RankPolicy{}, errors.New("stored rank policy integrity check failed")
	}
	created, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return RankPolicy{}, err
	}
	return RankPolicy{
		ExamID: definition.ExamID, ExamVersion: definition.ExamVersion, PolicyVersion: definition.PolicyVersion,
		Rank: definition.Rank, CriteriaHash: definition.CriteriaHash,
		RequiredCourseExerciseIDs:   definition.RequiredCourseExerciseIDs,
		RequiredPriorRanks:          definition.RequiredPriorRanks,
		ProjectEvidenceRequirements: definition.ProjectEvidenceRequirements,
		PolicyHash:                  storedHash, CreatedAt: created,
	}, nil
}

// RankPolicyForExam returns the newest registered policy for this exact exam
// version. Callers still receive an explicit not-found error when no policy is
// registered; this method does not alter exam or award behavior.
func (s *Store) RankPolicyForExam(examID, examVersion string) (RankPolicy, error) {
	var policyVersion int
	err := s.db.QueryRow(`SELECT COALESCE(MAX(policy_version),0) FROM rank_policies WHERE exam_id=? AND exam_version=?`, examID, examVersion).Scan(&policyVersion)
	if err != nil {
		return RankPolicy{}, err
	}
	if policyVersion == 0 {
		return RankPolicy{}, ErrRankPolicyNotFound
	}
	return s.RankPolicy(examID, examVersion, policyVersion)
}
