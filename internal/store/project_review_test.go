package store

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"academy/internal/projectreview"
	"academy/internal/projects"
	"academy/internal/runner"
)

func reviewFixture(t *testing.T) (*Store, ProjectSubmission, ProjectEvaluation, ed25519.PrivateKey, *projectreview.Registry) {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "reviews.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if _, err = s.CreateProfile("Ada", "light"); err != nil {
		t.Fatal(err)
	}
	p, ok := projects.Find("p3")
	if !ok {
		t.Fatal("P3 project not found")
	}
	files := cloneFiles(p.StarterFiles)
	sub, err := s.RecordProjectSubmission("local", "p3", files, files)
	if err != nil {
		t.Fatal(err)
	}
	criteria, ok := projects.CriteriaHash("p3")
	if !ok {
		t.Fatal("P3 criteria missing")
	}
	profile := ProjectEvaluationProfile{ProjectVersion: p.Version, CriteriaHash: criteria,
		CheckerVersion: "taskflow-v1", CheckerDigest: "sha256:" + strings.Repeat("a", 64),
		RuntimePolicyHash: "sha256:" + strings.Repeat("b", 64), RubricVersion: "taskflow-automatic-v1", RubricHash: "sha256:" + strings.Repeat("c", 64)}
	checks, _ := json.Marshal([]runner.Check{{Name: "p3.acceptance", Passed: true}})
	ev, err := s.RecordProjectEvaluation("local", sub.ID, profile, "automatic_pass", 0, true, string(checks))
	if err != nil {
		t.Fatal(err)
	}
	pub, priv, err := projectreview.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	registryJSON, _ := json.Marshal(map[string]any{"reviewers": []any{map[string]string{"reviewer_id": "operator-reviewer-1", "public_key": base64.StdEncoding.EncodeToString(pub)}}})
	registry, err := projectreview.LoadRegistry(strings.NewReader(string(registryJSON)))
	if err != nil {
		t.Fatal(err)
	}
	return s, sub, ev, priv, registry
}

func signedReview(t *testing.T, sub ProjectSubmission, ev ProjectEvaluation, priv ed25519.PrivateKey) projectreview.Attestation {
	t.Helper()
	a, err := projectreview.Sign(projectreview.Attestation{Version: projectreview.Version, ReviewerID: "operator-reviewer-1",
		SubmissionID: sub.ID, EvaluationID: ev.ID, ProjectID: sub.ProjectID, ProjectVersion: sub.ProjectVersion,
		ArtifactHash: strings.TrimPrefix(sub.ArtifactHash, "sha256:"), RubricVersion: "p3-human-review-v1",
		RubricHash: strings.Repeat("f", 64), Decision: "approve", Comments: "Reviewed exact artifact.", IssuedAt: "2026-10-16T12:00:00Z"}, priv)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestRecordProjectReviewBindsVerifiedImmutableEvidenceAndExactReplay(t *testing.T) {
	s, sub, ev, priv, registry := reviewFixture(t)
	a := signedReview(t, sub, ev, priv)
	if err := projectreview.Verify(registry, a); err != nil {
		t.Fatalf("verify signature: %v", err)
	}
	first, err := s.RecordProjectReview("local", a)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.RecordProjectReview("local", a)
	if err != nil || replay.ID != first.ID || replay.AttestationDigest != first.AttestationDigest {
		t.Fatalf("exact replay = %+v, err=%v", replay, err)
	}
	read, err := s.ProjectReview("local", first.ID)
	if err != nil || read.Attestation != a || read.AttestationDigest != first.AttestationDigest {
		t.Fatalf("read = %+v, err=%v", read, err)
	}
	if _, err = s.RecordProjectReview("other", a); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("foreign profile err=%v", err)
	}
	if _, err = s.DB().Exec(`UPDATE project_reviews SET comments='rewritten' WHERE id=?`, first.ID); err == nil {
		t.Fatal("review UPDATE succeeded")
	}
	if _, err = s.DB().Exec(`DELETE FROM project_reviews WHERE id=?`, first.ID); err == nil {
		t.Fatal("review DELETE succeeded")
	}
	if got, err := s.ProjectReview("local", first.ID); err != nil || got.Attestation.Comments != a.Comments {
		t.Fatalf("immutable review changed: %+v %v", got, err)
	}
}

func TestRecordProjectReviewRejectsUnboundAndUntrustedEvidence(t *testing.T) {
	s, sub, ev, priv, registry := reviewFixture(t)
	a := signedReview(t, sub, ev, priv)
	if err := projectreview.Verify(registry, a); err != nil {
		t.Fatalf("verify signature before persistence: %v", err)
	}
	if _, err := s.RecordProjectReview("local", a); err != nil {
		t.Fatalf("valid signed attestation rejected: %v", err)
	}
	bad := a
	bad.Signature = strings.Repeat("A", len(a.Signature))
	if err := projectreview.Verify(registry, bad); err == nil {
		t.Fatal("tampered signature verified")
	}
	bad = signedReview(t, sub, ev, priv)
	bad.ProjectID = "p2"
	if _, err := s.RecordProjectReview("local", bad); err == nil {
		t.Fatal("unsupported project accepted")
	}
	bad = signedReview(t, sub, ev, priv)
	bad.ArtifactHash = strings.Repeat("d", 64)
	if _, err := s.RecordProjectReview("local", bad); err == nil {
		t.Fatal("stale artifact accepted")
	}
	bad = signedReview(t, sub, ev, priv)
	bad.EvaluationID = "missing-evaluation"
	if _, err := s.RecordProjectReview("local", bad); !errors.Is(err, ErrProjectEvidenceNotFound) {
		t.Fatalf("mismatched evaluation err=%v", err)
	}
	_, err := s.DB().Exec(`INSERT INTO project_reviews(id,profile_id,submission_id,evaluation_id,project_id,project_version,
		artifact_hash,reviewer_id,attestation_version,rubric_version,rubric_hash,decision,comments,issued_at,signature,
		attestation_digest,attestation_json,created_at) VALUES('manual-fk-check','local',?,?,'p3',?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		sub.ID, "no-such-evaluation", sub.ProjectVersion, sub.ArtifactHash, "operator-reviewer-1", projectreview.Version,
		"p3-human-review-v1", strings.Repeat("f", 64), "approve", "test", "2026-10-16T12:00:00Z", "signature",
		"sha256:"+strings.Repeat("1", 64), `{}`, "2026-10-16T12:00:00Z")
	if err == nil {
		t.Fatal("database accepted review with an evaluation not linked to the exact submission")
	}
}
