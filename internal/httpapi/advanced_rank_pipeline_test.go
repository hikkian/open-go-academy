package httpapi

import (
	"academy/internal/assessment"
	"academy/internal/content"
	"academy/internal/projectreview"
	"academy/internal/projectrunner"
	"academy/internal/projects"
	"academy/internal/runner"
	"academy/internal/store"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// This executor runs only the fixed trusted exam references below with the
// assessment's own driver. Project grading remains an explicit API boundary
// stub, as this test is about rank-policy/evidence orchestration, not OCI.
type advancedPipelineExamRunner struct{}

func (advancedPipelineExamRunner) Available(context.Context) bool { return true }
func (advancedPipelineExamRunner) Execute(ctx context.Context, req runner.Request) (runner.Result, error) {
	dir, err := os.MkdirTemp("", "oga-advanced-exam-*")
	if err != nil {
		return runner.Result{}, err
	}
	defer os.RemoveAll(dir)
	files := make([]string, 0, len(req.Files))
	for name, source := range req.Files {
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
			return runner.Result{}, err
		}
		files = append(files, path)
	}
	args := append([]string{"run"}, files...)
	args = append(args, req.Args...)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOTOOLCHAIN=local", "GOPROXY=off", "GOSUMDB=off", "CGO_ENABLED=0")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err = cmd.Run()
	result := runner.Result{Stdout: stdout.String(), Stderr: stderr.String(), Status: "ok"}
	if err != nil {
		result.Status, result.ExitCode = "error", 1
		if exit, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exit.ExitCode()
		}
	}
	return result, nil
}

type advancedPipelineProjectRunner struct {
	profiles map[string]projectrunner.EvaluationProfile
}

func (advancedPipelineProjectRunner) Available(context.Context) bool { return true }
func (advancedPipelineProjectRunner) SupportsProject(_ context.Context, id string) bool {
	return id == "p3" || id == "p4" || id == "p5"
}
func (advancedPipelineProjectRunner) Execute(context.Context, projectrunner.Request) (runner.Result, error) {
	return runner.Result{Status: "ok"}, nil
}
func (r advancedPipelineProjectRunner) ProjectProfile(_ context.Context, id string) (projectrunner.EvaluationProfile, bool) {
	p, ok := r.profiles[id]
	return p, ok
}
func (r advancedPipelineProjectRunner) ExecuteWithProfile(_ context.Context, req projectrunner.Request) (runner.Result, projectrunner.EvaluationProfile, error) {
	profile, ok := r.profiles[req.ProjectID]
	if !ok {
		return runner.Result{}, projectrunner.EvaluationProfile{}, fmt.Errorf("no fixture profile for %s", req.ProjectID)
	}
	return runner.Result{Status: "ok", Passed: true, Checks: []runner.Check{{Name: "fixture deterministic checks", Passed: true}}}, profile, nil
}

type advancedPipelineReviewer struct {
	id      string
	private ed25519.PrivateKey
}

func newAdvancedPipelineReviewer(t *testing.T, id string) advancedPipelineReviewer {
	t.Helper()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return advancedPipelineReviewer{id: id, private: private}
}

