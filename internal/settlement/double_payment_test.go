//go:build integration

package settlement

import (
	"context"
	"sync"
	"testing"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestBackToBackCycles_NoDoublePayment(t *testing.T) {
	ctx := context.Background()

	providerID := seedProvider(ctx, t, "b2b@example.com", "GA_B2B")

	acctID := seedUsageAndDeduction(ctx, t, providerID, "b2b-c1@example.com", "25.00")

	agg := NewAggregator(testQueries)
	sub := &mockSubmitter{
		submitPayoutsFunc: func(_ context.Context, _ uuid.UUID, payouts []ProviderPayout) ([]PayoutResult, error) {
			results := make([]PayoutResult, len(payouts))
			for i, p := range payouts {
				results[i] = PayoutResult{ProviderID: p.ProviderID, TxHash: "tx_b2b_1", Status: TransactionSuccess}
			}
			return results, nil
		},
	}
	rec := NewReconciler(testPool, testQueries)
	cycle := newTestCycle(agg, sub, rec)
	cycle.minThreshold = decimal.Zero

	if err := cycle.Run(ctx); err != nil {
		t.Fatalf("first cycle Run: %v", err)
	}

	batches, err := testQueries.ListSettlementBatches(ctx, repository.ListSettlementBatchesParams{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListSettlementBatches: %v", err)
	}
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch after first cycle, got %d", len(batches))
	}
	if batches[0].Status != repository.BatchStatusCompleted {
		t.Errorf("batch status = %s, want completed", batches[0].Status)
	}

	assertLedgerEntriesMarked(ctx, t, acctID)

	sub2 := &mockSubmitter{
		submitPayoutsFunc: func(_ context.Context, _ uuid.UUID, payouts []ProviderPayout) ([]PayoutResult, error) {
			t.Error("second cycle should not submit payouts (no unsettled payouts)")
			return nil, nil
		},
	}
	cycle2 := newTestCycle(agg, sub2, rec)
	cycle2.minThreshold = decimal.Zero

	if err := cycle2.Run(ctx); err != nil {
		t.Fatalf("second cycle Run: %v", err)
	}

	batches2, err := testQueries.ListSettlementBatches(ctx, repository.ListSettlementBatchesParams{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListSettlementBatches after second cycle: %v", err)
	}
	if len(batches2) != 1 {
		t.Errorf("expected 1 batch total after second cycle, got %d — new batch would double-pay", len(batches2))
	}

	assertLedgerEntriesMarked(ctx, t, acctID)
}

func TestMidCycleCrash_AfterCreateBatch_RecoverySafety(t *testing.T) {
	ctx := context.Background()

	providerID := seedProvider(ctx, t, "crash@example.com", "GA_CRASH")

	acctID := seedUsageAndDeduction(ctx, t, providerID, "crash-c1@example.com", "50.00")

	rec := NewReconciler(testPool, testQueries)

	payouts := []ProviderPayout{
		{
			ProviderID:    providerID,
			Amount:        decimal.NewFromFloat(50.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GA_CRASH",
		},
	}

	crashBatchID, _, err := rec.CreateBatch(ctx, payouts)
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	crashBatch, err := testQueries.GetSettlementBatchByID(ctx, crashBatchID)
	if err != nil {
		t.Fatalf("GetSettlementBatchByID: %v", err)
	}
	if crashBatch.Status != repository.BatchStatusPending {
		t.Errorf("crash batch status = %s, want pending", crashBatch.Status)
	}

	agg := NewAggregator(testQueries)
	sub := &mockSubmitter{
		submitPayoutsFunc: func(_ context.Context, _ uuid.UUID, payouts []ProviderPayout) ([]PayoutResult, error) {
			results := make([]PayoutResult, len(payouts))
			for i, p := range payouts {
				results[i] = PayoutResult{ProviderID: p.ProviderID, TxHash: "tx_crash_recovery", Status: TransactionSuccess}
			}
			return results, nil
		},
	}
	cycle := newTestCycle(agg, sub, rec)
	cycle.minThreshold = decimal.Zero

	if err := cycle.Run(ctx); err != nil {
		t.Fatalf("recovery cycle Run: %v", err)
	}

	batches, err := testQueries.ListSettlementBatches(ctx, repository.ListSettlementBatchesParams{Limit: 10, Offset: 0})
	if err != nil {
		t.Fatalf("ListSettlementBatches: %v", err)
	}
	if len(batches) != 2 {
		t.Fatalf("expected 2 batches (1 crashed + 1 recovery), got %d", len(batches))
	}

	crashBatchAfter, err := testQueries.GetSettlementBatchByID(ctx, crashBatchID)
	if err != nil {
		t.Fatalf("GetSettlementBatchByID for crash batch: %v", err)
	}
	if crashBatchAfter.Status != repository.BatchStatusPending {
		t.Errorf("crash batch status changed to %s, should remain pending (never finalized)", crashBatchAfter.Status)
	}

	var newBatch *repository.SettlementBatch
	for i := range batches {
		if batches[i].ID != crashBatchID {
			newBatch = &batches[i]
			break
		}
	}
	if newBatch == nil {
		t.Fatal("no new batch created by recovery cycle")
	}
	if newBatch.Status != repository.BatchStatusCompleted {
		t.Errorf("new batch status = %s, want completed", newBatch.Status)
	}

	newEntries, err := testQueries.ListSettlementEntriesByBatch(ctx, newBatch.ID)
	if err != nil {
		t.Fatalf("ListSettlementEntriesByBatch: %v", err)
	}
	for _, e := range newEntries {
		if e.Status != repository.SettlementEntryStatusCompleted {
			t.Errorf("new entry %s status = %s, want completed", e.ID, e.Status)
		}
	}

	crashEntries, err := testQueries.ListSettlementEntriesByBatch(ctx, crashBatchID)
	if err != nil {
		t.Fatalf("ListSettlementEntriesByBatch for crash batch: %v", err)
	}
	for _, e := range crashEntries {
		if e.Status != repository.SettlementEntryStatusPending {
			t.Errorf("crash entry %s status changed to %s, should remain pending", e.ID, e.Status)
		}
	}

	assertLedgerEntriesMarked(ctx, t, acctID)

	var dedupCount int
	err = testPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ledger_entries
		 WHERE account_id = $1 AND entry_type = 'deduction' AND reference_id IS NOT NULL`,
		acctID,
	).Scan(&dedupCount)
	if err != nil {
		t.Fatalf("count ledger marks: %v", err)
	}
	if dedupCount > 1 {
		t.Errorf("deduction ledger entries marked %d times — double-payment detected", dedupCount)
	}
}

func TestFinalizeBatch_DoubleCall_NoReMark(t *testing.T) {
	ctx := context.Background()

	providerID := seedProvider(ctx, t, "double-finalize@example.com", "GA_DFIN")
	_, acctID := seedConsumerWithAccount(ctx, t, "double-finalize-c1@example.com")
	seedDeductionLedgerEntry(ctx, t, acctID, "30.00")

	reconciler := NewReconciler(testPool, testQueries)

	payouts := []ProviderPayout{
		{
			ProviderID:    providerID,
			Amount:        decimal.NewFromFloat(30.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GA_DFIN",
		},
	}

	batchID, _, err := reconciler.CreateBatch(ctx, payouts)
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	results := []PayoutResult{
		{
			ProviderID: providerID,
			TxHash:     "tx_dfinal",
			Status:     TransactionSuccess,
		},
	}

	if err := reconciler.FinalizeBatch(ctx, batchID, results); err != nil {
		t.Fatalf("first FinalizeBatch: %v", err)
	}

	var markCountAfterFirst int
	err = testPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ledger_entries
		 WHERE account_id = $1 AND entry_type = 'deduction' AND reference_id IS NOT NULL
		 AND reference_type = 'settlement_batch'`,
		acctID,
	).Scan(&markCountAfterFirst)
	if err != nil {
		t.Fatalf("count marks after first finalize: %v", err)
	}

	if err := reconciler.FinalizeBatch(ctx, batchID, results); err != nil {
		t.Fatalf("second FinalizeBatch (should be no-op): %v", err)
	}

	var markCountAfterSecond int
	err = testPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ledger_entries
		 WHERE account_id = $1 AND entry_type = 'deduction' AND reference_id IS NOT NULL
		 AND reference_type = 'settlement_batch'`,
		acctID,
	).Scan(&markCountAfterSecond)
	if err != nil {
		t.Fatalf("count marks after second finalize: %v", err)
	}

	if markCountAfterSecond != markCountAfterFirst {
		t.Errorf("ledger mark count changed after second FinalizeBatch: %d → %d — re-mark detected",
			markCountAfterFirst, markCountAfterSecond)
	}

	if markCountAfterSecond != 1 {
		t.Errorf("expected exactly 1 deduction marked, got %d", markCountAfterSecond)
	}
}

func TestConcurrentCycles_NoRace(t *testing.T) {
	ctx := context.Background()

	p1 := seedProvider(ctx, t, "concurrent-p1@example.com", "GA_CONC_1")
	p2 := seedProvider(ctx, t, "concurrent-p2@example.com", "GA_CONC_2")

	_, acct1 := seedConsumerWithAccount(ctx, t, "concurrent-c1@example.com")
	_, acct2 := seedConsumerWithAccount(ctx, t, "concurrent-c2@example.com")

	seedDeductionLedgerEntry(ctx, t, acct1, "40.00")
	seedDeductionLedgerEntry(ctx, t, acct2, "60.00")

	reconciler := NewReconciler(testPool, testQueries)

	payouts := []ProviderPayout{
		{
			ProviderID:    p1,
			Amount:        decimal.NewFromFloat(40.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GA_CONC_1",
		},
		{
			ProviderID:    p2,
			Amount:        decimal.NewFromFloat(60.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GA_CONC_2",
		},
	}

	var wg sync.WaitGroup
	batchIDs := make([]uuid.UUID, 2)
	errs := make([]error, 2)

	for i := range 2 {
		wg.Add(1)

		go func(idx int) {
			defer wg.Done()

			id, _, err := reconciler.CreateBatch(ctx, payouts)
			batchIDs[idx] = id
			errs[idx] = err
		}(i)
	}

	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d CreateBatch error: %v", i, err)
		}
		if batchIDs[i] == uuid.Nil {
			t.Errorf("goroutine %d got nil batch ID", i)
		}
	}

	if batchIDs[0] == batchIDs[1] {
		t.Error("both goroutines got the same batch ID — should be different batches")
	}

	var totalBatches int
	err := testPool.QueryRow(ctx, `SELECT COUNT(*) FROM settlement_batches`).Scan(&totalBatches)
	if err != nil {
		t.Fatalf("count batches: %v", err)
	}
	if totalBatches != 2 {
		t.Errorf("expected exactly 2 batches, got %d — DB corruption or partial state", totalBatches)
	}

	for _, batchID := range batchIDs {
		batch, err := testQueries.GetSettlementBatchByID(ctx, batchID)
		if err != nil {
			t.Errorf("GetSettlementBatchByID(%s): %v", batchID, err)
			continue
		}
		if batch.EntryCount != 2 {
			t.Errorf("batch %s entry_count = %d, want 2", batchID, batch.EntryCount)
		}

		entries, err := testQueries.ListSettlementEntriesByBatch(ctx, batchID)
		if err != nil {
			t.Errorf("ListSettlementEntriesByBatch(%s): %v", batchID, err)
			continue
		}
		if len(entries) != 2 {
			t.Errorf("batch %s has %d entries, want 2", batchID, len(entries))
		}
	}
}


