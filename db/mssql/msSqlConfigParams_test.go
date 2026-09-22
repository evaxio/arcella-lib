package mssql

import (
	"path/filepath"
	"testing"

	"axgit.vixiv.ru/snake/arcella-lib/app/v3/config/basic"
)

func TestConfigMSDBDefaults(t *testing.T) {
	t.Setenv("cfg", filepath.Join(t.TempDir(), "does-not-exist.yml"))
	for _, env := range []string{
		"MSSQL_DSN", "MSSQL_MAX_OPEN_CONNS", "MSSQL_MAX_IDLE_CONNS", "MSSQL_CONN_MAX_LIFETIME", "MSSQL_CONN_MAX_IDLE_TIME",
	} {
		t.Setenv(env, "")
	}

	var cfg ConfigMSDB
	if err := basic.NewBasicConfig(&cfg).Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DSN != "" {
		t.Errorf("DSN = %q, want empty", cfg.DSN)
	}
	if cfg.MaxOpenConns != -1 {
		t.Errorf("MaxOpenConns = %d, want -1", cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns != -1 {
		t.Errorf("MaxIdleConns = %d, want -1", cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime != 3600 {
		t.Errorf("ConnMaxLifetime = %d, want 3600", cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime != 1800 {
		t.Errorf("ConnMaxIdleTime = %d, want 1800", cfg.ConnMaxIdleTime)
	}
}
