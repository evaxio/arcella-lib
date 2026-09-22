package mssql

import (
	"context"
	"database/sql"
	"errors"
	log "log/slog"
	"sync"
	"time"

	_ "github.com/microsoft/go-mssqldb"
	mssql "github.com/microsoft/go-mssqldb"
)

const startupTimeout = 10 * time.Second

type MsSQL struct {
	config *ConfigMSDB
	mu     sync.Mutex
	db     *sql.DB
}

func NewMsSQL(config *ConfigMSDB) *MsSQL {
	return &MsSQL{config: config}
}

func (ms *MsSQL) GetName() string {
	return "MsSql"
}

func (ms *MsSQL) Start(ctx context.Context) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if ms.db != nil {
		return errors.New("mssql is already started")
	}
	var err error
	ms.db, err = sql.Open("sqlserver", ms.config.DSN)
	if err == nil {
		if ms.config.MaxOpenConns > 0 {
			ms.db.SetMaxOpenConns(ms.config.MaxOpenConns)
		}
		if ms.config.MaxIdleConns > 0 {
			ms.db.SetMaxIdleConns(ms.config.MaxIdleConns)
		}
		if ms.config.ConnMaxLifetime > 0 {
			ms.db.SetConnMaxLifetime(time.Duration(ms.config.ConnMaxLifetime) * time.Second)
		}
		if ms.config.ConnMaxIdleTime > 0 {
			ms.db.SetConnMaxIdleTime(time.Duration(ms.config.ConnMaxIdleTime) * time.Second)
		}
		log.Debug("Start sqlserver")
		err = ms.pingWithRetry(ctx)
		if err != nil {
			log.Error("Start ping error", log.String("Message", err.Error()))
			ms.db.Close()
			ms.db = nil
		}
	}
	return err
}

func (ms *MsSQL) pingWithRetry(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	pingCtx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()
	var err error
	for attempt := 0; ; attempt++ {
		if err = ms.db.PingContext(pingCtx); err == nil {
			return nil
		}
		if attempt >= 3 || !isTransient(err) {
			return err
		}
		log.Warn("Start ping error, retry", log.Int("attempt", attempt+1), log.String("Message", err.Error()))
		time.Sleep(time.Second)
	}
}

func (ms *MsSQL) Stop() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if ms.db == nil {
		return nil
	}
	err := ms.db.Close()
	ms.db = nil
	return err
}

func (ms *MsSQL) GetDB() *sql.DB {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.db
}

func isTransient(err error) bool {
	var sqlErr mssql.Error
	if !errors.As(err, &sqlErr) {
		return false
	}

	switch sqlErr.Number {
	case
		// Connection-establishment and transport transient errors.
		64, 233, 4060, 4221,
		10053, 10054,
		10928, 10929,

		// Azure SQL failover, throttling, and availability errors.
		40020, 40143, 40166,
		40197, 40501, 40540, 40613,
		42108, 42109,
		49918, 49919, 49920,

		// Common retryable statement-level contention errors.
		1205, 1222:
		return true
	default:
		return false
	}
}
