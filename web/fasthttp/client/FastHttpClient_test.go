package client

import (
	appPkgWeb "axgit.vixiv.ru/snake/arcella-lib/web"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFastHttpClientMethodAndBody(t *testing.T) {
	var gotMethod, gotBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	fh := NewFastHttpClient(&ConfigFastHttpClient{
		RequestTimeout:  time.Second,
		ReadTimeout:     time.Second,
		WriteTimeout:    time.Second,
		IdleConnTimeout: time.Second,
		DNSTimeout:      time.Second,
	})

	resp, err := fh.DORequest(&appPkgWeb.RequestDTS{RequestUrl: ts.URL, RequestType: "GET"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("method = %s, want GET", gotMethod)
	}
	if gotBody != "" {
		t.Fatalf("nil body must be sent empty, got %q", gotBody)
	}

	resp, err = fh.DORequest(&appPkgWeb.RequestDTS{RequestUrl: ts.URL})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet {
		t.Fatalf("empty RequestType must default to GET, got %s", gotMethod)
	}
	if gotBody != "" {
		t.Fatalf("nil body must be sent empty, got %q", gotBody)
	}

	resp, err = fh.DORequest(&appPkgWeb.RequestDTS{
		RequestUrl:  ts.URL,
		RequestType: "POST",
		Body:        map[string]any{"a": 1},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost {
		t.Fatalf("method = %s, want POST", gotMethod)
	}
	if gotBody != `{"a":1}` {
		t.Fatalf("body = %q, want %q", gotBody, `{"a":1}`)
	}
}
