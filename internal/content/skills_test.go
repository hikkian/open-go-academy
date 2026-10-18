package content

import (
	"fmt"
	"slices"
	"testing"
)

func TestCompetencyMetadataRejectsUnknownDuplicateOrMissingIDs(t *testing.T) {
	for _, skills := range [][]string{{"unknown"}, {"sql", "sql"}, nil} {
		if err := validateExerciseSkills(Exercise{ID: "fixture", Kind: "function", Skills: skills}); err == nil {
			t.Fatalf("invalid competency list accepted: %v", skills)
		}
	}
	if err := validateExerciseSkills(Exercise{ID: "fixture", Kind: "function", Skills: []string{"sql", "reliability"}}); err != nil {
		t.Fatal(err)
	}
}

func TestAllAdvancedCompetenciesUseKnownMeasuredPracticeTags(t *testing.T) {
	manifest, err := Manifest()
	if err != nil {
		t.Fatal(err)
	}
	ids := append([]string(nil), manifest.Published...)
	seen := make(map[string]struct{}, len(manifest.Published)+88)
	for _, id := range ids {
		seen[id] = struct{}{}
	}
	for _, track := range []struct {
		prefix string
		count  int
	}{{"middle", 40}, {"middleplus", 24}, {"senior", 24}} {
		for n := 1; n <= track.count; n++ {
			id := fmt.Sprintf("%s-%02d", track.prefix, n)
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	catalog, err := loadLessons(ids)
	if err != nil {
		t.Fatal(err)
	}
	for _, lesson := range catalog.AllLessons() {
		if lesson.Rank != "Middle" && lesson.Rank != "Middle+" && lesson.Rank != "Senior" {
			continue
		}
		for _, exercise := range lesson.Exercises {
			if err := validateExerciseSkills(exercise); err != nil {
				t.Fatal(err)
			}
			if slices.Contains(exercise.Skills, "communication") {
				t.Errorf("%s helper is not a reviewed written artifact", exercise.ID)
			}
		}
	}
	for _, test := range []struct{ id, skill string }{{"middle-01", "concurrency"}, {"middle-26", "sql"}, {"middleplus-12", "distributed"}, {"senior-22", "architecture"}} {
		lesson, ok := catalog.FindLesson(test.id)
		if !ok {
			t.Fatalf("missing lesson %s", test.id)
		}
		for _, exercise := range lesson.Exercises {
			if !slices.Contains(exercise.Skills, test.skill) {
				t.Errorf("%s missing topical skill %s", exercise.ID, test.skill)
			}
		}
	}
}
