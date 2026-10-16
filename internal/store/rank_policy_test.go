package store

import (
	"errors"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func validMiddlePolicy() RankPolicy {
	fingerprints := []string{"sha256:" + strings.Repeat("1", 64)}
	return validPolicyWithReviewGate("middle-final", "2", "Middle", "exam-criteria-sha256", "service-project", "p3", fingerprints, 1)
}

func validPolicyWithReviewGate(examID, version, rank, criteria, requirementID, projectID string, fingerprints []string, quorum int) RankPolicy {
	fingerprints = append([]string(nil), fingerprints...)
	sort.Strings(fingerprints)
	reviewTrustVersion := "review-trust-v1"
	previousRank := map[string]string{"Middle": "Junior+", "Middle+": "Middle", "Senior": "Middle+"}[rank]
	return RankPolicy{
		ExamID: examID, ExamVersion: version, PolicyVersion: 1,
		Rank: rank, CriteriaHash: criteria,
		RequiredCourseExerciseIDs: []string{"middle-01", "middle-02"},
		RequiredPriorRanks:        []string{previousRank},
		ProjectEvidenceRequirements: []ProjectEvidenceRequirement{{
			RequirementID: requirementID, ProjectID: projectID, ProjectVersion: "3",
			CriteriaHash: "project-criteria-sha256", ArtifactSchema: ProjectArtifactSchemaVersion,
			CheckerVersion: "checker-4", CheckerDigest: "sha256:checker", RuntimePolicyHash: "sha256:runtime",
			RubricVersion: "rubric-2", RubricHash: "sha256:rubric", RequirePass: true,
			ManualReviewRubricVersion: "human-v1", ManualReviewRubricHash: strings.Repeat("a", 64),
			ReviewTrustPolicyVersion: reviewTrustVersion, ReviewTrustPolicyHash: ComputeReviewTrustPolicyHash(reviewTrustVersion, fingerprints),
			AllowedReviewerKeyFingerprints: fingerprints, MinimumDistinctReviewerKeyFingerprints: quorum,
		}},
	}
}

func TestRankPolicyImmutableRegistrationReplayAndRead(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "rank-policy.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.RegisterExam("middle-final", "2", "Middle", "exam-criteria-sha256"); err != nil {
		t.Fatal(err)
	}
	want := validMiddlePolicy()
	first, err := s.RegisterRankPolicy(want)
	if err != nil {
		t.Fatal(err)
	}
	if first.PolicyHash == "" || first.PolicyHash[:7] != "sha256:" {
		t.Fatalf("invalid policy hash %q", first.PolicyHash)
	}
	second, err := s.RegisterRankPolicy(want)
	if err != nil {
		t.Fatalf("identical replay: %v", err)
	}
	if first.PolicyHash != second.PolicyHash || !reflect.DeepEqual(first.RequiredCourseExerciseIDs, second.RequiredCourseExerciseIDs) {
		t.Fatalf("replay changed policy: first=%+v second=%+v", first, second)
	}
	read, err := s.RankPolicyForExam("middle-final", "2")
	if err != nil || read.PolicyHash != first.PolicyHash || !reflect.DeepEqual(read.ProjectEvidenceRequirements, first.ProjectEvidenceRequirements) {
		t.Fatalf("read=%+v err=%v", read, err)
	}
}

func TestRankPolicyPersistsAfterReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "persist.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.RegisterExam("middle-final", "2", "Middle", "exam-criteria-sha256"); err != nil {
		t.Fatal(err)
	}
	want, err := s.RegisterRankPolicy(validMiddlePolicy())
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	got, err := s.RankPolicy("middle-final", "2", 1)
	if err != nil || got.PolicyHash != want.PolicyHash || !reflect.DeepEqual(got.RequiredCourseExerciseIDs, want.RequiredCourseExerciseIDs) {
		t.Fatalf("reopened policy=%+v err=%v", got, err)
	}
}

