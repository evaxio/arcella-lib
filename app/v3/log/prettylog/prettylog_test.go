package prettylog

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestHandleFormatsRecord(t *testing.T) {
	var out bytes.Buffer
	h := New(&out, nil)
	when := time.Date(2026, 1, 2, 3, 4, 5, 123456000, time.UTC)
	rec := slog.NewRecord(when, slog.LevelInfo, "hello world", 0)
	rec.AddAttrs(slog.String("k", "v"), slog.Int("n", 42))
	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	s := out.String()
	for _, want := range []string{"INF ", "hello world", "k=v", "n=42", "26.01.02 03:04:05.1234", " ["} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q:\n%s", want, s)
		}
	}
	if !strings.HasSuffix(s, "\n") {
		t.Error("record must end with newline")
	}
}

func TestHandleLevelFilter(t *testing.T) {
	var out bytes.Buffer
	h := New(&out, &Options{Level: slog.LevelError})
	logger := slog.New(h)
	logger.Info("filtered-out")
	logger.Error("kept")
	s := out.String()
	if strings.Contains(s, "filtered-out") {
		t.Error("Info record passed the Error level filter")
	}
	if !strings.Contains(s, "kept") {
		t.Error("Error record missing")
	}
}

func TestHandleCustomLevelNoPanic(t *testing.T) {
	var out bytes.Buffer
	h := New(&out, nil)
	rec := slog.NewRecord(time.Now(), slog.Level(99), "custom-level", 0)
	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if !strings.Contains(out.String(), "99") {
		t.Errorf("custom level not rendered:\n%s", out.String())
	}
}

func TestMessageTruncationRuneSafe(t *testing.T) {
	var out bytes.Buffer
	h := New(&out, nil)
	msg := strings.Repeat("аб", 600) // 1200 bytes > maxStrSize
	rec := slog.NewRecord(time.Now(), slog.LevelInfo, msg, 0)
	if err := h.Handle(context.Background(), rec); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	s := out.String()
	if !strings.Contains(s, "...") {
		t.Error("truncated message must end with ...")
	}
	if !utf8.Valid(out.Bytes()) {
		t.Error("truncation split a multi-byte rune")
	}
}

func TestGroupsAndAttrs(t *testing.T) {
	var out bytes.Buffer
	h := New(&out, nil)
	h2 := h.WithGroup("grp").WithAttrs([]slog.Attr{slog.String("a", "b")})
	logger := slog.New(h2)
	logger.Info("m")
	s := out.String()
	if !strings.Contains(s, "grp: ") {
		t.Errorf("group missing:\n%s", s)
	}
	if !strings.Contains(s, "[a=b]") {
		t.Errorf("group attrs missing:\n%s", s)
	}
}

func TestBufferPool(t *testing.T) {
	b := newBuffer()
	b.WriteString("hello")
	b.WriteByte('!')
	if string(*b) != "hello!" {
		t.Errorf("buffer content = %q", string(*b))
	}
	b.Free()

	b2 := newBuffer()
	b2.WriteString("world")
	b2.WriteStringIf(true, "?")
	b2.WriteStringIf(false, "x")
	if string(*b2) != "world?" {
		t.Errorf("buffer2 content = %q", string(*b2))
	}
	b2.Free()

	// oversized buffers must not be returned to the pool (no crash expected)
	big := newBuffer()
	*big = append(*big, make([]byte, 32<<10)...)
	big.Free()
}

func TestBufferWriteInterface(t *testing.T) {
	var b buffer
	n, err := b.Write([]byte("abc"))
	if err != nil || n != 3 {
		t.Errorf("Write = %d, %v; want 3, nil", n, err)
	}
}
