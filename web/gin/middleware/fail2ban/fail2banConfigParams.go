package fail2ban

import "time"

type ConfigFail2Ban struct {
	Disabled      bool          `yaml:"disabled"      env:"WEB_FAIL2BAN_DISABLED"                `
	LogBannedPath bool          `yaml:"logBannedPath" env:"WEB_FAIL2BAN_LOG_BANNED"              `
	WatchPeriod   time.Duration `yaml:"watchPeriod"   env:"WEB_FAIL2BAN_PERIOD"     default:"1m" `
	WatchCount    int           `yaml:"maxCount"      env:"WEB_FAIL2BAN_MAX"        default:"3"  `
	BanPeriod     time.Duration `yaml:"banTime"       env:"WEB_FAIL2BAN_BLOCK"      default:"15m"`
}
