package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Opt-in only: this test migrates and writes to the database named explicitly in
// LINKHUB_TEST_DATABASE_URL. It removes only its own uniquely named fixtures.
func TestPostgresOwnershipAndAtomicTags(t *testing.T) {
	dsn := os.Getenv("LINKHUB_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("set LINKHUB_TEST_DATABASE_URL to an isolated disposable PostgreSQL database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	s, e := OpenPG(ctx, dsn)
	if e != nil {
		t.Fatal("open isolated PostgreSQL:", e)
	}
	defer s.Close()
	b := make([]byte, 12)
	if _, e = rand.Read(b); e != nil {
		t.Fatal(e)
	}
	suffix := hex.EncodeToString(b)
	emailA, emailB := "a-"+suffix+"@example.test", "b-"+suffix+"@example.test"
	defer func() {
		_, _ = s.pool.Exec(context.Background(), `DELETE FROM users WHERE email_normalized IN ($1,$2)`, emailA, emailB)
	}()
	a, e := s.CreateUser(emailA, []byte("fixture-hash"))
	if e != nil {
		t.Fatal(e)
	}
	buser, e := s.CreateUser(emailB, []byte("fixture-hash"))
	if e != nil {
		t.Fatal(e)
	}
	tagA, e := s.CreateTag(a.ID, TagInput{Name: "private"})
	if e != nil {
		t.Fatal(e)
	}
	tagB, e := s.CreateTag(buser.ID, TagInput{Name: "foreign"})
	if e != nil {
		t.Fatal(e)
	}
	enabled := true
	link, e := s.CreateLink(a.ID, LinkInput{Slug: "a-" + suffix, TargetURL: "https://example.test", Enabled: &enabled, Tags: []string{tagA.Name}})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.GetLink(buser.ID, link.ID); e != errNotFound {
		t.Fatalf("foreign read error=%v", e)
	}
	if e = s.DeleteLink(buser.ID, link.ID); e != errNotFound {
		t.Fatalf("foreign delete error=%v", e)
	}
	if _, e = s.CreateLink(a.ID, LinkInput{Slug: "b-" + suffix, TargetURL: "https://example.test", Enabled: &enabled, Tags: []string{tagB.Name}}); e != errNotFound {
		t.Fatalf("foreign tag attach error=%v", e)
	}
	links, e := s.ListLinks(a.ID, 100, 0)
	if e != nil {
		t.Fatal(e)
	}
	if len(links) != 1 || links[0].ID != link.ID {
		t.Fatalf("failed association was not rolled back: %#v", links)
	}
	second, e := s.CreateLink(a.ID, LinkInput{Slug: "c-" + suffix, TargetURL: "https://example.test/second", Enabled: &enabled, Tags: []string{tagA.Name}})
	if e != nil {
		t.Fatal("create second tagged link:", e)
	}
	poolCfg, e := pgxpool.ParseConfig(dsn)
	if e != nil {
		t.Fatal(e)
	}
	poolCfg.MaxConns = 1
	limitedPool, e := pgxpool.NewWithConfig(ctx, poolCfg)
	if e != nil {
		t.Fatal(e)
	}
	defer limitedPool.Close()
	limitedStore := &PGStore{pool: limitedPool}
	links, e = limitedStore.ListLinks(a.ID, 100, 0)
	if e != nil {
		t.Fatal("single-connection listing:", e)
	}
	if len(links) != 2 {
		t.Fatalf("single-connection listing returned %d links, want 2", len(links))
	}
	for _, listed := range links {
		if len(listed.Tags) != 1 || listed.Tags[0].ID != tagA.ID {
			t.Fatalf("tags for link %s: %#v (second fixture %s)", listed.ID, listed.Tags, second.ID)
		}
	}
	if _, e = s.CreateLink(a.ID, LinkInput{Slug: link.Slug, TargetURL: "https://other.example", Enabled: &enabled}); e != errConflict {
		t.Fatalf("duplicate slug error=%v", e)
	}
	var version string
	if e = s.pool.QueryRow(ctx, `SELECT version FROM schema_migrations WHERE version='0001_init'`).Scan(&version); e != nil {
		t.Fatal("migration version missing:", e)
	}
}
