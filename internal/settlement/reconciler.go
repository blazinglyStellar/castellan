package settlement

import (
	"context"
	"errors"
	"fmt"
	"math"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var (
	ErrNoPayouts       = errors.New("no payouts to settle")
	ErrMixedCurrencies = errors.New("all payouts must use the same currency")
)

type Reconciler struct {
	pool    *pgxpool.Pool
	queries *repository.Queries
}

func NewReconciler(pool *pgxpool.Pool, queries *repository.Queries) *Reconciler {
	return &Reconciler{
		pool:    pool,
		queries: queries,
	}
}

func (r *Reconciler) CreateBatch(
	ctx context.Context,
	payouts []ProviderPayout,
) (uuid.UUID, []repository.SettlementEntry, error) {
	if len(payouts) == 0 {
		return uuid.Nil, nil, ErrNoPayouts
	}
	if len(payouts) > math.MaxInt32 {
		return uuid.Nil, nil, errors.New("too many payouts")
	}

	batchCurrency := payouts[0].Currency
	total := decimal.Zero
	for _, p := range payouts {
		if p.Amount.Sign() <= 0 {
			return uuid.Nil, nil, fmt.Errorf("invalid payout amount for provider %s: %s", p.ProviderID, p.Amount)
		}
		if p.Currency != batchCurrency {
			return uuid.Nil, nil, ErrMixedCurrencies
		}
		total = total.Add(p.Amount)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	txQueries := r.queries.WithTx(tx)

	totalNumeric, err := DecimalToNumeric(total)
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("convert total to numeric: %w", err)
	}

	batch, err := txQueries.InsertSettlementBatch(ctx, repository.InsertSettlementBatchParams{
		Status:      repository.BatchStatusPending,
		TotalAmount: totalNumeric,
		Currency:    batchCurrency,
		EntryCount:  int32(len(payouts)), // #nosec G115 — bounds checked above
		TxHash:      pgtype.Text{Valid: false},
	})
	if err != nil {
		return uuid.Nil, nil, fmt.Errorf("insert settlement batch: %w", err)
	}

	entries := make([]repository.SettlementEntry, 0, len(payouts))
	for _, p := range payouts {
		amtNumeric, err := DecimalToNumeric(p.Amount)
		if err != nil {
			return uuid.Nil, nil, fmt.Errorf("convert payout amount to numeric: %w", err)
		}

		entry, err := txQueries.InsertSettlementEntry(ctx, repository.InsertSettlementEntryParams{
			BatchID:       batch.ID,
			ProviderID:    p.ProviderID,
			Amount:        amtNumeric,
			Currency:      p.Currency,
			WalletAddress: p.WalletAddress,
			Status:        repository.SettlementEntryStatusPending,
		})
		if err != nil {
			return uuid.Nil, nil, fmt.Errorf("insert settlement entry for provider %s: %w", p.ProviderID, err)
		}

		entries = append(entries, entry)
	}

	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, nil, fmt.Errorf("commit transaction: %w", err)
	}

	return batch.ID, entries, nil
}
