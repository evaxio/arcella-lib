package jasypt

import (
	"encoding/base64"
	"testing"
)

const testIterations = 1000

func TestEncryptDecryptRoundTrip(t *testing.T) {
	const password = "secret"
	const plainText = "hello, jasypt"

	enc, err := Encrypt(password, testIterations, plainText)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	dec, err := Decrypt(password, testIterations, enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if dec != plainText {
		t.Errorf("Decrypt = %q, want %q", dec, plainText)
	}
}

func TestDecryptWrongPassword(t *testing.T) {
	enc, err := Encrypt("right", testIterations, "data")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if _, err := Decrypt("wrong", testIterations, enc); err == nil {
		t.Errorf("Decrypt with the wrong password = nil error, want an error")
	}
}

// plaintext ending in bytes 0x01..0x08 must survive (padding must be read
// from the last byte, not stripped with TrimRight)
func TestPlaintextWithTrailingPadBytes(t *testing.T) {
	const password = "secret"
	for _, plainText := range []string{"abc\x05", "x\x01", "y\x08", "z\x01\x02\x03"} {
		enc, err := Encrypt(password, testIterations, plainText)
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", plainText, err)
		}
		dec, err := Decrypt(password, testIterations, enc)
		if err != nil {
			t.Fatalf("Decrypt(%q): %v", plainText, err)
		}
		if dec != plainText {
			t.Errorf("round-trip(%q) = %q", plainText, dec)
		}
	}
}

func TestBoundaryLengths(t *testing.T) {
	const password = "secret"
	for _, n := range []int{0, 7, 8, 9} {
		buf := make([]byte, n)
		for i := range buf {
			buf[i] = 'a' + byte(i%26)
		}
		plainText := string(buf)
		enc, err := Encrypt(password, testIterations, plainText)
		if err != nil {
			t.Fatalf("Encrypt(len=%d): %v", n, err)
		}
		dec, err := Decrypt(password, testIterations, enc)
		if err != nil {
			t.Fatalf("Decrypt(len=%d): %v", n, err)
		}
		if dec != plainText {
			t.Errorf("round-trip(len=%d) = %q, want %q", n, dec, plainText)
		}
	}
}

func TestDecryptGarbage(t *testing.T) {
	const password = "secret"
	if _, err := Decrypt(password, testIterations, "!!!not-base64!!!"); err == nil {
		t.Errorf("Decrypt(garbage base64) = nil error, want an error")
	}
	// valid base64, but the length is not a multiple of 8
	short := base64.StdEncoding.EncodeToString([]byte("12345"))
	if _, err := Decrypt(password, testIterations, short); err == nil {
		t.Errorf("Decrypt(bad length) = nil error, want an error")
	}
	// valid base64, only the salt, no ciphertext
	saltOnly := base64.StdEncoding.EncodeToString(make([]byte, 8))
	if _, err := Decrypt(password, testIterations, saltOnly); err == nil {
		t.Errorf("Decrypt(salt only) = nil error, want an error")
	}
}
