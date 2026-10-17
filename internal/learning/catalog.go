package learning

import (
	"academy/internal/assessment"
	"academy/internal/content"
	"academy/internal/store"
)

// WithCatalog adds signals from published, server-authored exercise mappings.
// It never derives rank or advanced competence from XP or client assertions.
func WithCatalog(snapshot Snapshot, p store.Progress, c *content.Catalog) Snapshot {
	if c == nil {
		return snapshot
	}
	latest := map[string]store.Attempt{}
	completed := map[string]bool{}
	for _, id := range p.Completed {
		completed[id] = true
	}
	for _, a := range p.Attempts {
		if old, ok := latest[a.ExerciseID]; !ok || a.ID > old.ID {
			latest[a.ExerciseID] = a
		}
	}
	reviews := map[string]bool{}
	for _, id := range snapshot.ReviewExercises {
		reviews[id] = true
	}
	addEvidence := func(exerciseID string, skills []string) {
		if !completed[exerciseID] || p.Evidence[exerciseID] <= 0 {
			return
		}
		for _, id := range skills {
			for i := range snapshot.Skills {
				skill := &snapshot.Skills[i]
				if skill.ID != id {
					continue
				}
				exists := false
				for _, old := range skill.Evidence {
					exists = exists || old == exerciseID
				}
				if !exists {
					skill.Evidence = append(skill.Evidence, exerciseID)
					skill.AttemptIDs = append(skill.AttemptIDs, p.Evidence[exerciseID])
				}
				if skill.Level < 2 {
					skill.Level = 2
				}
			}
		}
	}
	for _, lesson := range c.AllLessons() {
		for i := range snapshot.Ranks {
			if snapshot.Ranks[i].Name == lesson.Rank {
				snapshot.Ranks[i].ContentAvailable = true
			}
		}
		for _, ex := range lesson.Exercises {
			if a, ok := latest[ex.ID]; ok && !a.Passed && !reviews[ex.ID] {
				snapshot.ReviewExercises = append(snapshot.ReviewExercises, ex.ID)
				reviews[ex.ID] = true
			}
			addEvidence(ex.ID, ex.Skills)
		}
	}
	for id, skills := range map[string][]string{
		"p1:automatic": {"go-core", "testing"},
		"p2:automatic": {"http", "sql"},
		"p3:automatic": {"concurrency", "reliability"},
		"p4:automatic": {"performance", "distributed"},
		"p5:automatic": {"architecture"},
	} {
		addEvidence(id, skills)
	}
	for _, exam := range assessment.All() {
		for i := range snapshot.Ranks {
			if snapshot.Ranks[i].Name == exam.Rank {
				snapshot.Ranks[i].AssessmentAvailable = true
			}
		}
	}
	return snapshot
}
