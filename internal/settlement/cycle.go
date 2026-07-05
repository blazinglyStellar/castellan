package settlement

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type payoutAggregator interface {
	Aggregate(ctx context.Context) ([]ProviderPayout, error)
}

type payoutSubmitter interface {
	TransactionMonitor
	SubmitPayouts(ctx context.Context, batchID uuid.UUID, payouts []ProviderPayout) ([]PayoutResult, error)
}

type cycleReconciler interface {
	RecoverFailed(ctx context.Context, monitor TransactionMonitor) error
	CreateBatch(ctx context.Context, payouts []ProviderPayout) (uuid.UUID, []repository.SettlementEntry, error)
	FinalizeBatch(ctx context.Context, batchID uuid.UUID, results []PayoutResult) error
}

type settlementCycle struct {
	aggregator   payoutAggregator
	submitter    payoutSubmitter
	reconciler   cycleReconciler
	minThreshold decimal.Decimal
}

func NewCycle(agg *Aggregator, sub *StellarSubmitter, rec *Reconciler, minThreshold decimal.Decimal) Cycle {
	if agg == nil {
		panic("settlement: nil Aggregator")
	}
	if sub == nil {
		panic("settlement: nil StellarSubmitter")
	}
	if rec == nil {
		panic("settlement: nil Reconciler")
	}

	return &settlementCycle{
		aggregator:   agg,
		submitter:    sub,
		reconciler:   rec,
		minThreshold: minThreshold,
	}
}

func (c *settlementCycle) Run(ctx context.Context) error {
	start := time.Now()

	log := slog.With(slog.String("component", "settlement_cycle"))

	log.InfoContext(ctx, "settlement cycle starting")

	if err := c.reconciler.RecoverFailed(ctx, c.submitter); err != nil {
		log.WarnContext(ctx, "recover failed batches encountered errors",
			slog.String("error", err.Error()),
		)
	}

	payouts, err := c.aggregator.Aggregate(ctx)
	if err != nil {
		return fmt.Errorf("aggregate payouts: %w", err)
	}

	if len(payouts) == 0 {
		log.InfoContext(ctx, "no unsettled payouts, skipping cycle")
		return nil
	}

	total := sumAmountsDecimal(payouts)
	if total.LessThan(c.minThreshold) {
		log.InfoContext(ctx, "total amount below minimum threshold, skipping cycle",
			slog.String("total", total.String()),
			slog.String("threshold", c.minThreshold.String()),
		)
		return nil
	}

	batchID, _, err := c.reconciler.CreateBatch(ctx, payouts)
	if err != nil {
		return fmt.Errorf("create batch: %w", err)
	}

	results, subErr := c.submitter.SubmitPayouts(ctx, batchID, payouts)

	if finalizeErr := c.reconciler.FinalizeBatch(ctx, batchID, results); finalizeErr != nil {
		log.ErrorContext(ctx, "finalize batch failed",
			slog.String("batch_id", batchID.String()),
			slog.String("error", finalizeErr.Error()),
		)
	}

	successCount := 0
	failureCount := 0
	for _, r := range results {
		switch r.Status {
		case TransactionSuccess:
			successCount++
		case TransactionFailed, TransactionPending:
			failureCount++
		}
	}

	log.InfoContext(ctx, "settlement cycle complete",
		slog.String("batch_id", batchID.String()),
		slog.Int("entry_count", len(payouts)),
		slog.Int("success_count", successCount),
		slog.Int("failure_count", failureCount),
		slog.String("total_amount", total.String()),
		slog.Duration("duration", time.Since(start)),
	)

	if subErr != nil {
		return fmt.Errorf("submit payouts (partial): %w", subErr)
	}

	return nil
}

func sumAmountsDecimal(payouts []ProviderPayout) decimal.Decimal {
	if len(payouts) == 0 {
		return decimal.Zero
	}

	sum := payouts[0].Amount
	for i := 1; i < len(payouts); i++ {
		sum = sum.Add(payouts[i].Amount)
	}

	return sum
}

var _ Cycle = (*settlementCycle)(nil)

const defaultRunnerInterval = 5 * time.Minute

type Runner struct {
	cycle    Cycle
	interval time.Duration
	mu       sync.Mutex
}

func NewRunner(cycle Cycle, interval time.Duration) *Runner {
	if interval <= 0 {
		interval = defaultRunnerInterval
	}

	return &Runner{
		cycle:    cycle,
		interval: interval,
	}
}

func (r *Runner) Run(ctx context.Context) error {
	log := slog.With(slog.String("component", "settlement_runner"))

	log.InfoContext(ctx, "settlement runner starting",
		slog.String("interval", r.interval.String()),
	)

	r.runCycle(ctx, log)

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.InfoContext(ctx, "settlement runner shutting down")
			return context.Canceled

		case <-ticker.C:
			if !r.mu.TryLock() {
				log.WarnContext(ctx, "previous settlement cycle still running, skipping tick")
				continue
			}

			func() {
				defer r.mu.Unlock()
				r.runCycle(ctx, log)
			}()
		}
	}
}

func (r *Runner) runCycle(ctx context.Context, log *slog.Logger) {
	if err := r.cycle.Run(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			return
		}

		log.ErrorContext(ctx, "settlement cycle error", slog.String("error", err.Error()))
	}
}
