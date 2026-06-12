package server

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"castellan/internal/database"
)

type Server struct {
	port int

	db database.Service
}

func NewServer() (*http.Server, error) {
	port, _ := strconv.Atoi(os.Getenv("PORT"))

	db, err := database.New()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	srv := &Server{
		port: port,

		db: db,
	}

	httpServer := &http.Server{
		Addr:         net.JoinHostPort("", strconv.Itoa(srv.port)),
		Handler:      srv.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return httpServer, nil
}
