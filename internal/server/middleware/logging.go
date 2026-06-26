package middleware

import (
	"bufio"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"
)

var errHijackerNotSupported = errors.New("hijacker not supported")

type contextKey string

const requestIDKey contextKey = "request_id"

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
	return rw.w.Write(b)
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
	return h.Hijack()
}

func (rw *responseWriter) Push(target string, opts *http.PushOptions) error {
	p, ok := rw.w.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return p.Push(target, opts)
}

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

			requestID, _ := r.Context().Value(requestIDKey).(string)
			if requestID == "" {
				requestID = "unknown"
			}

			logger.InfoContext(
				r.Context(),
				"request completed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.statusCode),
				slog.Float64("latency_ms", float64(latency)),
				slog.String("request_id", requestID),
			)
		})
	}
}
