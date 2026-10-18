package httpapi

import (
	"academy/internal/assessment"
	"academy/internal/content"
	"academy/internal/credentials"
	"academy/internal/projectreview"
	"academy/internal/store"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSeniorCredentialFlow exercises the public API and persistent store as
// one learner journey. Reviewer and issuer keys are generated test fixtures;
// no production signing material is loaded or inferred.
func TestSeniorCredentialFlow(t *testing.T) {
	for _, tc := range []struct {
		name           string
		approvals      int
		wantCredential bool
	}{
		{name: "two independent approvals complete the rank and issue privately", approvals: 2, wantCredential: true},
		{name: "one approval cannot meet quorum", approvals: 1, wantCredential: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reviewers := []advancedPipelineReviewer{
				newAdvancedPipelineReviewer(t, "senior-reviewer-a"),
				newAdvancedPipelineReviewer(t, "senior-reviewer-b"),
			}
			registryEntries := make([]map[string]string, 0, len(reviewers))
			for _, reviewer := range reviewers {
				registryEntries = append(registryEntries, map[string]string{
					"reviewer_id": reviewer.id,
					"public_key":  base64.StdEncoding.EncodeToString(reviewer.private.Public().(ed25519.PublicKey)),
				})
			}
			registryJSON, err := json.Marshal(map[string]any{"reviewers": registryEntries})
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
			if _, err := s.CreateProfile("Senior flow fixture", "light"); err != nil {
				t.Fatal(err)
			}
			catalog, err := content.Load()
			if err != nil {
				t.Fatal(err)
			}
			api := New(s, catalog, advancedPipelineExamRunner{}, "senior-flow-session", "127.0.0.1:8086")
			api.ProjectRunner = advancedPipelineProjectRunner{profiles: advancedPipelineProfiles()}
			api.ProjectReviewers = registry
			_, issuerKey, err := ed25519.GenerateKey(rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			api.IssuerKey = issuerKey
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
				req := httptest.NewRequest(method, "http://127.0.0.1:8086"+path, bytes.NewReader(data))
				req.AddCookie(&http.Cookie{Name: "academy_session", Value: "senior-flow-session"})
				if method != http.MethodGet {
					req.Header.Set("X-CSRF-Token", "senior-flow-session")
				}
				w := httptest.NewRecorder()
				h.ServeHTTP(w, req)
				return w
			}

			exam, ok := assessment.Find("senior-exam", "")
			if !ok {
				t.Fatal("Senior exam is missing")
			}
			seedPipelinePriorRank(t, s, exam.RequiredRank)
			seedPipelineCourse(t, s, exam)
			submissionID := postAdvancedProject(t, request, "p5")
			postPipelineReview(t, request, submissionID, "p5", reviewers[0], "approve")
			if tc.approvals == 2 {
				postPipelineReview(t, request, submissionID, "p5", reviewers[1], "approve")
			}

			credentialList := func() []map[string]any {
				t.Helper()
				w := request(http.MethodGet, "/api/credentials", nil)
				if w.Code != http.StatusOK {
					t.Fatalf("GET credentials: %d %s", w.Code, w.Body.String())
				}
				var rows []map[string]any
				if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
					t.Fatal(err)
				}
				return rows
			}
			if !tc.wantCredential {
				if w := request(http.MethodPost, "/api/exams/senior-exam/attempts", map[string]any{}); w.Code != http.StatusConflict {
					t.Fatalf("one approval started Senior exam despite quorum=2: %d %s", w.Code, w.Body.String())
				}
				if rows := credentialList(); len(rows) != 0 {
					t.Fatalf("credential exists before a Senior award: %+v", rows)
				}
				return
			}

			started := startCatalogExam(t, catalogFixture{store: s, request: request}, exam.ID)
			payload := map[string]any{
				"attempt_token": started.Token,
				"revision":      started.Revision,
				"answers":       examAnswers(exam),
				"codes":         advancedPipelineExamCodes(t, exam),
			}
			submitted := request(http.MethodPost, "/api/exams/attempts/"+started.ID+"/submit", payload)
			if submitted.Code != http.StatusOK {
				t.Fatalf("submit complete Senior objectives: %d %s", submitted.Code, submitted.Body.String())
			}
			var examResult struct {
				Passed      bool   `json:"passed"`
				RankAwarded bool   `json:"rank_awarded"`
				Rank        string `json:"rank"`
				Score       int    `json:"score"`
			}
			if err := json.Unmarshal(submitted.Body.Bytes(), &examResult); err != nil {
				t.Fatal(err)
			}
			if !examResult.Passed || !examResult.RankAwarded || examResult.Rank != "Senior" || examResult.Score != 100 {
				t.Fatalf("Senior objective exam did not atomically award rank: %+v", examResult)
			}
			attempt, err := s.ExamAttempt("local", started.ID)
			if err != nil || attempt.Status != "completed" || !attempt.Passed {
				t.Fatalf("persisted Senior attempt = %+v, err=%v", attempt, err)
			}
			awards, err := s.RankAwards("local")
			if err != nil {
				t.Fatal(err)
			}
			seniorAwards := 0
			for _, award := range awards {
				if award.Rank == "Senior" && award.AttemptID == started.ID {
					seniorAwards++
				}
			}
			if seniorAwards != 1 {
				t.Fatalf("expected one atomic Senior rank award for attempt %s, got %+v", started.ID, awards)
			}

			issued := request(http.MethodPost, "/api/credentials", map[string]any{
				"attempt_id": started.ID, "name": "Псевдоним Senior", "consent": true,
			})
			if issued.Code != http.StatusCreated {
				t.Fatalf("issue Senior credential: %d %s", issued.Code, issued.Body.String())
			}
			var summary struct {
				ID        string `json:"id"`
				Published bool   `json:"published"`
				Rank      string `json:"rank"`
			}
			if err := json.Unmarshal(issued.Body.Bytes(), &summary); err != nil {
				t.Fatal(err)
			}
			if summary.ID == "" || summary.Published || summary.Rank != "Senior" {
				t.Fatalf("credential was not issued privately for Senior: %+v", summary)
			}
			records := credentialList()
			if len(records) != 1 || records[0]["id"] != summary.ID || records[0]["published"] != false {
				t.Fatalf("private credential not persisted exactly once: %+v", records)
			}
			if verify := request(http.MethodGet, fmt.Sprintf("/api/verify/%s", summary.ID), nil); verify.Code != http.StatusNotFound {
				t.Fatalf("private credential appeared in public verification: %d %s", verify.Code, verify.Body.String())
			}
			stored, err := s.Credential("local", summary.ID)
			if err != nil {
				t.Fatal(err)
			}
			var envelope credentials.Envelope
			if err := json.Unmarshal([]byte(stored.EnvelopeJSON), &envelope); err != nil {
				t.Fatal(err)
			}
			if err := credentials.Verify(issuerKey.Public().(ed25519.PublicKey), envelope); err != nil {
				t.Fatalf("test issuer public key rejected stored credential: %v", err)
			}
		})
	}
}
