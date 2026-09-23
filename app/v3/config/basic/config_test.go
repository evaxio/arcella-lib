package basic

import (
	"github.com/evaxio/arcella-lib/crypt/jasypt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigDefaultsAndEnv(t *testing.T) {
	t.Setenv("CFG_TEST_STR", "env-str")
	t.Setenv("CFG_TEST_INT", "77")

	type inner struct {
		Name string `default:"inner-name"`
	}
	type cfgT struct {
		Str    string  `default:"d-str" env:"CFG_TEST_STR"`
		Int    int     `default:"42" env:"CFG_TEST_INT"`
		Bool   bool    `default:"true"`
		Float  float64 `default:"1.25"`
		Dur    int64   `default:"1h"`
		NoDef  string
		Inner  inner
		PInner *inner
		Slice  []inner
		unexp  string
	}

	cfg := &cfgT{
		PInner: &inner{},
		Slice:  []inner{{Name: "s1"}, {}},
		unexp:  "keep-me",
	}
	if err := NewBasicConfig(cfg).Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Str != "env-str" {
		t.Errorf("Str = %q, want env-str", cfg.Str)
	}
	if cfg.Int != 77 {
		t.Errorf("Int = %d, want 77", cfg.Int)
	}
	if !cfg.Bool {
		t.Error("Bool = false, want true")
	}
	if cfg.Float != 1.25 {
		t.Errorf("Float = %v, want 1.25", cfg.Float)
	}
	if cfg.Dur != int64(time.Hour) {
		t.Errorf("Dur = %d, want %d", cfg.Dur, int64(time.Hour))
	}
	if cfg.NoDef != "" {
		t.Errorf("NoDef = %q, want empty", cfg.NoDef)
	}
	if cfg.Inner.Name != "inner-name" {
		t.Errorf("Inner.Name = %q, want inner-name", cfg.Inner.Name)
	}
	if cfg.PInner == nil || cfg.PInner.Name != "inner-name" {
		t.Errorf("PInner = %+v, want name inner-name", cfg.PInner)
	}
	if len(cfg.Slice) != 2 {
		t.Fatalf("Slice len = %d, want 2", len(cfg.Slice))
	}
	if cfg.Slice[0].Name != "s1" {
		t.Errorf("Slice[0].Name = %q, want s1 (set value must be preserved)", cfg.Slice[0].Name)
	}
	if cfg.Slice[1].Name != "inner-name" {
		t.Errorf("Slice[1].Name = %q, want inner-name (default)", cfg.Slice[1].Name)
	}
	if cfg.unexp != "keep-me" {
		t.Errorf("unexported field lost: %q", cfg.unexp)
	}
}

func TestConfigFromFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "config.yml")
	content := "str: file-str\nint: 5\n"
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("cfg", file)

	type cfgT struct {
		Str string `default:"d"`
		Int int    `default:"42"`
	}
	cfg := &cfgT{}
	if err := NewBasicConfig(cfg).Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Str != "file-str" {
		t.Errorf("Str = %q, want file-str", cfg.Str)
	}
	if cfg.Int != 5 {
		t.Errorf("Int = %d, want 5", cfg.Int)
	}
}

func TestConfigEncodedValue(t *testing.T) {
	t.Setenv("JASYPT_PASSWORD", "test-pass")
	t.Setenv("JASYPT_ITERATIONS", "100")
	enc, err := jasypt.Encrypt("test-pass", 100, "s3cret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	t.Setenv("CFG_TEST_SECRET", "ENC("+enc+")")

	type cfgT struct {
		Secret string `env:"CFG_TEST_SECRET"`
	}
	cfg := &cfgT{}
	if err := NewBasicConfig(cfg).Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Secret != "s3cret" {
		t.Errorf("Secret = %q, want s3cret", cfg.Secret)
	}
}

func TestConfigMissingFileSkipped(t *testing.T) {
	t.Setenv("cfg", filepath.Join(t.TempDir(), "does-not-exist.yml"))
	type cfgT struct {
		Str string `default:"d"`
	}
	cfg := &cfgT{}
	if err := NewBasicConfig(cfg).Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Str != "d" {
		t.Errorf("Str = %q, want d", cfg.Str)
	}
}

func TestLoadNotAPointerPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic for non-pointer config")
		}
	}()
	var cfgT struct{}
	NewBasicConfig(cfgT).Load()
}
