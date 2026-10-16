package projectrunner

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strings"
	"time"

	"academy/internal/runner"
)

var taskFlowUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-8][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

type taskFlowFixture struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	WorkspaceID     string `json:"workspace_id"`
	ProjectID       string `json:"project_id"`
	SecondProjectID string `json:"second_project_id"`
	TaskID          string `json:"task_id"`
	CommentID       string `json:"comment_id"`
	IdempotencyKey  string `json:"idempotency_key"`
}

const taskFlowOutboxSQL = `SET search_path=pg_catalog;
SELECT EXISTS (
 SELECT 1 FROM public.outbox o JOIN public.job_receipts r ON r.outbox_id=o.id
 WHERE o.project_id=:'project_id'::uuid AND o.idempotency_key=:'idempotency_key'
   AND o.payload->>'task_id'=:'task_id' AND o.kind='task.created'
   AND o.state='done' AND r.kind='task.created'
);`

func taskBaseURL(projectID string) string {
	if projectID == "p3" {
		return "http://127.0.0.1:8098"
	}
	return "http://127.0.0.1:8097"
}

func healthCheckName(projectID string) string {
	if projectID == "p3" {
		return "readyz"
	}
	return "healthz"
}

func decodeTaskFlowFixture(raw json.RawMessage) (taskFlowFixture, error) {
	var fixture taskFlowFixture
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if len(raw) == 0 || dec.Decode(&fixture) != nil {
		return fixture, errors.New("invalid TaskFlow checker state")
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return fixture, errors.New("trailing TaskFlow checker state")
	}
	if fixture.Email == "" || len(fixture.Email) > 254 || len(fixture.Password) < 12 || len(fixture.Password) > 256 || !taskFlowUUID.MatchString(fixture.WorkspaceID) || !taskFlowUUID.MatchString(fixture.ProjectID) || !taskFlowUUID.MatchString(fixture.SecondProjectID) || fixture.ProjectID == fixture.SecondProjectID || !taskFlowUUID.MatchString(fixture.TaskID) || !taskFlowUUID.MatchString(fixture.CommentID) || len(fixture.IdempotencyKey) < 1 || len(fixture.IdempotencyKey) > 128 {
		return fixture, errors.New("invalid TaskFlow checker state")
	}
	return fixture, nil
}

func (e *engine) taskFlowOutboxCheck(ctx context.Context, db, appRole string, raw json.RawMessage) runner.Check {
	check := runner.Check{Name: "TaskFlow task has a completed outbox receipt in PostgreSQL"}
	fixture, err := decodeTaskFlowFixture(raw)
	if err != nil {
		return check
	}
	args := []string{"exec", "-i", db, "psql", "-X", "-U", appRole, "-d", "academy", "-v", "ON_ERROR_STOP=1", "-qtA", "-v", "project_id=" + fixture.ProjectID, "-v", "idempotency_key=" + fixture.IdempotencyKey, "-v", "task_id=" + fixture.TaskID, "-f", "-"}
	out, err := e.run(ctx, []byte(taskFlowOutboxSQL), args...)
	check.Passed = err == nil && strings.TrimSpace(string(out)) == "t"
	return check
}

func (e *engine) waitTaskFlowOutbox(ctx context.Context, db, appRole string, raw json.RawMessage) runner.Check {
	deadline := time.NewTimer(12 * time.Second)
	defer deadline.Stop()
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		check := e.taskFlowOutboxCheck(ctx, db, appRole, raw)
		if check.Passed {
			return check
		}
		select {
		case <-ctx.Done():
			return runner.Check{Name: check.Name, Passed: false}
		case <-deadline.C:
			return runner.Check{Name: check.Name, Passed: false}
		case <-tick.C:
		}
	}
}

func (e *engine) taskFlowTables(ctx context.Context, db, appRole string) []runner.Check {
	tables := []string{"users", "sessions", "workspaces", "memberships", "projects", "tasks", "comments", "outbox", "job_receipts"}
	checks := make([]runner.Check, 0, len(tables))
	for _, table := range tables {
		// Table names are fixed constants; the app role is a server-generated job suffix.
		out, err := e.pod(ctx, "exec", db, "psql", "-X", "-U", appRole, "-d", "academy", "-tAc", "SELECT pg_catalog.to_regclass('public."+table+"') IS NOT NULL")
		checks = append(checks, runner.Check{Name: "TaskFlow PostgreSQL table " + table, Passed: err == nil && strings.TrimSpace(string(out)) == "t"})
	}
	return checks
}

func (e *engine) submitTaskFlow(ctx context.Context, job, db, app, dsn string, labels []string, files []byte) (runner.Result, error) {
	ready, _, err := e.waitReady(ctx, job, db, app, "p3", labels)
	if err != nil {
		return runner.Result{Status: "failed", ExitCode: 1, Checks: ready.Checks, Passed: false, Stderr: "TaskFlow не прошёл проверку readiness."}, nil
	}
	first, err := e.runChecker(ctx, job, db, labels, checkerInput{ProjectID: "p3", Phase: "initial", BaseURL: taskBaseURL("p3")})
	if err != nil {
		return runner.Result{}, err
	}
	checks := append([]runner.Check{}, ready.Checks...)
	checks = append(checks, first.Checks...)
	checks = append(checks, e.waitTaskFlowOutbox(ctx, db, "student_"+job, first.State))
	if _, err = e.pod(ctx, "rm", "-f", app); err != nil {
		return runner.Result{}, errors.New("cannot stop TaskFlow phase-one server")
	}
	app2 := "academy-" + job + "-app2"
	if _, err = e.runStudent(ctx, "p3", app2, db, files, dsn, false, labels); err != nil {
		return runner.Result{Status: "failed", ExitCode: 1, Passed: false, Checks: checks}, nil
	}
	ready2, _, err := e.waitReady(ctx, job, db, app2, "p3", labels)
	checks = append(checks, ready2.Checks...)
	if err != nil {
		return runner.Result{Status: "failed", ExitCode: 1, Passed: false, Checks: checks, Stderr: "TaskFlow после перезапуска не прошёл readiness."}, nil
	}
	second, err := e.runChecker(ctx, job, db, labels, checkerInput{ProjectID: "p3", Phase: "persistence", BaseURL: taskBaseURL("p3"), State: first.State})
	if err != nil {
		return runner.Result{}, err
	}
	checks = append(checks, second.Checks...)
	checks = append(checks, e.taskFlowTables(ctx, db, "student_"+job)...)
	passed := len(checks) > 0
	for _, check := range checks {
		if strings.TrimSpace(check.Name) == "" {
			return runner.Result{}, errors.New("invalid trusted TaskFlow check")
		}
		passed = passed && check.Passed
	}
	status := "failed"
	if passed {
		status = "automatic_pass"
	}
	return runner.Result{Status: status, ExitCode: 0, Passed: passed, Checks: checks}, nil
}
