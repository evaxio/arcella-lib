package gorm

import (
	"context"
	"errors"
	"testing"
	"time"

	gl "gorm.io/gorm/logger"
)

func TestLogModeReturnsReceiver(t *testing.T) {
	var _ gl.Interface = (*GormLoggerSlog)(nil)
	sl := New()
	for _, level := range []gl.LogLevel{gl.Silent, gl.Error, gl.Warn, gl.Info} {
		if got := sl.LogMode(level); got == nil {
			t.Fatalf("LogMode(%d) returned nil", int(level))
		}
	}
}

func TestLoggerCallsDoNotPanic(t *testing.T) {
	sl := New()
	ctx := context.Background()
	begin := time.Now()
	sl.LogMode(gl.Info)
	sl.Info(ctx, "info message")
	sl.Warn(ctx, "warn message")
	sl.Error(ctx, "error message")
	sl.Trace(ctx, begin, func() (string, int64) { return "SELECT 1", 1 }, nil)
	sl.Trace(ctx, begin, func() (string, int64) { return "SELECT 1", 0 }, errors.New("boom"))
}
