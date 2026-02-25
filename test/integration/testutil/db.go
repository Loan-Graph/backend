package testutil

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_URL")
	if strings.TrimSpace(dsn) == "" {
		dsn = "postgres://loangraph:secret@localhost:5432/loangraph?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skip integration test (db connect init): %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skip integration test (db ping): %v", err)
	}
	return pool
}

func ApplyMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	paths := []string{
		filepath.Join("internal", "db", "migrations"),
		filepath.Join("..", "..", "internal", "db", "migrations"),
	}

	var migDir string
	for _, p := range paths {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			migDir = p
			break
		}
	}
	if migDir == "" {
		t.Fatalf("migrations directory not found")
	}

	files, err := filepath.Glob(filepath.Join(migDir, "*.up.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	sort.Strings(files)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read migration %s: %v", file, err)
		}
		for _, stmt := range strings.Split(string(content), ";") {
			q := strings.TrimSpace(stmt)
			if q == "" {
				continue
			}
			if _, err := pool.Exec(ctx, q); err != nil {
				t.Fatalf("exec migration %s: %v\nstmt=%s", file, err, q)
			}
		}
	}
}

func ResetTables(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	q := `
TRUNCATE TABLE
  outbox_jobs,
  chain_events,
  pools,
  passport_cache,
  repayments,
  loans,
  borrowers,
  lenders,
  auth_sessions,
  users,
  app_metadata
RESTART IDENTITY CASCADE
`
	if _, err := pool.Exec(ctx, q); err != nil {
		t.Fatalf("reset tables: %v", err)
	}
}
