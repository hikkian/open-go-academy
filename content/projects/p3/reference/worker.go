package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrLeaseLost = errors.New("job lease lost")

type Worker struct {
	store                           *Store
	concurrency, batch, maxAttempts int
	poll                            time.Duration
	running, draining               atomic.Bool
}

func NewWorker(s *Store, concurrency, batch, maxAttempts int) *Worker {
	if concurrency < 1 {
		concurrency = 1
	}
	if batch < 1 {
		batch = 1
	}
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	return &Worker{store: s, concurrency: concurrency, batch: batch, maxAttempts: maxAttempts, poll: 250 * time.Millisecond}
}

func (w *Worker) Ready() bool { return w.running.Load() && !w.draining.Load() }

func (w *Worker) Run(ctx context.Context) {
	w.draining.Store(false)
	runWorkers(ctx, w.concurrency, w.batch, w.poll, w.processOne, func(started bool) {
		if started {
			w.running.Store(true)
			return
		}
		w.draining.Store(true)
	})
	w.running.Store(false)
}

// runWorkers joins every worker after cancellation. process must honor ctx for
// I/O; callers can impose a bounded join deadline because this function is
// independently blocked on ctx and does not detach worker goroutines.
func runWorkers(ctx context.Context, concurrency, batch int, poll time.Duration, process func(context.Context) (bool, error), lifecycle func(bool)) {
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				worked, err := processBatch(ctx, batch, process)
				if err != nil {
					select {
					case <-ctx.Done():
						return
					case <-time.After(poll):
					}
					continue
				}
				if !worked {
					select {
					case <-ctx.Done():
						return
					case <-time.After(poll):
					}
				}
			}
		}()
	}
	lifecycle(true)
	<-ctx.Done()
	lifecycle(false)
	wg.Wait()
}

func processBatch(ctx context.Context, batch int, process func(context.Context) (bool, error)) (bool, error) {
	worked := false
	for i := 0; i < batch; i++ {
		if ctx.Err() != nil {
			return worked, ctx.Err()
		}
		ok, err := process(ctx)
		if err != nil {
			return worked, err
		}
		if !ok {
			return worked, nil
		}
		worked = true
	}
	return worked, nil
}

type claimedJob struct {
	ID, LeaseToken, Kind string
	Payload              []byte
	Attempts             int
}

func (w *Worker) processOne(ctx context.Context) (bool, error) {
	job, ok, err := w.claimOne(ctx)
	if err != nil || !ok {
		return ok, err
	}
	return w.applyClaim(ctx, job)
}

func (w *Worker) claimOne(ctx context.Context) (*claimedJob, bool, error) {
	tx, err := w.store.db.Begin(ctx)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback(ctx)
	// An expired final attempt is terminal even if its owner died before writing
	// a result. Earlier expired leases may be reclaimed with a new fencing token.
	if _, err = tx.Exec(ctx, `UPDATE outbox SET state='dead',locked_at=NULL,lease_token=NULL,last_error='worker lease expired at retry limit' WHERE state='processing' AND locked_at<now()-interval '30 seconds' AND attempts >= $1`, w.maxAttempts); err != nil {
		return nil, false, err
	}
	job := &claimedJob{}
	err = tx.QueryRow(ctx, `SELECT id::text,kind,payload,attempts FROM outbox WHERE state IN ('pending','processing') AND attempts < $1 AND available_at<=now() AND (locked_at IS NULL OR locked_at<now()-interval '30 seconds') ORDER BY created_at,id FOR UPDATE SKIP LOCKED LIMIT 1`, w.maxAttempts).Scan(&job.ID, &job.Kind, &job.Payload, &job.Attempts)
	if errors.Is(err, pgx.ErrNoRows) {
		// The terminal-expiry sweep above may have changed rows even when no
		// claimable job remains; commit it rather than letting the defer undo it.
		if err = tx.Commit(ctx); err != nil {
			return nil, false, err
		}
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	err = tx.QueryRow(ctx, `UPDATE outbox SET state='processing',locked_at=now(),attempts=attempts+1,lease_token=gen_random_uuid() WHERE id=$1 RETURNING lease_token::text,attempts`, job.ID).Scan(&job.LeaseToken, &job.Attempts)
	if err != nil {
		return nil, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, false, err
	}
	return job, true, nil
}

func (w *Worker) applyClaim(ctx context.Context, job *claimedJob) (bool, error) {
	apply, err := w.store.db.Begin(ctx)
	if err != nil {
		_, failErr := w.fail(ctx, job, err)
		return true, failErr
	}
	defer apply.Rollback(ctx)
	if job.Kind != "task.created" {
		_ = apply.Rollback(ctx)
		_, err = w.fail(ctx, job, fmt.Errorf("unsupported job kind"))
		return true, err
	}
	if _, err = apply.Exec(ctx, "INSERT INTO job_receipts(outbox_id,kind,payload) VALUES($1,$2,$3::jsonb) ON CONFLICT(outbox_id) DO NOTHING", job.ID, job.Kind, string(job.Payload)); err != nil {
		_ = apply.Rollback(ctx)
		_, failErr := w.fail(ctx, job, err)
		return true, failErr
	}
	tag, err := apply.Exec(ctx, "UPDATE outbox SET state='done',locked_at=NULL,lease_token=NULL,completed_at=now(),last_error='' WHERE id=$1 AND state='processing' AND lease_token=$2::uuid", job.ID, job.LeaseToken)
	if err != nil {
		_ = apply.Rollback(ctx)
		_, failErr := w.fail(ctx, job, err)
		return true, failErr
	}
	if tag.RowsAffected() == 0 {
		_ = apply.Rollback(ctx)
		return true, ErrLeaseLost
	}
	return true, apply.Commit(ctx)
}

// fail changes state only when the caller still owns the exact claim. Attempts
// are read from the locked row, so an old worker cannot requeue a newer failure.
func (w *Worker) fail(ctx context.Context, job *claimedJob, cause error) (bool, error) {
	delaySeconds := int64(time.Second << min(job.Attempts, 8) / time.Second)
	tag, err := w.store.db.Exec(ctx, `UPDATE outbox SET state=CASE WHEN attempts >= $3 THEN 'dead' ELSE 'pending' END,locked_at=NULL,lease_token=NULL,available_at=now()+($4 * interval '1 second'),last_error=$5 WHERE id=$1 AND state='processing' AND lease_token=$2::uuid`, job.ID, job.LeaseToken, w.maxAttempts, delaySeconds, truncate(cause.Error(), 300))
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// Drain handles only currently available durable jobs and respects its caller's deadline.
func (w *Worker) Drain(ctx context.Context) error {
	for {
		worked, err := processBatch(ctx, w.batch, w.processOne)
		if err != nil {
			return err
		}
		if !worked {
			return nil
		}
	}
}

func waitForWorker(ctx context.Context, done <-chan struct{}) bool {
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}
