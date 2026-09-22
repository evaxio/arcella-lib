package gin

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	ginFramework "github.com/gin-gonic/gin"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

func TestGinWebStart(t *testing.T) {
	port := freePort(t)
	gw := NewGinWeb(&ConfigGinWeb{
		Port:         port,
		StartMode:    "release",
		ReadTimeOut:  time.Second,
		WriteTimeOut: time.Second,
	})
	gw.GetEngine().GET("/ping", func(c *ginFramework.Context) {
		c.String(http.StatusOK, "pong")
	})

	if err := gw.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := gw.Start(context.Background()); err != nil {
		t.Fatalf("second Start: %v", err)
	}
	defer func() { _ = gw.Stop() }()

	url := fmt.Sprintf("http://127.0.0.1:%d/ping", port)
	var lastErr error
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK && string(b) == "pong" {
				return
			}
			lastErr = fmt.Errorf("status = %d, body = %q", resp.StatusCode, b)
		} else {
			lastErr = err
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server not reachable: %v", lastErr)
}