func TestManualReviewTrustBindingIsPartOfImmutablePolicyHash(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "rank-policy-review-hash.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.RegisterExam("middle-final", "2", "Middle", "exam-criteria-sha256"); err != nil {
		t.Fatal(err)
	}
	policy := validMiddlePolicy()
	r := &policy.ProjectEvidenceRequirements[0]
	r.ManualReviewRubricVersion = "review-v1"
	r.ManualReviewRubricHash = strings.Repeat("f", 64)
	r.ReviewTrustPolicyVersion = "registry-v1"
	r.AllowedReviewerKeyFingerprints = []string{"sha256:" + strings.Repeat("1", 64), "sha256:" + strings.Repeat("2", 64)}
	r.ReviewTrustPolicyHash = ComputeReviewTrustPolicyHash(r.ReviewTrustPolicyVersion, r.AllowedReviewerKeyFingerprints)
	r.MinimumDistinctReviewerKeyFingerprints = 2
	first, err := s.RegisterRankPolicy(policy)
	if err != nil {
		t.Fatal(err)
	}
	changed := policy
	changed.ProjectEvidenceRequirements = append([]ProjectEvidenceRequirement(nil), policy.ProjectEvidenceRequirements...)
	changed.ProjectEvidenceRequirements[0].AllowedReviewerKeyFingerprints = []string{"sha256:" + strings.Repeat("1", 64), "sha256:" + strings.Repeat("3", 64)}
	changed.ProjectEvidenceRequirements[0].ReviewTrustPolicyHash = ComputeReviewTrustPolicyHash("registry-v1", changed.ProjectEvidenceRequirements[0].AllowedReviewerKeyFingerprints)
	if _, err = s.RegisterRankPolicy(changed); err == nil {
		t.Fatal("changed review trust policy reused immutable policy version")
	}
	changed.ProjectEvidenceRequirements[0] = policy.ProjectEvidenceRequirements[0]
	changed.ProjectEvidenceRequirements[0].AllowedReviewerKeyFingerprints = append([]string(nil), policy.ProjectEvidenceRequirements[0].AllowedReviewerKeyFingerprints...)
	changed.ProjectEvidenceRequirements[0].AllowedReviewerKeyFingerprints = append(changed.ProjectEvidenceRequirements[0].AllowedReviewerKeyFingerprints, "sha256:"+strings.Repeat("3", 64))
	changed.ProjectEvidenceRequirements[0].MinimumDistinctReviewerKeyFingerprints = 3
	changed.ProjectEvidenceRequirements[0].ReviewTrustPolicyHash = ComputeReviewTrustPolicyHash("registry-v1", changed.ProjectEvidenceRequirements[0].AllowedReviewerKeyFingerprints)
	if _, err = s.RegisterRankPolicy(changed); err == nil {
		t.Fatal("changed reviewer quorum reused immutable policy version")
	}
	if first.PolicyHash == "" {
		t.Fatal("empty policy hash")
	}
	invalid := policy
	invalid.ProjectEvidenceRequirements = append([]ProjectEvidenceRequirement(nil), policy.ProjectEvidenceRequirements...)
	invalid.ProjectEvidenceRequirements[0].ReviewTrustPolicyHash = ""
	if _, err = s.RegisterRankPolicy(invalid); err == nil {
		t.Fatal("partial manual review binding accepted")
	}
}

func TestRankPolicyRejectsConflictingRegistration(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "conflict.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.RegisterExam("middle-final", "2", "Middle", "exam-criteria-sha256"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RegisterRankPolicy(validMiddlePolicy()); err != nil {
		t.Fatal(err)
	}
	conflict := validMiddlePolicy()
	conflict.RequiredCourseExerciseIDs = []string{"middle-01", "middle-03"}
	if _, err = s.RegisterRankPolicy(conflict); err == nil {
		t.Fatal("conflicting definition replaced immutable policy")
	}
}

