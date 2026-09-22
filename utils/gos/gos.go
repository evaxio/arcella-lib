package gos

import (
	"context"
	log "log/slog"
	"runtime/debug"
)

func SafeGoCtx(ctx context.Context, fn func(ctx context.Context)) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("🚩 panic", log.Any("r", r), log.String("stack", string(debug.Stack())))
			}
		}()
		fn(ctx)
	}()
}

func SafeGo(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error("🚩 panic", log.Any("r", r), log.String("stack", string(debug.Stack())))
			}
		}()
		fn()
	}()
}
