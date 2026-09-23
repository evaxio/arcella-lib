package postgres

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/evaxio/arcella-lib/app/v3/config/basic"
)

func TestConfigPostgreSQLDefaults(t *testing.T) {
	t.Setenv("cfg", filepath.Join(t.TempDir(), "does-not-exist.yml"))
	for _, env := range []string{
		"PG_DSN", "PG_CONS_MIN", "PG_CONS_MAX", "PG_HCHK_PERIOD", "PG_SHOW_INIT_PING", "PG_SHOW_INIT_PARAMS",
	} {
		t.Setenv(env, "")
	}

	var cfg ConfigPostgreSQL
	if err := basic.NewBasicConfig(&cfg).Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DSN != "" {
		t.Errorf("DSN = %q, want empty", cfg.DSN)
	}
	if cfg.MinConns != -1 {
		t.Errorf("MinConns = %d, want -1", cfg.MinConns)
	}
	if cfg.MaxConns != -1 {
		t.Errorf("MaxConns = %d, want -1", cfg.MaxConns)
	}
	if cfg.HealthCheckPeriod != 0 {
		t.Errorf("HealthCheckPeriod = %v, want 0", cfg.HealthCheckPeriod)
	}
	if cfg.MaxConnIdleTime != 30*time.Minute {
		t.Errorf("MaxConnIdleTime = %v, want 30m", cfg.MaxConnIdleTime)
	}
	if cfg.MaxConnLifetime != time.Hour {
		t.Errorf("MaxConnLifetime = %v, want 1h", cfg.MaxConnLifetime)
	}
	if cfg.MaxConnLifetimeJitter != 0 {
		t.Errorf("MaxConnLifetimeJitter = %v, want 0", cfg.MaxConnLifetimeJitter)
	}
	if cfg.WithInitPing {
		t.Error("WithInitPing = true, want false")
	}
	if cfg.ShowInitParams {
		t.Error("ShowInitParams = true, want false")
	}
}