func TestRankPolicyRejectsBadOrMismatchedExamBinding(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "binding.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.RegisterRankPolicy(validMiddlePolicy()); err == nil {
		t.Fatal("unregistered exam accepted")
	}
	if err = s.RegisterExam("middle-final", "2", "Middle+", "exam-criteria-sha256"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RegisterRankPolicy(validMiddlePolicy()); err == nil {
		t.Fatal("rank mismatch accepted")
	}
	if err = s.RegisterExam("middle-final", "3", "Middle", "different-criteria"); err != nil {
		t.Fatal(err)
	}
	p := validMiddlePolicy()
	p.ExamVersion = "3"
	if _, err = s.RegisterRankPolicy(p); err == nil {
		t.Fatal("criteria mismatch accepted")
	}
	if _, err = s.RankPolicy("absent", "1", 1); !errors.Is(err, ErrRankPolicyNotFound) {
		t.Fatalf("not-found error=%v", err)
	}
}

func TestRankPolicyRejectsInvalidAndDuplicateRequirements(t *testing.T) {
	cases := map[string]func(*RankPolicy){
		"unsupported rank":     func(p *RankPolicy) { p.Rank = "Junior" },
		"duplicate exercise":   func(p *RankPolicy) { p.RequiredCourseExerciseIDs = []string{"same", "same"} },
		"invalid exercise id":  func(p *RankPolicy) { p.RequiredCourseExerciseIDs = []string{"bad id"} },
		"duplicate prior rank": func(p *RankPolicy) { p.RequiredPriorRanks = []string{"Junior", "Junior"} },
		"invalid prior rank":   func(p *RankPolicy) { p.RequiredPriorRanks = []string{"Apprentice"} },
		"duplicate requirement id": func(p *RankPolicy) {
			p.ProjectEvidenceRequirements = append(p.ProjectEvidenceRequirements, p.ProjectEvidenceRequirements[0])
		},
		"unsupported artifact schema": func(p *RankPolicy) { p.ProjectEvidenceRequirements[0].ArtifactSchema = 99 },
		"invalid project version":     func(p *RankPolicy) { p.ProjectEvidenceRequirements[0].ProjectVersion = " " },
		"nonpassing project requirement": func(p *RankPolicy) {
			p.ProjectEvidenceRequirements[0].RequirePass = false
		},
		"missing project evidence": func(p *RankPolicy) {
			p.ProjectEvidenceRequirements = nil
		},
		"missing previous rank": func(p *RankPolicy) {
			p.RequiredPriorRanks = []string{"Zero"}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			p := validMiddlePolicy()
			mutate(&p)
			if err := validateRankPolicy(p); err == nil {
				t.Fatal("invalid policy accepted")
			}
		})
	}
}

func TestAdvancedRankPoliciesRequireTheirRankSpecificHumanReviewGate(t *testing.T) {
	cases := []struct {
		rank         string
		project      string
		minimum      int
		fingerprints []string
	}{
		{"Middle", "p3", 1, []string{"sha256:" + strings.Repeat("1", 64)}},
		{"Middle+", "p4", 1, []string{"sha256:" + strings.Repeat("2", 64)}},
		{"Senior", "p5", 2, []string{"sha256:" + strings.Repeat("3", 64), "sha256:" + strings.Repeat("4", 64)}},
	}
	for _, tc := range cases {
		t.Run(tc.rank+" accepts exact gate", func(t *testing.T) {
			p := validPolicyWithReviewGate("exam", "1", tc.rank, "criteria", "required-project", tc.project, tc.fingerprints, tc.minimum)
			if err := validateRankPolicy(p); err != nil {
				t.Fatal(err)
			}
		})
		t.Run(tc.rank+" rejects wrong project or quorum", func(t *testing.T) {
			p := validPolicyWithReviewGate("exam", "1", tc.rank, "criteria", "required-project", tc.project, tc.fingerprints, tc.minimum)
			p.ProjectEvidenceRequirements[0].ProjectID = map[string]string{"p3": "p4", "p4": "p3", "p5": "p4"}[tc.project]
			if err := validateRankPolicy(p); err == nil {
				t.Fatal("wrong rank project passed validation")
			}
			p = validPolicyWithReviewGate("exam", "1", tc.rank, "criteria", "required-project", tc.project, tc.fingerprints, tc.minimum)
			p.ProjectEvidenceRequirements[0].MinimumDistinctReviewerKeyFingerprints = tc.minimum - 1
			if err := validateRankPolicy(p); err == nil {
				t.Fatal("insufficient reviewer quorum passed validation")
			}
		})
	}
}