func newAdvancedPipelineAPI(t *testing.T, reviewers []advancedPipelineReviewer, profiles map[string]projectrunner.EvaluationProfile) (*store.Store, func(string, string, any) *httptest.ResponseRecorder) {
	t.Helper()
	entries := make([]map[string]string, 0, len(reviewers))
	for _, reviewer := range reviewers {
		entries = append(entries, map[string]string{"reviewer_id": reviewer.id, "public_key": base64.StdEncoding.EncodeToString(reviewer.private.Public().(ed25519.PublicKey))})
	}
	registryJSON, err := json.Marshal(map[string]any{"reviewers": entries})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := projectreview.LoadRegistry(bytes.NewReader(registryJSON))
	if err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if _, err := s.CreateProfile("Advanced rank fixture", "light"); err != nil {
		t.Fatal(err)
	}
	catalog, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	api := New(s, catalog, advancedPipelineExamRunner{}, "pipeline-session", "127.0.0.1:8084")
	api.ProjectRunner = advancedPipelineProjectRunner{profiles: profiles}
	api.ProjectReviewers = registry
	h := api.Handler(t.TempDir())
	request := func(method, path string, body any) *httptest.ResponseRecorder {
		t.Helper()
		var data []byte
		if body != nil {
			data, err = json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
		}
		req := httptest.NewRequest(method, "http://127.0.0.1:8084"+path, bytes.NewReader(data))
		req.AddCookie(&http.Cookie{Name: "academy_session", Value: "pipeline-session"})
		if method != http.MethodGet {
			req.Header.Set("X-CSRF-Token", "pipeline-session")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	return s, request
}

func advancedPipelineProfiles() map[string]projectrunner.EvaluationProfile {
	profiles := make(map[string]projectrunner.EvaluationProfile)
	for _, id := range []string{"p3", "p4", "p5"} {
		project, _ := projects.Find(id)
		criteria, _ := projects.CriteriaHash(id)
		hash := func(label string) string {
			sum := sha256.Sum256([]byte(label))
			return "sha256:" + hex.EncodeToString(sum[:])
		}
		checker, rubric := "taskflow-v1", "taskflow-automatic-v1"
		if id == "p4" {
			checker, rubric = "scalelab-deterministic-v1", "scalelab-deterministic-v1"
		}
		if id == "p5" {
			checker, rubric = "capstone-deterministic-v1", "capstone-deterministic-v1"
		}
		profiles[id] = projectrunner.EvaluationProfile{
			ProjectVersion: project.Version, CriteriaHash: criteria,
			CheckerVersion: checker, CheckerDigest: hash(id + "-checker"),
			RuntimePolicyHash: hash(id + "-runtime"), RubricVersion: rubric, RubricHash: hash(id + "-rubric"),
		}
	}
	return profiles
}

func seedPipelineCourse(t *testing.T, s *store.Store, exam *assessment.Exam) {
	t.Helper()
	for _, id := range exam.Prerequisites {
		if _, err := s.RecordAttempt(id, map[string]string{"main.go": "package main"}, true, "fixture course evidence"); err != nil {
			t.Fatal(err)
		}
	}
}

func seedPipelinePriorRank(t *testing.T, s *store.Store, prior string) {
	t.Helper()
	// Prior-rank evidence is fixture setup. The rank under test is always
	// issued through the HTTP attempt/submit path below.
	id := "fixture-prior-" + strings.ToLower(strings.ReplaceAll(prior, "+", "plus"))
	if err := s.RegisterExam(id, "1", prior, "fixture-prior-rank-policy"); err != nil {
		t.Fatal(err)
	}
	attemptID := id + "-attempt"
	if _, err := s.DB().Exec(`INSERT INTO exam_attempts
		(id, profile_id, exam_id, version, rank, criteria_hash, token_hash, status, score, passed, created_at, finished_at, checks_json, evidence_json)
		VALUES (?, 'local', ?, '1', ?, 'fixture-prior-rank-policy', 'fixture-token-hash', 'completed', 100, 1, '2026-10-17T00:00:00Z', '2026-10-17T00:00:01Z', '[]', '{}')`,
		attemptID, id, prior); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DB().Exec(`INSERT INTO rank_awards (profile_id, exam_id, version, rank, attempt_id, awarded_at)
		VALUES ('local', ?, '1', ?, ?, '2026-10-17T00:00:01Z')`, id, prior, attemptID); err != nil {
		t.Fatal(err)
	}
}

func postAdvancedProject(t *testing.T, request func(string, string, any) *httptest.ResponseRecorder, projectID string) string {
	t.Helper()
	project, ok := projects.Find(projectID)
	if !ok {
		t.Fatalf("missing project %s", projectID)
	}
	files := cloneProjectFiles(project.StarterFiles)
	if projectID == "p3" {
		files["reference_test.go"] = "package main\nimport \"testing\"\nfunc TestX(t *testing.T) {}\n"
	}
	if projectID == "p5" {
		for i := 1; i <= 3; i++ {
			files[fmt.Sprintf("design/adr/ADR-00%d-decision.md", i)] = "# Decision\n\nFixture records a reviewed trade-off.\n"
		}
	}
	response := request(http.MethodPost, "/api/projects/"+projectID+"/submit", map[string]any{"files": files})
	if response.Code != http.StatusOK {
		t.Fatalf("%s project submit %d: %s", projectID, response.Code, response.Body.String())
	}
	var result struct {
		Passed      bool `json:"passed"`
		RankAwarded bool `json:"rank_awarded"`
		Evidence    struct {
			SubmissionID     string `json:"submission_id"`
			EvaluationStatus string `json:"evaluation_status"`
		} `json:"project_evidence"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if !result.Passed || result.RankAwarded || result.Evidence.SubmissionID == "" || result.Evidence.EvaluationStatus != "automatic_pass" {
		t.Fatalf("%s did not persist automatic evidence without awarding: %+v", projectID, result)
	}
	return result.Evidence.SubmissionID
}

func postPipelineReview(t *testing.T, request func(string, string, any) *httptest.ResponseRecorder, submissionID string, projectID string, reviewer advancedPipelineReviewer, decision string) {
	t.Helper()
	packetResponse := request(http.MethodGet, "/api/projects/"+projectID+"/submissions/"+submissionID+"/review-packet", nil)
	if packetResponse.Code != http.StatusOK {
		t.Fatalf("review packet %s: %d %s", projectID, packetResponse.Code, packetResponse.Body.String())
	}
	var packet struct {
		ProjectID    string `json:"project_id"`
		SubmissionID string `json:"submission_id"`
		EvaluationID string `json:"evaluation_id"`
		ArtifactHash string `json:"artifact_hash"`
		Rubric       struct {
			Version string `json:"version"`
			Hash    string `json:"hash"`
		} `json:"rubric"`
	}
	if err := json.Unmarshal(packetResponse.Body.Bytes(), &packet); err != nil {
		t.Fatal(err)
	}
	attestation := projectreview.Attestation{
		Version: projectreview.Version, ReviewerID: reviewer.id, SubmissionID: packet.SubmissionID,
		EvaluationID: packet.EvaluationID, ProjectID: packet.ProjectID, ProjectVersion: mustProjectVersion(t, projectID),
		ArtifactHash: strings.TrimPrefix(packet.ArtifactHash, "sha256:"), RubricVersion: packet.Rubric.Version,
		RubricHash: packet.Rubric.Hash, Decision: decision, Comments: "Integration fixture decision.", IssuedAt: "2026-10-17T00:00:00Z",
	}
	signed, err := projectreview.Sign(attestation, reviewer.private)
	if err != nil {
		t.Fatal(err)
	}
	response := request(http.MethodPost, "/api/projects/"+projectID+"/submissions/"+submissionID+"/reviews", signed)
	if response.Code != http.StatusCreated {
		t.Fatalf("review %s/%s decision=%s: %d %s", projectID, reviewer.id, decision, response.Code, response.Body.String())
	}
}

func mustProjectVersion(t *testing.T, id string) string {
	t.Helper()
	project, ok := projects.Find(id)
	if !ok {
		t.Fatalf("missing project %s", id)
	}
	return project.Version
}

var advancedPipelineReferences = map[string]string{
	"middle-exam-retry-decision":        `package main\nfunc RetryDecision(retryable bool, attemptsRemaining int) string { if retryable && attemptsRemaining > 0 { return "retry" }; return "terminal" }`,
	"middle-exam-backoff-seconds":       `package main\nfunc BackoffSeconds(baseSeconds, attempt, maxSeconds int) int { if baseSeconds <= 0 || attempt <= 0 || maxSeconds <= 0 { return 0 }; if baseSeconds >= maxSeconds { return maxSeconds }; if attempt > 30 { attempt = 30 }; delay := baseSeconds; for i := 1; i < attempt && delay < maxSeconds; i++ { if delay > maxSeconds/2 { return maxSeconds }; delay *= 2 }; if delay > maxSeconds { return maxSeconds }; return delay }`,
	"middle-exam-commit-allowed":        `package main\nfunc CommitAllowed(validated, conflictFree, contextActive bool) bool { return validated && conflictFree && contextActive }`,
	"middleplus-exam-compare-benchmark": `package main\nfunc CompareBenchmark(baselineRuns int, baselineMedianNS int64, candidateRuns int, candidateMedianNS int64, minimumRuns int) string { if baselineRuns < minimumRuns || candidateRuns < minimumRuns || baselineMedianNS <= 0 || candidateMedianNS <= 0 { return "insufficient" }; if candidateMedianNS < baselineMedianNS { return "improved" }; if candidateMedianNS > baselineMedianNS { return "regressed" }; return "equal" }`,
	"middleplus-exam-slo-budget":        `package main\nfunc SLOBudget(targetP95MS, observedP95MS, errorBasisPoints, maxErrorBasisPoints int) string { if targetP95MS <= 0 || observedP95MS < 0 || errorBasisPoints < 0 || maxErrorBasisPoints <= 0 { return "invalid" }; if observedP95MS > targetP95MS || errorBasisPoints > maxErrorBasisPoints { return "breach" }; return "within" }`,
	"middleplus-exam-fault-gate":        `package main\nfunc FaultGate(recovered, dataLoss, errorBudgetAvailable bool) string { if dataLoss { return "stop" }; if !recovered || !errorBudgetAvailable { return "hold" }; return "proceed" }`,
	"senior-exam-action-priority":       `package main\nfunc ActionPriority(impact, likelihood int) string { score := impact * likelihood; if score >= 9 { return "now" }; if score >= 4 { return "plan" }; return "track" }`,
	"senior-exam-decision-review":       `package main\nfunc DecisionReview(triggerObserved, evidenceUpdated, ownerAssigned bool) string { if !triggerObserved { return "wait" }; if !evidenceUpdated { return "gather_evidence" }; if !ownerAssigned { return "assign_owner" }; return "revisit_decision" }`,
	"senior-exam-release-gate":          `package main\nfunc ReleaseGate(acceptancePassed, rollbackReady, independentReviewApproved, highRiskOpen bool) bool { return acceptancePassed && rollbackReady && independentReviewApproved && !highRiskOpen }`,
}

func advancedPipelineExamCodes(t *testing.T, exam *assessment.Exam) map[string]string {
	t.Helper()
	codes := make(map[string]string, len(exam.Tasks))
	for _, task := range exam.Tasks {
		spec := exam.FunctionSpec(task.ID)
		if spec == nil {
			t.Fatalf("task %s has no function grading spec", task.ID)
		}
		source, ok := advancedPipelineReferences[spec.ExerciseID()]
		if !ok {
			t.Fatalf("missing trusted fixture reference %s", spec.ExerciseID())
		}
		source = strings.ReplaceAll(source, `\n`, "\n")
		codes[task.ID] = source
	}
	return codes
}

func TestAdvancedRankPipelineRequiresProjectReviewAndAwardsMiddleMiddlePlusSenior(t *testing.T) {
	for _, tc := range []struct {
		rank, examID, projectID string
		quorum                  int
	}{
		{"Middle", "middle-exam", "p3", 1}, {"Middle+", "middleplus-exam", "p4", 1}, {"Senior", "senior-exam", "p5", 2},
	} {
		t.Run(tc.rank, func(t *testing.T) {
			reviewers := []advancedPipelineReviewer{newAdvancedPipelineReviewer(t, "reviewer-a"), newAdvancedPipelineReviewer(t, "reviewer-b")}
			s, request := newAdvancedPipelineAPI(t, reviewers, advancedPipelineProfiles())
			exam, ok := assessment.Find(tc.examID, "")
			if !ok {
				t.Fatalf("missing exam %s", tc.examID)
			}
			seedPipelinePriorRank(t, s, exam.RequiredRank)
			seedPipelineCourse(t, s, exam)
			submissionID := postAdvancedProject(t, request, tc.projectID)
			// A signed reject must make the project ineligible even after its
			// automatic checks pass. The same reviewer's later approval supersedes it.
			postPipelineReview(t, request, submissionID, tc.projectID, reviewers[0], "reject")
			if got := request(http.MethodPost, "/api/exams/"+tc.examID+"/attempts", map[string]any{}); got.Code != http.StatusConflict {
				t.Fatalf("rejected project review did not block exam start: %d %s", got.Code, got.Body.String())
			}
			postPipelineReview(t, request, submissionID, tc.projectID, reviewers[0], "approve")
			if tc.quorum == 2 {
				postPipelineReview(t, request, submissionID, tc.projectID, reviewers[1], "approve")
			}

			// An objectively failed exam is completed without an award; the learner
			// can then start a fresh attempt with the current pinned evidence.
			failedAttempt := startCatalogExam(t, catalogFixture{store: s, request: request}, tc.examID)
			failedPayload := map[string]any{"attempt_token": failedAttempt.Token, "revision": failedAttempt.Revision, "answers": map[string]string{}, "codes": map[string]string{}}
			failedResponse := request(http.MethodPost, "/api/exams/attempts/"+failedAttempt.ID+"/submit", failedPayload)
			if failedResponse.Code != http.StatusOK {
				t.Fatalf("failed exam submit %d: %s", failedResponse.Code, failedResponse.Body.String())
			}
			var failedResult struct {
				Passed      bool `json:"passed"`
				RankAwarded bool `json:"rank_awarded"`
			}
			if err := json.Unmarshal(failedResponse.Body.Bytes(), &failedResult); err != nil {
				t.Fatal(err)
			}
			if failedResult.Passed || failedResult.RankAwarded {
				t.Fatalf("failed exam awarded rank: %+v", failedResult)
			}
			priorAwards, err := s.RankAwards("local")
			if err != nil {
				t.Fatal(err)
			}
			for _, award := range priorAwards {
				if award.Rank == tc.rank {
					t.Fatalf("rank %s awarded after failed attempt", tc.rank)
				}
			}

			started := startCatalogExam(t, catalogFixture{store: s, request: request}, tc.examID)
			payload := map[string]any{"attempt_token": started.Token, "revision": started.Revision, "answers": examAnswers(exam), "codes": advancedPipelineExamCodes(t, exam)}
			response := request(http.MethodPost, "/api/exams/attempts/"+started.ID+"/submit", payload)
			if response.Code != http.StatusOK {
				t.Fatalf("exam submit %d: %s", response.Code, response.Body.String())
			}
			var result struct {
				Passed      bool   `json:"passed"`
				RankAwarded bool   `json:"rank_awarded"`
				Rank        string `json:"rank"`
				Score       int    `json:"score"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if !result.Passed || !result.RankAwarded || result.Rank != tc.rank || result.Score != 100 {
				t.Fatalf("objective exam did not award exact rank: %+v body=%s", result, response.Body.String())
			}
			if replay := request(http.MethodPost, "/api/exams/attempts/"+started.ID+"/submit", payload); replay.Code != http.StatusConflict {
				t.Fatalf("completed exam replay status=%d body=%s", replay.Code, replay.Body.String())
			}
			awards, err := s.RankAwards("local")
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			for _, award := range awards {
				if award.Rank == tc.rank {
					count++
				}
			}
			if count != 1 {
				t.Fatalf("rank %s awards=%d, all=%+v", tc.rank, count, awards)
			}
		})
	}
}
