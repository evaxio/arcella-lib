package utils

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNumbersAndComma(t *testing.T) {
	tests := []struct{ in, want string }{
		{"abc1,23def456", "1,23456"},
		{"1.000,500", "1000,500"},
		{"no-digits-here", ""},
		{"", ""},
		{"12345", "12345"},
	}
	for _, tt := range tests {
		if got := NumbersAndComma(tt.in); got != tt.want {
			t.Errorf("NumbersAndComma(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSieveOfEratosthenes(t *testing.T) {
	want := []int{2, 3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97}
	if got := SieveOfEratosthenes(2, 100); !sameInts(got, want) {
		t.Errorf("SieveOfEratosthenes(2, 100) = %v, want %v", got, want)
	}
	if got := SieveOfEratosthenes(0, 100); !sameInts(got, want) {
		t.Errorf("SieveOfEratosthenes(0, 100) = %v, want %v", got, want)
	}
	if got := SieveOfEratosthenes(0, 0); len(got) != 0 {
		t.Errorf("SieveOfEratosthenes(0, 0) = %v, want empty", got)
	}
	if got := SieveOfEratosthenes(0, 1); len(got) != 0 {
		t.Errorf("SieveOfEratosthenes(0, 1) = %v, want empty", got)
	}
	if got := SieveOfEratosthenes(2, 2); len(got) != 0 {
		t.Errorf("SieveOfEratosthenes(2, 2) = %v, want empty", got)
	}
	if got := SieveOfEratosthenes(2, 3); !sameInts(got, []int{2}) {
		t.Errorf("SieveOfEratosthenes(2, 3) = %v, want [2]", got)
	}
	if got := SieveOfEratosthenes(90, 100); !sameInts(got, []int{97}) {
		t.Errorf("SieveOfEratosthenes(90, 100) = %v, want [97]", got)
	}
}

func TestSleepRNDGuard(t *testing.T) {
	// must not panic on non-positive limits
	SleepRND(0)
	SleepRND(-5)
	SleepRNDMin(-1, 0)
	SleepRNDMin(3, -2)
	SleepRNDMilliSec(0, 0)
	SleepRNDMilliSec(-4, -1)
}

func TestStreamToString(t *testing.T) {
	in := "hello stream" + strings.Repeat(" world", 1000)
	if got := StreamToString(strings.NewReader(in)); got != in {
		t.Errorf("StreamToString len = %d, want %d", len(got), len(in))
	}
	if got := StreamToString(strings.NewReader("")); got != "" {
		t.Errorf("StreamToString(empty) = %q, want empty", got)
	}
}

type fakeReadCloser struct {
	reader io.Reader
	closed bool
}

func (f *fakeReadCloser) Read(p []byte) (int, error) {
	return f.reader.Read(p)
}
func (f *fakeReadCloser) Close() error {
	f.closed = true
	return nil
}

func TestReadCloserToString(t *testing.T) {
	rc := &fakeReadCloser{reader: strings.NewReader("the content")}
	got := ReadCloserToString(rc)
	if got != "the content" {
		t.Errorf("ReadCloserToString = %q, want %q", got, "the content")
	}
	if !rc.closed {
		t.Errorf("ReadCloserToString did not close the reader")
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	from := filepath.Join(dir, "from.txt")
	to := filepath.Join(dir, "to.txt")
	content := []byte("file content 123")
	if err := os.WriteFile(from, content, 0o644); err != nil {
		t.Fatalf("WriteFile(from): %v", err)
	}
	if err := CopyFile(from, to); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}
	got, err := os.ReadFile(to)
	if err != nil {
		t.Fatalf("ReadFile(to): %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("copied content = %q, want %q", got, content)
	}
	if err := CopyFile(filepath.Join(dir, "missing.txt"), to); err == nil {
		t.Errorf("CopyFile(missing source) = nil error, want error")
	}
}

func TestFileExists(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.txt")
	if FileExists(p) {
		t.Errorf("FileExists(%q) = true for a missing file", p)
	}
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if !FileExists(p) {
		t.Errorf("FileExists(%q) = false for an existing file", p)
	}
}
