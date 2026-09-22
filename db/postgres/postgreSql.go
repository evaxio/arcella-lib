package postgres

import (
	"context"
	"database/sql"
	"errors"
	log "log/slog"
	"math/rand"
	"regexp"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

var notifyNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

const opTimeout = 5 * time.Second

const (
	restartBaseDelay = 5 * time.Second
	restartMaxDelay  = 5 * time.Minute
)

type PostgreSQL struct {
	config   *ConfigPostgreSQL
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	dbpool   *pgxpool.Pool
	stdlibDB *sql.DB
	// restart
	restarting         bool
	lastRestartAttempt time.Time
	restartBackoff     time.Duration
	// notify
	conn           *pgxpool.Conn
	notificationCh chan *pgconn.Notification
	notifyName     string
	errorCh        chan string
}

type PGOption func(s *PostgreSQL)

func OptionNotifyListener(notifyName string) PGOption {
	return func(s *PostgreSQL) {
		s.notifyName = notifyName
	}
}

func NewPostgreSQL(config *ConfigPostgreSQL, opts ...PGOption) *PostgreSQL {
	log.Debug("PostgreSQL")
	p := PostgreSQL{
		config:         config,
		errorCh:        make(chan string, 1),
		notificationCh: make(chan *pgconn.Notification, 64),
	}
	for _, opt := range opts {
		opt(&p)
	}
	return &p
}

func (s *PostgreSQL) GetName() string {
	return "PostgreSQL"
}

func (s *PostgreSQL) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.restarting {
		return errors.New("restart is in progress")
	}
	return s.doStart(ctx)
}

func (s *PostgreSQL) doStart(ctx context.Context) error {
	// s.ctx = ctx
	s.ctx, s.cancel = context.WithCancel(context.Background())
	cfg, err := pgxpool.ParseConfig(s.config.DSN)
	if err != nil {
		return err
	}
	if s.config.MinConns > -1 {
		cfg.MinConns = s.config.MinConns
	}
	if s.config.MaxConns > -1 {
		cfg.MaxConns = s.config.MaxConns
	}
	if s.config.HealthCheckPeriod.Seconds() > 0 {
		cfg.HealthCheckPeriod = s.config.HealthCheckPeriod
	}
	if s.config.MaxConnIdleTime > 0 {
		cfg.MaxConnIdleTime = s.config.MaxConnIdleTime
	}
	if s.config.MaxConnLifetime > 0 {
		cfg.MaxConnLifetime = s.config.MaxConnLifetime
	}
	if s.config.MaxConnLifetimeJitter > 0 {
		cfg.MaxConnLifetimeJitter = s.config.MaxConnLifetimeJitter
	}
	s.dbpool, err = pgxpool.NewWithConfig(s.ctx, cfg)
	if err == nil {
		s.stdlibDB = stdlib.OpenDBFromPool(s.dbpool)
		if s.config.ShowInitParams {
			log.Info("Start",
				log.Int64("MinConns", int64(s.config.MinConns)),
				log.Int64("MaxConns", int64(s.config.MaxConns)),
				log.Duration("HealthCheckPeriod", s.config.HealthCheckPeriod),
				log.Duration("MaxConnIdleTime", s.config.MaxConnIdleTime),
				log.Duration("MaxConnLifetime", s.config.MaxConnLifetime),
				log.Duration("MaxConnLifetimeJitter", s.config.MaxConnLifetimeJitter),
				log.Bool("WithInitPing", s.config.WithInitPing),
			)
		}
		if s.notifyName != "" { // notify
			if !notifyNamePattern.MatchString(s.notifyName) {
				return errors.New("invalid notify channel name: " + s.notifyName)
			}
			// LISTEN is per-session: it must run on the same conn that reads the notifications
			if s.conn, err = s.dbpool.Acquire(s.ctx); err == nil {
				if _, err = s.conn.Exec(context.Background(), "LISTEN "+s.notifyName); err == nil {
					log.Debug("Ready to wait notify", log.String("eventName", s.notifyName))
					go s.listenForEvent(s.conn, s.ctx)
					return s.dbpool.Ping(ctx)
				}
				s.conn.Release()
				s.conn = nil
				return err
			}
			return err
		}
		if s.config.WithInitPing {
			log.Info("WithInitPing")
			return s.dbpool.Ping(ctx)
		}
	}
	return err
}

func (s *PostgreSQL) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.doStop()
}

