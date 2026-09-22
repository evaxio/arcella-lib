package json

import (
	json "github.com/goccy/go-json"
	log "log/slog"
)

type Codec struct{}

func NewCodec() *Codec {
	return &Codec{}
}

func (jc *Codec) EncodeStruct(data interface{}) []byte {
	b, err := json.Marshal(data)
	if err != nil {
		log.Error("JSON encode failed", "error", err)
	}
	return b
}

func (jc *Codec) DecodeStruct(data []byte, target interface{}) error {
	return json.Unmarshal(data, target)
}
