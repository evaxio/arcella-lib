package gorm

import (
	"context"
	log "log/slog"
	"sync/atomic"
	"time"

	gl "gorm.io/gorm/logger"
)

type GormLoggerSlog struct {
	level atomic.Int32
}

func New() *GormLoggerSlog {
	return &GormLoggerSlog{}
}

func (sl *GormLoggerSlog) LogMode(level gl.LogLevel) gl.Interface {
	log.Debug("🌌: LogMode", log.Int("level", int(level)))
	sl.level.Store(int32(level))
	return sl
}

// params and SQL may contain query values (credentials, PII) - do not log them

func (sl *GormLoggerSlog) Info(c context.Context, s string, params ...interface{}) {
	if sl.level.Load() >= int32(gl.Info) {
		log.Info("🌌: " + s)
	}
}

func (sl *GormLoggerSlog) Warn(c context.Context, s string, params ...interface{}) {
	if sl.level.Load() >= int32(gl.Warn) {
		log.Warn("🌌: " + s)
	}
}

func (sl *GormLoggerSlog) Error(c context.Context, s string, params ...interface{}) {
	if sl.level.Load() >= int32(gl.Error) {
		log.Error("🌌: " + s)
	}
}

func (sl *GormLoggerSlog) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if err != nil {
		if sl.level.Load() >= int32(gl.Error) {
			log.Error("🌌: Trace", log.String("message", err.Error()), log.Duration("time", time.Since(begin)))
		}
		return
	}
	if sl.level.Load() < int32(gl.Info) {
		return
	}
	_, rowsAffected := fc()
	log.Debug("🌌: Trace", log.Duration("time", time.Since(begin)), log.Int64("rowsAffected", rowsAffected))
}