func (s *PostgreSQL) doStop() error {
	if s.notifyName != "" { // notify
		if s.conn != nil {
			s.conn.Release()
			s.conn = nil
			log.Debug("notify listener closed")
		}
	}
	if s.cancel != nil {
		log.Debug("do cancel")
		s.cancel()
		s.cancel = nil
	}
	if s.dbpool != nil {
		s.dbpool.Close()
		s.dbpool = nil
	}
	s.stdlibDB = nil
	return nil
}

func (s *PostgreSQL) GetPgxDB() *pgxpool.Pool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dbpool
}

func (s *PostgreSQL) GetDB() *sql.DB {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dbpool == nil {
		return nil
	}
	if s.stdlibDB == nil {
		s.stdlibDB = stdlib.OpenDBFromPool(s.dbpool)
	}
	return s.stdlibDB
}

func (s *PostgreSQL) LogStats() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logStats()
}

func (s *PostgreSQL) logStats() {
	if s.dbpool == nil {
		return
	}
	stat := s.dbpool.Stat()
	log.Debug("LogStats",
		log.Int64("MaxConns", int64(stat.MaxConns())),
		log.Int64("AcquiredConns", int64(stat.AcquiredConns())),
		log.Int64("TotalConns", int64(stat.TotalConns())),
		log.Int64("IdleConns", int64(stat.IdleConns())),
	)
}

func (s *PostgreSQL) DoRestart() error {
	s.mu.Lock()
	if s.restarting {
		s.mu.Unlock()
		return errors.New("restart is in progress")
	}
	s.restarting = true
	if s.restartBackoff == 0 {
		s.restartBackoff = restartBaseDelay
	} else {
		s.restartBackoff *= 2
		if s.restartBackoff > restartMaxDelay {
			s.restartBackoff = restartMaxDelay
		}
	}
	delay := jitterDuration(s.restartBackoff)
	s.lastRestartAttempt = time.Now()
	s.logStats()
	log.Debug("DoRestart", log.Duration("delay", delay))
	stopErr := s.doStop()
	s.mu.Unlock()
	if stopErr != nil {
		s.mu.Lock()
		s.restarting = false
		s.mu.Unlock()
		log.Error("Stop error", log.String("Message", stopErr.Error()))
		return stopErr
	}
	time.Sleep(delay)
	s.mu.Lock()
	startErr := s.doStart(context.Background())
	if startErr != nil {
		log.Error("Start error", log.String("Message", startErr.Error()))
	} else {
		s.restartBackoff = 0
	}
	s.restarting = false
	s.mu.Unlock()
	return startErr
}

func jitterDuration(d time.Duration) time.Duration {
	jitter := d / 5
	if jitter == 0 {
		return d
	}
	return d - jitter + time.Duration(rand.Int63n(int64(2*jitter)+1))
}

func (s *PostgreSQL) DoPing() {
	log.Debug("DoPing")
	s.mu.Lock()
	parent := s.ctx
	pool := s.dbpool
	s.mu.Unlock()
	if pool == nil {
		return
	}
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, opTimeout)
	defer cancel()
	err := pool.Ping(ctx)
	if err != nil {
		log.Error("Ping error", log.String("Message", err.Error()))
		s.DoRestart()
	}
}

func (s *PostgreSQL) Query(sql string, args ...any) (pgx.Rows, error) {
	s.mu.Lock()
	ctx, pool := s.ctx, s.dbpool
	s.mu.Unlock()
	return pool.Query(ctx, sql, args...)
}

func (s *PostgreSQL) QueryRow(sql string, args ...any) pgx.Row {
	s.mu.Lock()
	ctx, pool := s.ctx, s.dbpool
	s.mu.Unlock()
	return pool.QueryRow(ctx, sql, args...)
}

func (s *PostgreSQL) Exec(sql string, args ...any) (pgconn.CommandTag, error) {
	s.mu.Lock()
	ctx, pool := s.ctx, s.dbpool
	s.mu.Unlock()
	return pool.Exec(ctx, sql, args...)
}

func (s *PostgreSQL) GetListenChan() (chan *pgconn.Notification, chan string) {
	return s.notificationCh, s.errorCh
}

func (s *PostgreSQL) listenForEvent(conn *pgxpool.Conn, ctx context.Context) {
	for {
		if notificationEvent, err := conn.Conn().WaitForNotification(ctx); err == nil {
			select {
			case s.notificationCh <- notificationEvent:
			default:
				log.Warn("notification channel is full, dropping notification", log.String("channel", notificationEvent.Channel))
			}
		} else {
			log.Warn("WaitForNotification", log.String("Message", err.Error()))
			s.errorCh <- err.Error()
			go s.DoPing()
			break
		}
	}
}
