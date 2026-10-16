package main

import (
	"errors"
	"time"
)

var ErrNotImplemented = errors.New("prototype operation is not implemented")

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

// Queue is an in-memory experiment. The production design must identify the
// transactional storage primitive that would provide equivalent guarantees.
type Queue struct{}

func NewQueue() *Queue { return &Queue{} }

// Enqueue should validate input and make repeated (tenant, event ID) delivery
// idempotent. Decide which payload conflict is an error.
func (q *Queue) Enqueue(event Event) (bool, error) {
	return false, ErrNotImplemented
}

// Claim should choose one eligible event and issue a monotonically increasing
// fencing token. Use the caller's fixed time to keep tests deterministic.
func (q *Queue) Claim(workerID string, now time.Time, lease time.Duration) (*Claim, error) {
	return nil, ErrNotImplemented
}

// Complete must reject a worker whose lease has expired or whose token is stale.
func (q *Queue) Complete(eventID, tenantID string, fenceToken uint64, now time.Time) error {
	return ErrNotImplemented
}
