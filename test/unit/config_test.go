package unit

import (
	"os"
	"testing"

	"github.com/loangraph/backend/internal/config"
)

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_MAX_CONNS", "")
	t.Setenv("DB_MIN_CONNS", "")
	t.Setenv("DB_MAX_CONN_LIFETIME", "")

	cfg := config.Load()

	if cfg.Port != "8090" {
		t.Fatalf("expected default port 8090, got %s", cfg.Port)
	}
	if cfg.Env != "local" {
		t.Fatalf("expected default env local, got %s", cfg.Env)
	}
	if cfg.DBMaxConns != 25 {
		t.Fatalf("expected default DBMaxConns 25, got %d", cfg.DBMaxConns)
	}
}

func TestLoadConfigOverrides(t *testing.T) {
	t.Setenv("PORT", "9000")
	t.Setenv("APP_ENV", "dev")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/db")
	t.Setenv("DB_MAX_CONNS", "10")
	t.Setenv("DB_MIN_CONNS", "1")
	t.Setenv("DB_MAX_CONN_LIFETIME", "10m")

	cfg := config.Load()

	if cfg.Port != "9000" || cfg.Env != "dev" {
		t.Fatalf("config overrides not applied: %+v", cfg)
	}
	if cfg.DatabaseURL != "postgres://user:pass@localhost:5432/db" {
		t.Fatalf("database url override not applied")
	}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}
