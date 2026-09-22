package HmacSHA256

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestEncodeKnownVector(t *testing.T) {
	data := "The quick brown fox jumps over the lazy dog"
	secret := "secret-key-123"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(data))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	if got := Encode(data, secret); got != want {
		t.Errorf("Encode = %s, want %s", got, want)
	}
}

func TestEncodeEmptyData(t *testing.T) {
	secret := "k"
	mac := hmac.New(sha256.New, []byte(secret))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if got := Encode("", secret); got != want {
		t.Errorf("Encode(empty) = %s, want %s", got, want)
	}
}
