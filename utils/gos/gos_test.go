package gos

import (
	"context"
	"testing"
	"time"
)

func TestSafeGo(t *testing.T) {
	done := make(chan struct{})
	SafeGo(func() { close(done) })
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("function did not run")
	}
}

func TestSafeGoRecoversPanic(t *testing.T) {
	done := make(chan struct{})
	SafeGo(func() {
		defer close(done)
		panic("boom")
	})
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("panic was not recovered (goroutine crashed)")
	}
}

func TestSafeGoCtx(t *testing.T) {
	got := make(chan context.Context, 1)
	SafeGoCtx(context.Background(), func(ctx context.Context) { got <- ctx })
	select {
	case c := <-got:
		if c == nil {
			t.Error("ctx is nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("function did not run")
	}
}
