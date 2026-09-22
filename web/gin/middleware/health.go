package middleware

import "github.com/gin-gonic/gin"

// Deprecated: no-op; kept for compatibility.
func HealthPoint() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
