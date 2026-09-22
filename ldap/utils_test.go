package ldap

import (
	"strconv"
	"testing"
)

func TestIsGUIDValidSize(t *testing.T) {
	tests := []struct {
		guid string
		want bool
	}{
		{"12345678123456781234567812345678", true},
		{"12345678-1234-5678-1234-567812345678", true},
		{"{12345678-1234-5678-1234-567812345678}", true},
		{"1234567812345678123456781234567", false},
		{"123456781234567812345678123456789", false},
		{"", false},
		{"12345678-1234-5678-1234-56781234567", false},
	}
	for _, tt := range tests {
		if got := IsGUIDValidSize(tt.guid); got != tt.want {
			t.Errorf("IsGUIDValidSize(%q) = %v, want %v", tt.guid, got, tt.want)
		}
	}
}

// mixed-endian LDAP objectGUID bytes for 12345678-1234-5678-1234-567812345678
var testGUIDBytes = []byte{
	0x78, 0x56, 0x34, 0x12,
	0x34, 0x12,
	0x78, 0x56,
	0x12, 0x34, 0x56, 0x78, 0x12, 0x34, 0x56, 0x78,
}

func TestConvertToDashedString(t *testing.T) {
	const want = "12345678-1234-5678-1234-567812345678"
	if got := ConvertToDashedString(testGUIDBytes); got != want {
		t.Errorf("ConvertToDashedString = %q, want %q", got, want)
	}
	if got := ConvertToDashedString(nil); got != "" {
		t.Errorf("ConvertToDashedString(nil) = %q, want empty", got)
	}
	if got := ConvertToDashedString([]byte{0x12, 0x34}); got != "" {
		t.Errorf("ConvertToDashedString(short) = %q, want empty", got)
	}
}

func TestGuidToOctetString(t *testing.T) {
	const guid = "12345678-1234-5678-1234-567812345678"
	const want = `\78\56\34\12\34\12\78\56\12\34\56\78\12\34\56\78`
	if got := GuidToOctetString(guid); got != want {
		t.Errorf("GuidToOctetString = %q, want %q", got, want)
	}
	if got := GuidToOctetString("short"); got != "" {
		t.Errorf("GuidToOctetString(short) = %q, want empty", got)
	}
	if got := GuidToOctetString(""); got != "" {
		t.Errorf("GuidToOctetString(empty) = %q, want empty", got)
	}
}

// TestGuidRoundTrip converts LDAP bytes to the dashed form, then to the octet
// string, and back to bytes again, expecting the original bytes.
func TestGuidRoundTrip(t *testing.T) {
	dashed := ConvertToDashedString(testGUIDBytes)
	octets := GuidToOctetString(dashed)

	// the octet string groups are already in the raw (mixed-endian) byte order
	back := make([]byte, 16)
	for i := 0; i < 16; i++ {
		v, err := strconv.ParseUint(octets[3*i+1:3*i+3], 16, 8)
		if err != nil {
			t.Fatalf("parsing octet group %d: %v", i, err)
		}
		back[i] = byte(v)
	}
	for i := range back {
		if back[i] != testGUIDBytes[i] {
			t.Errorf("round-trip byte %d = 0x%02X, want 0x%02X", i, back[i], testGUIDBytes[i])
		}
	}
}

func TestDecodeSID(t *testing.T) {
	// S-1-5-18 in the byte layout decodeSID expects:
	// revision, subauthority count, authority (big-endian), subauthorities (little-endian)
	sid18 := []byte{
		0x01, 0x01,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x05,
		0x12, 0x00, 0x00, 0x00,
	}
	if got := decodeSID(sid18); got != "S-1-5-18" {
		t.Errorf("decodeSID(S-1-5-18) = %q, want %q", got, "S-1-5-18")
	}
	if got := decodeSID(nil); got != "" {
		t.Errorf("decodeSID(nil) = %q, want empty", got)
	}
	if got := decodeSID([]byte{0x01}); got != "" {
		t.Errorf("decodeSID(short) = %q, want empty", got)
	}
	// subauthority count claims more bytes than are present: must not panic
	if got := decodeSID([]byte{0x01, 0xFF, 0, 0, 0, 0, 0, 0}); got != "" {
		t.Errorf("decodeSID(truncated) = %q, want empty", got)
	}
}

func TestGetDateFromString(t *testing.T) {
	// the layout has no zone token, so the time is taken as-is in the
	// Asia/Novosibirsk location and the wall clock is preserved
	if got := getDateFromString("20240304121804Z"); got != "2024-03-04 12:18:04" {
		t.Errorf("getDateFromString = %q, want %q", got, "2024-03-04 12:18:04")
	}
	if got := getDateFromString("garbage"); got != "garbage" {
		t.Errorf("getDateFromString(garbage) = %q, want the input back", got)
	}
}
