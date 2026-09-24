package basic

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/evaxio/arcella-lib/crypt/jasypt"
)

const (
	envJasyptPassword   = "JASYPT_PASSWORD"
	envJasyptIterations = "JASYPT_ITERATIONS"

	defaultJasyptIterations = 1000
)

type CryptDecoder struct {
	password   string
	iterations int
}

func NewDecoder() *CryptDecoder {
	iterations := defaultJasyptIterations
	if v := os.Getenv(envJasyptIterations); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			iterations = n
		}
	}
	password := os.Getenv(envJasyptPassword)
	if password == "" {
		slog.Warn("JASYPT_PASSWORD is not set!!!!")
	}
	return &CryptDecoder{
		password:   password,
		iterations: iterations,
	}
}

func (cd *CryptDecoder) Decode(plainText string) (string, error) {
	return jasypt.Decrypt(cd.password, cd.iterations, plainText)
}
