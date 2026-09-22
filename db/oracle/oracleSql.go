package oracle

import (
	"context"
	"database/sql"
	"errors"
	log "log/slog"
	"sync"
	"time"

	goOra "github.com/sijms/go-ora/v2"
)

const opTimeout = 5 * time.Second

const startupTimeout = 10 * time.Second

type OracleSQL struct {
	config     *ConfigOracleDB
	mu         sync.Mutex
	connection *sql.DB
	ctx        context.Context
}

func NewOracleDB(config *ConfigOracleDB) *OracleSQL {
	return &OracleSQL{
		config: config,
	}
}

func (s *OracleSQL) GetName() string {
	return "OracleSQL"
}

func (s *OracleSQL) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.doStart(ctx)
}

func (s *OracleSQL) doStart(ctx context.Context) (err error) {
	s.ctx = ctx
	urlOptions := map[string]string{}
	if len(s.config.TraceFile) > 0 {
		urlOptions["trace file"] = s.config.TraceFile // "trace.log"
		log.Debug("Trace", log.String("file", s.config.TraceFile))
	}
	for k, v := range s.config.UrlParams {
		log.Debug("options", log.String(k, v))
		urlOptions[k] = v
	}
	databaseURL := s.config.Dsn
	if databaseURL == "" {
		databaseURL = goOra.BuildUrl(s.config.Server, s.config.Port, s.config.Service, s.config.User, s.config.Password, urlOptions)
	}

	if s.connection, err = sql.Open("oracle", databaseURL); err == nil {
		log.Debug("connected", log.Any("Status", s.connection.Stats()))
		if s.config.MaxOpenConns > 0 {
			s.connection.SetMaxOpenConns(s.config.MaxOpenConns)
		}
		if s.config.MaxIdleConns > 0 {
			s.connection.SetMaxIdleConns(s.config.MaxIdleConns)
		}
		if s.config.ConnMaxLifetime > 0 {
			s.connection.SetConnMaxLifetime(time.Duration(s.config.ConnMaxLifetime) * time.Second)
		}
		if s.config.ConnMaxIdleTime > 0 {
			s.connection.SetConnMaxIdleTime(time.Duration(s.config.ConnMaxIdleTime) * time.Second)
		}
		if !s.config.InitWithoutPing {
			parent := ctx
			if parent == nil {
				parent = context.Background()
			}
			pingCtx, cancel := context.WithTimeout(parent, startupTimeout)
			defer cancel()
			err = s.connection.PingContext(pingCtx)
		} else {
			log.Warn("InitWithoutPing")
		}
	}
	// if s.connection, err = goOra.NewConnection(databaseURL); err == nil {err = s.connection.Ping()}
	return err
}

func (s *OracleSQL) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.doStop()
}

func (s *OracleSQL) doStop() error {
	if s.connection == nil {
		return nil
	}
	err := s.connection.Close()
	s.connection = nil
	return err
}

// Deprecated: checkConnection is not used; call DoPing and DoRestart directly.
func (s *OracleSQL) checkConnection() *sql.DB {
	// log.Debug("checkConnection", log.Any("Driver", s.connection.Driver()))
	if err := s.DoPing(); err != nil {
		log.Error("checkConnection", log.String("Message", err.Error()))
		s.DoRestart()
		s.mu.Lock()
		var openConnections int
		if s.connection != nil {
			openConnections = s.connection.Stats().OpenConnections
		}
		s.mu.Unlock()
		log.Debug("DoRestart.done", log.Int("OpenConnections", openConnections))
	}
	// log.Debug("pinged")
	return s.GetDB()
}

func (s *OracleSQL) GetDB() *sql.DB {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.connection
	// return s.connection
}

func (s *OracleSQL) DoRestart() {
	s.mu.Lock()
	defer s.mu.Unlock()
	log.Debug("DoRestart")
	stopped := s.connection
	err := s.doStop()
	if err != nil {
		log.Error("Stop error", log.String("Message", err.Error()))
	} else if stopped != nil {
		log.Debug("stopped", log.Any("Driver", stopped.Driver()))
	}
	err = s.doStart(s.ctx)
	if err != nil {
		log.Error("Start error", log.String("Message", err.Error()))
		log.Error("connection is unavailable, GetDB returns nil")
	}
}

func (s *OracleSQL) DoPing() error {
	//log.Debug("doPing", log.Any("status", s.connection.Stats()))
	s.mu.Lock()
	parent := s.ctx
	conn := s.connection
	s.mu.Unlock()
	if conn == nil {
		return errors.New("connection is nil")
	}
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, opTimeout)
	defer cancel()
	return conn.PingContext(ctx)
	//return nil
}
