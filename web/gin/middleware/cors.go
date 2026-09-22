package middleware

import (
	"net/http"

	log "log/slog"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware(port string) gin.HandlerFunc {
	log.Debug("CORS Middleware")
	allowedOrigin := "http://localhost:" + port
	return func(c *gin.Context) {
		if c.GetHeader("Origin") == allowedOrigin {
			c.Writer.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			c.Writer.Header().Set("Access-Control-Max-Age", "86400")
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "X-Requested-With, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, token, Accept, Origin, Cache-Control, X-Requested-With, Set-Cookie")
			//c.Writer.Header().Set("Access-Control-Allow-Methods", "OPTIONS, POST, PATCH, GET, PUT, DELETE")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, GET, PUT, DELETE")
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(http.StatusOK)
				return
			}
		}
		c.Next()
	}
}
