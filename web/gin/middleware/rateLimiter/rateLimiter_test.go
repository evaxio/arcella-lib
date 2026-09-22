package rateLimiter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRateLimiter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(RateLimiter(&ConfigRateLimiter{Limit: 1, Seconds: 1}))
	handlerHits := 0
	router.GET("/", func(c *gin.Context) {
		handlerHits++
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("first request: status = %d, want 200", w.Code)
	}

	w = httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("second request: status = %d, want 429", w.Code)
	}
	if w.Body.String() == "" {
		t.Fatal("429 response must have a body")
	}
	if handlerHits != 1 {
		t.Fatalf("handler chain must be aborted on 429, handler hits = %d, want 1", handlerHits)
	}
}
