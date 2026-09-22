package log

import (
	log "log/slog"
	"net/http"
	"strings"
	"time"
)

// LoggingMiddleware example:
//
//	client := &http.Client {
//	       Transport: LoggingMiddleware(http.DefaultTransport),
//	}
func LoggingMiddleware(next http.RoundTripper) http.RoundTripper {
	return &loggingTransport{next}
}

type loggingTransport struct {
	rt http.RoundTripper
}

func maskSensitiveHeaders(h http.Header) http.Header {
	masked := make(http.Header, len(h))
	for k, v := range h {
		switch {
		case strings.EqualFold(k, "Authorization"), strings.EqualFold(k, "Proxy-Authorization"), strings.EqualFold(k, "Set-Cookie"):
			masked[k] = []string{"***"}
		default:
			masked[k] = v
		}
	}
	return masked
}

func (lt *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !log.Default().Enabled(req.Context(), log.LevelDebug) {
		return lt.rt.RoundTrip(req)
	}
	// Логируем запрос
	log.Debug("Request", log.String("method", req.Method), log.String("url", req.URL.String()))
	log.Debug("Header", log.Any("value", maskSensitiveHeaders(req.Header)))

	// Логируем размер тела запроса, не читая его
	if req.Body != nil {
		log.Debug("Body", log.Int("size", int(req.ContentLength)))
	}

	// Замеряем время выполнения
	start := time.Now()
	resp, err := lt.rt.RoundTrip(req)
	duration := time.Since(start)

	// Логируем ответ
	if err != nil {
		log.Error("response", log.String("message", err.Error()))
		return resp, err
	}

	log.Debug("response", log.Int("code", resp.StatusCode), log.String("status", resp.Status), log.Duration("duration", duration))
	log.Debug("headers", log.Any("value", maskSensitiveHeaders(resp.Header)))

	return resp, err
}
