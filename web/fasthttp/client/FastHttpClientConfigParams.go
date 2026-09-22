package client

import "time"

type ConfigFastHttpClient struct {
	RequestTimeout      time.Duration `yaml:"requestTimeout"      env:"HTTP_CLIENT_REQUEST_TIMEOUT" default:"60s"`
	ReadTimeout         time.Duration `yaml:"readTimeout"         env:"HTTP_CLIENT_READ_TIMEOUT"    default:"60s" `
	WriteTimeout        time.Duration `yaml:"writeTimeout"        env:"HTTP_CLIENT_WRITE_TIMEOUT"   default:"60s" `
	IdleConnTimeout     time.Duration `yaml:"idleConnTimeout"     env:"HTTP_CLIENT_IDLE_TIMEOUT"    default:"1h"`
	DNSTimeout          time.Duration `yaml:"dnsTimeout"          env:"HTTP_CLIENT_DNS_TIMEOUT"     default:"1h"`
	MaxResponseBodySize int64         `yaml:"maxResponseBodySize" env:"HTTP_CLIENT_MAX_RESPONSE_BODY_SIZE" default:"67108864"`
}
