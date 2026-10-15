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
		for i, rank := range s.Ranks {
			if rank.AssessmentAvailable != (i < 3) {
				t.Fatal("assessment availability differs from published catalog", rank)
			}
		}
		if s.Verified {
			t.Fatal("project evidence assigned rank")
		}
	}
}
