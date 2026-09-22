package mssql

type ConfigMSDB struct {
	DSN             string `yaml:"dsn"             env:"MSSQL_DSN"`                               // "sqlserver://user:password@localhost:1433?database=mydb"
	MaxOpenConns    int    `yaml:"maxOpenConns"    env:"MSSQL_MAX_OPEN_CONNS"    default:"-1"`    // -1 = unlimited
	MaxIdleConns    int    `yaml:"maxIdleConns"    env:"MSSQL_MAX_IDLE_CONNS"    default:"-1"`    // -1 = unlimited
	ConnMaxLifetime int    `yaml:"connMaxLifetime" env:"MSSQL_CONN_MAX_LIFETIME" default:"3600"`  // seconds, 0 = driver default
	ConnMaxIdleTime int    `yaml:"connMaxIdleTime" env:"MSSQL_CONN_MAX_IDLE_TIME" default:"1800"` // seconds, 0 = driver default
}
