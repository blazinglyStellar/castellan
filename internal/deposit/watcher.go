package deposit

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"castellan/internal/repository/db"
	"castellan/internal/stellar"
)

const pollInterval = 30 * time.Second

type Watcher struct {
	queries       repository.Querier
	cfg           stellar.Config
	creditHandler *CreditHandler
}

func NewWatcher(queries repository.Querier, cfg stellar.Config, pool *pgxpool.Pool, logger *slog.Logger) *Watcher {
	return &Watcher{
		queries:       queries,
		cfg:           cfg,
		creditHandler: NewCreditHandler(pool, queries, cfg, logger),
	}
}

func (w *Watcher) Run(ctx context.Context) error {
	slog.InfoContext(ctx, "deposit watcher starting",
		slog.String("horizon_url", w.cfg.HorizonURL),
		slog.String("network", w.cfg.Network),
	)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "deposit watcher stopped")
			return context.Canceled
		case <-ticker.C:
			if err := w.poll(ctx); err != nil {
				if !errors.Is(err, context.Canceled) {
					slog.ErrorContext(ctx, "deposit poll failed", slog.String("error", err.Error()))
				}
			}
		}
	}
}

func (w *Watcher) poll(_ context.Context) error {
	// TODO: fetch Stellar payments to hot wallet address from Horizon,
	// convert each to PaymentOperation, and call w.creditHandler.CreditDeposit(ctx, op).
	return nil
}
