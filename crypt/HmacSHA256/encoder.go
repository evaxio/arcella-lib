package HmacSHA256

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
)

func Encode(data, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// Verify reports whether encoded is a valid base64 HMAC-SHA256 of data under
// secret, using a constant-time comparison.
func Verify(data, secret, encoded string) bool {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return false
	}
	return hmac.Equal(mac.Sum(nil), decoded)
}
