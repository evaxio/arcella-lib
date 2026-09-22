package requestLog

import (
	"io"
	log "log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func postWithBody(t *testing.T, body string) string {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(GetRequestLog())
	router.POST("/", func(c *gin.Context) {
		b, err := io.ReadAll(c.Request.Body)
		if err != nil {
			t.Errorf("handler body read error: %v", err)
		}
		c.String(http.StatusOK, string(b))
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	return w.Body.String()
}

func TestRequestLogBodyIntactDebugDisabled(t *testing.T) {
	prev := log.Default()
	log.SetDefault(log.New(log.NewTextHandler(io.Discard, &log.HandlerOptions{Level: log.LevelInfo})))
	defer log.SetDefault(prev)

	body := `{"hello":"world","padding":"xxxxxxxxxxxxxxxxxxxxxxxx"}`
	if got := postWithBody(t, body); got != body {
		t.Fatalf("body not intact: got %q, want %q", got, body)
	}
}

func TestRequestLogBodyIntactDebugEnabled(t *testing.T) {
	prev := log.Default()
	log.SetDefault(log.New(log.NewTextHandler(io.Discard, &log.HandlerOptions{Level: log.LevelDebug})))
	defer log.SetDefault(prev)

	body := `{"hello":"world"}`
	if got := postWithBody(t, body); got != body {
		t.Fatalf("body not intact: got %q, want %q", got, body)
	}
}
