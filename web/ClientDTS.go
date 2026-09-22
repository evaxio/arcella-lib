package web

import (
	"encoding/json"
)

type RequestDTS struct {
	RequestUrl  string            `json:"url"`
	RequestType string            `json:"type"` // default GET
	Body        interface{}       `json:"body,omitempty"`
	Auth        AuthData          `json:"auth,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Preset      string            `json:"preset,omitempty"`
}

const C_AuthBasicType = "basic"

type AuthData struct {
	Type string `json:"type"`
	Host string `json:"host"`
	User string `json:"-"`
	Pass string `json:"-"`
}

func (ad *AuthData) IsBasic() bool {
	return ad.Type == C_AuthBasicType && ad.User != ""
}

type ResponseDTS struct {
	StatusCode int
	Status     string // e.g. "200 OK"
	Body       string
	Header     map[string][]string
}

type ResponseBytesDTS struct {
	StatusCode int
	Status     string // e.g. "200 OK"
	Body       []byte
	Header     map[string][]string
}

func (resp *ResponseDTS) String() string {
	if b, err := json.Marshal(resp); err == nil {
		return string(b)
	} else {
		return ""
	}
}
