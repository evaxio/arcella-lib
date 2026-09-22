package aes

import (
	"encoding/base64"
	"testing"
)

func TestCryptDeCryptRoundTrip(t *testing.T) {
	const secret = "my secret key"
	const text = "hello, AES world! 123"

	enc, err := Crypt(text, secret)
	if err != nil {
		t.Fatalf("Crypt: %v", err)
	}
	if enc == text {
		t.Fatal("ciphertext equals the plaintext")
	}
	dec, err := DeCrypt(enc, secret)
	if err != nil {
		t.Fatalf("DeCrypt: %v", err)
	}
	if dec != text {
		t.Errorf("DeCrypt = %q, want %q", dec, text)
	}
}

func TestDeCryptTamperedCiphertext(t *testing.T) {
	const secret = "my secret key"
	const text = "tamper with me"

	enc, err := Crypt(text, secret)
	if err != nil {
		t.Fatalf("Crypt: %v", err)
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("DecodeString: %v", err)
	}
	raw[len(raw)-1] ^= 0xFF // corrupt the last ciphertext byte
	tampered := base64.StdEncoding.EncodeToString(raw)

	if _, err := DeCrypt(tampered, secret); err == nil {
		t.Errorf("DeCrypt(tampered) = nil error, want an authentication error")
	}
}

func TestDeCryptWrongSecret(t *testing.T) {
	enc, err := Crypt("text", "key-one")
	if err != nil {
		t.Fatalf("Crypt: %v", err)
	}
	if _, err := DeCrypt(enc, "key-two"); err == nil {
		t.Errorf("DeCrypt with the wrong secret = nil error, want an error")
	}
}
