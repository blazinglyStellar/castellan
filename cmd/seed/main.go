package main

import (
	"context"
	"log"

	"castellan/internal/database"
	"castellan/internal/repository/db"
	"castellan/internal/seed"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	dbService, err := database.New()
	if err != nil {
		log.Fatalf("database connection: %v", err)
	}

	queries := repository.New(dbService.Pool())

	ctx := context.Background()
	if err := seed.Run(ctx, dbService.Pool(), queries); err != nil {
		_ = dbService.Close() // #nosec G104
		log.Fatalf("seed failed: %v", err)
	}

	_ = dbService.Close() // #nosec G104
	log.Println("seed complete")
}
