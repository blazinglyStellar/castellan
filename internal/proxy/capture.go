package proxy

import (
	"io"
	"net/http"
)

type captureResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (w *captureResponseWriter) WriteHeader(statusCode int) {
	if w.statusCode == 0 {
		w.statusCode = statusCode
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *captureResponseWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	w.bytesWritten += int64(n)

	return n, err //nolint:wrapcheck // passthrough to inner writer
}

type countingReadCloser struct {
	io.ReadCloser
	counter *int64
}

func (r *countingReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	*r.counter += int64(n)

	return n, err //nolint:wrapcheck // passthrough to inner reader
}
