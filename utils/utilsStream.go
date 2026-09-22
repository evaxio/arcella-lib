package utils

import (
	"io"
	"strings"
)

// StreamToByte reads the whole reader; read errors are swallowed, so the
// returned data may be partial.
func StreamToByte(reader io.Reader) []byte {
	decompressed, _ := io.ReadAll(reader)
	return decompressed
}

func StreamToString(reader io.Reader) string {
	var buf strings.Builder
	io.Copy(&buf, reader)
	return buf.String()
}

// ReadCloserToString reads the whole reader and closes it; read errors are
// swallowed, so the returned data may be partial.
func ReadCloserToString(reader io.ReadCloser) string {
	defer reader.Close()
	readed, _ := io.ReadAll(reader)
	return string(readed)
}
