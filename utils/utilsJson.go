package utils

import (
	json "github.com/goccy/go-json"
	log "log/slog"
)

func ToJsonB(a any) []byte {
	if b, err := json.Marshal(a); err == nil {
		return b
	} else {
		log.Error("JSON marshal failed", "error", err)
		return nil
	}
}

func ToJson(a any) string {
	if b, err := json.Marshal(a); err == nil {
		return string(b)
	} else {
		return err.Error()
	}
}

func ToJsonPretty(a any) string {
	if b, err := json.MarshalIndent(a, "", "\t"); err == nil {
		return string(b)
	} else {
		return err.Error()
	}
}
