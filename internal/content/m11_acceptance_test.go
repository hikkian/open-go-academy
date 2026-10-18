package content

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"academy/internal/runner"
)

var lessonWord = regexp.MustCompile(`[\pL\pN]+([-'’][\pL\pN]+)*`)

// TestM11AdvancedCurriculumAcceptance runs the final content/Go-code gate for
// all advanced lessons, whether they are published in the local catalog or
// still candidates for publication:
// ACADEMY_ACCEPT_M11_ADVANCED=1 go test -count=1 -timeout=1h ./internal/content -run '^TestM11AdvancedCurriculumAcceptance$'
// To rerun a small subset after a text-only fix, set ACADEMY_M11_LESSON_IDS to
// a comma-separated list of advanced lesson IDs. Catalog/rank/order validation remains global.
func TestM11AdvancedCurriculumAcceptance(t *testing.T) {
	if os.Getenv("ACADEMY_ACCEPT_M11_ADVANCED") != "1" {
		t.Skip("set ACADEMY_ACCEPT_M11_ADVANCED=1 to run full M11 curriculum acceptance")
	}

	manifest, err := Manifest()
	if err != nil {
		t.Fatal(err)
	}
	draftIDs := make([]string, 0, 88)
	for _, track := range []struct {
		prefix string
		count  int
	}{{"middle", 40}, {"middleplus", 24}, {"senior", 24}} {
		for n := 1; n <= track.count; n++ {
			draftIDs = append(draftIDs, fmt.Sprintf("%s-%02d", track.prefix, n))
		}
	}
	selected := make(map[string]struct{})
	if raw := strings.TrimSpace(os.Getenv("ACADEMY_M11_LESSON_IDS")); raw != "" {
		known := make(map[string]struct{}, len(draftIDs))
		for _, id := range draftIDs {
			known[id] = struct{}{}
		}
		for _, id := range strings.Split(raw, ",") {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if _, ok := known[id]; !ok {
				t.Fatalf("unknown M11 lesson filter %q", id)
			}
			selected[id] = struct{}{}
		}
		if len(selected) == 0 {
			t.Fatal("ACADEMY_M11_LESSON_IDS must contain at least one lesson ID")
		}
	}
	allIDs := make([]string, 0, len(manifest.Published)+len(draftIDs))
	seenIDs := make(map[string]struct{}, cap(allIDs))
	for _, id := range append(append([]string(nil), manifest.Published...), draftIDs...) {
		if _, exists := seenIDs[id]; exists {
			continue
		}
		seenIDs[id] = struct{}{}
		allIDs = append(allIDs, id)
	}
	catalog, err := loadLessons(allIDs)
	if err != nil {
		t.Fatalf("candidate catalog failed production validation: %v", err)
	}
	if got, want := len(catalog.Lessons), len(allIDs); got != want {
		t.Fatalf("loaded %d lessons, want %d", got, want)
	}

	engine := runner.New()
	engine.Scope = fmt.Sprintf("m11-acceptance-%d-%d", os.Getpid(), time.Now().UnixNano())
	preflightCtx, preflightCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := engine.Preflight(preflightCtx); err != nil {
		preflightCancel()
		t.Fatalf("isolated runner preflight failed; candidate code was not executed: %v", err)
	}
	preflightCancel()

	seenOrders := make(map[int]string, len(draftIDs))
	ranks := map[string]string{"middle": "Middle", "middleplus": "Middle+", "senior": "Senior"}
	for _, id := range draftIDs {
		lesson, ok := catalog.FindLesson(id)
		if !ok {
			t.Fatalf("candidate lesson %s missing from catalog", id)
		}
		prefix := strings.SplitN(id, "-", 2)[0]
		if lesson.Rank != ranks[prefix] {
			t.Errorf("%s has rank %q, want %q", id, lesson.Rank, ranks[prefix])
		}
		base := map[string]int{"middle": 40, "middleplus": 80, "senior": 104}[prefix]
		chapter, scanErr := strconv.Atoi(strings.TrimPrefix(id, prefix+"-"))
		if scanErr != nil {
			t.Errorf("%s has invalid chapter suffix", id)
		} else if want := base + chapter; lesson.Order != want {
			t.Errorf("%s has order %d, want global M11 order %d", id, lesson.Order, want)
		}
		if old, exists := seenOrders[lesson.Order]; exists {
			t.Errorf("%s and %s share global order %d", old, id, lesson.Order)
		} else {
			seenOrders[lesson.Order] = id
		}
		if len(selected) > 0 {
			if _, ok := selected[id]; !ok {
				continue
			}
		}
		if words := len(lessonWord.FindAllString(lesson.Body, -1)); words < 1800 || words > 2200 {
			t.Errorf("%s has %d lexical words, want 1800–2200", id, words)
		}
		groups, exampleErr := documentationExamples(lesson.Body)
		if exampleErr != nil {
			t.Errorf("%s Go examples have invalid metadata: %v", id, exampleErr)
			continue
		}
		if len(groups) < 3 {
			t.Errorf("%s has %d Go example groups, want at least 3", id, len(groups))
			continue
		}
		for _, group := range groups {
			t.Run(fmt.Sprintf("%s-example-%s", id, group.ID), func(t *testing.T) {
				trustedCommand(t, group.Files, nil, group.Mode)
			})
		}
		if len(lesson.Exercises) != 3 {
			t.Errorf("%s has %d exercises, want 3", id, len(lesson.Exercises))
			continue
		}
		for _, exercise := range lesson.Exercises {
			spec, ok := catalog.Grading(exercise.ID)
			if !ok {
				t.Errorf("%s has no production grader %s", id, exercise.ID)
				continue
			}
			starterRequest, starterErr := spec.Request(map[string]string{exercise.StarterFile: exercise.StarterCode}, false)
			if starterErr != nil {
				t.Errorf("%s starter %s violates its production source contract: %v", id, exercise.ID, starterErr)
			} else {
				// These starter files are trusted repository content. Compile them
				// without running their intentionally incomplete bodies.
				t.Run(id+"-starter-"+exercise.ID, func(t *testing.T) {
					trustedCommand(t, starterRequest.Files, nil, "build")
				})
			}
			reference, ok := catalog.Reference(exercise.ID)
			if !ok {
				t.Errorf("%s has no private reference %s", id, exercise.ID)
				continue
			}
			request, requestErr := spec.Request(map[string]string{"main.go": reference}, true)
			if requestErr != nil {
				t.Errorf("%s reference %s cannot produce production grader request: %v", id, exercise.ID, requestErr)
				continue
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			result, runErr := engine.Execute(ctx, request)
			cancel()
			if runErr != nil {
				t.Fatalf("%s reference %s isolated runner failed: %v", id, exercise.ID, runErr)
			}
			if !engine.Ready() {
				t.Fatalf("isolated runner cleanup failed after %s reference %s; stopped before another container", id, exercise.ID)
			}
			checks := spec.Check(result, true)
			if len(checks) == 0 {
				t.Errorf("%s reference %s returned no production checks", id, exercise.ID)
			}
			for _, check := range checks {
				if !check.Passed {
					t.Errorf("%s reference %s failed production check %q", id, exercise.ID, check.Name)
				}
			}
		}
	}
}
