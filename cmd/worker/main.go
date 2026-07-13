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
	"castellan/internal/settlement"
	"castellan/internal/stellar"

	"github.com/shopspring/decimal"

	_ "github.com/joho/godotenv/autoload"
)

const (
	shutdownGracePeriod       = 5 * time.Second
	defaultSettlementInterval = 5 * time.Minute
)

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

	creditHandler := deposit.NewCreditHandler(dbSvc.Pool(), stellarCfg)
	watcher := deposit.NewWatcher(queries, stellarCfg, creditHandler)
	go func() {
		if err := watcher.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("watcher error", slog.String("error", err.Error()))
		}
	}()

	settlementInterval := settlementIntervalFromEnv()
	settlementMinThreshold := settlementMinThresholdFromEnv()

	agg := settlement.NewAggregator(queries)
	submitter := settlement.NewStellarSubmitter(dbSvc.Pool(), queries, stellarCfg)
	reconciler := settlement.NewReconciler(dbSvc.Pool(), queries)
	cycle := settlement.NewCycle(agg, submitter, reconciler, settlementMinThreshold)
	runner := settlement.NewRunner(cycle, settlementInterval)

	slog.Info("settlement worker starting",
		slog.String("interval", settlementInterval.String()),
		slog.String("min_threshold", settlementMinThreshold.String()),
	)

	go func() {
		if err := runner.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("settlement runner error", slog.String("error", err.Error()))
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down workers")
	time.Sleep(shutdownGracePeriod)

	return nil
}

func settlementIntervalFromEnv() time.Duration {
	v := os.Getenv("SETTLEMENT_INTERVAL")
	if v == "" {
		return defaultSettlementInterval
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		slog.Warn("invalid SETTLEMENT_INTERVAL, using default",
			slog.String("value", v),
			slog.String("error", err.Error()),
		)

		return defaultSettlementInterval
	}

	return d
}

func settlementMinThresholdFromEnv() decimal.Decimal {
	v := os.Getenv("SETTLEMENT_MIN_THRESHOLD")
	if v == "" {
		return decimal.Zero
	}

	d, err := decimal.NewFromString(v)
	if err != nil {
		slog.Warn("invalid SETTLEMENT_MIN_THRESHOLD, using default",
			slog.String("value", v),
			slog.String("error", err.Error()),
		)

		return decimal.Zero
	}

	if d.IsNegative() {
		slog.Warn("SETTLEMENT_MIN_THRESHOLD must be non-negative, using default",
			slog.String("value", v),
		)

		return decimal.Zero
	}

	return d
}
