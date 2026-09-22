package basic

import (
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

type AthenticateUser func(username, password string) bool

func BasicAuthFunc(authFunc AthenticateUser) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := strings.SplitN(c.Request.Header.Get("Authorization"), " ", 2)
		if len(auth) != 2 || auth[0] != "Basic" {
			respondWithUnauthorized(c)
			return
		}
		payload, _ := base64.StdEncoding.DecodeString(auth[1])
		pair := strings.SplitN(string(payload), ":", 2)
		if len(pair) != 2 || !authFunc(pair[0], pair[1]) {
			respondWithUnauthorized(c)
			return
		}
		c.Next()
	}
}

func respondWithUnauthorized(c *gin.Context) {
	c.Header("WWW-Authenticate", "Basic realm=\"Authorization Required\"")
	c.JSON(http.StatusUnauthorized, gin.H{"Message": "Unauthorized"})
	c.Abort()
}
