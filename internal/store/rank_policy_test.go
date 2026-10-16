package store

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func validMiddlePolicy() RankPolicy {
	return RankPolicy{
		ExamID: "middle-final", ExamVersion: "2", PolicyVersion: 1,
		Rank: "Middle", CriteriaHash: "exam-criteria-sha256",
		RequiredCourseExerciseIDs: []string{"middle-01", "middle-02"},
		RequiredPriorRanks:        []string{"Junior+"},
		ProjectEvidenceRequirements: []ProjectEvidenceRequirement{{
			RequirementID: "service-project", ProjectID: "service", ProjectVersion: "3",
			CriteriaHash: "project-criteria-sha256", ArtifactSchema: ProjectArtifactSchemaVersion,
			CheckerVersion: "checker-4", CheckerDigest: "sha256:checker", RuntimePolicyHash: "sha256:runtime",
			RubricVersion: "rubric-2", RubricHash: "sha256:rubric", RequirePass: true,
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
