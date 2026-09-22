package otel

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/trace"
)

func TestNewSpanID(t *testing.T) {
	o := NewOtello(&ConfigOtello{})
	seen := make(map[trace.SpanID]bool, 1000)
	for i := 0; i < 1000; i++ {
		sid := o.NewSpanID()
		if !sid.IsValid() {
			t.Fatalf("invalid span id: %v", sid)
		}
		if seen[sid] {
			t.Fatalf("duplicate span id: %v", sid)
		}
		seen[sid] = true
	}
}

func TestStartStopUnreachable(t *testing.T) {
	o := NewOtello(&ConfigOtello{Url: "127.0.0.1:1", ServiceName: "test-svc"})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := o.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if !o.Inited() {
		t.Fatal("Inited = false after successful Start")
	}
	if err := o.Stop(); err != nil {
		t.Logf("Stop flushed pending telemetry to unreachable collector: %v", err)
	}
}

func TestStopWithoutStart(t *testing.T) {
	o := NewOtello(&ConfigOtello{Url: "127.0.0.1:1", ServiceName: "test-svc"})
	if err := o.Stop(); err != nil {
		t.Fatalf("Stop without Start: %v", err)
	}
}
