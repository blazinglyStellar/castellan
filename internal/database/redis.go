package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

const redisConnectTimeout = 5 * time.Second

// RedisClient wraps a go-redis client to provide connectivity to Redis.
type RedisClient struct {
	client *redis.Client
}

var (
	redisURL    = os.Getenv("REDIS_URL")
	redisClient *RedisClient
)

// NewRedis creates or returns a singleton Redis client from the REDIS_URL env var.
func NewRedis() (*RedisClient, error) {
	if redisClient != nil {
		return redisClient, nil
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}

	client := redis.NewClient(opts)

	ctx, cancel := context.WithTimeout(context.Background(), redisConnectTimeout)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}

	redisClient = &RedisClient{client: client}

	return redisClient, nil
}

// Close shuts down the Redis client connection.
func (r *RedisClient) Close() error {
	if err := r.client.Close(); err != nil {
		return fmt.Errorf("close redis client: %w", err)
	}

	return nil
}

// Client returns the underlying go-redis client.
func (r *RedisClient) Client() *redis.Client {
	return r.client
}
