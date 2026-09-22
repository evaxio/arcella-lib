package pb

import (
	pbenc "github.com/2xxn/go-raw-protobuf"
)

// Codec is reflection-based (go-raw-protobuf) and meant for ad-hoc structs;
// use google.golang.org/protobuf generated types on hot paths.
type Codec struct{}

func NewCodec() *Codec {
	return &Codec{}
}

func (jc *Codec) EncodeStruct(data interface{}) []byte {
	return pbenc.EncodeStruct(data)
}

func (jc *Codec) DecodeStruct(data []byte, target interface{}) error {
	return pbenc.DecodeStruct(data, target)
}
