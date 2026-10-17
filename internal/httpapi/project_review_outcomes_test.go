package httpapi

import (
	"academy/internal/content"
	"academy/internal/projects"
	"academy/internal/store"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestProjectManualRubricIsVisibleBeforeAnySubmission(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.CreateProfile("rubric fixture", "light"); err != nil {
		t.Fatal(err)
	}
	catalog, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	api := New(s, catalog, nil, "rubric-session", "127.0.0.1:8084")
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8084/api/projects/p3", nil)
	request.AddCookie(&http.Cookie{Name: "academy_session", Value: "rubric-session"})
	response := httptest.NewRecorder()
	api.Handler(t.TempDir()).ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("project before submit: %d %s", response.Code, response.Body.String())
	}
	var project struct {
		ManualReviewRubric *manualReviewRubric `json:"manual_review_rubric"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &project); err != nil {
		t.Fatal(err)
	}
	if project.ManualReviewRubric == nil || !strings.Contains(strings.Join(project.ManualReviewRubric.Requirements, "\n"), "M33-slo-error-budget") {
		t.Fatal("student cannot read required review evidence before project submission")
	}
}

func TestP3ManualRubricBindsMiddleOutcomeEvidence(t *testing.T) {
	project, ok := projects.Find("p3")
	if !ok {
		t.Fatal("P3 missing")
	}
	original := append([]string(nil), project.Requirements...)
	rubric := buildManualReviewRubric(project)
	if rubric.Version != project.Version+"-manual-review-v2" {
		t.Fatal("P3 manual review version was not advanced", rubric.Version)
	}
	text := strings.Join(rubric.Requirements, "\n")
	for _, id := range []string{"M27-access-boundaries", "M28-dependency-graph", "M29-request-boundaries", "M30-32-operations", "M33-slo-error-budget", "M34-35-fault-resilience", "M36-40-release-evidence"} {
		if !strings.Contains(text, "["+id+"]") {
			t.Errorf("manual rubric omits %s", id)
		}
	}
	if !reflect.DeepEqual(project.Requirements, original) {
		t.Fatal("manual rubric mutated automatic project requirements")
	}
	if rubric.Hash == "" || buildManualReviewRubric(project).Hash != rubric.Hash {
		t.Fatal("manual rubric hash is not deterministic")
	}
	project.Requirements = append(append([]string(nil), project.Requirements...), "Changed review requirement")
	if buildManualReviewRubric(project).Hash == rubric.Hash {
		t.Fatal("review change must invalidate a pinned rubric")
	}
	for _, id := range []string{"p4", "p5"} {
		other, _ := projects.Find(id)
		if got := buildManualReviewRubric(other).Version; got != other.Version+"-manual-review-v1" {
			t.Errorf("unrelated rubric %s changed: %s", id, got)
		}
	}
}
