package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errNotFound = errors.New("not found")
var errConflict = errors.New("conflict")

//go:embed migrations/*.sql
var migrations embed.FS

type PGStore struct{ pool *pgxpool.Pool }

func OpenPG(ctx context.Context, dsn string) (*PGStore, error) {
	s, e := openPool(dsn)
	if e != nil {
		return nil, e
	}
	if e = s.Migrate(ctx); e != nil {
		s.Close()
		return nil, e
	}
	return s, nil
}
func openPool(dsn string) (*PGStore, error) {
	cfg, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		return nil, fmt.Errorf("invalid database configuration")
	}
	cfg.MaxConns = 8
	cfg.MinConns = 0
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.ConnConfig.ConnectTimeout = 5 * time.Second
	cfg.ConnConfig.RuntimeParams["statement_timeout"] = "5000"
	cfg.ConnConfig.RuntimeParams["lock_timeout"] = "2000"
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	p, e := pgxpool.NewWithConfig(ctx, cfg)
	if e != nil {
		return nil, e
	}
	s := &PGStore{p}
	return s, nil
}

func RollbackMigrations(ctx context.Context, dsn string) error {
	s, e := openPool(dsn)
	if e != nil {
		return e
	}
	defer s.Close()
	conn, e := s.pool.Acquire(ctx)
	if e != nil {
		return e
	}
	defer conn.Release()
	if _, e = conn.Exec(ctx, `SELECT pg_advisory_lock(716203041)`); e != nil {
		return e
	}
	defer conn.Exec(context.Background(), `SELECT pg_advisory_unlock(716203041)`)
	var migrationTable bool
	if e = conn.QueryRow(ctx, `SELECT to_regclass('public.schema_migrations') IS NOT NULL`).Scan(&migrationTable); e != nil {
		return e
	}
	if !migrationTable {
		return nil
	}
	var exists bool
	if e = conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version='0001_init')`).Scan(&exists); e != nil {
		return e
	}
	if !exists {
		return nil
	}
	raw, e := migrations.ReadFile("migrations/0001_init.down.sql")
	if e != nil {
		return e
	}
	tx, e := conn.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, string(raw), pgx.QueryExecModeSimpleProtocol); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version='0001_init'`); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *PGStore) Close() { s.pool.Close() }
