package cookies

import "github.com/gin-gonic/gin"

type ICookieManager interface {
	IsAuthorized(c *gin.Context) bool
	SetAuthorized(c *gin.Context, user, domain string)
}
