package middleware

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

var errHijackerNotSupported = errors.New("hijacker not supported")

type responseWriter struct {
	w          http.ResponseWriter
	statusCode int
	wrote      bool
}

func (rw *responseWriter) Header() http.Header {
	return rw.w.Header()
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if rw.wrote {
		return
	}
	rw.statusCode = statusCode
	rw.wrote = true
	rw.w.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wrote {
		rw.statusCode = http.StatusOK
		rw.wrote = true
	}
	n, err := rw.w.Write(b)
	if err != nil {
		return n, fmt.Errorf("write response: %w", err)
	}
	return n, nil
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.w.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := rw.w.(http.Hijacker)
	if !ok {
		return nil, nil, errHijackerNotSupported
	}
	conn, buf, err := h.Hijack()
	if err != nil {
		return nil, nil, fmt.Errorf("hijack response: %w", err)
	}
	return conn, buf, nil
}

func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	p, ok := rw.w.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	if err := p.Push(target, opts); err != nil {
		return fmt.Errorf("push response: %w", err)
	}
	return nil
}

// sensitiveHeaders lists request headers whose values may carry credentials
// and must be redacted before logging.
var sensitiveHeaders = []string{
	"Authorization",
	"Proxy-Authorization",
	"Cookie",
	"X-Api-Key",
	"X-Auth-Token",
}

// redactHeaders returns a clone of h with credential-bearing header values
// replaced by "[REDACTED]" so that API keys and session tokens never appear in
// log output.
func redactHeaders(h http.Header) http.Header {
	out := h.Clone()
	for _, name := range sensitiveHeaders {
		if out.Get(name) != "" {
			out.Set(name, "[REDACTED]")
		}
	}
	return out
}

// RequestLogger logs every request on completion with method, path, status,
// latency, and request ID. Credential-bearing headers (Authorization, Cookie,
// Proxy-Authorization, X-Api-Key, X-Auth-Token) are replaced with "[REDACTED]"
// so that API keys and session tokens never appear in log output. A sanitized
// authorization attr is appended only when the header is present on the request.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{w: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			latency := time.Since(start).Milliseconds()

			requestID := GetRequestID(r.Context())
			if requestID == "" {
				requestID = "unknown"
			}

			safe := redactHeaders(r.Header)

			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.statusCode),
				slog.Float64("latency_ms", float64(latency)),
				slog.String("request_id", requestID),
			}
			if safe.Get("Authorization") != "" {
				attrs = append(attrs, slog.String("authorization", safe.Get("Authorization")))
			}
			logger.LogAttrs(r.Context(), slog.LevelInfo, "request completed", attrs...)
		})
	}
}
