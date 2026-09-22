package log

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type countingBody struct {
	r io.Reader
}

func (cb *countingBody) Read(p []byte) (int, error) {
	return cb.r.Read(p)
}

func (cb *countingBody) Close() error {
	return nil
}

func TestLoggingMiddlewareDoesNotConsumeBody(t *testing.T) {
	want := strings.Repeat("payload-", 1024)
	var got string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		w.Write([]byte("ok"))
	}))
	defer ts.Close()

	client := &http.Client{Transport: LoggingMiddleware(http.DefaultTransport)}
	body := &countingBody{r: strings.NewReader(want)}
	req, err := http.NewRequest(http.MethodPost, ts.URL, body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if got != want {
		t.Fatalf("server received %d bytes, want %d (body must not be truncated)", len(got), len(want))
	}
	if _, ok := req.Body.(*countingBody); !ok {
		t.Fatal("middleware replaced the request body, the body must not be read")
	}
}
