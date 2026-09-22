package client

import (
	appPkgWeb "axgit.vixiv.ru/snake/arcella-lib/web"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestConfig() *ConfigHTTPClient {
	return &ConfigHTTPClient{
		MaxIdleConns:       10,
		IdleConnTimeout:    time.Second,
		RequestTimeout:     time.Second,
		DisableCompression: true,
	}
}

func TestGetClientReused(t *testing.T) {
	c := NewHTTPClient(newTestConfig())
	c1 := c.GetClient()
	c2 := c.GetClient()
	if c1 != c2 {
		t.Fatal("GetClient must return the same *http.Client")
	}
	if c1.Transport == nil {
		t.Fatal("Transport must not be nil")
	}
	if c1.Transport != c2.Transport {
		t.Fatal("Transport must be reused between GetClient calls")
	}

	c.SetFixedCipherRuleWithURL("https://example.com/")
	c3 := c.GetClient()
	if c3 == c1 {
		t.Fatal("client must be rebuilt after the cipher rule changes")
	}
	c.SetFixedCipherRuleWithURL("https://example.com/")
	c4 := c.GetClient()
	if c4 != c3 {
		t.Fatal("client must not be rebuilt when the cipher rule is unchanged")
	}
}

func TestDORequestGetHeaders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Method", r.Method)
		w.Header().Set("X-Req-Header", r.Header.Get("X-Req-Header"))
		w.Header().Set("X-Client-Header", r.Header.Get("X-Client-Header"))
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	c := NewHTTPClient(newTestConfig())
	c.AddHeader("X-Client-Header", "from-client")
	resp, err := c.DORequest(&appPkgWeb.RequestDTS{
		RequestUrl: ts.URL,
		Headers:    map[string]string{"X-Req-Header": "from-req"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header["X-Method"][0]; got != http.MethodGet {
		t.Fatalf("method = %s, want GET", got)
	}
	if got := resp.Header["X-Req-Header"][0]; got != "from-req" {
		t.Fatalf("request header = %q, want %q", got, "from-req")
	}
	if got := resp.Header["X-Client-Header"][0]; got != "from-client" {
		t.Fatalf("client header = %q, want %q", got, "from-client")
	}
}

func TestDORequestPostJSONBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Method", r.Method)
		b, _ := io.ReadAll(r.Body)
		w.Write(b)
	}))
	defer ts.Close()

	c := NewHTTPClient(newTestConfig())
	resp, err := c.DORequest(&appPkgWeb.RequestDTS{
		RequestUrl:  ts.URL,
		RequestType: "POST",
		Body:        map[string]any{"key": "value"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := resp.Header["X-Method"][0]; got != http.MethodPost {
		t.Fatalf("method = %s, want POST", got)
	}
	if resp.Body != `{"key":"value"}` {
		t.Fatalf("body = %q, want %q", resp.Body, `{"key":"value"}`)
	}
}

func TestDORequestGzipResponse(t *testing.T) {
	payload := []byte("gzip compressed payload")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		gz.Write(payload)
		gz.Close()
	}))
	defer ts.Close()

	c := NewHTTPClient(newTestConfig())
	resp, err := c.DORequest(&appPkgWeb.RequestDTS{RequestUrl: ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Body != string(payload) {
		t.Fatalf("DORequest body = %q, want %q", resp.Body, payload)
	}

	respB, err := c.DORequestBytes(&appPkgWeb.RequestDTS{RequestUrl: ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(respB.Body, payload) {
		t.Fatalf("DORequestBytes body = %q, want %q", respB.Body, payload)
	}
}

func TestDORequestBasicAuth(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Auth", r.Header.Get("Authorization"))
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	c := NewHTTPClient(newTestConfig())
	resp, err := c.DORequest(&appPkgWeb.RequestDTS{
		RequestUrl: ts.URL,
		Auth:       appPkgWeb.AuthData{Type: appPkgWeb.C_AuthBasicType, User: "u1", Pass: "p1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("u1:p1"))
	if got := resp.Header["X-Auth"][0]; got != want {
		t.Fatalf("authorization = %q, want %q", got, want)
	}
}
