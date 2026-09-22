package types

import (
	"bytes"
	"testing"
	"time"
)

func TestNullFloat64Scan(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		wantErr bool
		want    float64
		valid   bool
	}{
		{name: "null", value: nil, want: 0, valid: false},
		{name: "value", value: 3.25, want: 3.25, valid: true},
		{name: "negative", value: -1.5, want: -1.5, valid: true},
		{name: "int64", value: int64(42), want: 42, valid: true},
		{name: "bytes", value: []byte("3.25"), want: 3.25, valid: true},
		{name: "bytes invalid", value: []byte("abc"), wantErr: true},
		{name: "wrong type", value: "3.25", wantErr: true},
		{name: "wrong type int", value: 1, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nf NullFloat64
			err := nf.Scan(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Scan(%v) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
			if nf.Valid != tt.valid {
				t.Errorf("Valid = %v, want %v", nf.Valid, tt.valid)
			}
			if nf.value != tt.want {
				t.Errorf("value = %v, want %v", nf.value, tt.want)
			}
		})
	}
}

func TestNullFloat64Value(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		var nf NullFloat64
		v, err := nf.Value()
		if err != nil {
			t.Fatalf("Value() error = %v", err)
		}
		if v != nil {
			t.Errorf("Value() = %v, want nil", v)
		}
	})
	t.Run("valid round-trip", func(t *testing.T) {
		nf := NullFloat64{value: 2.5, Valid: true}
		v, err := nf.Value()
		if err != nil {
			t.Fatalf("Value() error = %v", err)
		}
		if v != 2.5 {
			t.Errorf("Value() = %v, want 2.5", v)
		}
		var back NullFloat64
		if err := back.Scan(v); err != nil {
			t.Fatalf("Scan(%v) error = %v", v, err)
		}
		if !back.Valid || back.value != 2.5 {
			t.Errorf("round-trip = %+v, want valid 2.5", back)
		}
	})
}

func TestNullStringScan(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		wantErr bool
		want    string
		valid   bool
	}{
		{name: "null", value: nil, want: "", valid: false},
		{name: "value", value: "hello", want: "hello", valid: true},
		{name: "empty string", value: "", want: "", valid: true},
		{name: "bytes", value: []byte("hello"), want: "hello", valid: true},
		{name: "wrong type", value: 42, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ns NullString
			err := ns.Scan(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Scan(%v) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
			if ns.Valid != tt.valid {
				t.Errorf("Valid = %v, want %v", ns.Valid, tt.valid)
			}
			if ns.Val != tt.want {
				t.Errorf("Val = %q, want %q", ns.Val, tt.want)
			}
		})
	}
}

func TestNullStringValue(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		var ns NullString
		v, err := ns.Value()
		if err != nil {
			t.Fatalf("Value() error = %v", err)
		}
		if v != nil {
			t.Errorf("Value() = %v, want nil", v)
		}
		if got := ns.ValidValue(); got != "" {
			t.Errorf("ValidValue() = %q, want empty", got)
		}
	})
	t.Run("valid round-trip", func(t *testing.T) {
		ns := NullString{Val: "abc", Valid: true}
		v, err := ns.Value()
		if err != nil {
			t.Fatalf("Value() error = %v", err)
		}
		if v != "abc" {
			t.Errorf("Value() = %v, want abc", v)
		}
		var back NullString
		if err := back.Scan(v); err != nil {
			t.Fatalf("Scan(%v) error = %v", v, err)
		}
		if !back.Valid || back.Val != "abc" {
			t.Errorf("round-trip = %+v, want valid abc", back)
		}
		if got := back.ValidValue(); got != "abc" {
			t.Errorf("ValidValue() = %q, want abc", got)
		}
	})
}

func TestNullTimeScan(t *testing.T) {
	fixed := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)
	tests := []struct {
		name    string
		value   any
		wantErr bool
		want    time.Time
		valid   bool
	}{
		{name: "null", value: nil, want: time.Time{}, valid: false},
		{name: "value", value: fixed, want: fixed, valid: true},
		{name: "rfc3339", value: "2024-05-06T07:08:09Z", want: fixed, valid: true},
		{name: "datetime", value: "2024-05-06 07:08:09", want: fixed, valid: true},
		{name: "bytes", value: []byte("2024-05-06T07:08:09Z"), want: fixed, valid: true},
		{name: "wrong type", value: "2024-05-06", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var nt NullTime
			err := nt.Scan(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Scan(%v) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
			if nt.Valid != tt.valid {
				t.Errorf("Valid = %v, want %v", nt.Valid, tt.valid)
			}
			if !nt.Time.Equal(tt.want) {
				t.Errorf("Time = %v, want %v", nt.Time, tt.want)
			}
		})
	}
}

func TestNullTimeValue(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		var nt NullTime
		v, err := nt.Value()
		if err != nil {
			t.Fatalf("Value() error = %v", err)
		}
		if v != nil {
			t.Errorf("Value() = %v, want nil", v)
		}
	})
	t.Run("valid round-trip", func(t *testing.T) {
		fixed := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)
		nt := NullTime{Time: fixed, Valid: true}
		v, err := nt.Value()
		if err != nil {
			t.Fatalf("Value() error = %v", err)
		}
		if tv, ok := v.(time.Time); !ok || !tv.Equal(fixed) {
			t.Errorf("Value() = %v, want %v", v, fixed)
		}
		var back NullTime
		if err := back.Scan(v); err != nil {
			t.Fatalf("Scan(%v) error = %v", v, err)
		}
		if !back.Valid || !back.Time.Equal(fixed) {
			t.Errorf("round-trip = %+v, want valid %v", back, fixed)
		}
	})
}

func TestBitBoolScan(t *testing.T) {
	tests := []struct {
		name    string
		src     any
		wantErr bool
		want    BitBool
	}{
		{name: "null", src: nil, want: false},
		{name: "true", src: []byte{1}, want: true},
		{name: "false", src: []byte{0}, want: false},
		{name: "non-zero", src: []byte{7}, want: false},
		{name: "empty", src: []byte{}, wantErr: true},
		{name: "wrong type", src: "1", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b BitBool
			err := b.Scan(tt.src)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Scan(%v) error = %v, wantErr %v", tt.src, err, tt.wantErr)
			}
			if b != tt.want {
				t.Errorf("BitBool = %v, want %v", b, tt.want)
			}
		})
	}
}

func TestBitBoolValue(t *testing.T) {
	t.Run("true", func(t *testing.T) {
		v, err := BitBool(true).Value()
		if err != nil {
			t.Fatalf("Value() error = %v", err)
		}
		if !bytes.Equal(v.([]byte), []byte{1}) {
			t.Errorf("Value() = %v, want [1]", v)
		}
	})
	t.Run("false", func(t *testing.T) {
		v, err := BitBool(false).Value()
		if err != nil {
			t.Fatalf("Value() error = %v", err)
		}
		if !bytes.Equal(v.([]byte), []byte{0}) {
			t.Errorf("Value() = %v, want [0]", v)
		}
	})
	t.Run("round-trip", func(t *testing.T) {
		for _, want := range []BitBool{true, false} {
			v, err := want.Value()
			if err != nil {
				t.Fatalf("Value() error = %v", err)
			}
			var b BitBool
			if err := b.Scan(v); err != nil {
				t.Fatalf("Scan(%v) error = %v", v, err)
			}
			if b != want {
				t.Errorf("round-trip = %v, want %v", b, want)
			}
		}
	})
}
