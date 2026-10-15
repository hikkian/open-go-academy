package store

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func passedCredentialAttempt(t *testing.T, s *Store) ExamAttempt {
	t.Helper()
	a, err := s.StartExam("local", "zero-final", "1", "token", nil)
	if err != nil {
		t.Fatal(err)
	}
	a, err = s.ClaimExamAttempt("local", "zero-final", a.ID, "token", a.Revision, `{}`, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	a, err = s.FinishExamAttempt("local", a.ID, 100, true, `[]`)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func credentialFor(a ExamAttempt) Credential {
	return Credential{ID: strings.Repeat("a", 32), ProfileID: "local", AttemptID: a.ID, ExamID: a.ExamID, Version: a.Version, Rank: a.Rank, CriteriaHash: a.CriteriaHash, KeyID: strings.Repeat("b", 64), EnvelopeJSON: `{"payload":"signed"}`, Published: true}
}

func TestCredentialRequiresVerifiedAwardAndCannotBeForgedOrReissued(t *testing.T) {
	s, _ := newExamStore(t)
	active, err := s.StartExam("local", "zero-final", "1", "token", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.IssueCredential("local", credentialFor(active)); !errors.Is(err, ErrCredentialPrerequisites) {
		t.Fatalf("unverified attempt issued credential: %v", err)
	}
	if err = s.CancelExamAttempt("local", active.ID); err != nil {
		t.Fatal(err)
	}
	passed := passedCredentialAttempt(t, s)
	forged := credentialFor(passed)
	forged.Rank = "Senior"
	if _, err = s.IssueCredential("local", forged); !errors.Is(err, ErrCredentialPrerequisites) {
		t.Fatalf("forged rank accepted: %v", err)
	}
	valid := credentialFor(passed)
	issued, err := s.IssueCredential("local", valid)
	if err != nil {
		t.Fatal(err)
	}
	if issued.Published {
		t.Fatal("issue honored caller's published=true")
	}
	if _, err = s.IssueCredential("local", valid); !errors.Is(err, ErrCredentialConflict) {
		t.Fatalf("duplicate attempt issue err=%v", err)
	}
	if _, err = s.IssueCredential("other", valid); !errors.Is(err, ErrCredentialPrerequisites) {
		t.Fatalf("foreign profile issue err=%v", err)
	}
	if _, err = s.Credential("other", issued.ID); !errors.Is(err, ErrCredentialNotFound) {
		t.Fatalf("cross-profile read err=%v", err)
	}
	if _, err = s.PublicCredential(issued.ID); !errors.Is(err, ErrCredentialNotFound) {
		t.Fatalf("private credential exposed: %v", err)
	}
	tooLarge := valid
	tooLarge.EnvelopeJSON = `{"payload":"` + strings.Repeat("x", MaxCredentialEnvelopeBytes) + `"}`
	if _, err = s.IssueCredential("local", tooLarge); err == nil {
		t.Fatal("oversize envelope accepted")
	}
	invalidID := valid
	invalidID.ID = "BAD"
	if _, err = s.IssueCredential("local", invalidID); err == nil {
		t.Fatal("invalid credential ID accepted")
	}
}

func TestCredentialPrivacyPersistenceAuditAndRevocation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "credentials.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.CreateProfile("Ada", "dark"); err != nil {
		t.Fatal(err)
	}
	if err = s.RegisterExam("zero-final", "1", "Zero", "criteria"); err != nil {
		t.Fatal(err)
	}
	if _, err = s.RecordAttempt("hello", map[string]string{}, true, "ok"); err != nil {
		t.Fatal(err)
	}
	awarded := passedCredentialAttempt(t, s)
	issued, err := s.IssueCredential("local", credentialFor(awarded))
	if err != nil {
		t.Fatal(err)
	}
	private, err := s.PassportPublic("local")
	if err != nil || private {
		t.Fatalf("passport default=%v err=%v", private, err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	reloaded, err := s.Credential("local", issued.ID)
	if err != nil || reloaded.Published {
		t.Fatalf("persisted credential=%+v err=%v", reloaded, err)
	}
	if err = s.SetCredentialPublished("local", issued.ID, true); err != nil {
		t.Fatal(err)
	}
	if err = s.SetPassportPublic("local", true); err != nil {
		t.Fatal(err)
	}
	public, err := s.PassportPublic("local")
	if err != nil || !public {
		t.Fatalf("passport public=%v err=%v", public, err)
	}
	if _, err = s.PublicCredential(issued.ID); err != nil {
		t.Fatalf("published credential unavailable: %v", err)
	}
	if err = s.RevokeCredential("local", issued.ID, "operator requested revocation"); err != nil {
		t.Fatal(err)
	}
	if err = s.RevokeCredential("local", issued.ID, "duplicate reason"); err != nil {
		t.Fatalf("idempotent revoke: %v", err)
	}
	publicAfterRevoke, err := s.PublicCredential(issued.ID)
	if err != nil || publicAfterRevoke.RevokedAt == nil || !publicAfterRevoke.Published {
		t.Fatalf("published revoked credential should remain verifiable: %+v err=%v", publicAfterRevoke, err)
	}
	got, err := s.Credential("local", issued.ID)
	if err != nil || got.RevokedAt == nil || !got.Published {
		t.Fatalf("revoked credential=%+v err=%v", got, err)
	}
	credentials, err := s.Credentials("local")
	if err != nil || len(credentials) != 1 {
		t.Fatalf("credentials=%v err=%v", credentials, err)
	}
	var events int
	var revoked int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM credential_audit WHERE credential_id=?`, issued.ID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM credential_audit WHERE credential_id=? AND action='revoked'`, issued.ID).Scan(&revoked); err != nil {
		t.Fatal(err)
	}
	if events != 3 || revoked != 1 {
		t.Fatalf("credential audit events=%d revoked=%d", events, revoked)
	}
	var auditHasReason int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('credential_audit') WHERE name='reason'`).Scan(&auditHasReason); err != nil {
		t.Fatal(err)
	}
	if auditHasReason != 0 {
		t.Fatal("free-form revocation reason is stored in audit")
	}
	var envelope any
	if err = json.Unmarshal([]byte(got.EnvelopeJSON), &envelope); err != nil {
		t.Fatalf("envelope did not persist: %v", err)
	}
}

func TestPassportPrivacyRequiresLocalProfileAndCredentialConflicts(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "empty.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.PassportPublic("local"); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("missing profile privacy query err=%v", err)
	}
	if err = s.SetPassportPublic("local", true); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("missing profile privacy set err=%v", err)
	}
	if _, err = s.Credential("local", "missing"); !errors.Is(err, ErrCredentialNotFound) {
		t.Fatalf("missing credential err=%v", err)
	}
}
