package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"castellan/internal/database"
	"castellan/internal/provider"
	"castellan/internal/repository/db"

	"github.com/google/uuid"
	_ "github.com/joho/godotenv/autoload"
)

func main() {
	if err := run(); err != nil {
		slog.Error("seed failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()
	logger := slog.Default()

	dbSvc, err := database.New()
	if err != nil {
		return fmt.Errorf("database connection: %w", err)
	}
	defer dbSvc.Close()

	queries := repository.New(dbSvc.Pool())

	const seedEmail = "seed-user@castellan.dev"

	if _, err := dbSvc.Pool().Exec(ctx,
		`INSERT INTO users (email) VALUES ($1) ON CONFLICT (email) DO NOTHING`,
		seedEmail,
	); err != nil {
		return fmt.Errorf("create seed user: %w", err)
	}
	var userID uuid.UUID
	if err := dbSvc.Pool().QueryRow(ctx,
		`SELECT id FROM users WHERE email = $1`, seedEmail,
	).Scan(&userID); err != nil {
		return fmt.Errorf("lookup seed user: %w", err)
	}

	if err := provider.SeedProviders(ctx, queries, userID); err != nil {
		return fmt.Errorf("seed providers: %w", err)
	}

	logger.Info("seed complete", slog.String("user_id", userID.String()))

	return nil
}
