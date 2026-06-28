package proxy

import (
	"os"
	"time"
)

const (
	defaultConnectTimeout = 5 * time.Second
	defaultReadTimeout    = 30 * time.Second
	defaultWriteTimeout   = 30 * time.Second
)

type Config struct {
	ConnectTimeout time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
}

func DefaultConfig() Config {
	return Config{
		ConnectTimeout: defaultConnectTimeout,
		ReadTimeout:    defaultReadTimeout,
		WriteTimeout:   defaultWriteTimeout,
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

	return cfg
}
