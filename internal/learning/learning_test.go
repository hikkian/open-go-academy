package learning

import (
	"academy/internal/store"
	"testing"
)

func TestSkillsRequireVerifiedEvidenceAndNeverRankFromXP(t *testing.T) {
	s := Build(store.Progress{XP: 999999, Attempts: []store.Attempt{{ID: 1, ExerciseID: "hello", Passed: false}}})
	if s.CurrentRank != "Zero" || s.Verified {
		t.Fatal("XP assigned rank")
	}
	for _, skill := range s.Skills {
		if skill.Level != 0 {
			t.Fatal("unverified skill", skill)
		}
	}
	if len(s.ReviewExercises) != 1 {
		t.Fatal("failed exercise missing from review")
	}
	s = Build(store.Progress{Completed: []string{"hello", "repair", "two-lines"}, XP: 30, Evidence: map[string]int64{"hello": 2, "repair": 3, "two-lines": 4}, Attempts: []store.Attempt{{ID: 1, ExerciseID: "hello", Passed: false}, {ID: 2, ExerciseID: "hello", Passed: true}}})
	if s.CurrentRank != "Zero" || s.Verified {
		t.Fatal("lesson promoted rank")
	}
	for _, skill := range s.Skills {
		want := 0
		if skill.Rank == "Zero" {
			want = 2
		}
		if skill.Level != want {
			t.Fatal(skill)
		}
	}
	if len(s.Ranks) != 6 || len(s.ReviewExercises) != 0 {
		t.Fatal(s)
	}
	for i, r := range s.Ranks {
		if i > 0 && (r.AssessmentAvailable || r.ContentAvailable) {
			t.Fatal("unpublished rank advertised ready", r)
		}
	}
}

func TestClaimedCompletionWithoutPersistedEvidenceIsNotMastery(t *testing.T) {
	s := Build(store.Progress{Completed: []string{"hello"}, XP: 1000})
	if s.Skills[0].Level != 0 {
		t.Fatal("client-like claimed completion accepted")
	}
}

func TestOnlyPersistedAwardVerifiesRank(t *testing.T) {
	base := Build(store.Progress{XP: 999999})
	if WithAwards(base, nil).Verified {
		t.Fatal("XP verified rank")
	}
	got := WithAwards(base, []store.RankAward{{Rank: "Zero", ExamID: "zero-core", Version: "1.0.0"}})
	if !got.Verified || got.CurrentRank != "Zero" {
		t.Fatal("valid persisted Zeroaward ignored")
	}
}
