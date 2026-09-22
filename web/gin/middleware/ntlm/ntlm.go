package ntlm

// https://github.com/vadimi/go-http-ntlm/blob/master/negotiator.go
// https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-nlmp/b34032e5-3aae-4bc6-84c3-c6d80eadf7f2

import (
	"encoding/base64"
	"encoding/binary"
	log "log/slog"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
)

const (
	unameStr  string = "username"
	domainStr string = "domain"
)
const (
	Authorization string = "Authorization"
	NTLM          string = "NTLM"
	Unauthorized  int    = 401
)

const (
	empty                            = 0x0000     // just a mock
	negotiateUnicode                 = 0x0001     // Text strings are in unicode
	negotiateOEM                     = 0x0002     // Text strings are in OEM
	requestTarget                    = 0x0004     // Server return its auth realm
	negotiateSign                    = 0x0010     // Request signature capability
	negotiateSeal                    = 0x0020     // Request confidentiality
	negotiateLMKey                   = 0x0080     // Generate session key
	negotiateNTLM                    = 0x0200     // NTLM authentication
	negotiateLocalCall               = 0x4000     // client/server on same machine
	negotiateAlwaysSign              = 0x8000     // Sign for all security levels
	negotiateExtendedSessionSecurity = 0x80000    // Extended session security
	negotiateVersion                 = 0x02000000 // negotiate version flag
	negotiate128                     = 0x20000000 // 128-bit session key negotiation
	negotiateKeyExch                 = 0x40000000 // Key exchange
	negotiate56                      = 0x80000000 // 56-bit encryption

	negotiateFlags uint32 = negotiateAlwaysSign |
		negotiateExtendedSessionSecurity |
		//	negotiateKeyExch |
		//	negotiate128 |
		//	negotiate56 |
		negotiateNTLM |
		//  requestTarget |
		//  negotiateOEM |
		negotiateUnicode |
		//	negotiateVersion |
		empty
)

var (
	put32     = binary.LittleEndian.PutUint32
	put16     = binary.LittleEndian.PutUint16
	get16     = binary.LittleEndian.Uint16
	encBase64 = base64.StdEncoding.EncodeToString
	decBase64 = base64.StdEncoding.DecodeString
)

// ntlmAuthChallenge is the constant 40-byte NTLM type-2 (challenge) message.
var ntlmAuthChallenge = func() []byte {
	retMsg := make([]byte, 40)
	copy(retMsg, "NTLMSSP\x00") // protocol
	put32(retMsg[8:], 2)
	put32(retMsg[12:], 0)
	put32(retMsg[16:], 40)
	put32(retMsg[20:], negotiateFlags)
	put32(retMsg[24:], 33554944)
	put32(retMsg[28:], 0)
	put32(retMsg[32:], 0)
	put16(retMsg[36:], 0)
	return retMsg
}()

// pathMatches reports whether reqPath is the protected path or a sub-path of it.
func pathMatches(reqPath, path string) bool {
	return reqPath == path || strings.HasPrefix(reqPath, path+"/")
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

// Deprecated: the NTLM type-3 password proof is not validated (username-only verification) and the challenge is constant; do not rely on this middleware as strong authentication.
func NTLMMiddleware(path string, opts ...Option) gin.HandlerFunc {
	o := &middlewareOptions{}
	for _, opt := range opts {
		opt(o)
	}
	return func(c *gin.Context) {
		// log.Debug("Compare ", log.String("urlPath", c.Request.URL.Path), log.String("path", path))
		if pathMatches(c.Request.URL.Path, path) {
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
					sendUnauthorized(c, NTLM+" "+encBase64(ntlmAuthChallenge))
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
					c.Set(unameStr, username)
					c.Set(domainStr, domain)
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

func sendUnauthorized(c *gin.Context, value string) {
	// log.Debug("sendUnauthorized", log.String("value", value))
	c.Writer.Header().Set("WWW-Authenticate", value)
	c.Writer.Header().Set("Connection", "keep-alive")
	c.AbortWithStatus(Unauthorized)
}
