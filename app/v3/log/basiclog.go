package log

import (
	pkgPretty "axgit.vixiv.ru/snake/arcella-lib/app/v3/log/prettylog"
	"io"
	"log/slog"
	"os"
)

func NewLog() slog.Handler {
	return pkgPretty.New(os.Stdout, &pkgPretty.Options{Level: slog.LevelDebug})
}

func NewLogWithLevel(level slog.Level, w io.Writer) slog.Handler {
	return pkgPretty.New(w, &pkgPretty.Options{Level: level})
}
