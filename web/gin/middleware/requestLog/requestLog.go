package requestLog

import (
	"bytes"
	"github.com/gin-gonic/gin"
	"io"
	log "log/slog"
)

func GetRequestLog() gin.HandlerFunc {
	log.Debug("Request log")
	return func(c *gin.Context) {
		if log.Default().Enabled(c, log.LevelDebug) {
			log.Debug("Request",
				log.String("IP", c.ClientIP()),
				log.String("Method", c.Request.Method),
				log.String("Path", c.Request.URL.Path))
			if c.Request.ContentLength > 0 {
				log.Debug("Request", log.Int("BodySize", int(c.Request.ContentLength)))
			}
			if c.Request.Body != nil {
				if byteBody, err := io.ReadAll(io.LimitReader(c.Request.Body, 64<<10)); err != nil {
					log.Error("Error body reading", log.String("message", err.Error()))
				} else {
					c.Request.Body = io.NopCloser(bytes.NewReader(byteBody))
				}
			}
		}
		c.Next() // Process the request
	}
}
