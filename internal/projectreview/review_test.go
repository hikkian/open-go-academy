package projectreview

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func fixture(t *testing.T) (Attestation, ed25519.PublicKey, ed25519.PrivateKey, *Registry) {
	t.Helper()
	pub, priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	a := Attestation{Version: 1, ReviewerID: "reviewer-1", SubmissionID: "sub-123", EvaluationID: "eval-456", ProjectID: "p5", ProjectVersion: "1.2.0", ArtifactHash: Hash([]byte("artifact")), RubricVersion: "2026-01", RubricHash: Hash([]byte("rubric")), Decision: "approve", Comments: "Reviewed exact submitted artifact.", IssuedAt: "2026-10-16T12:00:00Z"}
	reg := &Registry{keys: map[string]ed25519.PublicKey{"reviewer-1": pub}}
	return a, pub, priv, reg
}

func TestCanonicalDeterministicAndSignature(t *testing.T) {
	a, _, priv, reg := fixture(t)
	first, err := canonical(a)
	if err != nil {
		t.Fatal(err)
	}
	second, err := canonical(a)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) || !strings.HasPrefix(string(first), domain) {
		t.Fatalf("canonical bytes differ or lack domain: %q", first)
	}
	want := domain + `{"version":1,"reviewer_id":"reviewer-1","submission_id":"sub-123","evaluation_id":"eval-456","project_id":"p5","project_version":"1.2.0","artifact_hash":"` + a.ArtifactHash + `","rubric_version":"2026-01","rubric_hash":"` + a.RubricHash + `","decision":"approve","comments":"Reviewed exact submitted artifact.","issued_at":"2026-10-16T12:00:00Z"}`
	if string(first) != want {
		t.Fatalf("canonical field order/encoding changed:\n got %q\nwant %q", first, want)
	}
	signed, err := Sign(a, priv)
	if err != nil {
		t.Fatal(err)
	}
	if err := Verify(reg, signed); err != nil {
		t.Fatal(err)
	}
	if _, err := Sign(signed, priv); err == nil {
		t.Fatal("resigning accepted")
	}
}

func TestVerifyRejectsTamperingBoundFields(t *testing.T) {
	a, _, priv, reg := fixture(t)
	signed, err := Sign(a, priv)
	if err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*Attestation){
		"artifact_hash":   func(a *Attestation) { a.ArtifactHash = Hash([]byte("changed artifact")) },
		"rubric_hash":     func(a *Attestation) { a.RubricHash = Hash([]byte("changed rubric")) },
		"rubric_version":  func(a *Attestation) { a.RubricVersion = "2026-02" },
		"evaluation_id":   func(a *Attestation) { a.EvaluationID = "eval-other" },
		"submission_id":   func(a *Attestation) { a.SubmissionID = "sub-other" },
		"project_id":      func(a *Attestation) { a.ProjectID = "p4" },
		"project_version": func(a *Attestation) { a.ProjectVersion = "2.0" },
		"decision":        func(a *Attestation) { a.Decision = "reject" },
		"comments":        func(a *Attestation) { a.Comments = "changed" },
		"issued_at":       func(a *Attestation) { a.IssuedAt = "2026-10-17T12:00:00Z" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := signed
			mutate(&changed)
			if err := Verify(reg, changed); err == nil {
				t.Fatal("tampering accepted")
			}
		})
	}
}

func TestVerifyRejectsNonCanonicalSignatureBase64(t *testing.T) {
	a, _, priv, reg := fixture(t)
	signed, err := Sign(a, priv)
	if err != nil {
		t.Fatal(err)
	}
	signed.Signature += "\n"
	if err := Verify(reg, signed); err == nil {
		t.Fatal("non-canonical signature encoding accepted")
	}
}

func TestVerifyRejectsUnknownReviewerAndMalformedAttestations(t *testing.T) {
	a, _, priv, reg := fixture(t)
	signed, err := Sign(a, priv)
	if err != nil {
		t.Fatal(err)
	}
	unknown := signed
	unknown.ReviewerID = "other-reviewer"
	if err := Verify(reg, unknown); err == nil {
		t.Fatal("unknown reviewer accepted")
	}
	bad := []struct {
		name   string
		mutate func(*Attestation)
	}{
		{"version", func(a *Attestation) { a.Version = 2 }}, {"reviewer", func(a *Attestation) { a.ReviewerID = "bad id" }},
		{"hash", func(a *Attestation) { a.ArtifactHash = "xyz" }}, {"decision", func(a *Attestation) { a.Decision = "maybe" }},
		{"comments", func(a *Attestation) { a.Comments = "bad\x00text" }}, {"signature", func(a *Attestation) { a.Signature = "!" }},
		{"timestamp", func(a *Attestation) { a.IssuedAt = "yesterday" }},
	}
	for _, test := range bad {
		t.Run(test.name, func(t *testing.T) {
			changed := signed
			test.mutate(&changed)
			if err := Verify(reg, changed); err == nil {
				t.Fatal("invalid attestation accepted")
			}
		})
	}
}

func TestLoadRegistryStrictAndValidated(t *testing.T) {
	pub, _, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	valid := `{"reviewers":[{"reviewer_id":"r1","public_key":"` + base64.StdEncoding.EncodeToString(pub) + `"}]}`
	reg, err := LoadRegistry(strings.NewReader(valid))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reg.keys["r1"]; !ok {
		t.Fatal("key absent")
	}
	cases := []string{
		`{}`, `{"reviewers":[]}`, `{"reviewers":[{"reviewer_id":"bad id","public_key":"` + base64.StdEncoding.EncodeToString(pub) + `"}]}`,
		`{"reviewers":[{"reviewer_id":"r","public_key":"short"}]}`,
		`{"reviewers":[{"reviewer_id":"r","public_key":"` + base64.StdEncoding.EncodeToString(pub) + `"},{"reviewer_id":"r","public_key":"` + base64.StdEncoding.EncodeToString(pub) + `"}]}`,
		`{"reviewers":[],"unexpected":true}`, `{"reviewers":[]} {}`,
	}
	for i, raw := range cases {
		t.Run(string(rune('a'+i)), func(t *testing.T) {
			if _, err := LoadRegistry(strings.NewReader(raw)); err == nil {
				t.Fatal("malformed registry accepted")
			}
		})
	}
	var roundTrip Attestation
	data, _ := json.Marshal(Attestation{Version: 1})
	if err := json.Unmarshal(data, &roundTrip); err != nil {
		t.Fatal(err)
	}
}
