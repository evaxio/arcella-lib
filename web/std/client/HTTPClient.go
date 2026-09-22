package client

import (
	appPkgWeb "axgit.vixiv.ru/snake/arcella-lib/web"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io"
	log "log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type HTTPClient struct {
	config                  *ConfigHTTPClient
	mu                      sync.RWMutex
	hClient                 *http.Client
	header                  map[string]string
	clientRoundTripper      http.RoundTripper
	clientRoundTripperAdded bool
	fixedCipherRule         string
}

func NewHTTPClient(config *ConfigHTTPClient) *HTTPClient {
	return &HTTPClient{
		config: config,
		header: make(map[string]string, 15),
	}
}

func (h *HTTPClient) GetURL(host, url string) string {
	if host == "" {
		return url
	} else {
		urlPath := strings.TrimSuffix(host, "/") + "/" + url
		// log.Debug("GetURL", log.String("url", urlPath))
		return urlPath
	}
}

func (h *HTTPClient) basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (h *HTTPClient) SetTransport(tr http.RoundTripper) {
	h.mu.Lock()
	h.clientRoundTripperAdded = true
	h.clientRoundTripper = tr
	if h.hClient != nil {
		h.hClient.CloseIdleConnections()
	}
	h.hClient = nil
	h.mu.Unlock()
}

func (h *HTTPClient) SetFixedCipherRuleWithURL(sUrl string) {
	h.mu.Lock()
	if h.fixedCipherRule != sUrl {
		h.fixedCipherRule = sUrl
		if h.hClient != nil {
			h.hClient.CloseIdleConnections()
		}
		h.hClient = nil
	}
	h.mu.Unlock()
}

func (h *HTTPClient) buildClient() *http.Client {
	hc := &http.Client{Timeout: h.config.RequestTimeout}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: h.config.InsecureSkipVerify,
	}

	if h.fixedCipherRule != "" {
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.MinVersion = tls.VersionTLS12
		if u, err := url.Parse(h.fixedCipherRule); err == nil {
			tlsConfig.ServerName = u.Hostname()
			log.Debug("Server", log.String("name", tlsConfig.ServerName))
		}
		tlsConfig.CipherSuites = []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		}
	}

	if h.clientRoundTripperAdded {
		hc.Transport = h.clientRoundTripper
	} else {
		tr := &http.Transport{
			MaxIdleConns:        h.config.MaxIdleConns,
			MaxIdleConnsPerHost: h.config.MaxIdleConnsPerHost,
			IdleConnTimeout:     h.config.IdleConnTimeout,
			DisableCompression:  h.config.DisableCompression,
			TLSHandshakeTimeout: 10 * time.Second,
		}
		tr.TLSClientConfig = tlsConfig
		hc.Transport = tr
	}
	return hc
}

// GetClient returns the shared *http.Client. It is built once and rebuilt
// only when the fixed cipher rule or the round tripper changes.
func (h *HTTPClient) GetClient() *http.Client {
	h.mu.RLock()
	client := h.hClient
	h.mu.RUnlock()
	if client != nil {
		return client
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hClient == nil {
		h.hClient = h.buildClient()
	}
	return h.hClient
}

// Close releases idle connections held by the underlying http.Client.
func (h *HTTPClient) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hClient != nil {
		h.hClient.CloseIdleConnections()
		h.hClient = nil
	}
}

func (h *HTTPClient) getReqBodyReader(request *appPkgWeb.RequestDTS) io.Reader {
	var requestBody []byte
	switch request.Body.(type) {
	case string:
		requestBody = []byte(request.Body.(string))
	default:
		var err error
		if requestBody, err = json.Marshal(request.Body); err != nil {
			log.Error("Marshal", log.String("error", err.Error()))
			return http.NoBody
		}
	}
	if len(requestBody) > 0 {
		// log.Debug("GetReqBodyReader")
		return bytes.NewReader(requestBody)
	} else {
		return http.NoBody
	}

}

// newResponseBodyReader returns the reader for the response body, wrapping it
// with a gzip reader when the server sent a gzip encoded body.
func newResponseBodyReader(resp *http.Response) (io.ReadCloser, error) {
	// Check that the server actually sent compressed data
	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		return gz, nil
	default:
		return resp.Body, nil
	}
}

