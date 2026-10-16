package httpapi

import (
	"academy/internal/content"
	"academy/internal/projectrunner"
	"academy/internal/projects"
	"academy/internal/runner"
	"academy/internal/store"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type p45EvidenceExecutor struct {
	profiles map[string]projectrunner.EvaluationProfile
	calls    int
}

func (*p45EvidenceExecutor) Available(context.Context) bool { return true }
func (*p45EvidenceExecutor) SupportsProject(_ context.Context, id string) bool {
	return id == "p4" || id == "p5"
}
func (*p45EvidenceExecutor) Execute(context.Context, projectrunner.Request) (runner.Result, error) {
	return runner.Result{Status: "ok"}, nil
}
func (e *p45EvidenceExecutor) ExecuteWithProfile(_ context.Context, req projectrunner.Request) (runner.Result, projectrunner.EvaluationProfile, error) {
	e.calls++
	profile := e.profiles[req.ProjectID]
	count := 4
	if req.ProjectID == "p5" {
		count = 6
	}
	checks := make([]runner.Check, count)
	for i := range checks {
		checks[i] = runner.Check{Name: "machine-check-" + string(rune('1'+i)), Passed: true}
	}
	return runner.Result{Status: "ok", ExitCode: 0, Passed: true, Checks: checks}, profile, nil
}

func TestP4P5SubmitPersistsRunnerOKAsAutomaticEvidenceWithoutAward(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.CreateProfile("Ada", "light"); err != nil {
		t.Fatal(err)
	}
	catalog, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	profiles := map[string]projectrunner.EvaluationProfile{}
	for _, id := range []string{"p4", "p5"} {
		project, ok := projects.Find(id)
		if !ok {
			t.Fatalf("%s missing", id)
		}
		criteria, _ := projects.CriteriaHash(id)
		checker := "scalelab-deterministic-v1"
		if id == "p5" {
			checker = "capstone-deterministic-v1"
		}
		hash := func(value string) string {
			sum := sha256.Sum256([]byte(value))
			return "sha256:" + hex.EncodeToString(sum[:])
		}
		profiles[id] = projectrunner.EvaluationProfile{
			ProjectVersion: project.Version, CriteriaHash: criteria, CheckerVersion: checker,
			CheckerDigest: hash(id + ":checker"), RuntimePolicyHash: hash(id + ":runtime"),
			RubricVersion: checker, RubricHash: hash(id + ":rubric"),
		}
	}
	api := New(s, catalog, nil, "session", "127.0.0.1:8081")
	api.ProjectRunner = &p45EvidenceExecutor{profiles: profiles}
	handler := api.Handler(t.TempDir())
	for _, id := range []string{"p4", "p5"} {
		t.Run(id, func(t *testing.T) {
			project, _ := projects.Find(id)
			files := cloneProjectFiles(project.StarterFiles)
			if id == "p5" {
				for i := 1; i <= 3; i++ {
					files["design/adr/ADR-00"+string(rune('0'+i))+"-decision.md"] = "# Decision\n\nReviewed trade-off.\n"
				}
			}
			body, _ := json.Marshal(map[string]any{"files": files})
			req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8081/api/projects/"+id+"/submit", bytes.NewReader(body))
			req.AddCookie(&http.Cookie{Name: "academy_session", Value: "session"})
			req.Header.Set("X-CSRF-Token", "session")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			if response.Code != http.StatusOK {
				t.Fatalf("submit returned %d: %s", response.Code, response.Body.String())
			}
			var payload struct {
				Passed      bool `json:"passed"`
				RankAwarded bool `json:"rank_awarded"`
				Evidence    struct {
					EvaluationStatus string `json:"evaluation_status"`
				} `json:"project_evidence"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if !payload.Passed || payload.RankAwarded || payload.Evidence.EvaluationStatus != "automatic_pass" {
				t.Fatalf("P4/P5 machine evidence or rank boundary incorrect: %+v", payload)
			}
			var count int
			if err := s.DB().QueryRow(`SELECT COUNT(*) FROM project_evaluations WHERE project_id=? AND evaluation_status='automatic_pass'`, id).Scan(&count); err != nil || count != 1 {
				t.Fatalf("persisted evaluation count=%d err=%v", count, err)
			}
			var awards int
			if err := s.DB().QueryRow("SELECT COUNT(*) FROM rank_awards").Scan(&awards); err != nil || awards != 0 {
				t.Fatal("project Submit awarded a rank", err)
			}
		})
	}
	p4, _ := projects.Find("p4")
	attack := cloneProjectFiles(p4.StarterFiles)
	attack["main.go"] = "package main\nimport \"os\"\nfunc init(){ os.Exit(0) }\n"
	attack["early_exit_test.go"] = "package main\nimport (\"os\"; \"testing\")\nfunc TestEarly(t *testing.T){ os.Exit(0) }\n"
	body, _ := json.Marshal(map[string]any{"files": attack})
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8081/api/projects/p4/submit", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "academy_session", Value: "session"})
	req.Header.Set("X-CSRF-Token", "session")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	if response.Code != http.StatusUnprocessableEntity || !strings.Contains(response.Body.String(), "не может") {
		t.Fatalf("assessment process hook was not rejected at API boundary: %d %s", response.Code, response.Body.String())
	}
	if api.ProjectRunner.(*p45EvidenceExecutor).calls != 2 {
		t.Fatal("invalid process hook reached the trusted runner")
	}
}
