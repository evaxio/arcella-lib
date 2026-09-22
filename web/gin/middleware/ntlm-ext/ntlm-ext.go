package ntlm

// https://github.com/vadimi/go-http-ntlm/blob/master/negotiator.go
// https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-nlmp/b34032e5-3aae-4bc6-84c3-c6d80eadf7f2

import (
	pkgCM "axgit.vixiv.ru/snake/arcella-lib/web/gin/middleware/ntlm-ext/cookies"
	"encoding/base64"
	"encoding/binary"
	"github.com/gin-gonic/gin"
	log "log/slog"
	"strings"
	"unicode"
)

const (
	Authorization string = "Authorization"
	NTLM          string = "NTLM"
	Step2NTLM            = "NTLM TlRMTVNTUAACAAAAAAAAACgAAAABgggAAAIAAgAAAAAAAAAAAAAAAA=="
	Unauthorized  int    = 401
)

var (
	get16     = binary.LittleEndian.Uint16
	encBase64 = base64.StdEncoding.EncodeToString
	decBase64 = base64.StdEncoding.DecodeString
)

// UserVerifier checks that the user is allowed (e.g. exists in LDAP and the account is enabled).
// The middleware is fail-closed: without a verifier every NTLM request is rejected.
type UserVerifier func(username, domain string) error

type Option func(*middlewareOptions)

type middlewareOptions struct {
	verifier UserVerifier
}

func WithUserVerifier(v UserVerifier) Option {
	return func(o *middlewareOptions) {
		o.verifier = v
	}
}

func NTLMMiddleware(path string, cm pkgCM.ICookieManager, opts ...Option) gin.HandlerFunc {
	o := &middlewareOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return func(c *gin.Context) {
		reqPath := c.Request.URL.Path
		if reqPath == path || strings.HasPrefix(reqPath, path+"/") {
			if !cm.IsAuthorized(c) {
				c.Writer.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type, Content-Range, Content-Disposition, Content-Description,Origin, X-Requested-With, sessionId")
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
				if authHeader := c.GetHeader(Authorization); authHeader != "" { // step 2
					stringArr := strings.Split(authHeader, " ")
					if len(stringArr) != 2 || stringArr[0] != NTLM {
						log.Error("Incorrect auth header format")
						sendUnauthorized(c, NTLM)
						return
					}
					byteData, err := decBase64(stringArr[1])
					if err != nil {
						log.Error("DecodeString", log.String("Message", err.Error()))
						sendUnauthorized(c, NTLM)
						return
					}
					switch {
					case len(byteData) > 8 && byteData[8] == 1: // step 2
						sendUnauthorized(c, Step2NTLM)
					case len(byteData) > 8 && byteData[8] == 3: // step 3
						username, okUser := getNTLMField(byteData, 38)
						domain, okDomain := getNTLMField(byteData, 30)
						if !okUser || !okDomain {
							log.Error("NTLM: cannot parse username/domain from auth message")
							sendUnauthorized(c, NTLM)
							return
						}
						if o.verifier == nil {
							log.Error("NTLM: user verifier is not configured, denying access (fail-closed)")
							sendUnauthorized(c, NTLM)
							return
						}
						if verr := o.verifier(username, domain); verr != nil {
							log.Error("NTLM: user verification failed", log.String("user", username), log.String("error", verr.Error()))
							sendUnauthorized(c, NTLM)
							return
						}
						cm.SetAuthorized(c, username, domain)
					default:
						log.Error("NTLM: unsupported message type")
						sendUnauthorized(c, NTLM)
						return
					}
				} else {
					sendUnauthorized(c, NTLM) // step 1
				}
			}
			c.Next()
		}
	}
}

func removeZeros(value string) string {
	return strings.Map(func(r rune) rune {
		if r == 0 || r > unicode.MaxASCII {
			return -1
		}
		return r
	}, value)
}

func getNTLMField(data []byte, fieldBase int) (string, bool) {
	if fieldBase+8 > len(data) {
		return "", false
	}
	length := int(get16(data[fieldBase:]))
	strStart := int(get16(data[fieldBase+4:]))
	if strStart > len(data) || length > len(data)-strStart {
		return "", false
	}
	return removeZeros(string(data[strStart : strStart+length])), true
}

func sendUnauthorized(c *gin.Context, value string) {
	c.Writer.Header().Set("WWW-Authenticate", value)
	c.Writer.Header().Set("Connection", "keep-alive")
	c.AbortWithStatus(Unauthorized)
}
