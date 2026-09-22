package bbolt

import (
	"context"
	"fmt"
	log "log/slog"
)

// Deprecated: BBLogger is not wired to any bolt.Options.Logger; it is kept for compatibility only.
type BBLogger struct {
}

func (b *BBLogger) Debug(v ...interface{}) {
	if log.Default().Enabled(context.Background(), log.LevelDebug) {
		log.Debug(fmt.Sprintf("%v", v...))
	}
}

func (b *BBLogger) Debugf(format string, v ...interface{}) {
	if log.Default().Enabled(context.Background(), log.LevelDebug) {
		log.Debug(fmt.Sprintf(format, v...))
	}
}

func (b *BBLogger) Error(v ...interface{}) {
	if log.Default().Enabled(context.Background(), log.LevelError) {
		log.Error(fmt.Sprintf("%v", v...))
	}
}

func (b *BBLogger) Errorf(format string, v ...interface{}) {
	if log.Default().Enabled(context.Background(), log.LevelError) {
		log.Error(fmt.Sprintf(format, v...))
	}
}

func (b *BBLogger) Info(v ...interface{}) {
	if log.Default().Enabled(context.Background(), log.LevelInfo) {
		log.Info(fmt.Sprintf("%v", v...))
	}
}

func (b *BBLogger) Infof(format string, v ...interface{}) {
	if log.Default().Enabled(context.Background(), log.LevelInfo) {
		log.Info(fmt.Sprintf(format, v...))
	}
}

func (b *BBLogger) Warning(v ...interface{}) {
	if log.Default().Enabled(context.Background(), log.LevelWarn) {
		log.Warn(fmt.Sprintf("%v", v...))
	}
}

func (b *BBLogger) Warningf(format string, v ...interface{}) {
	if log.Default().Enabled(context.Background(), log.LevelWarn) {
		log.Warn(fmt.Sprintf(format, v...))
	}
}

func (b *BBLogger) Fatal(v ...interface{}) {
	if log.Default().Enabled(context.Background(), log.LevelError) {
		log.Error("FATAL: " + fmt.Sprintf("%v", v...))
	}
}

func (b *BBLogger) Fatalf(format string, v ...interface{}) {
	if log.Default().Enabled(context.Background(), log.LevelError) {
		log.Error("FATAL: " + fmt.Sprintf(format, v...))
	}
}

func (b *BBLogger) Panic(v ...interface{}) {
	if log.Default().Enabled(context.Background(), log.LevelError) {
		log.Error("PANIC: " + fmt.Sprintf("%v", v...))
	}
}

func (b *BBLogger) Panicf(format string, v ...interface{}) {
	if log.Default().Enabled(context.Background(), log.LevelError) {
		log.Error("PANIC: " + fmt.Sprintf(format, v...))
	}
}