func (s *PGStore) Migrate(ctx context.Context) error {
	conn, e := s.pool.Acquire(ctx)
	if e != nil {
		return e
	}
	defer conn.Release()
	if _, e = conn.Exec(ctx, `SELECT pg_advisory_lock(716203041)`); e != nil {
		return e
	}
	defer conn.Exec(context.Background(), `SELECT pg_advisory_unlock(716203041)`)
	if _, e = conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); e != nil {
		return e
	}
	raw, e := migrations.ReadFile("migrations/0001_init.up.sql")
	if e != nil {
		return e
	}
	var exists bool
	e = conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version='0001_init')`).Scan(&exists)
	if e != nil || exists {
		return e
	}
	tx, e := conn.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	if _, e = tx.Exec(ctx, string(raw), pgx.QueryExecModeSimpleProtocol); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES('0001_init')`); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (s *PGStore) CreateUser(email string, password []byte) (User, error) {
	ctx, cancel := operationContext()
	defer cancel()
	var u User
	e := s.pool.QueryRow(ctx, `INSERT INTO users(email_normalized,password_hash) VALUES($1,$2) RETURNING id::text,email_normalized,role`, email, password).Scan(&u.ID, &u.Email, &u.Role)
	if e != nil {
		return u, translate(e)
	}
	return u, nil
}
func (s *PGStore) UserByEmail(email string) (User, []byte, error) {
	ctx, cancel := operationContext()
	defer cancel()
	var u User
	var h []byte
	e := s.pool.QueryRow(ctx, `SELECT id::text,email_normalized,password_hash,role FROM users WHERE email_normalized=$1`, email).Scan(&u.ID, &u.Email, &h, &u.Role)
	if e != nil {
		return u, nil, translate(e)
	}
	return u, h, nil
}
func (s *PGStore) CreateSession(th, ch, uid string, exp time.Time) error {
	ctx, cancel := operationContext()
	defer cancel()
	_, e := s.pool.Exec(ctx, `INSERT INTO sessions(token_hash,csrf_hash,user_id,expires_at) VALUES($1,$2,$3,$4)`, th, ch, uid, exp)
	return e
}
func (s *PGStore) Session(th string) (User, string, time.Time, error) {
	ctx, cancel := operationContext()
	defer cancel()
	var u User
	var csrf string
	var exp time.Time
	e := s.pool.QueryRow(ctx, `SELECT u.id::text,u.email_normalized,u.role,s.csrf_hash,s.expires_at FROM sessions s JOIN users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.expires_at>now()`, th).Scan(&u.ID, &u.Email, &u.Role, &csrf, &exp)
	if e != nil {
		return u, "", exp, translate(e)
	}
	return u, csrf, exp, nil
}
func (s *PGStore) DeleteSession(th string) error {
	ctx, cancel := operationContext()
	defer cancel()
	_, e := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, th)
	return e
}
func scanLink(row pgx.Row) (Link, error) {
	var l Link
	e := row.Scan(&l.ID, &l.Slug, &l.TargetURL, &l.Enabled, &l.CreatedAt, &l.UpdatedAt)
	return l, e
}
func (s *PGStore) CreateLink(owner string, in LinkInput) (Link, error) {
	ctx, cancel := operationContext()
	defer cancel()
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return Link{}, e
	}
	defer tx.Rollback(ctx)
	var l Link
	e = tx.QueryRow(ctx, `INSERT INTO links(owner_id,slug,target_url,enabled) VALUES($1,$2,$3,$4) RETURNING id::text,slug,target_url,enabled,created_at,updated_at`, owner, in.Slug, in.TargetURL, *in.Enabled).Scan(&l.ID, &l.Slug, &l.TargetURL, &l.Enabled, &l.CreatedAt, &l.UpdatedAt)
	if e != nil {
		return l, translate(e)
	}
	if e = replaceTags(ctx, tx, owner, l.ID, in.Tags); e != nil {
		return l, translate(e)
	}
	l.Tags, e = readTags(ctx, tx, l.ID)
	if e != nil {
		return l, e
	}
	return l, tx.Commit(ctx)
}
func (s *PGStore) ListLinks(owner string, limit, offset int) ([]Link, error) {
	ctx, cancel := operationContext()
	defer cancel()
	rows, e := s.pool.Query(ctx, `SELECT id::text,slug,target_url,enabled,created_at,updated_at FROM links WHERE owner_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3`, owner, limit, offset)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Link{}
	for rows.Next() {
		l, e := scanLink(rows)
		if e != nil {
			return nil, e
		}
		l.Tags = []Tag{}
		out = append(out, l)
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	rows.Close()
	if len(out) == 0 {
		return out, nil
	}
	indices := make(map[string]int, len(out))
	ids := make([]string, len(out))
	for i := range out {
		indices[out[i].ID] = i
		ids[i] = out[i].ID
	}
	tagRows, e := s.pool.Query(ctx, `SELECT lt.link_id::text,t.id::text,t.name FROM link_tags lt JOIN tags t ON t.id=lt.tag_id WHERE lt.link_id::text=ANY($1::text[]) ORDER BY t.name_normalized,t.id`, ids)
	if e != nil {
		return nil, e
	}
	defer tagRows.Close()
	for tagRows.Next() {
		var linkID string
		var tag Tag
		if e = tagRows.Scan(&linkID, &tag.ID, &tag.Name); e != nil {
			return nil, e
		}
		if i, ok := indices[linkID]; ok {
			out[i].Tags = append(out[i].Tags, tag)
		}
	}
	if e = tagRows.Err(); e != nil {
		return nil, e
	}
	return out, nil
}
func (s *PGStore) GetLink(owner, id string) (Link, error) {
	ctx, cancel := operationContext()
	defer cancel()
	l, e := scanLink(s.pool.QueryRow(ctx, `SELECT id::text,slug,target_url,enabled,created_at,updated_at FROM links WHERE id=$1 AND owner_id=$2`, id, owner))
	if e != nil {
		return Link{}, translate(e)
	}
	l.Tags, e = readTags(ctx, s.pool, l.ID)
	return l, e
}
func (s *PGStore) UpdateLink(owner, id string, in LinkInput) (Link, error) {
	ctx, cancel := operationContext()
	defer cancel()
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return Link{}, e
	}
	defer tx.Rollback(ctx)
	l, e := scanLink(tx.QueryRow(ctx, `UPDATE links SET slug=$3,target_url=$4,enabled=$5,updated_at=now() WHERE id=$1 AND owner_id=$2 RETURNING id::text,slug,target_url,enabled,created_at,updated_at`, id, owner, in.Slug, in.TargetURL, *in.Enabled))
	if e != nil {
		return Link{}, translate(e)
	}
	if e = replaceTags(ctx, tx, owner, id, in.Tags); e != nil {
		return Link{}, translate(e)
	}
	l.Tags, e = readTags(ctx, tx, id)
	if e != nil {
		return Link{}, e
	}
	return l, tx.Commit(ctx)
}
func (s *PGStore) DeleteLink(owner, id string) error {
	ctx, cancel := operationContext()
	defer cancel()
	tag, e := s.pool.Exec(ctx, `DELETE FROM links WHERE id=$1 AND owner_id=$2`, id, owner)
	if e != nil {
		return translate(e)
	}
	if tag.RowsAffected() == 0 {
		return errNotFound
	}
	return nil
}
func (s *PGStore) CreateTag(owner string, in TagInput) (Tag, error) {
	ctx, cancel := operationContext()
	defer cancel()
	var t Tag
	e := s.pool.QueryRow(ctx, `INSERT INTO tags(owner_id,name,name_normalized) VALUES($1,$2,$3) RETURNING id::text,name`, owner, strings.TrimSpace(in.Name), strings.ToLower(strings.TrimSpace(in.Name))).Scan(&t.ID, &t.Name)
	return t, translate(e)
}
func (s *PGStore) ListTags(owner string) ([]Tag, error) {
	ctx, cancel := operationContext()
	defer cancel()
	rows, e := s.pool.Query(ctx, `SELECT id::text,name FROM tags WHERE owner_id=$1 ORDER BY name_normalized,id`, owner)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	v := []Tag{}
	for rows.Next() {
		var t Tag
		if e = rows.Scan(&t.ID, &t.Name); e != nil {
			return nil, e
		}
		v = append(v, t)
	}
	return v, rows.Err()
}
func (s *PGStore) UpdateTag(owner, id string, in TagInput) (Tag, error) {
	ctx, cancel := operationContext()
	defer cancel()
	var t Tag
	e := s.pool.QueryRow(ctx, `UPDATE tags SET name=$3,name_normalized=$4 WHERE id=$1 AND owner_id=$2 RETURNING id::text,name`, id, owner, strings.TrimSpace(in.Name), strings.ToLower(strings.TrimSpace(in.Name))).Scan(&t.ID, &t.Name)
	if e != nil {
		return t, translate(e)
	}
	return t, nil
}
func (s *PGStore) DeleteTag(owner, id string) error {
	ctx, cancel := operationContext()
	defer cancel()
	tag, e := s.pool.Exec(ctx, `DELETE FROM tags WHERE id=$1 AND owner_id=$2`, id, owner)
	if e != nil {
		return translate(e)
	}
	if tag.RowsAffected() == 0 {
		return errNotFound
	}
	return nil
}
func (s *PGStore) RedirectTarget(slug string) (string, error) {
	ctx, cancel := operationContext()
	defer cancel()
	var target string
	e := s.pool.QueryRow(ctx, `SELECT target_url FROM links WHERE slug=$1 AND enabled=true`, slug).Scan(&target)
	if e != nil {
		return "", translate(e)
	}
	return target, nil
}
func operationContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 6*time.Second)
}

