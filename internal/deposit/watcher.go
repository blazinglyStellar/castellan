package deposit

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"castellan/internal/repository/db"
	"castellan/internal/stellar"
)

const pollInterval = 30 * time.Second

type Watcher struct {
	queries repository.Querier
	cfg     stellar.Config
}

func NewWatcher(queries repository.Querier, cfg stellar.Config) *Watcher {
	return &Watcher{queries: queries, cfg: cfg}
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
	return nil
}
