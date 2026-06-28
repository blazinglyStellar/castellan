package proxy

import (
	"os"
	"strconv"
	"time"
)

const (
	defaultConnectTimeout  = 5 * time.Second
	defaultReadTimeout     = 30 * time.Second
	defaultWriteTimeout    = 30 * time.Second
	defaultRetryMaxRetries = 3
	defaultRetryBaseDelay  = 200 * time.Millisecond
	defaultRetryMaxDelay   = 3 * time.Second
)

type Config struct {
	ConnectTimeout  time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	RetryMaxRetries int
	RetryBaseDelay  time.Duration
	RetryMaxDelay   time.Duration
}

func DefaultConfig() Config {
	return Config{
		ConnectTimeout:  defaultConnectTimeout,
		ReadTimeout:     defaultReadTimeout,
		WriteTimeout:    defaultWriteTimeout,
		RetryMaxRetries: defaultRetryMaxRetries,
		RetryBaseDelay:  defaultRetryBaseDelay,
		RetryMaxDelay:   defaultRetryMaxDelay,
	}
}

func ConfigFromEnv() Config {
	cfg := DefaultConfig()

	if v := os.Getenv("UPSTREAM_CONNECT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.ConnectTimeout = d
		}
	}

	if v := os.Getenv("UPSTREAM_READ_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.ReadTimeout = d
		}
	}

	if v := os.Getenv("UPSTREAM_WRITE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.WriteTimeout = d
		}
	}

	if v := os.Getenv("RETRY_MAX_RETRIES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.RetryMaxRetries = n
		}
	}

	if v := os.Getenv("RETRY_BASE_DELAY"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.RetryBaseDelay = d
		}
	}

	if v := os.Getenv("RETRY_MAX_DELAY"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			cfg.RetryMaxDelay = d
		}
	}

	return cfg
}
