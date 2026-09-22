package client

// Deprecated: not implemented; no transport or request methods.
type HTTPClientGorilla struct {
	config *ConfigHTTPClientHTTPClientGorilla
}

func NewHTTPClient(config *ConfigHTTPClientHTTPClientGorilla) *HTTPClientGorilla {
	return &HTTPClientGorilla{
		config: config,
	}
}
