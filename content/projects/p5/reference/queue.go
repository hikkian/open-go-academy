package main

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

var (
	ErrInvalidEvent    = errors.New("invalid event")
	ErrPayloadConflict = errors.New("event ID already exists with a different payload")
	ErrInvalidClaim    = errors.New("invalid worker or lease")
	ErrLeaseExpired    = errors.New("lease expired")
	ErrStaleFence      = errors.New("stale fencing token")
	ErrNotClaimed      = errors.New("event is not currently claimed")
)

type Event struct {
	ID       string
	TenantID string
	Payload  []byte
}

type Claim struct {
	Event      Event
	WorkerID   string
	FenceToken uint64
	LeaseUntil time.Time
}

type record struct {
	event    Event
	state    state
	claim    Claim
	sequence uint64
}

type state uint8

const (
	pending state = iota
	leased
	done
)

type queueKey struct {
	tenantID string
	eventID  string
}

// Queue serializes the state transition for this in-memory experiment. A
// production implementation must replace this mutex with an atomic datastore
// transaction/conditional update shared by every worker process.
type Queue struct {
	mu       sync.Mutex
	rows     map[queueKey]*record
	sequence uint64
	fence    uint64
}

func NewQueue() *Queue { return &Queue{rows: make(map[queueKey]*record)} }

func key(tenantID, eventID string) queueKey { return queueKey{tenantID: tenantID, eventID: eventID} }

func (q *Queue) Enqueue(event Event) (bool, error) {
	if event.ID == "" || event.TenantID == "" || len(event.Payload) == 0 || len(event.Payload) > 64*1024 {
		return false, ErrInvalidEvent
	}
	k := key(event.TenantID, event.ID)
	q.mu.Lock()
	defer q.mu.Unlock()
	if existing, ok := q.rows[k]; ok {
		if !bytes.Equal(existing.event.Payload, event.Payload) {
			return false, ErrPayloadConflict
		}
		return false, nil
	}
	q.sequence++
	event.Payload = bytes.Clone(event.Payload)
	q.rows[k] = &record{event: event, state: pending, sequence: q.sequence}
	return true, nil
}

// Claim grants at most one live lease for an event. Fixed caller-provided time
// makes lease-expiry tests deterministic and exposes clock ownership as a
// deliberate production decision rather than hiding it in this model.
func (q *Queue) Claim(workerID string, now time.Time, lease time.Duration) (*Claim, error) {
	if workerID == "" || lease <= 0 {
		return nil, ErrInvalidClaim
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	eligible := make([]*record, 0)
	for _, row := range q.rows {
		if row.state == pending || (row.state == leased && !now.Before(row.claim.LeaseUntil)) {
			eligible = append(eligible, row)
		}
	}
	if len(eligible) == 0 {
		return nil, nil
	}
	sort.Slice(eligible, func(i, j int) bool { return eligible[i].sequence < eligible[j].sequence })
	row := eligible[0]
	q.fence++
	row.state = leased
	row.claim = Claim{
		Event:    Event{ID: row.event.ID, TenantID: row.event.TenantID, Payload: bytes.Clone(row.event.Payload)},
		WorkerID: workerID, FenceToken: q.fence, LeaseUntil: now.Add(lease),
	}
	return cloneClaim(row.claim), nil
}

// Complete is the fencing point: a slow worker cannot commit after lease
// expiry or after another worker has acquired a newer token.
func (q *Queue) Complete(eventID, tenantID string, fenceToken uint64, now time.Time) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	row, ok := q.rows[key(tenantID, eventID)]
	if !ok || row.state != leased {
		return ErrNotClaimed
	}
	if row.claim.FenceToken != fenceToken {
		return ErrStaleFence
	}
	if !now.Before(row.claim.LeaseUntil) {
		return ErrLeaseExpired
	}
	row.state = done
	return nil
}

func (q *Queue) State(tenantID, eventID string) (string, bool) {
	q.mu.Lock()
	defer q.mu.Unlock()
	row, ok := q.rows[key(tenantID, eventID)]
	if !ok {
		return "", false
	}
	switch row.state {
	case pending:
		return "pending", true
	case leased:
		return "leased", true
	case done:
		return "done", true
	default:
		return "unknown", true
	}
}

func cloneClaim(claim Claim) *Claim {
	claim.Event.Payload = bytes.Clone(claim.Event.Payload)
	return &claim
}

func (c Claim) String() string {
	return fmt.Sprintf("%s/%s worker=%s fence=%d until=%s", c.Event.TenantID, c.Event.ID, c.WorkerID, c.FenceToken, c.LeaseUntil.UTC().Format(time.RFC3339Nano))
}
