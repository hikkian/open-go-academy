package httpapi

import (
	"academy/internal/content"
	"academy/internal/projects"
	"academy/internal/store"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProjectEvidenceLookupIsScopedAndReadOnly(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.CreateProfile("Ada", "light"); err != nil {
		t.Fatal(err)
	}
	p, ok := projects.Find("p3")
	if !ok {
		t.Fatal("p3 is not in project catalog")
	}
	criteriaHash, ok := projects.CriteriaHash("p3")
	if !ok {
		t.Fatal("p3 criteria hash unavailable")
	}
	submission, err := s.RecordProjectSubmission("local", "p3", cloneProjectFiles(p.StarterFiles), cloneProjectFiles(p.StarterFiles))
	if err != nil {
		t.Fatal(err)
	}
	profile := store.ProjectEvaluationProfile{
		ProjectVersion: p.Version, CriteriaHash: criteriaHash,
		CheckerVersion: "taskflow-v1", CheckerDigest: "sha256:" + strings.Repeat("a", 64),
		RuntimePolicyHash: "sha256:" + strings.Repeat("b", 64),
		RubricVersion:     "taskflow-automatic-v1", RubricHash: "sha256:" + strings.Repeat("c", 64),
	}
	evaluation, err := s.RecordProjectEvaluation("local", submission.ID, profile, "automatic_pass", 0, true, `[{"name":"criteria.one","passed":true}]`)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	h := New(s, catalog, nil, "session", "127.0.0.1:8081").Handler(t.TempDir())
	lookup := func(projectID, submissionID string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8081/api/projects/"+projectID+"/submissions/"+submissionID, nil)
		req.AddCookie(&http.Cookie{Name: "academy_session", Value: "session"})
		res := httptest.NewRecorder()
		h.ServeHTTP(res, req)
		return res
	}
	if got := lookup("p2", submission.ID); got.Code != http.StatusNotFound {
		t.Fatalf("cross-project lookup status=%d body=%s", got.Code, got.Body.String())
	}
	if got := lookup("p3", "unknown-submission"); got.Code != http.StatusNotFound {
		t.Fatalf("unknown submission status=%d body=%s", got.Code, got.Body.String())
	}
	got := lookup("p3", submission.ID)
	if got.Code != http.StatusOK {
		t.Fatalf("lookup status=%d body=%s", got.Code, got.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(got.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	for key, want := range map[string]string{
		"submission_id": submission.ID, "evaluation_id": evaluation.ID,
		"project_id": "p3", "project_version": p.Version, "criteria_hash": criteriaHash,
		"artifact_hash": submission.ArtifactHash, "executed_source_hash": submission.ExecutedSourceHash,
		"evaluation_status": "automatic_pass", "checker_version": profile.CheckerVersion,
		"checker_digest": profile.CheckerDigest, "runtime_policy_hash": profile.RuntimePolicyHash,
		"rubric_version": profile.RubricVersion, "rubric_hash": profile.RubricHash,
	} {
		if payload[key] != want {
			t.Errorf("%s=%v, want %q", key, payload[key], want)
		}
	}
	if payload["passed"] != true || payload["exit_code"] != float64(0) || payload["checks_hash"] == "" || payload["submission_created_at"] == "" || payload["evaluation_created_at"] == "" {
		t.Fatalf("incomplete persisted evidence projection: %v", payload)
	}
	for _, private := range []string{"profile_id", "files", "executed_source", "checks_json", "api_key", "secret"} {
		if _, exists := payload[private]; exists {
			t.Errorf("lookup exposed private field %q", private)
		}
	}
	var awards int
	if err := s.DB().QueryRow(`SELECT COUNT(*) FROM rank_awards`).Scan(&awards); err != nil || awards != 0 {
		t.Fatalf("read-only lookup changed rank awards: count=%d err=%v", awards, err)
	}
}
