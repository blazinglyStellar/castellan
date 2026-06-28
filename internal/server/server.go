package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	_ "github.com/joho/godotenv/autoload"
	"github.com/shopspring/decimal"

	"castellan/internal/database"
	"castellan/internal/gateway"
	"castellan/internal/provider"
	"castellan/internal/proxy"
	"castellan/internal/repository/db"
	"castellan/internal/server/middleware"
)

type Server struct {
	port int

	db      database.Service
	proxy   *proxy.Proxy
	balance middleware.BalanceChecker
	ledger  gateway.LedgerService
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
		if closeErr := databaseService.Close(); closeErr != nil {
			slog.Warn("failed to close database after resolver error",
				slog.String("error", closeErr.Error()),
			)
		}

		return nil, fmt.Errorf("failed to create provider resolver: %w", err)
	}

	balancer := middleware.BalanceCheckerFunc(func(ctx context.Context, ownerID uuid.UUID) (decimal.Decimal, error) {
		balance, err := queries.GetAccountBalance(ctx, ownerID)
		if err != nil {
			return decimal.Zero, fmt.Errorf("balance unavailable: %w", err)
		}

		f64, err := balance.Float64Value()
		if err != nil {
			return decimal.Zero, fmt.Errorf("failed to convert balance: %w", err)
		}
		if !f64.Valid {
			return decimal.Zero, errors.New("balance is null")
		}

		return decimal.NewFromFloat(f64.Float64), nil
	})

	pxy := proxy.NewReverseProxy(resolver, slog.Default())

	srv := &Server{
		port:    port,
		db:      databaseService,
		proxy:   pxy,
		balance: balancer,
		ledger:  gateway.NoopLedger{},
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
