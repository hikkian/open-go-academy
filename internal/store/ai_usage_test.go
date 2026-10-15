package store

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

func TestAIUsageReservationsPersistAndFinishOnce(t *testing.T) {
	p := filepath.Join(t.TempDir(), "ai.db")
	s, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	day := "2026-10-15"
	id, err := s.ReserveAI(day, 60, 100)
	if err != nil {
		t.Fatal(err)
	}
	u, err := s.AIUsage(day)
	if err != nil || u.ReservedTokens != 60 || u.Requests != 1 {
		t.Fatalf("reserved usage=%+v err=%v", u, err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	u, err = s.AIUsage(day)
	if err != nil || u.ReservedTokens != 60 || u.Requests != 1 {
		t.Fatalf("reopened usage=%+v err=%v", u, err)
	}
	if _, err = s.ReserveAI(day, 41, 100); !errors.Is(err, ErrAIBudget) {
		t.Fatalf("oversubscription err=%v", err)
	}
	if err = s.FinishAI(id, 2, 3, false, true); err != nil {
		t.Fatal(err)
	}
	if err = s.FinishAI(id, 2, 3, false, true); !errors.Is(err, ErrAIUsageConflict) {
		t.Fatalf("duplicate finish err=%v", err)
	}
	u, err = s.AIUsage(day)
	if err != nil || u.UsedTokens != 5 || u.ReservedTokens != 0 || u.Requests != 1 || u.FailedRequests != 0 {
		t.Fatalf("finished usage=%+v err=%v", u, err)
	}
	failID, err := s.ReserveAI(day, 10, 100)
	if err != nil {
		t.Fatal(err)
	}
	if err = s.FinishAI(failID, 1, 2, false, false); err != nil {
		t.Fatal(err)
	}
	u, err = s.AIUsage(day)
	if err != nil || u.UsedTokens != 15 || u.ReservedTokens != 0 || u.FailedRequests != 1 || u.EstimatedRequests != 1 {
		t.Fatalf("failed usage=%+v err=%v", u, err)
	}
	if _, err = s.ReserveAI("2026-10-16", 100, 100); err != nil {
		t.Fatalf("new day should have a separate budget: %v", err)
	}
}

func TestAIUsageConcurrentReservationsCannotExceedBudget(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "concurrent-ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	const workers = 20
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := s.ReserveAI("2026-10-15", 10, 100); errs <- e }()
	}
	wg.Wait()
	close(errs)
	reserved, denied := 0, 0
	for e := range errs {
		switch {
		case e == nil:
			reserved++
		case errors.Is(e, ErrAIBudget):
			denied++
		default:
			t.Fatalf("unexpected reservation error: %v", e)
		}
	}
	if reserved != 10 || denied != 10 {
		t.Fatalf("reserved=%d denied=%d, want 10 each", reserved, denied)
	}
	u, err := s.AIUsage("2026-10-15")
	if err != nil || u.ReservedTokens != 100 || u.Requests != 10 {
		t.Fatalf("usage=%+v err=%v", u, err)
	}
}

func TestAIUsageValidationAndPendingReservationSurvivesRestart(t *testing.T) {
	badDays := []string{"2026-1-01", "2026-02-30", "2026/10/15", ""}
	s, err := Open(filepath.Join(t.TempDir(), "validation-ai.db"))
	if err != nil {
		t.Fatal(err)
	}
	for _, day := range badDays {
		if _, e := s.AIUsage(day); e == nil {
			t.Errorf("AIUsage accepted invalid day %q", day)
		}
		if _, e := s.ReserveAI(day, 1, 10); e == nil {
			t.Errorf("ReserveAI accepted invalid day %q", day)
		}
	}
	for _, tc := range []struct{ tokens, budget int }{{0, 10}, {-1, 10}, {1, 0}, {1, -1}, {1, 1_000_001}, {1_000_001, 1_000_001}} {
		if _, e := s.ReserveAI("2026-10-15", tc.tokens, tc.budget); e == nil {
			t.Errorf("ReserveAI accepted tokens=%d budget=%d", tc.tokens, tc.budget)
		}
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "pending-ai.db")
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.ReserveAI("2026-10-15", 7, 7); err != nil {
		t.Fatal(err)
	}
	if err = s.Close(); err != nil {
		t.Fatal(err)
	}
	s, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err = s.ReserveAI("2026-10-15", 1, 7); !errors.Is(err, ErrAIBudget) {
		t.Fatalf("pending reserve was released on restart: %v", err)
	}
	if err = s.FinishAI(9999, 0, 0, false, true); !errors.Is(err, ErrAIUsageNotFound) {
		t.Fatalf("missing reservation err=%v", err)
	}
	if err = s.FinishAI(1, -1, 0, false, true); err == nil {
		t.Fatal("negative input usage accepted")
	}
}
