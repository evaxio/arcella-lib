package rateLimiter

import (
	log "log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"golang.org/x/time/rate"
)

const (
	limiterEvictThreshold = 1024
	limiterEvictInterval  = 30 * time.Second
	limiterIdleTimeout    = 5 * time.Minute
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func RateLimiter(cfg *ConfigRateLimiter) gin.HandlerFunc {
	var mu sync.Mutex
	limits := make(map[string]*ipLimiter)
	var lastSweep time.Time
	log.Debug("Limiter", log.Int("requests", cfg.Limit), log.Int("per seconds", cfg.Seconds))
	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		lim := limits[ip]
		if lim == nil {
			lim = &ipLimiter{limiter: rate.NewLimiter(rate.Limit(cfg.Seconds), cfg.Limit)} // 1,10 =  10 requests per second
			limits[ip] = lim
		}
		lim.lastSeen = time.Now()
		if len(limits) > limiterEvictThreshold && time.Since(lastSweep) > limiterEvictInterval {
			now := time.Now()
			for k, entry := range limits {
				if now.Sub(entry.lastSeen) > limiterIdleTimeout {
					delete(limits, k)
				}
			}
			lastSweep = now
		}
		mu.Unlock()
		if lim.limiter.Allow() {
			c.Next()
		} else {
			log.Debug("🚩 RateLimiter", log.String("from", c.Request.RemoteAddr))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"message": "Limit exceed",
			})
		}
	}
}
