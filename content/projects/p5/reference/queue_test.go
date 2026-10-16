package main

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func event() Event {
	return Event{ID: "evt-7", TenantID: "tenant-a", Payload: []byte("order.updated:v1")}
}

func TestEnqueueIdempotencyIsTenantScopedAndPayloadStable(t *testing.T) {
	q := NewQueue()
	if inserted, err := q.Enqueue(event()); err != nil || !inserted {
		t.Fatalf("first enqueue = %v, %v", inserted, err)
	}
	if inserted, err := q.Enqueue(event()); err != nil || inserted {
		t.Fatalf("replay = %v, %v; want false, nil", inserted, err)
	}
	conflict := event()
	conflict.Payload = []byte("different")
	if _, err := q.Enqueue(conflict); !errors.Is(err, ErrPayloadConflict) {
		t.Fatalf("conflicting replay error = %v", err)
	}
	otherTenant := event()
	otherTenant.TenantID = "tenant-b"
	if inserted, err := q.Enqueue(otherTenant); err != nil || !inserted {
		t.Fatalf("other tenant enqueue = %v, %v", inserted, err)
	}
}

func TestConcurrentWorkersCannotClaimSameLiveLease(t *testing.T) {
	q := NewQueue()
	if _, err := q.Enqueue(event()); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0)
	const workers = 32
	var wg sync.WaitGroup
	var mu sync.Mutex
	claims := 0
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			claim, err := q.Claim(workerName(i), now, time.Minute)
			if err != nil {
				t.Errorf("claim: %v", err)
				return
			}
			if claim != nil {
				mu.Lock()
				claims++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if claims != 1 {
		t.Fatalf("live claims = %d; want exactly 1", claims)
	}
}

func TestExpiredLeaseCanBeReclaimedButOldFenceCannotComplete(t *testing.T) {
	q := NewQueue()
	if _, err := q.Enqueue(event()); err != nil {
		t.Fatal(err)
	}
	start := time.Unix(1_800_000_000, 0)
	old, err := q.Claim("worker-old", start, 10*time.Second)
	if err != nil || old == nil {
		t.Fatalf("old claim = %#v, %v", old, err)
	}
	if err := q.Complete(old.Event.ID, old.Event.TenantID, old.FenceToken, start.Add(10*time.Second)); !errors.Is(err, ErrLeaseExpired) {
		t.Fatalf("completion at expiry = %v", err)
	}
	newClaim, err := q.Claim("worker-new", start.Add(10*time.Second), time.Minute)
	if err != nil || newClaim == nil {
		t.Fatalf("reclaim = %#v, %v", newClaim, err)
	}
	if newClaim.FenceToken <= old.FenceToken {
		t.Fatalf("fence did not increase: old=%d new=%d", old.FenceToken, newClaim.FenceToken)
	}
	if err := q.Complete(old.Event.ID, old.Event.TenantID, old.FenceToken, start.Add(11*time.Second)); !errors.Is(err, ErrStaleFence) {
		t.Fatalf("stale completion = %v", err)
	}
	if err := q.Complete(newClaim.Event.ID, newClaim.Event.TenantID, newClaim.FenceToken, start.Add(11*time.Second)); err != nil {
		t.Fatalf("current completion: %v", err)
	}
	if state, _ := q.State("tenant-a", "evt-7"); state != "done" {
		t.Fatalf("state = %q; want done", state)
	}
}

func TestLeaseAndEventValidation(t *testing.T) {
	q := NewQueue()
	if _, err := q.Enqueue(Event{ID: "", TenantID: "t", Payload: []byte("x")}); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("invalid event error = %v", err)
	}
	if _, err := q.Claim("", time.Now(), time.Second); !errors.Is(err, ErrInvalidClaim) {
		t.Fatalf("invalid claim error = %v", err)
	}
	if _, err := q.Claim("worker", time.Now(), 0); !errors.Is(err, ErrInvalidClaim) {
		t.Fatalf("zero lease error = %v", err)
	}
}

func workerName(i int) string { return "worker-" + string(rune('A'+i)) }
