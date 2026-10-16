package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"time"

	"academy/internal/assessment"
	"academy/internal/projectrunner"
	"academy/internal/projects"
	"academy/internal/store"
)

// trustedProjectProfileReader is implemented by the project-runner client.
// Requiring its authenticated profile endpoint ensures policies are pinned to
// the exact evaluator that can produce evidence in this installation.
type trustedProjectProfileReader interface {
	ProjectProfile(context.Context, string) (projectrunner.EvaluationProfile, bool)
}

type advancedRankProject struct {
	rank      string
	projectID string
	quorum    int
}

var advancedRankProjects = []advancedRankProject{
	{rank: "Middle", projectID: "p3", quorum: 1},
	{rank: "Middle+", projectID: "p4", quorum: 1},
	{rank: "Senior", projectID: "p5", quorum: 2},
}

// registerAdvancedRankPolicies leaves ranks unavailable unless both the
// operator's reviewer allowlist and the exact trusted runner profiles exist.
// Existing policy versions are immutable: an identical current definition is
// reused, while any evaluator or reviewer-key change gets a new version.
func (a *API) registerAdvancedRankPolicies(ctx context.Context) error {
	if a == nil || a.Store == nil || a.ProjectReviewers == nil || a.ProjectRunner == nil {
		return nil
	}
	profileReader, ok := a.ProjectRunner.(trustedProjectProfileReader)
	if !ok {
		return nil
	}
	fingerprints := a.ProjectReviewers.KeyFingerprints()
	if len(fingerprints) == 0 {
		return nil
	}
	for _, definition := range advancedRankProjects {
		if len(fingerprints) < definition.quorum {
			continue
		}
		exam, ok := advancedExamForRank(definition.rank)
		if !ok {
			continue
		}
		project, ok := projects.Find(definition.projectID)
		if !ok {
			continue
		}
		criteriaHash, ok := projects.CriteriaHash(definition.projectID)
		if !ok {
			continue
		}
		profile, ok := profileReader.ProjectProfile(ctx, definition.projectID)
		if !ok || profile.ProjectVersion != project.Version || profile.CriteriaHash != criteriaHash {
			continue
		}
		rubric := buildManualReviewRubric(project)
		requirement := store.ProjectEvidenceRequirement{
			RequirementID: "project-" + definition.projectID,
			ProjectID:     project.ID, ProjectVersion: project.Version,
			CriteriaHash: criteriaHash, ArtifactSchema: store.ProjectArtifactSchemaVersion,
			CheckerVersion: profile.CheckerVersion, CheckerDigest: profile.CheckerDigest,
			RuntimePolicyHash: profile.RuntimePolicyHash,
			RubricVersion:     profile.RubricVersion, RubricHash: profile.RubricHash,
			RequirePass:               true,
			ManualReviewRubricVersion: rubric.Version, ManualReviewRubricHash: rubric.Hash,
			ReviewTrustPolicyVersion:               "review-registry-v1",
			AllowedReviewerKeyFingerprints:         append([]string(nil), fingerprints...),
			MinimumDistinctReviewerKeyFingerprints: definition.quorum,
		}
		requirement.ReviewTrustPolicyHash = store.ComputeReviewTrustPolicyHash(requirement.ReviewTrustPolicyVersion, requirement.AllowedReviewerKeyFingerprints)
		policy := store.RankPolicy{
			ExamID: exam.ID, ExamVersion: exam.Version, PolicyVersion: 1,
			Rank: exam.Rank, CriteriaHash: exam.Hash(),
			RequiredCourseExerciseIDs:   append([]string(nil), exam.Prerequisites...),
			RequiredPriorRanks:          []string{exam.RequiredRank},
			ProjectEvidenceRequirements: []store.ProjectEvidenceRequirement{requirement},
		}
		if err := a.registerImmutableRankPolicy(policy); err != nil {
			return err
		}
	}
	return nil
}

func advancedExamForRank(rank string) (*assessment.Exam, bool) {
	for _, exam := range assessment.All() {
		if exam.Rank == rank && advancedRank(rank) {
			return exam, true
		}
	}
	return nil, false
}

