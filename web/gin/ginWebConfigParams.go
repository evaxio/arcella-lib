package gin

import (
	f2b "axgit.vixiv.ru/snake/arcella-lib/web/gin/middleware/fail2ban"
	"time"
)

type ConfigGinWeb struct {
	Port         int                `yaml:"port"          default:"8080"      env:"WEB_PORT"`
	StartMode    string             `yaml:"startMode"     default:"debug"     env:"WEB_MODE"`
	UseLogger    bool               `yaml:"logger"                            env:"WEB_LOGGER"`
	Static       string             `yaml:"static"                            env:"WEB_STATIC_DIR"`
	ReadTimeOut  time.Duration      `yaml:"readTimeOut"   default:"10s"       env:"WEB_READ_TIMEOUT"`
	WriteTimeOut time.Duration      `yaml:"writeTimeOut"  default:"10s"       env:"WEB_WRITE_TIMEOUT"`
	Fail2Ban     f2b.ConfigFail2Ban `yaml:"fail2ban"`
	RequestLog   bool               `yaml:"requestLog"                        env:"WEB_REQUEST_LOG"`
}
