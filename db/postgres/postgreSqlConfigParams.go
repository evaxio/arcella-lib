package postgres

import "time"

type ConfigPostgreSQL struct {
	DSN                   string        `yaml:"dsn"                    env:"PG_DSN"`
	MinConns              int32         `yaml:"minConns"               env:"PG_CONS_MIN"    default:"-1"`
	MaxConns              int32         `yaml:"maxConns"               env:"PG_CONS_MAX"    default:"-1"`
	HealthCheckPeriod     time.Duration `yaml:"healthCheckPeriod"      env:"PG_HCHK_PERIOD" default:"0s"`
	MaxConnIdleTime       time.Duration `yaml:"maxConnIdleTime"                             default:"30m"`
	MaxConnLifetime       time.Duration `yaml:"maxConnLifetime"                             default:"1h"`
	MaxConnLifetimeJitter time.Duration `yaml:"maxConnLifetimeJitter"                       default:"0s"`
	WithInitPing          bool          `yaml:"withInitPing"           env:"PG_SHOW_INIT_PING"`
	ShowInitParams        bool          `yaml:"showInitParams"         env:"PG_SHOW_INIT_PARAMS"`
}
