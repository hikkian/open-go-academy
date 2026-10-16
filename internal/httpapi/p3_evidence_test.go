package httpapi

import (
	"academy/internal/content"
	"academy/internal/projectrunner"
	"academy/internal/projects"
	"academy/internal/runner"
	"academy/internal/store"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

type p3EvidenceExecutor struct {
	profile projectrunner.EvaluationProfile
	calls   int
	last    projectrunner.Request
}

func (*p3EvidenceExecutor) Available(context.Context) bool                    { return true }
func (*p3EvidenceExecutor) SupportsProject(_ context.Context, id string) bool { return id == "p3" }
func (e *p3EvidenceExecutor) Execute(_ context.Context, q projectrunner.Request) (runner.Result, error) {
	return runner.Result{}, nil
}
func (e *p3EvidenceExecutor) ExecuteWithProfile(_ context.Context, q projectrunner.Request) (runner.Result, projectrunner.EvaluationProfile, error) {
	e.calls++
	e.last = q
	checks := make([]runner.Check, 45)
	for i := range checks {
		checks[i] = runner.Check{Name: "taskflow.check." + strconv.Itoa(i+1), Passed: true}
	}
	return runner.Result{Status: "automatic_pass", ExitCode: 0, Passed: true, Checks: checks}, e.profile, nil
}

func TestP3SubmitPersistsImmutableArtifactAndTrustedEvaluation(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.CreateProfile("Ada", "light"); err != nil {
		t.Fatal(err)
	}
	catalog, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	project, _ := projects.Find("p3")
	criteriaHash, _ := projects.CriteriaHash("p3")
	digest := "sha256:" + strings.Repeat("a", 64)
	runtimeHash := "sha256:" + strings.Repeat("b", 64)
	rubricHash := "sha256:" + strings.Repeat("c", 64)
	worker := &p3EvidenceExecutor{profile: projectrunner.EvaluationProfile{
		ProjectVersion: project.Version, CriteriaHash: criteriaHash, CheckerVersion: "taskflow-v1",
		CheckerDigest: digest, RuntimePolicyHash: runtimeHash,
		RubricVersion: "taskflow-automatic-v1", RubricHash: rubricHash,
	}}
	api := New(s, catalog, nil, "session", "127.0.0.1:8081")
	api.ProjectRunner = worker
	handler := api.Handler(t.TempDir())
	files := cloneProjectFiles(project.StarterFiles)
	files["reference_test.go"] = "package main\nimport \"testing\"\nfunc TestX(t *testing.T) {}\n"
	files["Dockerfile"] = "export-only; never executed"
	files["compose.yaml"] = "export-only: true\n"
	files[".env.example"] = "SAMPLE=not-a-secret\n"
	request := func() *httptest.ResponseRecorder {
		body, _ := json.Marshal(map[string]any{"files": files})
		req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8081/api/projects/p3/submit", bytes.NewReader(body))
		req.AddCookie(&http.Cookie{Name: "academy_session", Value: "session"})
		req.Header.Set("X-CSRF-Token", "session")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}

	first := request()
	if first.Code != http.StatusOK {
		t.Fatalf("P3 submit status=%d body=%s", first.Code, first.Body.String())
	}
	var payload struct {
		Passed          bool           `json:"passed"`
		Checks          []runner.Check `json:"checks"`
		RankAwarded     bool           `json:"rank_awarded"`
		ProjectEvidence struct {
			SubmissionID       string `json:"submission_id"`
			EvaluationID       string `json:"evaluation_id"`
			ProjectVersion     string `json:"project_version"`
			CriteriaHash       string `json:"criteria_hash"`
			ArtifactHash       string `json:"artifact_hash"`
			ExecutedSourceHash string `json:"executed_source_hash"`
			CheckerVersion     string `json:"checker_version"`
			CheckerDigest      string `json:"checker_digest"`
			RuntimePolicyHash  string `json:"runtime_policy_hash"`
			RubricVersion      string `json:"rubric_version"`
			RubricHash         string `json:"rubric_hash"`
			ChecksHash         string `json:"checks_hash"`
		} `json:"project_evidence"`
	}
	if err = json.Unmarshal(first.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Passed || payload.RankAwarded || len(payload.Checks) != 45 {
		t.Fatalf("P3 result or rank boundary incorrect: %+v", payload)
	}
	evidence := payload.ProjectEvidence
	if evidence.SubmissionID == "" || evidence.EvaluationID == "" || evidence.ProjectVersion != project.Version ||
		evidence.CriteriaHash != criteriaHash || evidence.ArtifactHash == evidence.ExecutedSourceHash ||
		evidence.CheckerDigest != digest || evidence.RuntimePolicyHash != runtimeHash || evidence.RubricHash != rubricHash || evidence.ChecksHash == "" {
		t.Fatalf("missing or conflated project evidence: %+v", evidence)
	}
	if _, ok := worker.last.Files["Dockerfile"]; ok {
		t.Fatal("export-only Dockerfile reached the runner")
	}
	if _, ok := worker.last.Files["compose.yaml"]; ok {
		t.Fatal("export-only Compose file reached the runner")
	}
	if _, ok := worker.last.Files[".env.example"]; ok {
		t.Fatal("export-only environment file reached the runner")
	}

	replay := request()
	var replayPayload map[string]any
	if replay.Code != http.StatusOK || json.Unmarshal(replay.Body.Bytes(), &replayPayload) != nil {
		t.Fatalf("same artifact replay failed: %d %s", replay.Code, replay.Body.String())
	}
	replayEvidence := replayPayload["project_evidence"].(map[string]any)
	if replayEvidence["submission_id"] != evidence.SubmissionID || replayEvidence["evaluation_id"] != evidence.EvaluationID {
		t.Fatalf("same immutable project evaluation created duplicate evidence: %v", replayEvidence)
	}
	var submissions, evaluations, awards int
	if err = s.DB().QueryRow(`SELECT COUNT(*) FROM project_submissions`).Scan(&submissions); err != nil || submissions != 1 {
		t.Fatalf("project submissions=%d err=%v", submissions, err)
	}
	if err = s.DB().QueryRow(`SELECT COUNT(*) FROM project_evaluations`).Scan(&evaluations); err != nil || evaluations != 1 {
		t.Fatalf("project evaluations=%d err=%v", evaluations, err)
	}
	if err = s.DB().QueryRow(`SELECT COUNT(*) FROM rank_awards`).Scan(&awards); err != nil || awards != 0 {
		t.Fatalf("project evaluation issued rank: awards=%d err=%v", awards, err)
	}
}

func cloneProjectFiles(files map[string]string) map[string]string {
	clone := make(map[string]string, len(files))
	for name, contents := range files {
		clone[name] = contents
	}
	return clone
}
