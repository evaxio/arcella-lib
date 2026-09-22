package gorm

type GormConfig struct {
	DSN            string `yaml:"dsn"            env:"GORM_DSN"                                `
	SQLiteFileName string `yaml:"sqLiteFileName" env:"GORM_SQLITE_FILE"`
	Migrate        bool   `yaml:"migrate"        env:"GORM_MIGRATE"       default:"false"      `
}
