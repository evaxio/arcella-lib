package trueConfClient

import (
	"github.com/evaxio/arcella-lib/app/v3/config/basic"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTokenTestServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/bridge/api/client/v1/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
	return httptest.NewTLSServer(mux)
}

func TestGetNewToken(t *testing.T) {
	srv := newTokenTestServer(t, http.StatusOK, `{"access_token":"tok123","token_type":"Bearer","expires_at":1893456000}`)
	defer srv.Close()

	tc := NewTcClient(&ConfigTcClient{Server: strings.TrimPrefix(srv.URL, "https://"), User: "u", Password: "p"})
	tc.httpClient = srv.Client()
	err, token := tc.GetNewToken()
	if err != nil {
		t.Fatalf("GetNewToken: %v", err)
	}
	if token != "tok123" {
		t.Errorf("token = %q, want %q", token, "tok123")
	}
}

func TestGetNewTokenNon2xx(t *testing.T) {
	srv := newTokenTestServer(t, http.StatusUnauthorized, `{"error":"invalid_grant"}`)
	defer srv.Close()

	tc := NewTcClient(&ConfigTcClient{Server: strings.TrimPrefix(srv.URL, "https://"), User: "u", Password: "p"})
	tc.httpClient = srv.Client()
	err, token := tc.GetNewToken()
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if token != "" {
		t.Errorf("token = %q, want empty", token)
	}
}

func TestConfigDefaults(t *testing.T) {
	var cfg ConfigTcClient
	if err := basic.NewBasicConfig(&cfg).Load(); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should default to true")
	}
}

func TestSendMsgStopped(t *testing.T) {
	tc := NewTcClient(&ConfigTcClient{Server: "localhost"})
	if err := tc.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := tc.SendMsg("hi", "chat1", "text"); err == nil {
		t.Fatal("expected error when stopped")
	}
}

func TestPushWSMsgQueueFull(t *testing.T) {
	tc := NewTcClient(&ConfigTcClient{Server: "localhost"})
	for i := 0; i < cap(tc.wsMsg); i++ {
		if err := tc.pushWSMsg(WSMsg{Type: 1, Text: []byte("x")}); err != nil {
			t.Fatalf("pushWSMsg[%d]: %v", i, err)
		}
	}
	done := make(chan error, 1)
	go func() {
		done <- tc.pushWSMsg(WSMsg{Type: 1, Text: []byte("x")})
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected queue-full error")
		}
	case <-time.After(wsMsgQueueFullTimeout + 2*time.Second):
		t.Fatal("pushWSMsg blocked forever")
	}
}
