// Package db provides the PostgreSQL connection pool and all query helpers
// for the governance API. All queries use pgx v5 directly — no ORM.
package db

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool is the shared connection pool. Initialised once at startup.
var Pool *pgxpool.Pool

// Connect creates and validates the connection pool from the DATABASE_URL environment variable.
// DATABASE_URL format: postgres://user:password@host:port/dbname?sslmode=disable
func Connect(ctx context.Context) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = defaultDSN()
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("db: parse config: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return fmt.Errorf("db: create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("db: ping: %w", err)
	}
	Pool = pool
	return nil
}

// defaultDSN assembles the DSN from individual environment variables.
// Used when DATABASE_URL is not set (e.g. local development with podman compose).
func defaultDSN() string {
	host := envOr("POSTGRES_HOST", "localhost")
	port := envOr("POSTGRES_PORT", "5432")
	user := envOr("POSTGRES_USER", "governance")
	pass := envOr("POSTGRES_PASSWORD", "governance")
	dbname := envOr("POSTGRES_DB", "governance")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, dbname)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// RunMigrations applies all pending SQL migration files from the given directory.
// Files are applied in alphabetical order; each is run in its own transaction.
// Idempotent: skips files already recorded in the migrations table.
func RunMigrations(ctx context.Context, migrationsDir string) error {
	if err := ensureMigrationsTable(ctx); err != nil {
		return err
	}
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("db: read migrations dir %q: %w", migrationsDir, err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		applied, err := isMigrationApplied(ctx, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		path := fmt.Sprintf("%s/%s", migrationsDir, name)
		sql, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("db: read migration %q: %w", name, err)
		}
		if err := applyMigration(ctx, name, string(sql)); err != nil {
			return fmt.Errorf("db: apply migration %q: %w", name, err)
		}
	}
	return nil
}

func ensureMigrationsTable(ctx context.Context) error {
	_, err := Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			name       TEXT PRIMARY KEY,
			applied_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)`)
	return err
}

func isMigrationApplied(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := Pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE name = $1)`, name,
	).Scan(&exists)
	return exists, err
}

func applyMigration(ctx context.Context, name, sql string) error {
	tx, err := Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, sql); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO schema_migrations (name) VALUES ($1)`, name,
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ============================================================
// JSON helpers for JSONB columns
// ============================================================

// MarshalJSON marshals v to []byte. Panics only on programming errors (non-marshalable types).
func MarshalJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("db.MarshalJSON: %v", err))
	}
	return b
}

// CollectRows converts pgx.Rows to a slice of T using pgx.RowToStructByName.
func CollectRows[T any](rows pgx.Rows, err error) ([]T, error) {
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowToStructByName[T])
}
