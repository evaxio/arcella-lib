package gin

import (
	"log"
	"log/slog"
)

type serverErrorLogWriter struct{}

func (*serverErrorLogWriter) Write(p []byte) (int, error) {
	slog.Error("Gin: ", slog.String("message", string(p)))
	return len(p), nil
}

func NewServerErrorLog() *log.Logger {
	return log.New(&serverErrorLogWriter{}, "", 0)
}
