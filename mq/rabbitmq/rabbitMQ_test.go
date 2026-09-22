package rabbitmq

import (
	"axgit.vixiv.ru/snake/arcella-lib/app/v3/config/basic"
	"context"
	"testing"
	"time"
)

func TestNewRabbitMQClampsConcurrency(t *testing.T) {
	cfg := &ConfigRabbitMQ{ConcurrencyCount: 0, RetryDelay: "5s"}
	r := NewRabbitMQ(cfg, "q", nil)
	if r.GetConf().ConcurrencyCount != 1 {
		t.Errorf("ConcurrencyCount = %d, want 1", r.GetConf().ConcurrencyCount)
	}
}

func TestStartEmptyUrl(t *testing.T) {
	cfg := &ConfigRabbitMQ{Url: "", ConcurrencyCount: 1, RetryDelay: "5s"}
	r := NewRabbitMQ(cfg, "q", nil)
	if err := r.Start(context.Background()); err == nil {
		t.Fatal("expected error for empty url")
	}
}

func TestStartStopWithoutConnection(t *testing.T) {
	cfg := &ConfigRabbitMQ{Url: "amqp://127.0.0.1:1", ConcurrencyCount: 1, RetryDelay: "1s"}
	r := NewRabbitMQ(cfg, "q", nil)
	if err := r.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := r.Stop(); err != nil {
		t.Errorf("Stop: %v", err)
	}
	if r.IsConnected() {
		t.Error("IsConnected = true after Stop")
	}
	if err := r.Stop(); err != nil {
		t.Errorf("second Stop: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
}

func TestProduceNotConnected(t *testing.T) {
	cfg := &ConfigRabbitMQ{Url: "", ConcurrencyCount: 1, RetryDelay: "5s"}
	r := NewRabbitMQ(cfg, "q", nil)
	data := []byte("payload")
	if err := r.Produce("q", &data); err == nil {
		t.Fatal("expected error when not connected")
	}
	if err := r.ProduceData("q", map[string]string{"k": "v"}); err == nil {
		t.Fatal("expected error when not connected")
	}
}

func TestProduceDisabled(t *testing.T) {
	cfg := &ConfigRabbitMQ{Url: "amqp://127.0.0.1:1", ConcurrencyCount: 1, Disabled: true, RetryDelay: "5s"}
	r := NewRabbitMQ(cfg, "q", nil)
	data := []byte("payload")
	if err := r.ProduceData("q", map[string]string{"k": "v"}); err != nil {
		t.Errorf("ProduceData disabled: %v", err)
	}
	if err := r.Start(context.Background()); err != nil {
		t.Errorf("Start disabled: %v", err)
	}
	if err := r.Stop(); err != nil {
		t.Errorf("Stop disabled: %v", err)
	}
	_ = data
}

func TestConfigDefaults(t *testing.T) {
	var cfg ConfigRabbitMQ
	if err := basic.NewBasicConfig(&cfg).Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ConcurrencyCount != 1 {
		t.Errorf("ConcurrencyCount = %d, want 1", cfg.ConcurrencyCount)
	}
	if cfg.RetryDelay != "5s" {
		t.Errorf("RetryDelay = %q, want %q", cfg.RetryDelay, "5s")
	}
	if !cfg.Quorum {
		t.Error("Quorum = false, want true")
	}
}
