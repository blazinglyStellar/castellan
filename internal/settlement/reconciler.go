package settlement

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

var (
	ErrNoPayouts              = errors.New("no payouts to settle")
	ErrMixedCurrencies        = errors.New("all payouts must use the same currency")
	ErrBatchNotFound          = errors.New("settlement batch not found")
	ErrNoResults              = errors.New("no payout results provided")
	ErrResultProviderMismatch = errors.New("payout result references provider not in batch")
	ErrDuplicateResult        = errors.New("duplicate payout result for provider")
	ErrMissingResults         = errors.New("missing payout results for one or more providers")
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

func (r *Reconciler) FinalizeBatch(
	ctx context.Context,
	batchID uuid.UUID,
	results []PayoutResult,
) error {
	batch, err := r.queries.GetSettlementBatchByID(ctx, batchID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBatchNotFound
		}
		return fmt.Errorf("get settlement batch: %w", err)
	}

	if batch.Status == repository.BatchStatusCompleted {
		return nil
	}

	if len(results) == 0 {
		return ErrNoResults
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	txQueries := r.queries.WithTx(tx)

	entries, err := txQueries.ListSettlementEntriesByBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("list settlement entries: %w", err)
	}

	expectedProviders := make(map[uuid.UUID]repository.SettlementEntry, len(entries))
	for _, e := range entries {
		expectedProviders[e.ProviderID] = e
	}

	seenProviders := make(map[uuid.UUID]bool, len(results))
	allSuccess := true

	for _, result := range results {
		entry, ok := expectedProviders[result.ProviderID]
		if !ok {
			return fmt.Errorf("%w: %s", ErrResultProviderMismatch, result.ProviderID)
		}
		if seenProviders[result.ProviderID] {
			return fmt.Errorf("%w: %s", ErrDuplicateResult, result.ProviderID)
		}
		seenProviders[result.ProviderID] = true

		switch result.Status {
		case TransactionSuccess:
			_, updateErr := txQueries.UpdateSettlementEntryStatus(
				ctx,
				repository.UpdateSettlementEntryStatusParams{
					ID:     entry.ID,
					Status: repository.SettlementEntryStatusCompleted,
				},
			)
			if updateErr != nil {
				return fmt.Errorf("update entry %s to completed: %w", entry.ID, updateErr)
			}

			_, markErr := txQueries.MarkProviderLedgerEntriesSettled(
				ctx,
				repository.MarkProviderLedgerEntriesSettledParams{
					ProviderID:    result.ProviderID,
					ReferenceID:   pgtype.UUID{Bytes: batchID, Valid: true},
					ReferenceType: pgtype.Text{String: "settlement_batch", Valid: true},
				},
			)
			if markErr != nil {
				return fmt.Errorf("mark ledger entries for provider %s: %w", result.ProviderID, markErr)
			}

		case TransactionFailed:
			_, updateErr := txQueries.UpdateSettlementEntryStatus(
				ctx,
				repository.UpdateSettlementEntryStatusParams{
					ID:     entry.ID,
					Status: repository.SettlementEntryStatusFailed,
				},
			)
			if updateErr != nil {
				return fmt.Errorf("update entry %s to failed: %w", entry.ID, updateErr)
			}

			allSuccess = false

		case TransactionPending:
			allSuccess = false
		}
	}

	if len(seenProviders) != len(expectedProviders) {
		return fmt.Errorf("%w: batch %s has %d entries but %d results provided",
			ErrMissingResults, batchID, len(expectedProviders), len(seenProviders))
	}

	batchStatus := repository.BatchStatusCompleted
	if !allSuccess {
		batchStatus = repository.BatchStatusFailed
	}

	_, err = txQueries.UpdateSettlementBatchStatus(ctx, repository.UpdateSettlementBatchStatusParams{
		ID:     batchID,
		Status: batchStatus,
	})
	if err != nil {
		return fmt.Errorf("update batch status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *Reconciler) RecoverFailed(
	ctx context.Context,
	monitor TransactionMonitor,
) error {
	batches, err := r.queries.ListFailedSettlementBatches(ctx)
	if err != nil {
		return fmt.Errorf("list failed batches: %w", err)
	}

	var errs []error
	for _, batch := range batches {
		if err := r.recoverBatch(ctx, monitor, batch); err != nil {
			slog.WarnContext(
				ctx, "recover batch failed",
				slog.String("batch_id", batch.ID.String()),
				slog.String("error", err.Error()),
			)
			errs = append(errs, fmt.Errorf("batch %s: %w", batch.ID, err))
		}
	}

	return errors.Join(errs...)
}

func (r *Reconciler) recoverBatch(
	ctx context.Context,
	monitor TransactionMonitor,
	batch repository.SettlementBatch,
) error {
	entries, err := r.queries.ListSettlementEntriesByBatch(ctx, batch.ID)
	if err != nil {
		return fmt.Errorf("list entries: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	txQueries := r.queries.WithTx(tx)
	allResolved := true

	for _, entry := range entries {
		if entry.Status == repository.SettlementEntryStatusCompleted {
			continue
		}

		if entry.Status != repository.SettlementEntryStatusFailed || !entry.TxHash.Valid {
			allResolved = false
			continue
		}

		status, monErr := monitor.MonitorTransaction(ctx, entry.TxHash.String)
		if monErr != nil {
			return fmt.Errorf("monitor transaction %s: %w", entry.TxHash.String, monErr)
		}

		if status == TransactionSuccess {
			_, updateErr := txQueries.UpdateSettlementEntryStatus(
				ctx,
				repository.UpdateSettlementEntryStatusParams{
					ID:     entry.ID,
					Status: repository.SettlementEntryStatusCompleted,
				},
			)
			if updateErr != nil {
				return fmt.Errorf("update entry %s to completed: %w", entry.ID, updateErr)
			}

			_, markErr := txQueries.MarkProviderLedgerEntriesSettled(
				ctx,
				repository.MarkProviderLedgerEntriesSettledParams{
					ProviderID:    entry.ProviderID,
					ReferenceID:   pgtype.UUID{Bytes: batch.ID, Valid: true},
					ReferenceType: pgtype.Text{String: "settlement_batch", Valid: true},
				},
			)
			if markErr != nil {
				return fmt.Errorf("mark ledger entries for provider %s: %w", entry.ProviderID, markErr)
			}
		} else {
			allResolved = false
		}
	}

	if allResolved {
		_, err = txQueries.UpdateSettlementBatchStatus(
			ctx,
			repository.UpdateSettlementBatchStatusParams{
				ID:     batch.ID,
				Status: repository.BatchStatusCompleted,
			},
		)
		if err != nil {
			return fmt.Errorf("update batch status: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

func (r *Reconciler) GetSettlementHistory(
	ctx context.Context,
	limit, offset int32,
) ([]repository.SettlementBatch, map[uuid.UUID][]repository.SettlementEntry, error) {
	batches, err := r.queries.ListSettlementBatches(ctx, repository.ListSettlementBatchesParams{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("list settlement batches: %w", err)
	}

	entriesMap := make(map[uuid.UUID][]repository.SettlementEntry, len(batches))
	for _, batch := range batches {
		entries, err := r.queries.ListSettlementEntriesByBatch(ctx, batch.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("list entries for batch %s: %w", batch.ID, err)
		}
		entriesMap[batch.ID] = entries
	}

	return batches, entriesMap, nil
}

func (r *Reconciler) CountSettlementBatches(ctx context.Context) (int64, error) {
	count, err := r.queries.CountSettlementBatches(ctx)
	if err != nil {
		return 0, fmt.Errorf("count settlement batches: %w", err)
	}
	return count, nil
}

func (r *Reconciler) GetSettlementSummaryByOwner(ctx context.Context, ownerID uuid.UUID) (decimal.Decimal, error) {
	numeric, err := r.queries.GetSettlementSummaryByOwner(ctx, ownerID)
	if err != nil {
		return decimal.Zero, fmt.Errorf("get settlement summary: %w", err)
	}

	return NumericToDecimal(numeric)
}

func (r *Reconciler) GetSettlementMonthlyHistoryByOwner(ctx context.Context, ownerID uuid.UUID, limit int32) ([]MonthlySettlement, error) {
	rows, err := r.queries.GetSettlementMonthlyHistoryByOwner(ctx, repository.GetSettlementMonthlyHistoryByOwnerParams{
		OwnerID: ownerID,
		Limit:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("get settlement monthly history: %w", err)
	}

	items := make([]MonthlySettlement, 0, len(rows))
	for _, row := range rows {
		amt, err := NumericToDecimal(row.Amount)
		if err != nil {
			return nil, fmt.Errorf("convert amount: %w", err)
		}

		items = append(items, MonthlySettlement{
			Month:  row.Month,
			Amount: amt,
		})
	}

	return items, nil
}

func (r *Reconciler) GetSettlementHistoryByOwner(
	ctx context.Context,
	ownerID uuid.UUID,
	cursorTs time.Time,
	cursorID uuid.UUID,
	limit int32,
	status string,
) ([]repository.SettlementBatch, map[uuid.UUID][]repository.ListSettlementEntriesByBatchWithProviderRow, error) {
	var batches []repository.SettlementBatch
	var err error

	if status != "" {
		batches, err = r.queries.ListSettlementBatchesByOwnerFiltered(ctx, repository.ListSettlementBatchesByOwnerFilteredParams{
			OwnerID: ownerID,
			Limit:   limit,
			Column3: cursorTs,
			Column4: cursorID,
			Column5: status,
		})
	} else {
		batches, err = r.queries.ListSettlementBatchesByOwner(ctx, repository.ListSettlementBatchesByOwnerParams{
			OwnerID: ownerID,
			Limit:   limit,
			Column3: cursorTs,
			Column4: cursorID,
		})
	}
	if err != nil {
		return nil, nil, fmt.Errorf("list settlement batches by owner: %w", err)
	}

	entriesMap := make(map[uuid.UUID][]repository.ListSettlementEntriesByBatchWithProviderRow, len(batches))
	for _, batch := range batches {
		entries, err := r.queries.ListSettlementEntriesByBatchWithProvider(ctx, batch.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("list entries for batch %s: %w", batch.ID, err)
		}
		entriesMap[batch.ID] = entries
	}

	return batches, entriesMap, nil
}
