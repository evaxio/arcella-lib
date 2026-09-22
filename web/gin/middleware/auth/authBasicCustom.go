package auth

import (
	"context"
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

type BasicAuthCustom struct {
}

type AthenticateUser func(username, password string) bool

func NewBasicAuthCustom() *BasicAuthCustom {
	return &BasicAuthCustom{}
}
func (ba *BasicAuthCustom) GetName() string {
	return "BasicAuthCustom"
}
func (ba *BasicAuthCustom) Start(ctx context.Context) (err error) {
	return err
}
func (ba *BasicAuthCustom) Stop() (err error) {
	return err
}

func (ba *BasicAuthCustom) BasicAuthFunc(authFunc AthenticateUser) gin.HandlerFunc {
	return func(c *gin.Context) {
		//start := time.Now()
		auth := strings.SplitN(c.Request.Header.Get("Authorization"), " ", 2)

		if len(auth) != 2 || auth[0] != "Basic" {
			ba.respondWithUnauthorized(c)
			//log.Debugf("BasicAuthFunc took %s", time.Since(start))
			return
		}
		payload, _ := base64.StdEncoding.DecodeString(auth[1])
		pair := strings.SplitN(string(payload), ":", 2)

		if len(pair) != 2 || !authFunc(pair[0], pair[1]) {
			ba.respondWithUnauthorized(c)
			//log.Debugf("BasicAuthFunc took %s", time.Since(start))
			return
		}

		//log.Debugf("BasicAuthFunc took %s", time.Since(start))
		c.Next()

	}

}

/* Example
func (ba *BasicAuthCustom) authenticateUser(username, password string) bool {
	return true
} */

func (ba *BasicAuthCustom) respondWithUnauthorized(c *gin.Context) {
	c.Header("WWW-Authenticate", "Basic realm=\"Authorization Required\"")
	c.JSON(http.StatusUnauthorized, gin.H{"Message": "Unauthorized"})
	c.Abort()
}
