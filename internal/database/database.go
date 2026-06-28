package database

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload"
)

// Service defines the database operations exposed to the rest of the application.
type Service interface {
	Health() map[string]string

	Close() error

	Pool() *pgxpool.Pool
}

type service struct {
	pool *pgxpool.Pool
}

var (
	database   = os.Getenv("BLUEPRINT_DB_DATABASE")
	password   = os.Getenv("BLUEPRINT_DB_PASSWORD")
	username   = os.Getenv("BLUEPRINT_DB_USERNAME")
	port       = os.Getenv("BLUEPRINT_DB_PORT")
	host       = os.Getenv("BLUEPRINT_DB_HOST")
	schema     = os.Getenv("BLUEPRINT_DB_SCHEMA")
	dbInstance *service
)

const (
	maxOpenConns       = 40
	maxWaitCount       = 1000
	connectTimeoutSecs = 5
)

// New creates or returns a singleton pgxpool connection from BLUEPRINT_DB_* env vars.
func New() (Service, error) {
	if dbInstance != nil {
		return dbInstance, nil
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable&search_path=%s",
		username, password, net.JoinHostPort(host, port), database, schema)

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeoutSecs*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()

		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	dbInstance = &service{
		pool: pool,
	}

	return dbInstance, nil
}

func (s *service) Pool() *pgxpool.Pool {
	return s.pool
}

func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	err := s.pool.Ping(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)

		return stats
	}

	stats["status"] = "up"
	stats["message"] = "It's healthy"

	poolStats := s.pool.Stat()
	stats["total_connections"] = strconv.Itoa(int(poolStats.TotalConns()))
	stats["idle_connections"] = strconv.Itoa(int(poolStats.IdleConns()))
	stats["acquired_connections"] = strconv.Itoa(int(poolStats.AcquiredConns()))
	stats["max_connections"] = strconv.Itoa(int(poolStats.MaxConns()))

	if poolStats.AcquiredConns() > maxOpenConns {
		stats["message"] = "The database is experiencing heavy load."
	}

	if poolStats.EmptyAcquireCount() > int64(maxWaitCount) {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	return stats
}

func (s *service) Close() error {
	log.Printf("Disconnected from database: %s", database)
	s.pool.Close()

	if dbInstance == s {
		dbInstance = nil
	}

	return nil
}
