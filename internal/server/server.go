package server

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"castellan/internal/database"
	"castellan/internal/provider"
	"castellan/internal/proxy"
	"castellan/internal/repository/db"
)

type Server struct {
	port int

	db    database.Service
	proxy *proxy.Proxy
}

func NewServer() (*http.Server, error) {
	port, _ := strconv.Atoi(os.Getenv("PORT"))

	databaseService, err := database.New()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	queries := repository.New(databaseService.Pool())
	resolver, err := provider.NewDBResolver(queries)
	if err != nil {
		databaseService.Close()

		return nil, fmt.Errorf("failed to create provider resolver: %w", err)
	}

	pxy := proxy.NewReverseProxy(resolver, slog.Default())

	srv := &Server{
		port:  port,
		db:    databaseService,
		proxy: pxy,
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
