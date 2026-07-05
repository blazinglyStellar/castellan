//go:build integration

package settlement

import (
	"context"
	"testing"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func seedUsageAndDeduction(
	ctx context.Context, t *testing.T,
	providerID uuid.UUID, email, amount string,
) uuid.UUID {
	t.Helper()

	consumerID, accountID := seedConsumerWithAccount(ctx, t, email)
	_, err := testPool.Exec(ctx,
		`INSERT INTO usage_events (id, consumer_id, provider_id, request_cost, currency, request_id, status)
		 VALUES ($1, $2, $3, $4, 'XLM', $5, 'completed')`,
		uuid.New(), consumerID, providerID, amount, uuid.New().String())
	if err != nil {
		t.Fatalf("seed usage event: %v", err)
	}

	seedDeductionLedgerEntry(ctx, t, accountID, amount)

	return accountID
}

func assertLedgerEntriesMarked(ctx context.Context, t *testing.T, accountID uuid.UUID) {
	t.Helper()

	ledgerEntries, err := testQueries.ListLedgerEntriesByAccount(ctx, repository.ListLedgerEntriesByAccountParams{
		AccountID: accountID,
		Limit:     10,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("ListLedgerEntriesByAccount(%s): %v", accountID, err)
	}
	for _, le := range ledgerEntries {
		if le.EntryType != repository.EntryTypeDeduction {
			continue
		}
		if !le.ReferenceID.Valid {
			t.Errorf("ledger entry %s reference_id not set", le.ID)
		}
		if !le.ReferenceType.Valid || le.ReferenceType.String != "settlement_batch" {
			t.Errorf("ledger entry %s reference_type = %v, want settlement_batch", le.ID, le.ReferenceType)
		}
	}
}

func TestSettlementLifecycle_FullCycle_Success(t *testing.T) {
	ctx := context.Background()

	providerID := seedProvider(ctx, t, "lifecycle-full@example.com", "GA_FULL")

	acct1 := seedUsageAndDeduction(ctx, t, providerID, "lifecycle-full-c1@example.com", "10.00")
	acct2 := seedUsageAndDeduction(ctx, t, providerID, "lifecycle-full-c2@example.com", "5.75")

	agg := NewAggregator(testQueries)
	sub := &mockSubmitter{
		submitPayoutsFunc: func(_ context.Context, _ uuid.UUID, payouts []ProviderPayout) ([]PayoutResult, error) {
			results := make([]PayoutResult, len(payouts))
			for i, p := range payouts {
				results[i] = PayoutResult{ProviderID: p.ProviderID, TxHash: "tx_full_success", Status: TransactionSuccess}
			}
			return results, nil
		},
	}
	rec := NewReconciler(testPool, testQueries)
	cycle := newTestCycle(agg, sub, rec)
	cycle.minThreshold = decimal.Zero

	if err := cycle.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	batches, err := testQueries.ListSettlementBatches(ctx, repository.ListSettlementBatchesParams{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListSettlementBatches: %v", err)
	}
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(batches))
	}
	if batches[0].Status != repository.BatchStatusCompleted {
		t.Errorf("batch status = %s, want completed", batches[0].Status)
	}
	if !batches[0].CompletedAt.Valid {
		t.Errorf("batch completed_at not set")
	}

	entries, err := testQueries.ListSettlementEntriesByBatch(ctx, batches[0].ID)
	if err != nil {
		t.Fatalf("ListSettlementEntriesByBatch: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.Status != repository.SettlementEntryStatusCompleted {
			t.Errorf("entry %s status = %s, want completed", e.ID, e.Status)
		}
	}

	assertLedgerEntriesMarked(ctx, t, acct1)
	assertLedgerEntriesMarked(ctx, t, acct2)

	payouts, err := agg.Aggregate(ctx)
	if err != nil {
		t.Fatalf("Aggregate after cycle: %v", err)
	}
	if len(payouts) != 0 {
		t.Errorf("expected 0 unsettled payouts after cycle, got %d", len(payouts))
	}
}

func TestSettlementLifecycle_NoUnsettledPayouts(t *testing.T) {
	ctx := context.Background()

	agg := NewAggregator(testQueries)
	sub := &mockSubmitter{}
	rec := NewReconciler(testPool, testQueries)
	cycle := newTestCycle(agg, sub, rec)
	cycle.minThreshold = decimal.Zero

	if err := cycle.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	var batchCount int
	err := testPool.QueryRow(ctx, `SELECT COUNT(*) FROM settlement_batches`).Scan(&batchCount)
	if err != nil {
		t.Fatalf("count settlement_batches: %v", err)
	}
	if batchCount != 0 {
		t.Errorf("expected 0 batches, got %d", batchCount)
	}
}

func TestSettlementLifecycle_PartialStellarFailure(t *testing.T) {
	ctx := context.Background()

	provider1 := seedProvider(ctx, t, "lifecycle-partial-p1@example.com", "GA_PARTIAL_1")
	provider2 := seedProvider(ctx, t, "lifecycle-partial-p2@example.com", "GA_PARTIAL_2")

	acct1 := seedUsageAndDeduction(ctx, t, provider1, "lifecycle-partial-c1@example.com", "20.00")
	acct2 := seedUsageAndDeduction(ctx, t, provider2, "lifecycle-partial-c2@example.com", "30.00")

	agg := NewAggregator(testQueries)
	sub := &mockSubmitter{
		submitPayoutsFunc: func(_ context.Context, _ uuid.UUID, payouts []ProviderPayout) ([]PayoutResult, error) {
			results := make([]PayoutResult, len(payouts))
			for i, p := range payouts {
				if p.ProviderID == provider1 {
					results[i] = PayoutResult{ProviderID: p.ProviderID, TxHash: "tx_partial_ok", Status: TransactionSuccess}
				} else {
					results[i] = PayoutResult{ProviderID: p.ProviderID, Error: "stellar timeout", Status: TransactionFailed}
				}
			}
			return results, nil
		},
	}
	rec := NewReconciler(testPool, testQueries)
	cycle := newTestCycle(agg, sub, rec)
	cycle.minThreshold = decimal.Zero

	err := cycle.Run(ctx)
	if err == nil {
		t.Fatal("expected error from partial Stellar failure")
	}

	batches, err := testQueries.ListSettlementBatches(ctx, repository.ListSettlementBatchesParams{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListSettlementBatches: %v", err)
	}
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(batches))
	}
	if batches[0].Status != repository.BatchStatusFailed {
		t.Errorf("batch status = %s, want failed", batches[0].Status)
	}

	entries, err := testQueries.ListSettlementEntriesByBatch(ctx, batches[0].ID)
	if err != nil {
		t.Fatalf("ListSettlementEntriesByBatch: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	for _, e := range entries {
		if e.ProviderID == provider1 && e.Status != repository.SettlementEntryStatusCompleted {
			t.Errorf("provider1 entry status = %s, want completed", e.Status)
		}
		if e.ProviderID == provider2 && e.Status != repository.SettlementEntryStatusFailed {
			t.Errorf("provider2 entry status = %s, want failed", e.Status)
		}
	}

	assertLedgerEntriesMarked(ctx, t, acct1)

	ledgerEntries, err := testQueries.ListLedgerEntriesByAccount(ctx, repository.ListLedgerEntriesByAccountParams{
		AccountID: acct2,
		Limit:     10,
		Offset:    0,
	})
	if err != nil {
		t.Fatalf("ListLedgerEntriesByAccount(%s): %v", acct2, err)
	}
	for _, le := range ledgerEntries {
		if le.EntryType == repository.EntryTypeDeduction && le.ReferenceID.Valid {
			t.Errorf("failed provider ledger entry %s reference_id should not be set", le.ID)
		}
	}
}

func TestSettlementLifecycle_Recovery(t *testing.T) {
	ctx := context.Background()

	providerID := seedProvider(ctx, t, "lifecycle-recover@example.com", "GA_RECOVER")

	acctID := seedUsageAndDeduction(ctx, t, providerID, "lifecycle-recover-c@example.com", "50.00")

	reconciler := NewReconciler(testPool, testQueries)

	payouts := []ProviderPayout{
		{
			ProviderID:    providerID,
			Amount:        decimal.NewFromFloat(50.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GA_RECOVER",
		},
	}

	batchID, entries, err := reconciler.CreateBatch(ctx, payouts)
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	results := []PayoutResult{
		{ProviderID: providerID, Status: TransactionFailed},
	}
	if err := reconciler.FinalizeBatch(ctx, batchID, results); err != nil {
		t.Fatalf("FinalizeBatch: %v", err)
	}

	_, err = testPool.Exec(ctx,
		`UPDATE settlement_entries SET tx_hash = $1 WHERE id = $2`,
		"tx_recover", entries[0].ID)
	if err != nil {
		t.Fatalf("set tx_hash: %v", err)
	}

	monitor := &mockMonitor{status: TransactionSuccess}
	if err := reconciler.RecoverFailed(ctx, monitor); err != nil {
		t.Fatalf("RecoverFailed: %v", err)
	}

	batch, err := testQueries.GetSettlementBatchByID(ctx, batchID)
	if err != nil {
		t.Fatalf("GetSettlementBatchByID: %v", err)
	}
	if batch.Status != repository.BatchStatusCompleted {
		t.Errorf("batch status = %s, want completed", batch.Status)
	}

	entry, err := testQueries.GetSettlementEntryByID(ctx, entries[0].ID)
	if err != nil {
		t.Fatalf("GetSettlementEntryByID: %v", err)
	}
	if entry.Status != repository.SettlementEntryStatusCompleted {
		t.Errorf("entry status = %s, want completed", entry.Status)
	}

	assertLedgerEntriesMarked(ctx, t, acctID)
}

func TestSettlementLifecycle_FinalizeBatchIdempotent(t *testing.T) {
	ctx := context.Background()

	providerID := seedProvider(ctx, t, "lifecycle-idem@example.com", "GA_IDEM")

	_ = seedUsageAndDeduction(ctx, t, providerID, "lifecycle-idem-c@example.com", "75.00")

	reconciler := NewReconciler(testPool, testQueries)

	payouts := []ProviderPayout{
		{
			ProviderID:    providerID,
			Amount:        decimal.NewFromFloat(75.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GA_IDEM",
		},
	}

	batchID, _, err := reconciler.CreateBatch(ctx, payouts)
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	results := []PayoutResult{
		{
			ProviderID: providerID,
			TxHash:     "tx_idem",
			Status:     TransactionSuccess,
		},
	}

	err = reconciler.FinalizeBatch(ctx, batchID, results)
	if err != nil {
		t.Fatalf("first FinalizeBatch: %v", err)
	}

	err = reconciler.FinalizeBatch(ctx, batchID, results)
	if err != nil {
		t.Fatalf("second FinalizeBatch (should be no-op): %v", err)
	}

	batch, err := testQueries.GetSettlementBatchByID(ctx, batchID)
	if err != nil {
		t.Fatalf("GetSettlementBatchByID: %v", err)
	}
	if batch.Status != repository.BatchStatusCompleted {
		t.Errorf("batch status = %s, want completed", batch.Status)
	}
}
