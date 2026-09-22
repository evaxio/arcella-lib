package generator

import (
	"crypto/rand"
	"encoding/hex"
	log "log/slog"
	"math/big"
	mrand "math/rand"
	"time"
)

// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go

const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890!@#$%^&*()-=+"

// randomBytes fills b with random bytes, retrying once. Both public functions
// cannot report errors (API stability), so if reading entropy fails twice the
// buffer is filled from a time-seeded generator instead of panicking.
func randomBytes(b []byte) {
	if _, err := rand.Read(b); err == nil {
		return
	}
	if _, err := rand.Read(b); err == nil {
		return
	}
	log.Error("crypto/rand failed twice, using a time-seeded fallback")
	src := mrand.New(mrand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = byte(src.Intn(256))
	}
}

// randIndex returns a random index in [0, max) without modulo bias.
func randIndex(max int) int {
	if n, err := rand.Int(rand.Reader, big.NewInt(int64(max))); err == nil {
		return int(n.Int64())
	}
	log.Error("crypto/rand failed, using a time-seeded fallback")
	return mrand.New(mrand.NewSource(time.Now().UnixNano())).Intn(max)
}

func RandomBase16String(l int) string {
	buff := make([]byte, (l+1)/2)
	randomBytes(buff)
	str := hex.EncodeToString(buff)
	return str[:l] // strip 1 extra character we get from odd length results
}

func ShortID(length int) string {
	ll := len(chars)
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = chars[randIndex(ll)]
	}
	return string(b)
}