func (h *HTTPClient) readResponseBody(reader io.Reader) ([]byte, error) {
	if h.config.MaxResponseBodySize > 0 {
		return io.ReadAll(io.LimitReader(reader, h.config.MaxResponseBodySize))
	}
	return io.ReadAll(reader)
}

// newHTTPRequest builds the request from reqData applying the client level
// headers and basic auth.
func (h *HTTPClient) newHTTPRequest(reqData *appPkgWeb.RequestDTS) (*http.Request, error) {
	requestType := reqData.RequestType
	if requestType == "" {
		requestType = "GET"
	}
	req, err := http.NewRequest(requestType, reqData.RequestUrl, h.getReqBodyReader(reqData))
	if err != nil {
		return nil, err
	}
	if reqData.Headers != nil && len(reqData.Headers) > 0 {
		// log.Debug("add headers", log.Int("count", len(reqData.Headers)))
		for k, v := range reqData.Headers {
			req.Header.Add(k, v)
		}
	}

	h.mu.RLock()
	headers := make(map[string]string, len(h.header))
	for k, v := range h.header {
		headers[k] = v
	}
	h.mu.RUnlock()

	if len(headers) > 0 {
		debugEnabled := log.Default().Enabled(req.Context(), log.LevelDebug)
		for k, v := range headers {
			req.Header.Add(k, v)
			if debugEnabled {
				if strings.EqualFold(k, "Authorization") || strings.EqualFold(k, "Proxy-Authorization") {
					log.Debug("Header: ", log.String("name", k), log.String("value", "***"))
				} else {
					log.Debug("Header: ", log.String("name", k), log.String("value", v))
				}
			}
		}
	} else {
		log.Debug("🚩 no headers")
	}

	if reqData.Auth.IsBasic() {
		log.Debug("set basic auth")
		req.Header.Add("Authorization", "Basic "+h.basicAuth(reqData.Auth.User, reqData.Auth.Pass))
	}
	return req, nil
}

func copyResponseHeaders(h http.Header) map[string][]string {
	if len(h) == 0 {
		return nil
	}
	headers := make(map[string][]string, len(h))
	for k, v := range h {
		headers[k] = v
	}
	return headers
}

// DORequest TODO: DORequest with DORequestBytes - wtf ?
func (h *HTTPClient) DORequest(reqData *appPkgWeb.RequestDTS) (appPkgWeb.ResponseDTS, error) {
	response := appPkgWeb.ResponseDTS{}
	client := h.GetClient()
	req, err := h.newHTTPRequest(reqData)
	if err != nil {
		return response, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return response, err
	}
	response.StatusCode = resp.StatusCode
	response.Status = resp.Status
	if resp.Body != nil {
		reader, err := newResponseBodyReader(resp)
		if err != nil {
			resp.Body.Close()
			return response, err
		}
		defer reader.Close()
		resBody, err := h.readResponseBody(reader)
		if err != nil {
			return response, err
		}
		response.Body = string(resBody)
	}
	response.Header = copyResponseHeaders(resp.Header)
	return response, nil
}

// DORequestBytes TODO: DORequest with DORequestBytes - wtf ?
func (h *HTTPClient) DORequestBytes(reqData *appPkgWeb.RequestDTS) (appPkgWeb.ResponseBytesDTS, error) {
	response := appPkgWeb.ResponseBytesDTS{}
	client := h.GetClient()
	req, err := h.newHTTPRequest(reqData)
	if err != nil {
		return response, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return response, err
	}
	response.StatusCode = resp.StatusCode
	response.Status = resp.Status
	if resp.Body != nil {
		reader, err := newResponseBodyReader(resp)
		if err != nil {
			resp.Body.Close()
			return response, err
		}
		defer reader.Close()
		resBody, err := h.readResponseBody(reader)
		if err != nil {
			return response, err
		}
		response.Body = resBody
	}
	response.Header = copyResponseHeaders(resp.Header)
	return response, nil
}

func (h *HTTPClient) AddBasicAuth(u, p string) {
	log.Debug("AddBasicAuth")
	h.mu.Lock()
	h.header["Authorization"] = "Basic " + h.basicAuth(u, p)
	h.mu.Unlock()
}

func (h *HTTPClient) AddHeader(k, v string) {
	h.mu.Lock()
	h.header[k] = v
	h.mu.Unlock()
}
