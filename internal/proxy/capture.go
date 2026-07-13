package proxy

import (
	"io"
)

type countingReadCloser struct {
	io.ReadCloser
	counter *int64
}

func (r *countingReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	*r.counter += int64(n)

	return n, err //nolint:wrapcheck // passthrough to inner reader
}
