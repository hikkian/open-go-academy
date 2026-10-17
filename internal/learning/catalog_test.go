package learning

import (
	"academy/internal/content"
	"academy/internal/store"
	"testing"
)

func TestCatalogSkillsRequireBothCompletionAndEvidence(t *testing.T) {
	c, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	var exercise content.Exercise
	for _, lesson := range c.AllLessons() {
		for _, ex := range lesson.Exercises {
			if ex.Kind == "function" && len(ex.Skills) > 0 {
				exercise = ex
				break
			}
		}
		if exercise.ID != "" {
			break
		}
	}
	if exercise.ID == "" {
		t.Fatal("missing published function fixture")
	}
	for _, complete := range []bool{false, true} {
		p := store.Progress{Evidence: map[string]int64{exercise.ID: 42}}
		if complete {
			p.Completed = []string{exercise.ID}
		}
		s := WithCatalog(Build(p), p, c)
		found := false
		for _, skill := range s.Skills {
			for _, id := range skill.Evidence {
				if id == exercise.ID {
					found = true
				}
			}
		}
		if found != complete {
			t.Fatalf("completion=%v, evidence added=%v", complete, found)
		}
		if s.Verified {
			t.Fatal("catalog evidence assigned rank")
		}
	}
}

func TestPublishedAssessmentsAndVerifiedProjectSkillEvidence(t *testing.T) {
	c, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, verified := range []bool{false, true} {
		p := store.Progress{Completed: []string{"p2:automatic"}}
		if verified {
			p.Evidence = map[string]int64{"p2:automatic": 42}
		}
		s := WithCatalog(Build(p), p, c)
		for _, skill := range s.Skills {
			if skill.ID == "sql" || skill.ID == "http" {
				want := 0
				if verified {
					want = 2
				}
				if skill.Level != want {
					t.Fatal("project skill lacks both completion and evidence", skill)
				}
			}
		}
		for _, rank := range s.Ranks {
			if !rank.AssessmentAvailable {
				t.Fatal("assessment availability differs from published catalog", rank)
			}
		}
		if s.Verified {
			t.Fatal("project evidence assigned rank")
		}
	}
}

func TestP3ProjectEvidenceUpdatesMiddleSkillsButNeverAwardsRank(t *testing.T) {
	c, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, verified := range []bool{false, true} {
		p := store.Progress{Completed: []string{"p3:automatic"}}
		if verified {
			p.Evidence = map[string]int64{"p3:automatic": 99}
		}
		snapshot := WithCatalog(Build(p), p, c)
		for _, skillID := range []string{"concurrency", "reliability"} {
			var found *Skill
			for i := range snapshot.Skills {
				if snapshot.Skills[i].ID == skillID {
					found = &snapshot.Skills[i]
					break
				}
			}
			if found == nil {
				t.Fatalf("missing Middle skill %q", skillID)
			}
			want := 0
			if verified {
				want = 2
			}
			if found.Level != want {
				t.Fatalf("verified=%v skill=%+v want level %d", verified, found, want)
			}
		}
		if snapshot.Verified || snapshot.CurrentRank != "Zero" {
			t.Fatalf("project evidence assigned rank: %+v", snapshot)
		}
		for _, rank := range snapshot.Ranks {
			if rank.Name == "Middle" && !rank.AssessmentAvailable {
				t.Fatal("published Middle objective exam is missing from rank catalog")
			}
		}
	}
}

func TestP4P5AutomaticProjectEvidenceUpdatesSkillsButNeverAwardsRank(t *testing.T) {
	c, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	for projectID, skills := range map[string][]string{
		"p4:automatic": {"performance", "distributed"},
		"p5:automatic": {"architecture"},
	} {
		p := store.Progress{Completed: []string{projectID}, Evidence: map[string]int64{projectID: 101}}
		snapshot := WithCatalog(Build(p), p, c)
		for _, skillID := range skills {
			var found *Skill
			for i := range snapshot.Skills {
				if snapshot.Skills[i].ID == skillID {
					found = &snapshot.Skills[i]
					break
				}
			}
			if found == nil || found.Level != 2 {
				t.Fatalf("project=%s skill=%q not linked to persisted automatic evidence: %+v", projectID, skillID, found)
			}
		}
		if snapshot.Verified || snapshot.CurrentRank != "Zero" {
			t.Fatalf("project %s automatic evidence assigned rank: %+v", projectID, snapshot)
		}
		for _, skill := range snapshot.Skills {
			if skill.ID == "communication" && skill.Level != 0 {
				t.Fatal("automatic project result cannot prove reviewed communication", skill)
			}
		}
	}
}
