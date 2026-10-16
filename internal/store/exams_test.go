package store

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func newExamStore(t *testing.T) (*Store, string) {
	t.Helper()
	s, e := Open(filepath.Join(t.TempDir(), "exam.db"))
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = s.Close() })
	if _, e = s.CreateProfile("Ada", "dark"); e != nil {
		t.Fatal(e)
	}
	if e = s.RegisterExam("zero-final", "1", "Zero", "sha256:criteria-v1"); e != nil {
		t.Fatal(e)
	}
	if _, e = s.RecordAttempt("hello", map[string]string{"main.go": "package main"}, true, "ok"); e != nil {
		t.Fatal(e)
	}
	return s, "zero-final"
}

func TestExamUpgradeAttemptClaimAwardAndImmutableVersion(t *testing.T) {
	p := filepath.Join(t.TempDir(), "upgrade.db")
	s, e := Open(p)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CreateProfile("Ada", "dark"); e != nil {
		t.Fatal(e)
	}
	if e = s.RegisterExam("zero-final", "1", "Zero", "hash-one"); e != nil {
		t.Fatal(e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s, e = Open(p)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.RegisterExam("zero-final", "1", "Zero", "hash-two"); e == nil {
		t.Fatal("criteria version changed")
	}
	// Re-establish prerequisite evidence for this upgraded-store fixture.
	if _, e = s.RecordAttempt("hello", map[string]string{}, true, "ok"); e != nil {
		t.Fatal(e)
	}
	a, e := s.StartExam("local", "zero-final", "1", "secret-hash", []string{"hello"})
	if e != nil {
		t.Fatal(e)
	}
	b, _ := json.Marshal(a)
	if string(b) == "" || contains(string(b), "secret-hash") {
		t.Fatal("token hash leaked in API value")
	}
	a, e = s.ClaimExamAttempt("local", "zero-final", a.ID, "secret-hash", a.Revision, `{"q1":"a"}`, `{"main.go":"package main"}`)
	if e != nil || a.Status != "grading" {
		t.Fatalf("claim=%+v err=%v", a, e)
	}
	if _, e = s.ClaimExamAttempt("local", "zero-final", a.ID, "secret-hash", a.Revision, `{}`, `{}`); !errors.Is(e, ErrExamAttemptConflict) {
		t.Fatalf("replay claim err=%v", e)
	}
	if _, e = s.FinishExamAttempt("local", a.ID, 100, true, `{"checks":["all"]}`); e != nil {
		t.Fatal(e)
	}
	awards, e := s.RankAwards("local")
	if e != nil || len(awards) != 1 {
		t.Fatalf("awards=%v err=%v", awards, e)
	}
	profile, e := s.Profile()
	if e != nil || profile.Rank != "Zero" {
		t.Fatalf("rank=%+v err=%v", profile, e)
	}
}

func TestExamPrerequisiteProfileAndFailDoNotAward(t *testing.T) {
	s, _ := newExamStore(t)
	if _, e := s.StartExam("forged", "zero-final", "1", "hash", nil); !errors.Is(e, ErrProfileNotFound) {
		t.Fatalf("foreign profile err=%v", e)
	}
	if _, e := s.StartExam("local", "zero-final", "1", "hash", []string{"missing"}); !errors.Is(e, ErrExamPrerequisites) {
		t.Fatalf("prereq err=%v", e)
	}
	a, e := s.StartExam("local", "zero-final", "1", "hash", []string{"hello"})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.ClaimExamAttempt("local", "zero-final", a.ID, "wrong", a.Revision, `{}`, `{}`); !errors.Is(e, ErrExamToken) {
		t.Fatalf("token err=%v", e)
	}
	if _, e = s.ClaimExamAttempt("local", "zero-final", a.ID, "hash", a.Revision, `{}`, `{}`); e != nil {
		t.Fatal(e)
	}
	if _, e = s.FinishExamAttempt("local", a.ID, 99, false, `[]`); e != nil {
		t.Fatal(e)
	}
	if awards, e := s.RankAwards("local"); e != nil || len(awards) != 0 {
		t.Fatalf("failure awarded rank: %v %v", awards, e)
	}
	if _, e = s.StartExam("local", "zero-final", "1", "other", nil); e != nil {
		t.Fatalf("failed retake denied: %v", e)
	}
}

func TestAdvancedExamCannotAwardWithoutProjectEvidencePolicy(t *testing.T) {
	s, _ := newExamStore(t)
	for _, rank := range []string{"Middle", "Middle+", "Senior"} {
		t.Run(rank, func(t *testing.T) {
			examID := strings.ToLower(strings.ReplaceAll(rank, "+", "plus")) + "-final"
			token := examID + "-token"
			if err := s.RegisterExam(examID, "1", rank, examID+"-hash"); err != nil {
				t.Fatal(err)
			}
			attempt, err := s.StartExam("local", examID, "1", token, nil)
			if err != nil {
				t.Fatal(err)
			}
			claimed, err := s.ClaimExamAttempt("local", examID, attempt.ID, token, attempt.Revision, `{"answer":"a"}`, `{"main.go":"package main"}`)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = s.FinishExamAttempt("local", attempt.ID, 100, true, `[{"passed":true}]`); !errors.Is(err, ErrRankEvidenceRequired) {
				t.Fatalf("advanced exam auto-award err=%v, want project evidence gate", err)
			}
			pending, err := s.ExamAttempt("local", attempt.ID)
			if err != nil || pending.Status != "grading" || pending.Score != 0 || pending.Passed || pending.ChecksJSON != claimed.ChecksJSON || pending.AnswersJSON != claimed.AnswersJSON || pending.CodesJSON != claimed.CodesJSON {
				t.Fatalf("rejected advanced result mutated the attempt: %+v err=%v", pending, err)
			}
			if err = s.InterruptExamAttempt("local", attempt.ID); err != nil {
				t.Fatal(err)
			}
			recovered, err := s.ExamAttempt("local", attempt.ID)
			if err != nil || recovered.Status != "interrupted" || recovered.Score != 0 || recovered.Passed || recovered.AnswersJSON != claimed.AnswersJSON || recovered.CodesJSON != claimed.CodesJSON {
				t.Fatalf("interrupted advanced attempt did not preserve retry input: %+v err=%v", recovered, err)
			}
			retry, err := s.StartExam("local", examID, "1", token+"-retry", nil)
			if err != nil {
				t.Fatalf("new attempt denied after interrupted advanced pass: %v", err)
			}
			if err = s.CancelExamAttempt("local", retry.ID); err != nil {
				t.Fatal(err)
			}
		})
	}
	awards, err := s.RankAwards("local")
	if err != nil || len(awards) != 0 {
		t.Fatalf("advanced exam created award without project evidence: %v %v", awards, err)
	}
	profile, err := s.Profile()
	if err != nil || profile.Rank != "Zero" {
		t.Fatalf("advanced exam changed rank without evidence: %+v %v", profile, err)
	}
}

func TestExamRankPrerequisiteRequiresPersistedSameProfileAward(t *testing.T) {
	s, _ := newExamStore(t) // Includes XP and verified course progress, but no rank award.
	if err := s.RegisterExam("junior-final", "1", "Junior", "junior-hash"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.StartExam("local", "junior-final", "1", "junior-token", nil, "Zero"); !errors.Is(err, ErrExamPrerequisites) {
		t.Fatalf("XP/progress incorrectly satisfied Zero rank: %v", err)
	}
	// Legacy callers, including the Zero exam with no rank prerequisite, remain valid.
	zero, err := s.StartExam("local", "zero-final", "1", "zero-token", nil)
	if err != nil {
		t.Fatalf("legacy Zero start: %v", err)
	}
	zero, err = s.ClaimExamAttempt("local", zero.ExamID, zero.ID, "zero-token", zero.Revision, `{}`, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.FinishExamAttempt("local", zero.ID, 100, true, `[]`); err != nil {
		t.Fatal(err)
	}
	junior, err := s.StartExam("local", "junior-final", "1", "junior-token", nil, "Zero")
	if err != nil {
		t.Fatalf("persisted Zero award not accepted: %v", err)
	}
	if _, err = s.StartExam("other", "junior-final", "1", "another-token", nil, "Zero"); !errors.Is(err, ErrProfileNotFound) {
		t.Fatalf("cross-profile request used local rank award: %v", err)
	}
	var evidence struct {
		RankAwards map[string]rankAwardEvidence `json:"rank_awards"`
	}
	if err = json.Unmarshal([]byte(junior.EvidenceJSON), &evidence); err != nil {
		t.Fatal(err)
	}
	proof := evidence.RankAwards["Zero"]
	if proof.ExamID != "zero-final" || proof.Version != "1" || proof.AttemptID != zero.ID {
		t.Fatalf("rank award evidence=%+v", proof)
	}
	junior, err = s.ClaimExamAttempt("local", junior.ExamID, junior.ID, "junior-token", junior.Revision, `{}`, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`DELETE FROM rank_awards WHERE profile_id='local' AND rank='Zero'`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.FinishExamAttempt("local", junior.ID, 100, true, `[]`); err == nil {
		t.Fatal("finish accepted after previous-rank award was removed")
	}
	stored, err := s.ExamAttempt("local", junior.ID)
	if err != nil || stored.Status != "grading" {
		t.Fatalf("failed award transaction changed attempt: %+v err=%v", stored, err)
	}
	awards, err := s.RankAwards("local")
	if err != nil || len(awards) != 0 {
		t.Fatalf("failed finish added rank award: %v %v", awards, err)
	}
}

func TestExamRankPrerequisiteAwardMustRemainAtClaimAndBelongToProfile(t *testing.T) {
	s, _ := newExamStore(t)
	if err := s.RegisterExam("junior-final", "1", "Junior", "junior-hash"); err != nil {
		t.Fatal(err)
	}
	zero, err := s.StartExam("local", "zero-final", "1", "zero-token", nil)
	if err != nil {
		t.Fatal(err)
	}
	zero, err = s.ClaimExamAttempt("local", zero.ExamID, zero.ID, "zero-token", zero.Revision, `{}`, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.FinishExamAttempt("local", zero.ID, 100, true, `[]`); err != nil {
		t.Fatal(err)
	}
	junior, err := s.StartExam("local", "junior-final", "1", "junior-token", nil, "Zero")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.db.Exec(`DELETE FROM rank_awards WHERE profile_id='local' AND rank='Zero'`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.ClaimExamAttempt("local", junior.ExamID, junior.ID, "junior-token", junior.Revision, `{}`, `{}`); err == nil {
		t.Fatal("claim accepted after previous-rank award was removed")
	}
}

func TestExamConcurrentStartClaimAndRecovery(t *testing.T) {
	s, _ := newExamStore(t)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := s.StartExam("local", "zero-final", "1", "hash", nil); errs <- e }()
	}
	wg.Wait()
	close(errs)
	ok := 0
	for e := range errs {
		if e == nil {
			ok++
		}
	}
	if ok != 1 {
		t.Fatalf("successful concurrent starts=%d, want 1", ok)
	}
	as, e := s.ExamAttempts("local")
	if e != nil || len(as) != 1 {
		t.Fatalf("attempts=%v err=%v", as, e)
	}
	if _, e = s.ClaimExamAttempt("local", "zero-final", as[0].ID, "hash", as[0].Revision, `{}`, `{}`); e != nil {
		t.Fatal(e)
	}
	if e = s.RecoverExamAttempts(); e != nil {
		t.Fatal(e)
	}
	a, e := s.ExamAttempt("local", as[0].ID)
	if e != nil || a.Status != "interrupted" {
		t.Fatalf("recovered=%+v err=%v", a, e)
	}
}

func TestExamResultConsistencyAndDraftState(t *testing.T) {
	s, _ := newExamStore(t)
	a, e := s.StartExam("local", "zero-final", "1", "hash", nil)
	if e != nil {
		t.Fatal(e)
	}
	a, e = s.SaveExamDraft("local", a.ID, "hash", a.Revision, `{"x":1}`, `{}`)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.FinishExamAttempt("local", a.ID, 100, true, `[]`); !errors.Is(e, ErrExamAttemptConflict) {
		t.Fatalf("active finish err=%v", e)
	}
	if _, e = s.ClaimExamAttempt("local", "zero-final", a.ID, "hash", a.Revision, `{}`, `{}`); e != nil {
		t.Fatal(e)
	}
	if _, e = s.FinishExamAttempt("local", a.ID, 99, true, `[]`); e == nil {
		t.Fatal("inconsistent score/pass accepted")
	}
}

func TestExamDraftRevisionRejectsStaleTabAndSubmit(t *testing.T) {
	s, _ := newExamStore(t)
	firstTab, err := s.StartExam("local", "zero-final", "1", "hash", nil)
	if err != nil {
		t.Fatal(err)
	}
	secondTab := firstTab // Both tabs loaded revision zero.
	firstTab, err = s.SaveExamDraft("local", firstTab.ID, "hash", firstTab.Revision, `{"q":"newer"}`, `{"main.go":"newer source"}`)
	if err != nil {
		t.Fatal(err)
	}
	if firstTab.Revision != 1 {
		t.Fatalf("saved revision=%d, want 1", firstTab.Revision)
	}
	if _, err = s.SaveExamDraft("local", secondTab.ID, "hash", secondTab.Revision, `{"q":"stale"}`, `{"main.go":"stale source"}`); !errors.Is(err, ErrExamAttemptConflict) {
		t.Fatalf("stale draft save err=%v", err)
	}
	if _, err = s.ClaimExamAttempt("local", "zero-final", firstTab.ID, "hash", secondTab.Revision, `{"q":"stale submit"}`, `{}`); !errors.Is(err, ErrExamAttemptConflict) {
		t.Fatalf("stale submit err=%v", err)
	}
	current, err := s.ExamAttempt("local", firstTab.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Revision != 1 || current.AnswersJSON != `{"q":"newer"}` || current.CodesJSON != `{"main.go":"newer source"}` {
		t.Fatalf("newer draft was overwritten: %+v", current)
	}
	claimed, err := s.ClaimExamAttempt("local", "zero-final", current.ID, "hash", current.Revision, current.AnswersJSON, current.CodesJSON)
	if err != nil {
		t.Fatal(err)
	}
	if claimed.Status != "grading" || claimed.Revision != 2 {
		t.Fatalf("claim=%+v", claimed)
	}
}

func TestExamPayloadLimitAndBoundedHistory(t *testing.T) {
	s, _ := newExamStore(t)
	a, err := s.StartExam("local", "zero-final", "1", "hash", nil)
	if err != nil {
		t.Fatal(err)
	}
	largeCode := `{"main.go":"` + strings.Repeat("x", 300*1024) + `"}`
	if _, err = s.ClaimExamAttempt("local", "zero-final", a.ID, "hash", a.Revision, `{}`, largeCode); err != nil {
		t.Fatalf("valid 300KiB source rejected: %v", err)
	}
	if _, err = s.FinishExamAttempt("local", a.ID, 100, true, `[]`); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 105; i++ {
		a, err = s.StartExam("local", "zero-final", "1", "hash", nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = s.ClaimExamAttempt("local", "zero-final", a.ID, "hash", a.Revision, `{}`, `{}`); err != nil {
			t.Fatal(err)
		}
		if _, err = s.FinishExamAttempt("local", a.ID, 0, false, `[]`); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.ExamAttempts("local")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 100 {
		t.Fatalf("history returned %d rows, want 100", len(got))
	}
	var stored, awards int
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM exam_attempts`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if err = s.db.QueryRow(`SELECT COUNT(*) FROM rank_awards`).Scan(&awards); err != nil {
		t.Fatal(err)
	}
	if stored != 101 || awards != 1 {
		t.Fatalf("stored=%d awards=%d; expected 100 recent attempts plus protected award evidence", stored, awards)
	}
	all, err := s.RankAwards("local")
	if err != nil || len(all) != 1 {
		t.Fatalf("award evidence lost: %v %v", all, err)
	}
	if _, err = s.ExamAttempt("local", all[0].AttemptID); err != nil {
		t.Fatalf("protected award attempt lost: %v", err)
	}
}
