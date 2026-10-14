package httpapi

import (
	"sync"
	"time"
)

type quota struct {
	mu         sync.Mutex
	started    time.Time
	mutations  int
	executions int
}

func (q *quota) allow(execution bool) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	now := time.Now()
	if q.started.IsZero() || now.Sub(q.started) >= time.Minute {
		q.started = now
		q.mutations = 0
		q.executions = 0
	}
	if q.mutations >= 120 || execution && q.executions >= 12 {
		return false
	}
	q.mutations++
	if execution {
		q.executions++
	}
	return true
}
