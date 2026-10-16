package main

import (
	"context"
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/0001_init.up.sql migrations/0001_init.down.sql
var migrations embed.FS

func migrate(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version integer PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("migration setup failed")
	}
	var applied bool
	if err := db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=1)").Scan(&applied); err != nil {
		return fmt.Errorf("migration status unavailable")
	}
	if applied {
		return nil
	}
	sql, err := migrations.ReadFile("migrations/0001_init.up.sql")
	if err != nil {
		return fmt.Errorf("migration asset unavailable")
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("migration could not start")
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, string(sql), pgx.QueryExecModeSimpleProtocol); err != nil {
		return fmt.Errorf("schema migration failed")
	}
	if _, err = tx.Exec(ctx, "INSERT INTO schema_migrations(version) VALUES(1)"); err != nil {
		return fmt.Errorf("migration record failed")
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("migration commit failed")
	}
	return nil
}

func rollback(ctx context.Context, db *pgxpool.Pool) error {
	sql, err := migrations.ReadFile("migrations/0001_init.down.sql")
	if err != nil {
		return fmt.Errorf("rollback asset unavailable")
	}
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("rollback could not start")
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, string(sql), pgx.QueryExecModeSimpleProtocol); err != nil {
		return fmt.Errorf("schema rollback failed")
	}
	if _, err = tx.Exec(ctx, "DELETE FROM schema_migrations WHERE version=1"); err != nil {
		return fmt.Errorf("migration record could not be removed")
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("rollback commit failed")
	}
	return nil
}
