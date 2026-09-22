package uuid

import (
	"testing"
)

func TestUUID7Format(t *testing.T) {
	u := NewUUID7()
	s := u.String()
	if len(s) != 36 {
		t.Fatalf("String len = %d, want 36: %q", len(s), s)
	}
	if s[14] != '7' {
		t.Errorf("version nibble = %c, want 7: %q", s[14], s)
	}
	variant := s[19]
	if variant != '8' && variant != '9' && variant != 'a' && variant != 'b' {
		t.Errorf("variant = %c, want one of 89ab: %q", variant, s)
	}
}

func TestUUID7Unique(t *testing.T) {
	seen := make(map[string]bool, 1000)
	for i := 0; i < 1000; i++ {
		u := NewUUID7()
		if seen[u.String()] {
			t.Fatal("duplicate uuid generated")
		}
		seen[u.String()] = true
	}
}

func TestXorEncryptDecrypt(t *testing.T) {
	u := NewUUID7()
	enc := u.XorEncrypt()
	dec, err := u.XorDecrypt(enc)
	if err != nil {
		t.Fatalf("XorDecrypt: %v", err)
	}
	if dec != u.String() {
		t.Errorf("decrypted = %q, want %q", dec, u.String())
	}
	if _, err := u.XorDecrypt("not-base64!!"); err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestBase64AndBytes(t *testing.T) {
	u := NewUUID7()
	if len(u.Bytes()) != bytesLen {
		t.Fatalf("Bytes len = %d, want %d", len(u.Bytes()), bytesLen)
	}
	if u.Base64() == "" {
		t.Error("Base64 is empty")
	}
}
