package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"castellan/internal/database"
	"castellan/internal/deposit"
	"castellan/internal/repository/db"
	"castellan/internal/stellar"

	_ "github.com/joho/godotenv/autoload"
)

const shutdownGracePeriod = 5 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("worker failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("shutting down workers")
		cancel()
	}()

	dbSvc, err := database.New()
	if err != nil {
		return fmt.Errorf("database connection: %w", err)
	}
	defer dbSvc.Close()

	queries := repository.New(dbSvc.Pool())
	stellarCfg := stellar.ConfigFromEnv()

	watcher := deposit.NewWatcher(queries, stellarCfg)
	go func() {
		if err := watcher.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("watcher error", slog.String("error", err.Error()))
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down workers")
	time.Sleep(shutdownGracePeriod)

	return nil
}
