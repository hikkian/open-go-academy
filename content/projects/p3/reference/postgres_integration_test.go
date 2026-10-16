package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresTaskOutboxAndTenantIsolation(t *testing.T) {
	dsn := os.Getenv("TASKFLOW_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set TASKFLOW_TEST_DATABASE_URL to an explicitly disposable PostgreSQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	adminCfg, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	admin, e := pgxpool.NewWithConfig(ctx, adminCfg)
	if e != nil {
		t.Fatal(e)
	}
	defer admin.Close()
	var nonce string
	if e = admin.QueryRow(ctx, "SELECT replace(gen_random_uuid()::text,'-','')").Scan(&nonce); e != nil {
		t.Fatal(e)
	}
	schema := "p3_test_" + nonce
	if _, e = admin.Exec(ctx, "CREATE SCHEMA "+schema); e != nil {
		t.Fatal(e)
	}
	defer func() { _, _ = admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE") }()
	cfg, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema + ",public"
	db, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if e = migrate(ctx, db); e != nil {
		t.Fatal(e)
	}
	store := &Store{db: db}
	tokenA, e := store.Register(ctx, "a@example.test", "correct horse battery staple")
	if e != nil {
		t.Fatal(e)
	}
	userA, e := store.identity(ctx, tokenA)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = store.Login(ctx, "a@example.test", "incorrect password"); e != ErrDenied {
		t.Fatalf("bad password was accepted: %v", e)
	}
	loggedIn, e := store.Login(ctx, "a@example.test", "correct horse battery staple")
	if e != nil {
		t.Fatalf("valid login: %v", e)
	}
	if authenticated, e := store.identity(ctx, loggedIn); e != nil || authenticated.UserID != userA.UserID {
		t.Fatalf("session identity mismatch: %#v %v", authenticated, e)
	}
	tokenB, e := store.Register(ctx, "b@example.test", "another correct horse phrase")
	if e != nil {
		t.Fatal(e)
	}
	userB, e := store.identity(ctx, tokenB)
	if e != nil {
		t.Fatal(e)
	}
	var workspace string
	if e = db.QueryRow(ctx, "SELECT workspace_id::text FROM memberships WHERE user_id=$1", userB.UserID).Scan(&workspace); e != nil {
		t.Fatal(e)
	}
	if _, e = store.CreateProject(ctx, userA, workspace, "intrusion"); e != ErrDenied {
		t.Fatalf("cross-tenant project create: %v", e)
	}
	var ownWorkspace string
	if e = db.QueryRow(ctx, "SELECT workspace_id::text FROM memberships WHERE user_id=$1", userA.UserID).Scan(&ownWorkspace); e != nil {
		t.Fatal(e)
	}
	project, e := store.CreateProject(ctx, userA, ownWorkspace, "Roadmap")
	if e != nil {
		t.Fatal(e)
	}
	task, created, e := store.CreateTask(ctx, userA, project, "Ship slice", "first task", "create-task-1")
	if e != nil || !created {
		t.Fatalf("create task: %q %v %v", task, created, e)
	}
	replayed, created, e := store.CreateTask(ctx, userA, project, "different ignored title", "", "create-task-1")
	if e != nil || created || replayed != task {
		t.Fatalf("idempotent replay: %q %v %v", replayed, created, e)
	}
	projectB, e := store.CreateProject(ctx, userB, workspace, "Independent project")
	if e != nil {
		t.Fatal(e)
	}
	taskB, created, e := store.CreateTask(ctx, userB, projectB, "Same local key", "", "create-task-1")
	if e != nil || !created || taskB == task {
		t.Fatalf("same key in another project was not independent: id=%q created=%v err=%v", taskB, created, e)
	}
	if _, e = store.AddComment(ctx, userA, task, "comment"); e != nil {
		t.Fatal(e)
	}
	if _, e = store.AddComment(ctx, userB, task, "forbidden"); e != ErrDenied {
		t.Fatalf("cross-tenant comment: %v", e)
	}
	if items, e := store.ListTasks(ctx, userB, project, ""); e != nil || len(items.Items) != 0 {
		t.Fatalf("cross-tenant task list leaked: %#v %v", items, e)
	}
	if e = store.SetMemberRole(ctx, userA, ownWorkspace, "b@example.test", "editor"); e != nil {
		t.Fatal(e)
	}
	authorized := make(chan struct{})
	releaseTask := make(chan struct{})
	taskResult := make(chan error, 1)
	go func() {
		_, created, err := store.createTask(ctx, userB, project, "serialized before revocation", "", "role-race", func() { close(authorized); <-releaseTask })
		if err == nil && !created {
			err = errors.New("new task unexpectedly replayed")
		}
		taskResult <- err
	}()
	<-authorized
	revokeTx, e := db.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = revokeTx.Exec(ctx, "SET LOCAL lock_timeout='100ms'"); e != nil {
		t.Fatal(e)
	}
	_, e = revokeTx.Exec(ctx, "UPDATE memberships SET role='viewer' WHERE workspace_id=$1 AND user_id=$2", ownWorkspace, userB.UserID)
	var lockErr *pgconn.PgError
	if !errors.As(e, &lockErr) || lockErr.Code != "55P03" {
		_ = revokeTx.Rollback(ctx)
		t.Fatalf("role downgrade was not blocked by in-flight task authorization: %v", e)
	}
	_ = revokeTx.Rollback(ctx)
	close(releaseTask)
	if e = <-taskResult; e != nil {
		t.Fatalf("authorized task failed before downgrade: %v", e)
	}
	if e = store.SetMemberRole(ctx, userA, ownWorkspace, "b@example.test", "viewer"); e != nil {
		t.Fatal(e)
	}
	if _, e = store.CreateProject(ctx, userB, ownWorkspace, "viewer write"); e != ErrDenied {
		t.Fatalf("viewer created project: %v", e)
	}
	if _, _, e = store.CreateTask(ctx, userB, project, "viewer write", "", "viewer-task"); e != ErrDenied {
		t.Fatalf("viewer created task: %v", e)
	}
	if _, e = store.AddComment(ctx, userB, task, "viewer write"); e != ErrDenied {
		t.Fatalf("viewer created comment: %v", e)
	}
	visible, e := store.ListTasks(ctx, userB, project, "")
	if e != nil || len(visible.Items) != 2 {
		t.Fatalf("viewer read membership: %d %v", len(visible.Items), e)
	}
	workspaces, e := store.ListWorkspaces(ctx, userA, "")
	if e != nil || len(workspaces.Items) != 1 {
		t.Fatalf("workspace listing %#v %v", workspaces, e)
	}
	projects, e := store.ListProjects(ctx, userA, ownWorkspace, "")
	if e != nil || len(projects.Items) != 1 {
		t.Fatalf("project listing %#v %v", projects, e)
	}
	handler := NewAPI(store).Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces", nil)
	request.AddCookie(&http.Cookie{Name: "taskflow_session", Value: tokenA})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("HTTP workspace listing returned %d: %s", response.Code, response.Body.String())
	}
	var workspacePage page[workspaceRecord]
	if e = json.Unmarshal(response.Body.Bytes(), &workspacePage); e != nil || len(workspacePage.Items) != 1 {
		t.Fatalf("HTTP workspace page %#v: %v", workspacePage, e)
	}
	request = httptest.NewRequest(http.MethodPut, "/api/v1/workspaces/"+ownWorkspace+"/members/b@example.test", bytes.NewBufferString(`{"role":"viewer"}`))
	request.AddCookie(&http.Cookie{Name: "taskflow_session", Value: tokenB})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("non-owner membership change returned %d", response.Code)
	}
	request = httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+project+"/tasks", bytes.NewBufferString(`{"title":"API task","idempotency_key":"api-key"}`))
	request.AddCookie(&http.Cookie{Name: "taskflow_session", Value: tokenA})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var apiTask map[string]string
	if response.Code != http.StatusCreated || json.Unmarshal(response.Body.Bytes(), &apiTask) != nil {
		t.Fatalf("HTTP task create returned %d: %s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+project+"/tasks", bytes.NewBufferString(`{"title":"retry","idempotency_key":"api-key"}`))
	request.AddCookie(&http.Cookie{Name: "taskflow_session", Value: tokenA})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	var replay map[string]string
	if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &replay) != nil || replay["id"] != apiTask["id"] {
		t.Fatalf("HTTP idempotent retry returned %d: %s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodPost, "/api/v1/projects/"+project+"/tasks", bytes.NewBufferString(`{"title":"forbidden","idempotency_key":"other-key"}`))
	request.AddCookie(&http.Cookie{Name: "taskflow_session", Value: tokenB})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("HTTP cross-tenant task creation returned %d", response.Code)
	}
	failedLogout := NewAPI(store)
	failedLogout.revoke = func(context.Context, string) error { return errors.New("simulated database outage") }
	request = httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/current", nil)
	request.AddCookie(&http.Cookie{Name: "taskflow_session", Value: tokenA})
	response = httptest.NewRecorder()
	failedLogout.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("failed session revoke claimed success with %d", response.Code)
	}
	if _, e = store.identity(ctx, tokenA); e != nil {
		t.Fatalf("failed logout unexpectedly revoked token: %v", e)
	}
	request = httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/current", nil)
	request.AddCookie(&http.Cookie{Name: "taskflow_session", Value: tokenA})
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("successful logout returned %d", response.Code)
	}
	if _, e = store.identity(ctx, tokenA); !errors.Is(e, pgx.ErrNoRows) {
		t.Fatalf("successful logout left session valid: %v", e)
	}
	updated, e := store.UpdateTaskState(ctx, userA, task, "doing", 1)
	if e != nil || updated.State != "doing" || updated.Version != 2 {
		t.Fatalf("versioned update %#v %v", updated, e)
	}
	if _, e = store.UpdateTaskState(ctx, userA, task, "done", 1); e != ErrDenied {
		t.Fatalf("stale update was accepted: %v", e)
	}
	if e = store.DeleteTask(ctx, userB, task); e != ErrDenied {
		t.Fatalf("viewer deleted task: %v", e)
	}
	if e = store.DeleteTask(ctx, userA, "00000000-0000-0000-0000-000000000000"); e != ErrDenied {
		t.Fatalf("foreign task delete: %v", e)
	}
	var beforeRollback int
	if e = db.QueryRow(ctx, "SELECT count(*) FROM tasks WHERE project_id=$1", project).Scan(&beforeRollback); e != nil {
		t.Fatal(e)
	}
	if _, _, e = store.CreateTask(ctx, userA, project, "transaction rollback probe", "", strings.Repeat("k", 129)); e == nil {
		t.Fatal("oversized idempotency key bypassed database constraint")
	}
	var afterRollback int
	if e = db.QueryRow(ctx, "SELECT count(*) FROM tasks WHERE project_id=$1", project).Scan(&afterRollback); e != nil || afterRollback != beforeRollback {
		t.Fatalf("failed outbox insert left orphan task: before=%d after=%d err=%v", beforeRollback, afterRollback, e)
	}

	// Competing retries for one key all return one committed task and one event.
	type result struct {
		id      string
		created bool
		err     error
	}
	results := make(chan result, 8)
	for i := 0; i < 8; i++ {
		go func(i int) {
			id, created, e := store.CreateTask(ctx, userA, project, "parallel "+strconv.Itoa(i), "", "parallel-key")
			results <- result{id, created, e}
		}(i)
	}
	var canonical string
	createdCount := 0
	for i := 0; i < 8; i++ {
		r := <-results
		if r.err != nil {
			t.Fatal(r.err)
		}
		if canonical == "" {
			canonical = r.id
		}
		if r.id != canonical {
			t.Fatalf("idempotency race produced %s and %s", canonical, r.id)
		}
		if r.created {
			createdCount++
		}
	}
	if createdCount != 1 {
		t.Fatalf("parallel create count=%d", createdCount)
	}
	if _, e = db.Exec(ctx, `INSERT INTO tasks(project_id,title) SELECT $1,'page-'||n FROM generate_series(1,105) n`, project); e != nil {
		t.Fatal(e)
	}
	firstPage, e := store.ListTasks(ctx, userA, project, "")
	if e != nil || len(firstPage.Items) != 100 || firstPage.NextCursor == "" {
		t.Fatalf("first keyset page has %d items/cursor=%q error=%v", len(firstPage.Items), firstPage.NextCursor, e)
	}
	secondPage, e := store.ListTasks(ctx, userA, project, firstPage.NextCursor)
	if e != nil || len(secondPage.Items) != 9 || secondPage.NextCursor != "" {
		t.Fatalf("second keyset page has %d items/cursor=%q error=%v", len(secondPage.Items), secondPage.NextCursor, e)
	}
	var count int
	if e = db.QueryRow(ctx, "SELECT count(*) FROM outbox WHERE project_id=$1 AND idempotency_key=$2", project, "create-task-1").Scan(&count); e != nil || count != 1 {
		t.Fatalf("outbox count %d: %v", count, e)
	}
	w := NewWorker(store, 2, 4, 3)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); _, _ = w.processOne(ctx) }()
	}
	wg.Wait()
	if e = db.QueryRow(ctx, "SELECT count(*) FROM job_receipts WHERE outbox_id=(SELECT id FROM outbox WHERE project_id=$1 AND idempotency_key=$2)", project, "create-task-1").Scan(&count); e != nil || count != 1 {
		t.Fatalf("job receipt count %d: %v", count, e)
	}
	if e = w.Drain(ctx); e != nil {
		t.Fatalf("worker drain: %v", e)
	}
	var done string
	if e = db.QueryRow(ctx, "SELECT state FROM outbox WHERE project_id=$1 AND idempotency_key=$2", project, "create-task-1").Scan(&done); e != nil || done != "done" {
		t.Fatalf("outbox state %q: %v", done, e)
	}
	if _, e = db.Exec(ctx, `INSERT INTO outbox(project_id,idempotency_key,kind,payload) VALUES($1,'bad-job','unknown.kind','{}')`, project); e != nil {
		t.Fatal(e)
	}
	failingWorker := NewWorker(store, 1, 1, 2)
	staleClaim, claimed, e := failingWorker.claimOne(ctx)
	if e != nil || !claimed {
		t.Fatalf("first lease claim failed: %v", e)
	}
	if _, e = db.Exec(ctx, "UPDATE outbox SET locked_at=now()-interval '31 seconds' WHERE id=$1", staleClaim.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = failingWorker.processOne(ctx); e != nil {
		t.Fatal(e)
	}
	staleChanged, e := failingWorker.fail(ctx, staleClaim, errors.New("late stale failure"))
	if e != nil || staleChanged {
		t.Fatalf("stale lease modified newer attempt: changed=%v error=%v", staleChanged, e)
	}
	if _, e = failingWorker.processOne(ctx); e != nil {
		t.Fatal(e)
	}
	var dead string
	var attempts int
	if e = db.QueryRow(ctx, "SELECT state,attempts FROM outbox WHERE project_id=$1 AND idempotency_key='bad-job'", project).Scan(&dead, &attempts); e != nil || dead != "dead" || attempts != 2 {
		t.Fatalf("retry bound state=%q attempts=%d error=%v", dead, attempts, e)
	}
	if _, e = db.Exec(ctx, `INSERT INTO outbox(project_id,idempotency_key,kind,payload,state,attempts,locked_at,lease_token) VALUES($1,'only-expired-final','unknown.kind','{}','processing',2,now()-interval '31 seconds',gen_random_uuid())`, project); e != nil {
		t.Fatal(e)
	}
	if job, ok, e := failingWorker.claimOne(ctx); e != nil || ok || job != nil {
		t.Fatalf("expired final attempt should be terminal, not claimable: job=%#v claimed=%v error=%v", job, ok, e)
	}
	if e = db.QueryRow(ctx, "SELECT state FROM outbox WHERE project_id=$1 AND idempotency_key='only-expired-final'", project).Scan(&dead); e != nil || dead != "dead" {
		t.Fatalf("expired final attempt cleanup was not committed: state=%q error=%v", dead, e)
	}
	if job, ok, e := failingWorker.claimOne(ctx); e != nil || ok || job != nil {
		t.Fatalf("terminal expired job was reclaimed: job=%#v claimed=%v error=%v", job, ok, e)
	}
	if e = rollback(ctx, db); e != nil {
		t.Fatalf("rollback: %v", e)
	}
	if e = migrate(ctx, db); e != nil {
		t.Fatalf("migration after rollback: %v", e)
	}
}
