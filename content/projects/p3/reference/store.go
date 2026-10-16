package main

import (
	"context"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrDenied = errors.New("not found")

type Store struct{ db *pgxpool.Pool }
type identity struct{ UserID string }
type taskRecord struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Version     int    `json:"version"`
}
type commentRecord struct {
	ID        string    `json:"id"`
	AuthorID  string    `json:"author_id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}
type workspaceRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}
type projectRecord struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
}
type page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}

func (s *Store) Register(ctx context.Context, email, password string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) < 3 || len(email) > 254 || !strings.Contains(email, "@") || len(password) < 12 || len(password) > 256 {
		return "", errors.New("invalid credentials")
	}
	salt := make([]byte, 16)
	if _, e := rand.Read(salt); e != nil {
		return "", e
	}
	key, e := pbkdf2.Key(sha256.New, password, salt, 210000, 32)
	if e != nil {
		return "", e
	}
	encoded := base64.RawStdEncoding.EncodeToString(salt) + "$" + base64.RawStdEncoding.EncodeToString(key)
	tx, e := s.db.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(ctx)
	var uid string
	if e = tx.QueryRow(ctx, "INSERT INTO users(email,password_hash) VALUES($1,$2) RETURNING id::text", email, encoded).Scan(&uid); e != nil {
		return "", e
	}
	var wid string
	if e = tx.QueryRow(ctx, "INSERT INTO workspaces(name) VALUES('My workspace') RETURNING id::text").Scan(&wid); e != nil {
		return "", e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO memberships(workspace_id,user_id,role) VALUES($1,$2,'owner')", wid, uid); e != nil {
		return "", e
	}
	token, e := newToken()
	if e != nil {
		return "", e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,$3)", hashToken(token), uid, time.Now().Add(24*time.Hour)); e != nil {
		return "", e
	}
	return token, tx.Commit(ctx)
}
func (s *Store) Login(ctx context.Context, email, password string) (string, error) {
	var uid, stored string
	e := s.db.QueryRow(ctx, "SELECT id::text,password_hash FROM users WHERE email=$1", strings.ToLower(strings.TrimSpace(email))).Scan(&uid, &stored)
	if e != nil {
		return "", ErrDenied
	}
	parts := strings.Split(stored, "$")
	if len(parts) != 2 {
		return "", ErrDenied
	}
	salt, e := base64.RawStdEncoding.DecodeString(parts[0])
	if e != nil {
		return "", ErrDenied
	}
	want, e := base64.RawStdEncoding.DecodeString(parts[1])
	if e != nil {
		return "", ErrDenied
	}
	got, e := pbkdf2.Key(sha256.New, password, salt, 210000, 32)
	if e != nil || subtle.ConstantTimeCompare(got, want) != 1 {
		return "", ErrDenied
	}
	token, e := newToken()
	if e != nil {
		return "", e
	}
	_, e = s.db.Exec(ctx, "INSERT INTO sessions(token_hash,user_id,expires_at) VALUES($1,$2,$3)", hashToken(token), uid, time.Now().Add(24*time.Hour))
	return token, e
}
func (s *Store) identity(ctx context.Context, token string) (identity, error) {
	var i identity
	e := s.db.QueryRow(ctx, "SELECT user_id::text FROM sessions WHERE token_hash=$1 AND expires_at>now()", hashToken(token)).Scan(&i.UserID)
	return i, e
}
func (s *Store) RevokeSession(ctx context.Context, token string) error {
	_, err := s.db.Exec(ctx, "DELETE FROM sessions WHERE token_hash=$1", hashToken(token))
	return err
}
func (s *Store) CreateWorkspace(ctx context.Context, u identity, name string) (string, error) {
	tx, e := s.db.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(ctx)
	var id string
	e = tx.QueryRow(ctx, "INSERT INTO workspaces(name) VALUES($1) RETURNING id::text", strings.TrimSpace(name)).Scan(&id)
	if e != nil {
		return "", e
	}
	if _, e = tx.Exec(ctx, "INSERT INTO memberships(workspace_id,user_id,role) VALUES($1,$2,'owner')", id, u.UserID); e != nil {
		return "", e
	}
	return id, tx.Commit(ctx)
}
func (s *Store) CreateProject(ctx context.Context, u identity, wid, name string) (string, error) {
	var id string
	e := s.db.QueryRow(ctx, `INSERT INTO projects(workspace_id,name) SELECT m.workspace_id,$3 FROM memberships m WHERE m.workspace_id=$1 AND m.user_id=$2 AND m.role IN ('owner','editor') AND length($3) BETWEEN 1 AND 120 RETURNING id::text`, wid, u.UserID, strings.TrimSpace(name)).Scan(&id)
	if errors.Is(e, pgx.ErrNoRows) {
		return "", ErrDenied
	}
	return id, e
}
func (s *Store) SetMemberRole(ctx context.Context, actor identity, workspaceID, email, role string) error {
	var userID string
	e := s.db.QueryRow(ctx, `INSERT INTO memberships(workspace_id,user_id,role) SELECT w.id,u.id,$4 FROM workspaces w JOIN memberships owner ON owner.workspace_id=w.id AND owner.user_id=$2 AND owner.role='owner' JOIN users u ON u.email=$3 WHERE w.id=$1 AND $4 IN ('editor','viewer') ON CONFLICT(workspace_id,user_id) DO UPDATE SET role=EXCLUDED.role WHERE memberships.role IN ('editor','viewer') RETURNING user_id::text`, workspaceID, actor.UserID, strings.ToLower(strings.TrimSpace(email)), role).Scan(&userID)
	if errors.Is(e, pgx.ErrNoRows) {
		return ErrDenied
	}
	return e
}
func (s *Store) CreateTask(ctx context.Context, u identity, pid, title, description, key string) (string, bool, error) {
	return s.createTask(ctx, u, pid, title, description, key, nil)
}

func (s *Store) createTask(ctx context.Context, u identity, pid, title, description, key string, afterAuthorization func()) (string, bool, error) {
	tx, e := s.db.Begin(ctx)
	if e != nil {
		return "", false, e
	}
	defer tx.Rollback(ctx)
	// Lock membership through commit. A concurrent role downgrade either wins
	// before this check (and is denied) or waits until the authorized write commits.
	var role string
	e = tx.QueryRow(ctx, `SELECT m.role FROM projects p JOIN memberships m USING(workspace_id) WHERE p.id=$1 AND m.user_id=$2 FOR SHARE OF m`, pid, u.UserID).Scan(&role)
	if errors.Is(e, pgx.ErrNoRows) {
		return "", false, ErrDenied
	}
	if e != nil {
		return "", false, e
	}
	if role != "owner" && role != "editor" {
		return "", false, ErrDenied
	}
	if afterAuthorization != nil {
		afterAuthorization()
	}
	var previous string
	e = tx.QueryRow(ctx, "SELECT payload->>'task_id' FROM outbox WHERE project_id=$1 AND idempotency_key=$2", pid, key).Scan(&previous)
	if e == nil {
		return previous, false, tx.Commit(ctx)
	}
	if !errors.Is(e, pgx.ErrNoRows) {
		return "", false, e
	}
	var id string
	e = tx.QueryRow(ctx, `INSERT INTO tasks(project_id,title,description) SELECT p.id,$3,$4 FROM projects p JOIN memberships m ON m.workspace_id=p.workspace_id WHERE p.id=$1 AND m.user_id=$2 AND m.role IN ('owner','editor') RETURNING id::text`, pid, u.UserID, strings.TrimSpace(title), description).Scan(&id)
	if e != nil {
		if errors.Is(e, pgx.ErrNoRows) {
			return "", false, ErrDenied
		}
		return "", false, e
	}
	encoded, _ := json.Marshal(map[string]string{"task_id": id, "project_id": pid})
	tag, e := tx.Exec(ctx, "INSERT INTO outbox(project_id,idempotency_key,kind,payload) VALUES($1,$2,'task.created',$3::jsonb) ON CONFLICT(project_id,idempotency_key) DO NOTHING", pid, key, string(encoded))
	if e != nil {
		return "", false, e
	}
	if tag.RowsAffected() == 0 {
		if _, e = tx.Exec(ctx, "DELETE FROM tasks WHERE id=$1", id); e != nil {
			return "", false, e
		}
		var previous string
		if e = tx.QueryRow(ctx, "SELECT payload->>'task_id' FROM outbox WHERE project_id=$1 AND idempotency_key=$2", pid, key).Scan(&previous); e != nil {
			return "", false, ErrDenied
		}
		return previous, false, tx.Commit(ctx)
	}
	return id, true, tx.Commit(ctx)
}
func (s *Store) AddComment(ctx context.Context, u identity, taskID, body string) (string, error) {
	var id string
	e := s.db.QueryRow(ctx, `INSERT INTO comments(task_id,author_id,body) SELECT t.id,$2,$3 FROM tasks t JOIN projects p ON p.id=t.project_id JOIN memberships m ON m.workspace_id=p.workspace_id WHERE t.id=$1 AND m.user_id=$2 AND m.role IN ('owner','editor') RETURNING id::text`, taskID, u.UserID, strings.TrimSpace(body)).Scan(&id)
	if errors.Is(e, pgx.ErrNoRows) {
		return "", ErrDenied
	}
	return id, e
}

func (s *Store) ListWorkspaces(ctx context.Context, u identity, after string) (page[workspaceRecord], error) {
	rows, e := s.db.Query(ctx, `SELECT w.id::text,w.name,m.role FROM workspaces w JOIN memberships m ON m.workspace_id=w.id WHERE m.user_id=$1 AND ($2='' OR w.id>$2::uuid) ORDER BY w.id LIMIT 101`, u.UserID, after)
	if e != nil {
		return page[workspaceRecord]{}, e
	}
	defer rows.Close()
	out := page[workspaceRecord]{Items: make([]workspaceRecord, 0)}
	for rows.Next() {
		var v workspaceRecord
		if e = rows.Scan(&v.ID, &v.Name, &v.Role); e != nil {
			return out, e
		}
		out.Items = append(out.Items, v)
	}
	if e = rows.Err(); e != nil {
		return out, e
	}
	if len(out.Items) > 100 {
		out.NextCursor = out.Items[99].ID
		out.Items = out.Items[:100]
	}
	return out, nil
}
func (s *Store) ListProjects(ctx context.Context, u identity, workspaceID, after string) (page[projectRecord], error) {
	rows, e := s.db.Query(ctx, `SELECT p.id::text,p.workspace_id::text,p.name FROM projects p JOIN memberships m ON m.workspace_id=p.workspace_id WHERE p.workspace_id=$1 AND m.user_id=$2 AND ($3='' OR p.id>$3::uuid) ORDER BY p.id LIMIT 101`, workspaceID, u.UserID, after)
	if e != nil {
		return page[projectRecord]{}, e
	}
	defer rows.Close()
	out := page[projectRecord]{Items: make([]projectRecord, 0)}
	for rows.Next() {
		var v projectRecord
		if e = rows.Scan(&v.ID, &v.WorkspaceID, &v.Name); e != nil {
			return out, e
		}
		out.Items = append(out.Items, v)
	}
	if e = rows.Err(); e != nil {
		return out, e
	}
	if len(out.Items) > 100 {
		out.NextCursor = out.Items[99].ID
		out.Items = out.Items[:100]
	}
	return out, nil
}
func (s *Store) ListTasks(ctx context.Context, u identity, projectID, after string) (page[taskRecord], error) {
	rows, e := s.db.Query(ctx, `SELECT t.id::text,t.project_id::text,t.title,t.description,t.state,t.version FROM tasks t JOIN projects p ON p.id=t.project_id JOIN memberships m ON m.workspace_id=p.workspace_id WHERE p.id=$1 AND m.user_id=$2 AND ($3='' OR t.id>$3::uuid) ORDER BY t.id LIMIT 101`, projectID, u.UserID, after)
	if e != nil {
		return page[taskRecord]{}, e
	}
	defer rows.Close()
	result := page[taskRecord]{Items: make([]taskRecord, 0)}
	for rows.Next() {
		var t taskRecord
		if e = rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.State, &t.Version); e != nil {
			return result, e
		}
		result.Items = append(result.Items, t)
	}
	if e = rows.Err(); e != nil {
		return result, e
	}
	if len(result.Items) > 100 {
		result.NextCursor = result.Items[99].ID
		result.Items = result.Items[:100]
	}
	return result, nil
}
func (s *Store) DeleteTask(ctx context.Context, u identity, taskID string) error {
	tag, e := s.db.Exec(ctx, `DELETE FROM tasks t USING projects p,memberships m WHERE t.id=$1 AND t.project_id=p.id AND p.workspace_id=m.workspace_id AND m.user_id=$2 AND m.role='owner'`, taskID, u.UserID)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return ErrDenied
	}
	return nil
}
func (s *Store) UpdateTaskState(ctx context.Context, u identity, taskID, state string, version int) (taskRecord, error) {
	var t taskRecord
	e := s.db.QueryRow(ctx, `UPDATE tasks t SET state=$3,version=version+1,updated_at=now() FROM projects p,memberships m WHERE t.id=$1 AND t.project_id=p.id AND p.workspace_id=m.workspace_id AND m.user_id=$2 AND m.role IN ('owner','editor') AND t.version=$4 AND $3 IN ('open','doing','done') RETURNING t.id::text,t.project_id::text,t.title,t.description,t.state,t.version`, taskID, u.UserID, state, version).Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.State, &t.Version)
	if errors.Is(e, pgx.ErrNoRows) {
		return t, ErrDenied
	}
	return t, e
}
func (s *Store) ListComments(ctx context.Context, u identity, taskID, after string) (page[commentRecord], error) {
	rows, e := s.db.Query(ctx, `SELECT c.id::text,c.author_id::text,c.body,c.created_at FROM comments c JOIN tasks t ON t.id=c.task_id JOIN projects p ON p.id=t.project_id JOIN memberships m ON m.workspace_id=p.workspace_id WHERE t.id=$1 AND m.user_id=$2 AND ($3='' OR c.id>$3::uuid) ORDER BY c.id LIMIT 101`, taskID, u.UserID, after)
	if e != nil {
		return page[commentRecord]{}, e
	}
	defer rows.Close()
	result := page[commentRecord]{Items: make([]commentRecord, 0)}
	for rows.Next() {
		var c commentRecord
		if e = rows.Scan(&c.ID, &c.AuthorID, &c.Body, &c.CreatedAt); e != nil {
			return result, e
		}
		result.Items = append(result.Items, c)
	}
	if e = rows.Err(); e != nil {
		return result, e
	}
	if len(result.Items) > 100 {
		result.NextCursor = result.Items[99].ID
		result.Items = result.Items[:100]
	}
	return result, nil
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func hashToken(v string) []byte { x := sha256.Sum256([]byte(v)); return x[:] }
