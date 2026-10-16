package httpapi

import (
	"academy/internal/content"
	"academy/internal/projectreview"
	"academy/internal/projects"
	"academy/internal/store"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type projectReviewFixture struct {
	t          *testing.T
	s          *store.Store
	h          http.Handler
	project    projects.Project
	submission store.ProjectSubmission
	evaluation store.ProjectEvaluation
	private    ed25519.PrivateKey
	public     ed25519.PublicKey
}

func newProjectReviewFixture(t *testing.T, withRegistry bool) *projectReviewFixture {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if _, err := s.CreateProfile("Ada", "light"); err != nil {
		t.Fatal(err)
	}
	p, ok := projects.Find("p3")
	if !ok {
		t.Fatal("p3 missing from catalog")
	}
	files := cloneProjectFiles(p.StarterFiles)
	submission, err := s.RecordProjectSubmission("local", "p3", files, files)
	if err != nil {
		t.Fatal(err)
	}
	criteria, ok := projects.CriteriaHash("p3")
	if !ok {
		t.Fatal("p3 criteria hash unavailable")
	}
	profile := store.ProjectEvaluationProfile{
		ProjectVersion: p.Version, CriteriaHash: criteria, CheckerVersion: "taskflow-v1",
		CheckerDigest: "sha256:" + strings.Repeat("a", 64), RuntimePolicyHash: "sha256:" + strings.Repeat("b", 64),
		RubricVersion: "taskflow-automatic-v1", RubricHash: "sha256:" + strings.Repeat("c", 64),
	}
	evaluation, err := s.RecordProjectEvaluation("local", submission.ID, profile, "automatic_pass", 0, true, `[{"name":"criteria.one","passed":true}]`)
	if err != nil {
		t.Fatal(err)
	}
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	var registry *projectreview.Registry
	if withRegistry {
		data := fmt.Sprintf(`{"reviewers":[{"reviewer_id":"operator-reviewer","public_key":%q}]}`, base64.StdEncoding.EncodeToString(public))
		registry, err = projectreview.LoadRegistry(strings.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
	}
	catalog, err := content.Load()
	if err != nil {
		t.Fatal(err)
	}
	api := New(s, catalog, nil, "review-session", "127.0.0.1:8081")
	api.ProjectReviewers = registry
	return &projectReviewFixture{t: t, s: s, h: api.Handler(t.TempDir()), project: p, submission: submission, evaluation: evaluation, private: private, public: public}
}

func (f *projectReviewFixture) request(method, path string, body any) *httptest.ResponseRecorder {
	f.t.Helper()
	var data []byte
	if body != nil {
		var err error
		data, err = json.Marshal(body)
		if err != nil {
			f.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, "http://127.0.0.1:8081"+path, bytes.NewReader(data))
	req.AddCookie(&http.Cookie{Name: "academy_session", Value: "review-session"})
	if method != http.MethodGet {
		req.Header.Set("X-CSRF-Token", "review-session")
	}
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	f.h.ServeHTTP(res, req)
	return res
}

func (f *projectReviewFixture) packet() map[string]any {
	f.t.Helper()
	res := f.request(http.MethodGet, "/api/projects/p3/submissions/"+f.submission.ID+"/review-packet", nil)
	if res.Code != http.StatusOK {
		f.t.Fatalf("review packet returned %d: %s", res.Code, res.Body.String())
	}
	var packet map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &packet); err != nil {
		f.t.Fatal(err)
	}
	return packet
}

func (f *projectReviewFixture) attestation(packet map[string]any) projectreview.Attestation {
	f.t.Helper()
	rubric, ok := packet["rubric"].(map[string]any)
	if !ok {
		f.t.Fatalf("review packet has no rubric: %#v", packet)
	}
	a := projectreview.Attestation{
		Version: projectreview.Version, ReviewerID: "operator-reviewer", SubmissionID: f.submission.ID,
		EvaluationID: f.evaluation.ID, ProjectID: f.project.ID, ProjectVersion: f.project.Version,
		ArtifactHash:  strings.TrimPrefix(f.submission.ArtifactHash, "sha256:"),
		RubricVersion: rubric["version"].(string), RubricHash: rubric["hash"].(string),
		Decision: "approve", Comments: "Рубрика проверена оператором.", IssuedAt: "2026-10-16T12:00:00Z",
	}
	signed, err := projectreview.Sign(a, f.private)
	if err != nil {
		f.t.Fatal(err)
	}
	return signed
}

func TestProjectReviewPacketAndSignedReviewAreBoundAndIdempotent(t *testing.T) {
	f := newProjectReviewFixture(t, true)
	packet := f.packet()
	if packet["project_id"] != "p3" || packet["submission_id"] != f.submission.ID || packet["evaluation_id"] != f.evaluation.ID || packet["artifact_hash"] != strings.TrimPrefix(f.submission.ArtifactHash, "sha256:") || packet["evaluation_status"] != "automatic_pass" {
		t.Fatalf("packet does not bind exact passing evidence: %#v", packet)
	}
	if _, ok := packet["files"].(map[string]any); !ok {
		t.Fatalf("packet omitted exact submitted files: %#v", packet["files"])
	}
	template, ok := packet["attestation_template"].(map[string]any)
	if !ok || template["artifact_hash"] != strings.TrimPrefix(f.submission.ArtifactHash, "sha256:") || template["evaluation_id"] != f.evaluation.ID || template["decision"] != "" || template["signature"] != "" {
		t.Fatalf("packet has no unsigned template bound to the same evidence: %#v", packet["attestation_template"])
	}
	a := f.attestation(packet)
	post := func(a projectreview.Attestation) *httptest.ResponseRecorder {
		return f.request(http.MethodPost, "/api/projects/p3/submissions/"+f.submission.ID+"/reviews", a)
	}
	first := post(a)
	if first.Code != http.StatusCreated {
		t.Fatalf("valid review returned %d: %s", first.Code, first.Body.String())
	}
	var firstResult map[string]any
	if err := json.Unmarshal(first.Body.Bytes(), &firstResult); err != nil {
		t.Fatal(err)
	}
	keyHash := sha256.Sum256(f.public)
	wantFingerprint := "sha256:" + hex.EncodeToString(keyHash[:])
	if firstResult["decision"] != "approve" || firstResult["reviewer_id"] != a.ReviewerID || firstResult["artifact_hash"] != a.ArtifactHash || firstResult["rank_awarded"] != false || firstResult["attestation_digest"] == "" || firstResult["reviewer_key_fingerprint"] != wantFingerprint {
		t.Fatalf("unexpected persisted review response: %#v", firstResult)
	}
	second := post(a)
	if second.Code != http.StatusCreated {
		t.Fatalf("exact replay returned %d: %s", second.Code, second.Body.String())
	}
	var secondResult map[string]any
	if err := json.Unmarshal(second.Body.Bytes(), &secondResult); err != nil {
		t.Fatal(err)
	}
	if secondResult["review_id"] != firstResult["review_id"] || secondResult["attestation_digest"] != firstResult["attestation_digest"] {
		t.Fatalf("exact replay was not idempotent: first=%#v second=%#v", firstResult, secondResult)
	}
	var reviews, awards int
	if err := f.s.DB().QueryRow(`SELECT COUNT(*) FROM project_reviews`).Scan(&reviews); err != nil {
		t.Fatal(err)
	}
	if err := f.s.DB().QueryRow(`SELECT COUNT(*) FROM rank_awards`).Scan(&awards); err != nil {
		t.Fatal(err)
	}
	if reviews != 1 || awards != 0 {
		t.Fatalf("reviews=%d rank awards=%d, want 1 and 0", reviews, awards)
	}
	listed := f.request(http.MethodGet, "/api/projects/p3/submissions/"+f.submission.ID+"/reviews", nil)
	if listed.Code != http.StatusOK {
		t.Fatalf("review history returned %d: %s", listed.Code, listed.Body.String())
	}
	var history struct {
		Reviews []map[string]any `json:"reviews"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &history); err != nil {
		t.Fatal(err)
	}
	if len(history.Reviews) != 1 || history.Reviews[0]["decision"] != "approve" || history.Reviews[0]["reviewer_key_fingerprint"] != wantFingerprint {
		t.Fatalf("review history did not show the persisted decision: %#v", history)
	}
	if _, leaksSignature := history.Reviews[0]["signature"]; leaksSignature {
		t.Fatal("review history exposed the raw signed attestation signature")
	}
}

func TestProjectReviewRejectsMissingRegistryUnknownKeyAndBadSignature(t *testing.T) {
	t.Run("missing registry", func(t *testing.T) {
		f := newProjectReviewFixture(t, false)
		packet := f.packet()
		got := f.request(http.MethodPost, "/api/projects/p3/submissions/"+f.submission.ID+"/reviews", f.attestation(packet))
		if got.Code != http.StatusServiceUnavailable {
			t.Fatalf("status=%d body=%s", got.Code, got.Body.String())
		}
	})
	t.Run("unknown reviewer", func(t *testing.T) {
		f := newProjectReviewFixture(t, true)
		packet := f.packet()
		a := f.attestation(packet)
		a.ReviewerID = "not-allowlisted"
		_, otherPrivate, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		a.Signature = ""
		a, err = projectreview.Sign(a, otherPrivate)
		if err != nil {
			t.Fatal(err)
		}
		got := f.request(http.MethodPost, "/api/projects/p3/submissions/"+f.submission.ID+"/reviews", a)
		if got.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status=%d body=%s", got.Code, got.Body.String())
		}
	})
	t.Run("tampered signature", func(t *testing.T) {
		f := newProjectReviewFixture(t, true)
		packet := f.packet()
		a := f.attestation(packet)
		a.Comments = "edited after signature"
		got := f.request(http.MethodPost, "/api/projects/p3/submissions/"+f.submission.ID+"/reviews", a)
		if got.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status=%d body=%s", got.Code, got.Body.String())
		}
	})
	t.Run("noncanonical signature base64", func(t *testing.T) {
		f := newProjectReviewFixture(t, true)
		packet := f.packet()
		a := f.attestation(packet)
		// Go's base64 decoder ignores CR/LF. The API must still reject this
		// noncanonical encoding so exact signed JSON remains replay-stable.
		a.Signature += "\n"
		got := f.request(http.MethodPost, "/api/projects/p3/submissions/"+f.submission.ID+"/reviews", a)
		if got.Code != http.StatusUnprocessableEntity {
			t.Fatalf("status=%d body=%s", got.Code, got.Body.String())
		}
	})
}

func TestProjectReviewRejectsMismatchedRubricAndEvidenceBindings(t *testing.T) {
	f := newProjectReviewFixture(t, true)
	packet := f.packet()
	base := f.attestation(packet)
	post := func(path string, a projectreview.Attestation) *httptest.ResponseRecorder {
		return f.request(http.MethodPost, path, a)
	}
	cases := []struct {
		name, path string
		mutate     func(*projectreview.Attestation)
		want       int
	}{
		{"rubric version", "/api/projects/p3/submissions/" + f.submission.ID + "/reviews", func(a *projectreview.Attestation) { a.RubricVersion += "-other" }, http.StatusUnprocessableEntity},
		{"rubric hash", "/api/projects/p3/submissions/" + f.submission.ID + "/reviews", func(a *projectreview.Attestation) { a.RubricHash = strings.Repeat("0", 64) }, http.StatusUnprocessableEntity},
		{"artifact hash", "/api/projects/p3/submissions/" + f.submission.ID + "/reviews", func(a *projectreview.Attestation) { a.ArtifactHash = strings.Repeat("0", 64) }, http.StatusConflict},
		{"evaluation id", "/api/projects/p3/submissions/" + f.submission.ID + "/reviews", func(a *projectreview.Attestation) { a.EvaluationID = "unknown-evaluation" }, http.StatusNotFound},
		{"submission path", "/api/projects/p3/submissions/other-submission/reviews", func(*projectreview.Attestation) {}, http.StatusUnprocessableEntity},
		{"project path", "/api/projects/p4/submissions/" + f.submission.ID + "/reviews", func(*projectreview.Attestation) {}, http.StatusUnprocessableEntity},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := base
			tc.mutate(&a)
			signed, err := projectreview.Sign(projectreview.Attestation{
				Version: a.Version, ReviewerID: a.ReviewerID, SubmissionID: a.SubmissionID, EvaluationID: a.EvaluationID,
				ProjectID: a.ProjectID, ProjectVersion: a.ProjectVersion, ArtifactHash: a.ArtifactHash,
				RubricVersion: a.RubricVersion, RubricHash: a.RubricHash, Decision: a.Decision, Comments: a.Comments, IssuedAt: a.IssuedAt,
			}, f.private)
			if err != nil {
				t.Fatal(err)
			}
			got := post(tc.path, signed)
			if got.Code != tc.want {
				t.Fatalf("status=%d want=%d body=%s", got.Code, tc.want, got.Body.String())
			}
		})
	}
	var reviews int
	if err := f.s.DB().QueryRow(`SELECT COUNT(*) FROM project_reviews`).Scan(&reviews); err != nil {
		t.Fatal(err)
	}
	if reviews != 0 {
		t.Fatalf("rejected bindings persisted %d reviews", reviews)
	}
}
