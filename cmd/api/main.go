package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"castellan/internal/database"
	"castellan/internal/repository/db"
	"castellan/internal/seed"
	"castellan/internal/server"

	_ "github.com/joho/godotenv/autoload"
)

func gracefulShutdown(apiServer *http.Server, done chan bool) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	log.Println("shutting down gracefully, press Ctrl+C again to force")
	stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown with error: %v", err)
	}

	log.Println("Server exiting")

	done <- true
}

func main() {
	if os.Getenv("SEED") == "true" {
		log.Println("SEED=true: running database seed...")
		dbService, err := database.New()
		if err != nil {
			log.Fatalf("seed database connection: %v", err)
		}
		queries := repository.New(dbService.Pool())
		seedCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		if err := seed.Run(seedCtx, dbService.Pool(), queries); err != nil {
			cancel()
			_ = dbService.Close() // #nosec G104
			log.Fatalf("seed failed: %v", err)
		}
		cancel()
		_ = dbService.Close() // #nosec G104
		log.Println("seed complete")
	}

	srv, err := server.NewServer()
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	done := make(chan bool, 1)

	go gracefulShutdown(srv, done)

	err = srv.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		panic(fmt.Sprintf("http server error: %s", err))
	}

	<-done
	log.Println("Graceful shutdown complete.")
}