func (a *API) registerImmutableRankPolicy(candidate store.RankPolicy) error {
	latest, err := a.Store.RankPolicyForExam(candidate.ExamID, candidate.ExamVersion)
	switch {
	case errors.Is(err, store.ErrRankPolicyNotFound):
		candidate.PolicyVersion = 1
	case err != nil:
		return err
	default:
		candidate.PolicyVersion = latest.PolicyVersion
		if sameRankPolicyDefinition(candidate, latest) {
			return nil
		}
		candidate.PolicyVersion++
	}
	_, err = a.Store.RegisterRankPolicy(candidate)
	return err
}

// currentReviewTrustMatches compares the live operator registry with the set
// pinned into the newest immutable policy. A stale policy remains readable for
// existing attempts, but cannot admit a new attempt under changed trust keys.
func (a *API) currentReviewTrustMatches(examID, examVersion string) bool {
	if a == nil || a.Store == nil || a.ProjectReviewers == nil {
		return false
	}
	policy, err := a.Store.RankPolicyForExam(examID, examVersion)
	if err != nil || len(policy.ProjectEvidenceRequirements) != 1 {
		return false
	}
	return a.currentEvidenceRequirementMatches(policy.ProjectEvidenceRequirements[0])
}

func (a *API) attemptReviewTrustMatches(attempt store.ExamAttempt) bool {
	if a == nil || a.Store == nil || a.ProjectReviewers == nil {
		return false
	}
	var snapshot struct {
		RankPolicyVersion int    `json:"rank_policy_version"`
		RankPolicyHash    string `json:"rank_policy_hash"`
	}
	if json.Unmarshal([]byte(attempt.EvidenceJSON), &snapshot) != nil || snapshot.RankPolicyVersion < 1 || snapshot.RankPolicyHash == "" {
		return false
	}
	policy, err := a.Store.RankPolicy(attempt.ExamID, attempt.Version, snapshot.RankPolicyVersion)
	if err != nil || policy.PolicyHash != snapshot.RankPolicyHash || len(policy.ProjectEvidenceRequirements) != 1 {
		return false
	}
	latest, err := a.Store.RankPolicyForExam(attempt.ExamID, attempt.Version)
	if err != nil || latest.PolicyVersion != policy.PolicyVersion || latest.PolicyHash != policy.PolicyHash {
		return false
	}
	return a.currentEvidenceRequirementMatches(policy.ProjectEvidenceRequirements[0])
}

func (a *API) currentEvidenceRequirementMatches(requirement store.ProjectEvidenceRequirement) bool {
	if a == nil || a.ProjectReviewers == nil || a.ProjectRunner == nil ||
		!reflect.DeepEqual(requirement.AllowedReviewerKeyFingerprints, a.ProjectReviewers.KeyFingerprints()) {
		return false
	}
	profileReader, ok := a.ProjectRunner.(trustedProjectProfileReader)
	if !ok {
		return false
	}
	profile, ok := profileReader.ProjectProfile(context.Background(), requirement.ProjectID)
	if !ok {
		return false
	}
	project, ok := projects.Find(requirement.ProjectID)
	criteriaHash, criteriaOK := projects.CriteriaHash(requirement.ProjectID)
	manualRubric := buildManualReviewRubric(project)
	return ok && criteriaOK && profile.ProjectVersion == project.Version && profile.CriteriaHash == criteriaHash &&
		requirement.ProjectVersion == project.Version && requirement.CriteriaHash == criteriaHash &&
		profile.CheckerVersion == requirement.CheckerVersion && profile.CheckerDigest == requirement.CheckerDigest &&
		profile.RuntimePolicyHash == requirement.RuntimePolicyHash && profile.RubricVersion == requirement.RubricVersion &&
		profile.RubricHash == requirement.RubricHash && manualRubric.Version == requirement.ManualReviewRubricVersion &&
		manualRubric.Hash == requirement.ManualReviewRubricHash
}

func sameRankPolicyDefinition(left, right store.RankPolicy) bool {
	left.PolicyHash, right.PolicyHash = "", ""
	// Stored timestamps and hashes are metadata, not policy content.
	left.CreatedAt, right.CreatedAt = time.Time{}, time.Time{}
	return reflect.DeepEqual(left, right)
}
