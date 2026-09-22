package fail2ban

import (
	"github.com/gin-gonic/gin"
	log "log/slog"
	"net/http"
	"time"
)

type Fail2Ban struct {
	suspected     *Suspected
	suspended     *Suspended
	watchPeriod   time.Duration
	watchCnt      int
	LogBannedPath bool
}

func newFail2Ban(config *ConfigFail2Ban) *Fail2Ban {
	log.Debug("Fail2Ban", log.Duration("watchPeriod", config.WatchPeriod), log.Int("maxCount", config.WatchCount), log.Duration("banTime", config.BanPeriod), log.Bool("LogBannedPath", config.LogBannedPath))
	return &Fail2Ban{
		LogBannedPath: config.LogBannedPath,
		watchCnt:      config.WatchCount - 1,
		watchPeriod:   config.WatchPeriod,
		suspected:     NewSuspected(config.WatchPeriod),
		suspended:     NewSuspended(config.BanPeriod),
	}
}

func GetFail2Ban(config *ConfigFail2Ban) gin.HandlerFunc {
	f2ban := newFail2Ban(config)
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if f2ban.suspended.Is(ip) {
			if f2ban.LogBannedPath {
				log.Debug("BANNED", log.String("ip", ip), log.String("method", c.Request.Method), log.String("path", c.Request.URL.Path))
			}
			c.AbortWithStatus(http.StatusNotFound)
		} else {
			c.Next()
			if http.StatusNotFound == c.Writer.Status() {
				f2ban.processStatusNotFound(ip)
			}
		}
	}
}

func (fb *Fail2Ban) processStatusNotFound(ip string) {
	count := fb.suspected.Incr(ip)
	if count > fb.watchCnt { // BANNED
		fb.suspected.Del(ip)
		fb.suspended.Add(ip)
		if !fb.LogBannedPath { // just once if not logging wrong path
			log.Debug("BANNED", log.String("ip", ip))
		}
	}
}
