package oracle

type ConfigOracleDB struct {
	Server          string            `yaml:"server"           env:"ORA_HOST"`
	Port            int               `yaml:"port"             env:"ORA_PORT"    default:"1521"`
	Service         string            `yaml:"service"          env:"ORA_SERVICE"`
	User            string            `yaml:"user"             env:"ORA_USER"`
	Password        string            `yaml:"pass"             env:"ORA_PASS"`
	TraceFile       string            `yaml:"trace"            env:"ORA_TRACE"`
	Dsn             string            `yaml:"dsn"              env:"ORA_DSN"`
	UrlParams       map[string]string `yaml:"options"          env:"ORA_OPTIONS"`
	InitWithoutPing bool              `yaml:"initWithoutPing"  env:"ORA_INIT_WITHOUT_PING"`                   // test purpose
	MaxOpenConns    int               `yaml:"maxOpenConns"     env:"ORA_MAX_OPEN_CONNS"       default:"-1"`   // -1 = unlimited
	MaxIdleConns    int               `yaml:"maxIdleConns"     env:"ORA_MAX_IDLE_CONNS"       default:"-1"`   // -1 = unlimited
	ConnMaxLifetime int               `yaml:"connMaxLifetime"  env:"ORA_CONN_MAX_LIFETIME"    default:"3600"` // seconds, 0 = driver default
	ConnMaxIdleTime int               `yaml:"connMaxIdleTime"  env:"ORA_CONN_MAX_IDLE_TIME"   default:"1800"` // seconds, 0 = driver default
}
