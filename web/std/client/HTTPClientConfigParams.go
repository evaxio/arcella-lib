package client

import "time"

type ConfigHTTPClient struct {
	MaxIdleConns        int           `yaml:"maxIdleConns"         env:"HTTP_CLIENT_MAX_IDLE_CONNS"      default:"10" `
	MaxIdleConnsPerHost int           `yaml:"maxIdleConnsPerHost"  env:"HTTP_CLIENT_MAX_IDLE_CONNS_PER_HOST" default:"10"`
	IdleConnTimeout     time.Duration `yaml:"idleConnTimeout"      env:"HTTP_CLIENT_IDLE_CONN_TIMEOUT"   default:"30s"`
	RequestTimeout      time.Duration `yaml:"requestTimeout"       env:"HTTP_CLIENT_REQUEST_TIMEOUT"     default:"60s"`
	DisableCompression  bool          `yaml:"disableCompression"   env:"HTTP_CLIENT_DISABLE_COMPRESSION" default:"false"`
	MaxResponseBodySize int64         `yaml:"maxResponseBodySize"  env:"HTTP_CLIENT_MAX_RESPONSE_BODY_SIZE" default:"67108864"`
	InsecureSkipVerify  bool          `yaml:"InsecureSkipVerify"   env:"HTTP_CLIENT_SKIP_VERIFY"`
}
