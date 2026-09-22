package oracle

import (
	"path/filepath"
	"testing"

	"axgit.vixiv.ru/snake/arcella-lib/app/v3/config/basic"
)

func TestConfigOracleDBDefaults(t *testing.T) {
	t.Setenv("cfg", filepath.Join(t.TempDir(), "does-not-exist.yml"))
	for _, env := range []string{
		"ORA_HOST", "ORA_PORT", "ORA_SERVICE", "ORA_USER", "ORA_PASS", "ORA_TRACE", "ORA_DSN", "ORA_OPTIONS",
		"ORA_INIT_WITHOUT_PING", "ORA_MAX_OPEN_CONNS", "ORA_MAX_IDLE_CONNS", "ORA_CONN_MAX_LIFETIME", "ORA_CONN_MAX_IDLE_TIME",
	} {
		t.Setenv(env, "")
	}

	var cfg ConfigOracleDB
	if err := basic.NewBasicConfig(&cfg).Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server != "" {
		t.Errorf("Server = %q, want empty", cfg.Server)
	}
	if cfg.Port != 1521 {
		t.Errorf("Port = %d, want 1521", cfg.Port)
	}
	if cfg.InitWithoutPing {
		t.Error("InitWithoutPing = true, want false")
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
