package middleware

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

const (
	defaultMaxBodySizeBytes int64 = 10 * 1024 * 1024
	maxBodySizeEnvKey             = "MAX_BODY_SIZE"
)

// MaxBodySizeFromEnv reads MAX_BODY_SIZE from the environment, defaulting to 10 MB.
func MaxBodySizeFromEnv() int64 {
	raw := os.Getenv(maxBodySizeEnvKey)
	if raw == "" {
		return defaultMaxBodySizeBytes
	}

	maxBytes, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || maxBytes <= 0 {
		return defaultMaxBodySizeBytes
	}

	return maxBytes
}

// MaxBodySize middleware limits request body size. Returns 413 when Content-Length exceeds maxBytes.
func MaxBodySize(maxBytes int64) func(http.Handler) http.Handler {
	if maxBytes <= 0 {
		maxBytes = defaultMaxBodySizeBytes
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > maxBytes {
				_ = r.Body.Close()
				http.Error(
					w,
					http.StatusText(http.StatusRequestEntityTooLarge),
					http.StatusRequestEntityTooLarge,
				)
				return
			}

			rw := &sizeLimitResponseWriter{ResponseWriter: w}
			r.Body = &sizeLimitReadCloser{
				ReadCloser: http.MaxBytesReader(rw, r.Body, maxBytes),
				writer:     rw,
			}

			next.ServeHTTP(rw, r)
		})
	}
}

type sizeLimitResponseWriter struct {
	http.ResponseWriter

	tooLarge    bool
	wroteHeader bool
}

func (w *sizeLimitResponseWriter) WriteHeader(statusCode int) {
	if w.tooLarge {
		return
	}

	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *sizeLimitResponseWriter) Write(data []byte) (int, error) {
	if w.tooLarge {
		return 0, http.ErrBodyNotAllowed
	}

	w.wroteHeader = true

	n, err := w.ResponseWriter.Write(data)
	if err != nil {
		return n, fmt.Errorf("write size-limited response: %w", err)
	}

	return n, nil
}

func (w *sizeLimitResponseWriter) rejectRequestEntityTooLarge() {
	if w.tooLarge {
		return
	}

	w.tooLarge = true
	if !w.wroteHeader {
		http.Error(
			w.ResponseWriter,
			http.StatusText(http.StatusRequestEntityTooLarge),
			http.StatusRequestEntityTooLarge,
		)
	}
}

type sizeLimitReadCloser struct {
	io.ReadCloser

	writer *sizeLimitResponseWriter
}

func (r *sizeLimitReadCloser) Read(data []byte) (int, error) {
	n, err := r.ReadCloser.Read(data)

	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		r.writer.rejectRequestEntityTooLarge()
	}
	if err == nil {
		return n, nil
	}
	if errors.Is(err, io.EOF) {
		return n, io.EOF
	}

	return n, fmt.Errorf("read size-limited request body: %w", err)
}
