package proxy

import (
	"crypto/rand"
	"math/big"
	"net/http"
	"time"
)

const (
	defaultMaxRetries = 3
	defaultBaseDelay  = 200 * time.Millisecond
	defaultMaxDelay   = 3 * time.Second
	backoffFactor     = 2
	jitterDivisor     = 2
	jitterCoinFlipMod = 2
)

type RetryPolicy struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries: defaultMaxRetries,
		BaseDelay:  defaultBaseDelay,
		MaxDelay:   defaultMaxDelay,
	}
}

func (p RetryPolicy) ShouldRetry(statusCode int, err error) bool {
	if err != nil {
		return true
	}

	return statusCode >= 500 && statusCode < 600
}

func (p RetryPolicy) Backoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	if p.BaseDelay <= 0 || p.MaxDelay <= 0 {
		return 0
	}

	exp := p.BaseDelay
	for range attempt {
		if exp >= p.MaxDelay/backoffFactor {
			exp = p.MaxDelay
			break
		}
		exp *= backoffFactor
	}

	n, _ := rand.Int(rand.Reader, new(big.Int).SetInt64(int64(exp)))
	jitter := time.Duration(n.Int64()) / jitterDivisor
	n, _ = rand.Int(rand.Reader, new(big.Int).SetInt64(jitterCoinFlipMod))
	if n.Int64() == 0 {
		jitter = -jitter
	}

	delay := exp + jitter
	if delay > p.MaxDelay {
		delay = p.MaxDelay
	}
	if delay < 0 {
		delay = 0
	}

	return delay
}

func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
}
