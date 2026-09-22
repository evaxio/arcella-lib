package json

import (
	"reflect"
	"testing"
)

type innerStruct struct {
	A int
	B string
}

type outerStruct struct {
	X      int
	Y      innerStruct
	Z      []string
	Nested *innerStruct
}

func TestEncodeDecodeRoundTrip(t *testing.T) {
	codec := NewCodec()
	in := outerStruct{
		X:      42,
		Y:      innerStruct{A: 7, B: "seven"},
		Z:      []string{"a", "b", "c"},
		Nested: &innerStruct{A: -1, B: "minus"},
	}
	b := codec.EncodeStruct(in)
	if len(b) == 0 {
		t.Fatal("EncodeStruct returned an empty payload")
	}
	var out outerStruct
	if err := codec.DecodeStruct(b, &out); err != nil {
		t.Fatalf("DecodeStruct: %v", err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Errorf("round-trip mismatch: in=%+v out=%+v", in, out)
	}
}

func TestEncodeUnencodable(t *testing.T) {
	codec := NewCodec()
	b := codec.EncodeStruct(make(chan int))
	if len(b) != 0 {
		t.Errorf("EncodeStruct(chan) = %q, want an empty payload", b)
	}
}