type execer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func replaceTags(ctx context.Context, db execer, owner, link string, names []string) error {
	if _, e := db.Exec(ctx, `DELETE FROM link_tags WHERE link_id=$1`, link); e != nil {
		return e
	}
	for _, name := range names {
		tag, e := db.Exec(ctx, `INSERT INTO link_tags(link_id,tag_id) SELECT $1,id FROM tags WHERE owner_id=$2 AND name_normalized=$3`, link, owner, strings.ToLower(strings.TrimSpace(name)))
		if e != nil {
			return e
		}
		if tag.RowsAffected() != 1 {
			return errNotFound
		}
	}
	return nil
}

type queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func readTags(ctx context.Context, db queryer, link string) ([]Tag, error) {
	rows, e := db.Query(ctx, `SELECT t.id::text,t.name FROM tags t JOIN link_tags lt ON lt.tag_id=t.id WHERE lt.link_id=$1 ORDER BY t.name_normalized,t.id`, link)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Tag{}
	for rows.Next() {
		var t Tag
		if e = rows.Scan(&t.ID, &t.Name); e != nil {
			return nil, e
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
func translate(e error) error {
	if e == nil {
		return nil
	}
	if errors.Is(e, pgx.ErrNoRows) {
		return errNotFound
	}
	var p *pgconn.PgError
	if errors.As(e, &p) {
		switch p.Code {
		case "23505":
			return errConflict
		case "23503":
			return errNotFound
		case "22P02":
			return errNotFound
		}
	}
	return e
}
