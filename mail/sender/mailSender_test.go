package sender

import (
	"context"
	"testing"

	basic "axgit.vixiv.ru/snake/arcella-lib/app/v3/config/basic"
)

// ConfigMailSender has no `default:` tags, so Load() with no config source
// must leave the struct at its zero values.
func TestConfigLoadZeroValues(t *testing.T) {
	var cfg ConfigMailSender
	if err := basic.NewBasicConfig(&cfg).Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server != "" || cfg.User != "" || cfg.Password != "" || cfg.WithSSL || cfg.DryRun {
		t.Errorf("Load with no config source = %+v, want zero values", cfg)
	}
}

func TestConfigLoadEnv(t *testing.T) {
	t.Setenv("MAIL_SERVER", "smtp.example.com:587")
	t.Setenv("MAIL_LOGIN", "user@example.com")
	t.Setenv("MAIL_PASS", "pass")
	t.Setenv("MAIL_SSL", "true")

	var cfg ConfigMailSender
	if err := basic.NewBasicConfig(&cfg).Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server != "smtp.example.com:587" {
		t.Errorf("Server = %q, want %q", cfg.Server, "smtp.example.com:587")
	}
	if cfg.User != "user@example.com" {
		t.Errorf("User = %q, want %q", cfg.User, "user@example.com")
	}
	if cfg.Password != "pass" {
		t.Errorf("Password = %q, want %q", cfg.Password, "pass")
	}
	if !cfg.WithSSL {
		t.Errorf("WithSSL = false, want true (env MAIL_SSL)")
	}
}

func TestStartInvalidServer(t *testing.T) {
	tests := []string{"", "no-port-here", "host:notaport"}
	for _, server := range tests {
		m := NewMailSender(&ConfigMailSender{Server: server})
		if err := m.Start(context.Background()); err == nil {
			t.Errorf("Start(server=%q) = nil error, want a clear error", server)
		}
	}
}

func TestStartValidServer(t *testing.T) {
	m := NewMailSender(&ConfigMailSender{Server: "smtp.example.com:587"})
	if err := m.Start(context.Background()); err != nil {
		t.Errorf("Start(valid server) = %v, want nil", err)
	}
	if err := m.Stop(); err != nil {
		t.Errorf("Stop without a connection = %v, want nil", err)
	}
}

func TestSendMessageInvalidServer(t *testing.T) {
	m := NewMailSender(&ConfigMailSender{Server: "host:notaport"})
	err := m.SendMessage("from@example.com", "to@example.com", "body", "subject", "text")
	if err == nil {
		t.Errorf("SendMessage with an invalid server = nil error, want a clear error")
	}
}
