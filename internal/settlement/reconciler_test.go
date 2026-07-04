//go:build integration

package settlement

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"castellan/internal/repository/db"
)

func seedProvider(ctx context.Context, t *testing.T, email, payoutAddr string) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	_, err := testPool.Exec(ctx,
		`INSERT INTO users (id, email, payout_stellar_address) VALUES ($1, $2, $3)`,
		userID, email, payoutAddr)
	if err != nil {
		t.Fatalf("seed user: %v", err)
	}

	providerID := uuid.New()
	_, err = testPool.Exec(ctx,
		`INSERT INTO providers (id, owner_id, name, base_url) VALUES ($1, $2, $3, $4)`,
		providerID, userID, "test-provider-"+email, "https://example.com")
	if err != nil {
		t.Fatalf("seed provider: %v", err)
	}

	return providerID
}

type mockMonitor struct {
	status TransactionStatus
	err    error
}

func (m *mockMonitor) MonitorTransaction(_ context.Context, _ string) (TransactionStatus, error) {
	return m.status, m.err
}

func TestRecoverFailed_SalvagesPreviouslyFailed(t *testing.T) {
	ctx := context.Background()

	p1 := seedProvider(ctx, t, "recover-p1@example.com", "GA_RECOVER_1")
	p2 := seedProvider(ctx, t, "recover-p2@example.com", "GA_RECOVER_2")

	consumer1, acct1 := seedConsumerWithAccount(ctx, t, "recover-c1@example.com")
	consumer2, acct2 := seedConsumerWithAccount(ctx, t, "recover-c2@example.com")

	seedDeductionLedgerEntry(ctx, t, acct1, "15.00")
	seedDeductionLedgerEntry(ctx, t, acct2, "25.00")

	reconciler := NewReconciler(testPool, testQueries)

	payouts := []ProviderPayout{
		{
			ProviderID:    p1,
			Amount:        decimal.NewFromFloat(15.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GA_RECOVER_1",
		},
		{
			ProviderID:    p2,
			Amount:        decimal.NewFromFloat(25.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GA_RECOVER_2",
		},
	}

	batchID, entries, err := reconciler.CreateBatch(ctx, payouts)
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	results := []PayoutResult{
		{ProviderID: p1, Status: TransactionFailed},
		{ProviderID: p2, Status: TransactionFailed},
	}
	if err := reconciler.FinalizeBatch(ctx, batchID, results); err != nil {
		t.Fatalf("FinalizeBatch: %v", err)
	}

	_, err = testPool.Exec(ctx,
		`UPDATE settlement_entries SET tx_hash = $1 WHERE id = $2`,
		"tx_recover_1", entries[0].ID)
	if err != nil {
		t.Fatalf("set tx_hash on entry 1: %v", err)
	}
	_, err = testPool.Exec(ctx,
		`UPDATE settlement_entries SET tx_hash = $1 WHERE id = $2`,
		"tx_recover_2", entries[1].ID)
	if err != nil {
		t.Fatalf("set tx_hash on entry 2: %v", err)
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

	for _, entry := range entries {
		got, err := testQueries.GetSettlementEntryByID(ctx, entry.ID)
		if err != nil {
			t.Fatalf("GetSettlementEntryByID(%s): %v", entry.ID, err)
		}
		if got.Status != repository.SettlementEntryStatusCompleted {
			t.Errorf("entry %s status = %s, want completed", entry.ID, got.Status)
		}
	}

	for _, ledgerAcctID := range []uuid.UUID{acct1, acct2} {
		ledgerEntries, err := testQueries.ListLedgerEntriesByAccount(
			ctx,
			repository.ListLedgerEntriesByAccountParams{
				AccountID: ledgerAcctID,
				Limit:     10,
				Offset:    0,
			},
		)
		if err != nil {
			t.Fatalf("ListLedgerEntriesByAccount(%s): %v", ledgerAcctID, err)
		}
		for _, le := range ledgerEntries {
			if le.EntryType == repository.EntryTypeDeduction && !le.ReferenceID.Valid {
				t.Errorf("ledger entry %s reference_id not set after recovery", le.ID)
			}
		}
	}

	_ = consumer1
	_ = consumer2
}

func TestRecoverFailed_NoopWhenMonitorStillFails(t *testing.T) {
	ctx := context.Background()

	p1 := seedProvider(ctx, t, "recover-noop@example.com", "GA_NOOP")

	reconciler := NewReconciler(testPool, testQueries)

	payouts := []ProviderPayout{
		{
			ProviderID:    p1,
			Amount:        decimal.NewFromFloat(10.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GA_NOOP",
		},
	}

	batchID, entries, err := reconciler.CreateBatch(ctx, payouts)
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	results := []PayoutResult{
		{ProviderID: p1, Status: TransactionFailed},
	}
	if err := reconciler.FinalizeBatch(ctx, batchID, results); err != nil {
		t.Fatalf("FinalizeBatch: %v", err)
	}

	_, err = testPool.Exec(ctx,
		`UPDATE settlement_entries SET tx_hash = $1 WHERE id = $2`,
		"tx_noop", entries[0].ID)
	if err != nil {
		t.Fatalf("set tx_hash: %v", err)
	}

	monitor := &mockMonitor{status: TransactionFailed}
	if err := reconciler.RecoverFailed(ctx, monitor); err != nil {
		t.Fatalf("RecoverFailed: %v", err)
	}

	batch, err := testQueries.GetSettlementBatchByID(ctx, batchID)
	if err != nil {
		t.Fatalf("GetSettlementBatchByID: %v", err)
	}
	if batch.Status != repository.BatchStatusFailed {
		t.Errorf("batch status = %s, want failed (unchanged)", batch.Status)
	}

	got, err := testQueries.GetSettlementEntryByID(ctx, entries[0].ID)
	if err != nil {
		t.Fatalf("GetSettlementEntryByID: %v", err)
	}
	if got.Status != repository.SettlementEntryStatusFailed {
		t.Errorf("entry status = %s, want failed (unchanged)", got.Status)
	}
}

func TestGetSettlementHistory_ReturnsBatchesWithEntries(t *testing.T) {
	ctx := context.Background()

	p1 := seedProvider(ctx, t, "history-p1@example.com", "GA_HIST_1")
	p2 := seedProvider(ctx, t, "history-p2@example.com", "GA_HIST_2")

	reconciler := NewReconciler(testPool, testQueries)

	payouts1 := []ProviderPayout{
		{
			ProviderID:    p1,
			Amount:        decimal.NewFromFloat(10.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GA_HIST_1",
		},
	}
	payouts2 := []ProviderPayout{
		{
			ProviderID:    p2,
			Amount:        decimal.NewFromFloat(20.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GA_HIST_2",
		},
	}

	batchID1, _, err := reconciler.CreateBatch(ctx, payouts1)
	if err != nil {
		t.Fatalf("CreateBatch 1: %v", err)
	}
	batchID2, _, err := reconciler.CreateBatch(ctx, payouts2)
	if err != nil {
		t.Fatalf("CreateBatch 2: %v", err)
	}

	batches, entriesMap, err := reconciler.GetSettlementHistory(ctx, 10, 0)
	if err != nil {
		t.Fatalf("GetSettlementHistory: %v", err)
	}

	if len(batches) < 2 {
		t.Errorf("expected at least 2 batches, got %d", len(batches))
	}

	for _, batchID := range []uuid.UUID{batchID1, batchID2} {
		entries, ok := entriesMap[batchID]
		if !ok {
			t.Errorf("no entries map entry for batch %s", batchID)
			continue
		}
		if len(entries) != 1 {
			t.Errorf("batch %s has %d entries, want 1", batchID, len(entries))
		}
	}
}

func TestGetSettlementHistory_Pagination(t *testing.T) {
	ctx := context.Background()

	p1 := seedProvider(ctx, t, "pagination-p1@example.com", "GA_PAGE_1")

	reconciler := NewReconciler(testPool, testQueries)

	for range 3 {
		payouts := []ProviderPayout{
			{
				ProviderID:    p1,
				Amount:        decimal.NewFromFloat(5.00),
				Currency:      repository.CurrencyXLM,
				WalletAddress: "GA_PAGE_1",
			},
		}
		_, _, err := reconciler.CreateBatch(ctx, payouts)
		if err != nil {
			t.Fatalf("CreateBatch: %v", err)
		}
	}

	batches, entriesMap, err := reconciler.GetSettlementHistory(ctx, 2, 0)
	if err != nil {
		t.Fatalf("GetSettlementHistory(limit=2): %v", err)
	}
	if len(batches) != 2 {
		t.Errorf("expected 2 batches with limit=2, got %d", len(batches))
	}

	batchesPage2, _, err := reconciler.GetSettlementHistory(ctx, 2, 2)
	if err != nil {
		t.Fatalf("GetSettlementHistory(offset=2): %v", err)
	}
	if len(batchesPage2) != 1 {
		t.Errorf("expected 1 batch on page 2, got %d", len(batchesPage2))
	}

	for _, b := range batches {
		if _, ok := entriesMap[b.ID]; !ok {
			t.Errorf("missing entries for batch %s", b.ID)
		}
	}
}

func TestCreateBatch_InsertsBatchAndEntries(t *testing.T) {
	ctx := context.Background()

	p1 := seedProvider(ctx, t, "cb-p1@example.com", "GAXXXX1")
	p2 := seedProvider(ctx, t, "cb-p2@example.com", "GAXXXX2")
	p3 := seedProvider(ctx, t, "cb-p3@example.com", "GAXXXX3")

	reconciler := NewReconciler(testPool, testQueries)

	payouts := []ProviderPayout{
		{
			ProviderID:    p1,
			Amount:        decimal.NewFromFloat(10.50),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GAXXXX1",
		},
		{
			ProviderID:    p2,
			Amount:        decimal.NewFromFloat(25.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GAXXXX2",
		},
		{
			ProviderID:    p3,
			Amount:        decimal.NewFromFloat(5.75),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GAXXXX3",
		},
	}

	batchID, entries, err := reconciler.CreateBatch(ctx, payouts)
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	if batchID == uuid.Nil {
		t.Fatal("expected non-nil batch ID")
	}

	batch, err := testQueries.GetSettlementBatchByID(ctx, batchID)
	if err != nil {
		t.Fatalf("GetSettlementBatchByID: %v", err)
	}
	if batch.Status != repository.BatchStatusPending {
		t.Errorf("batch status = %s, want pending", batch.Status)
	}
	if batch.EntryCount != 3 {
		t.Errorf("batch entry_count = %d, want 3", batch.EntryCount)
	}
	if batch.Currency != repository.CurrencyXLM {
		t.Errorf("batch currency = %s, want XLM", batch.Currency)
	}

	wantTotal := decimal.NewFromFloat(41.25)
	gotTotal, err := NumericToDecimal(batch.TotalAmount)
	if err != nil {
		t.Fatalf("parse batch total_amount: %v", err)
	}
	if !gotTotal.Equal(wantTotal) {
		t.Errorf("batch total_amount = %s, want %s", gotTotal.String(), wantTotal.String())
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	entryMap := make(map[uuid.UUID]repository.SettlementEntry)
	for _, e := range entries {
		entryMap[e.ProviderID] = e
	}

	for _, p := range payouts {
		e, ok := entryMap[p.ProviderID]
		if !ok {
			t.Errorf("no entry for provider %s", p.ProviderID)
			continue
		}
		if e.BatchID != batchID {
			t.Errorf("entry batch_id = %s, want %s", e.BatchID, batchID)
		}
		if e.Status != repository.SettlementEntryStatusPending {
			t.Errorf("entry %s status = %s, want pending", e.ID, e.Status)
		}
		if e.WalletAddress != p.WalletAddress {
			t.Errorf("entry wallet_address = %s, want %s", e.WalletAddress, p.WalletAddress)
		}
		if e.Currency != repository.CurrencyXLM {
			t.Errorf("entry currency = %s, want XLM", e.Currency)
		}

		gotAmt, err := NumericToDecimal(e.Amount)
		if err != nil {
			t.Fatalf("parse entry amount: %v", err)
		}
		if !gotAmt.Equal(p.Amount) {
			t.Errorf("entry amount = %s, want %s", gotAmt.String(), p.Amount.String())
		}
	}
}

func TestCreateBatch_EmptyPayouts(t *testing.T) {
	ctx := context.Background()
	reconciler := NewReconciler(testPool, testQueries)

	_, _, err := reconciler.CreateBatch(ctx, nil)
	if err != ErrNoPayouts {
		t.Errorf("expected ErrNoPayouts, got %v", err)
	}

	_, _, err = reconciler.CreateBatch(ctx, []ProviderPayout{})
	if err != ErrNoPayouts {
		t.Errorf("expected ErrNoPayouts for empty slice, got %v", err)
	}
}

func TestCreateBatch_AtomicityOnFailure(t *testing.T) {
	ctx := context.Background()

	p1 := seedProvider(ctx, t, "atomic-p1@example.com", "GAATOM")

	reconciler := NewReconciler(testPool, testQueries)

	invalidProviderID := uuid.New()
	payouts := []ProviderPayout{
		{
			ProviderID:    p1,
			Amount:        decimal.NewFromFloat(100.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GAATOM",
		},
		{
			ProviderID:    invalidProviderID,
			Amount:        decimal.NewFromFloat(50.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GABOGUS",
		},
	}

	_, _, err := reconciler.CreateBatch(ctx, payouts)
	if err == nil {
		t.Fatal("expected error due to foreign key violation on second entry")
	}

	var batchCount int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM settlement_batches`).Scan(&batchCount)
	if err != nil {
		t.Fatalf("count settlement_batches: %v", err)
	}
	if batchCount != 0 {
		t.Errorf("expected 0 batches after rollback, got %d", batchCount)
	}

	var entryCount int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM settlement_entries`).Scan(&entryCount)
	if err != nil {
		t.Fatalf("count settlement_entries: %v", err)
	}
	if entryCount != 0 {
		t.Errorf("expected 0 entries after rollback, got %d", entryCount)
	}
}

func TestCreateBatch_SinglePayout(t *testing.T) {
	ctx := context.Background()

	p1 := seedProvider(ctx, t, "single@example.com", "GASINGLE")

	reconciler := NewReconciler(testPool, testQueries)

	payouts := []ProviderPayout{
		{
			ProviderID:    p1,
			Amount:        decimal.NewFromFloat(99.99),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GASINGLE",
		},
	}

	batchID, entries, err := reconciler.CreateBatch(ctx, payouts)
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	if batchID == uuid.Nil {
		t.Fatal("expected non-nil batch ID")
	}

	batch, err := testQueries.GetSettlementBatchByID(ctx, batchID)
	if err != nil {
		t.Fatalf("GetSettlementBatchByID: %v", err)
	}
	if batch.EntryCount != 1 {
		t.Errorf("batch entry_count = %d, want 1", batch.EntryCount)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].ProviderID != p1 {
		t.Errorf("entry provider_id = %s, want %s", entries[0].ProviderID, p1)
	}
}

func seedConsumerWithAccount(
	ctx context.Context, t *testing.T, email string,
) (userID uuid.UUID, accountID uuid.UUID) {
	t.Helper()

	userID = uuid.New()
	_, err := testPool.Exec(ctx,
		`INSERT INTO users (id, email) VALUES ($1, $2)`,
		userID, email)
	if err != nil {
		t.Fatalf("seed consumer user: %v", err)
	}

	accountID = uuid.New()
	_, err = testPool.Exec(ctx,
		`INSERT INTO accounts (id, owner_id, balance, currency) VALUES ($1, $2, 1000, 'XLM')`,
		accountID, userID)
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}

	return userID, accountID
}

func seedDeductionLedgerEntry(
	ctx context.Context, t *testing.T,
	accountID uuid.UUID, amount string,
) {
	t.Helper()

	_, err := testPool.Exec(ctx,
		`INSERT INTO ledger_entries (account_id, entry_type, amount, balance_after, currency, status)
		 VALUES ($1, 'deduction', $2, (SELECT balance FROM accounts WHERE id = $1), 'XLM', 'completed')`,
		accountID, amount)
	if err != nil {
		t.Fatalf("seed ledger entry: %v", err)
	}
}

func TestFinalizeBatch_AllSuccess(t *testing.T) {
	ctx := context.Background()

	p1 := seedProvider(ctx, t, "fb-all-p1@example.com", "GASUCCESS")
	consumer1, acct1 := seedConsumerWithAccount(ctx, t, "fb-all-c1@example.com")
	consumer2, acct2 := seedConsumerWithAccount(ctx, t, "fb-all-c2@example.com")

	seedDeductionLedgerEntry(ctx, t, acct1, "10.00")
	seedDeductionLedgerEntry(ctx, t, acct2, "5.75")

	reconciler := NewReconciler(testPool, testQueries)

	payouts := []ProviderPayout{
		{
			ProviderID:    p1,
			Amount:        decimal.NewFromFloat(15.75),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GASUCCESS",
		},
	}

	batchID, entries, err := reconciler.CreateBatch(ctx, payouts)
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	results := []PayoutResult{
		{
			ProviderID: p1,
			TxHash:     "abc123",
			Status:     TransactionSuccess,
		},
	}

	err = reconciler.FinalizeBatch(ctx, batchID, results)
	if err != nil {
		t.Fatalf("FinalizeBatch: %v", err)
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

	for _, ledgerAcctID := range []uuid.UUID{acct1, acct2} {
		ledgerEntries, err := testQueries.ListLedgerEntriesByAccount(
			ctx,
			repository.ListLedgerEntriesByAccountParams{
				AccountID: ledgerAcctID,
				Limit:     10,
				Offset:    0,
			},
		)
		if err != nil {
			t.Fatalf("ListLedgerEntriesByAccount: %v", err)
		}

		for _, le := range ledgerEntries {
			if le.EntryType == repository.EntryTypeDeduction {
				if !le.ReferenceID.Valid {
					t.Errorf("ledger entry %s reference_id not set", le.ID)
				} else if le.ReferenceID.Bytes != batchID {
					t.Errorf("ledger entry %s reference_id = %v, want %v", le.ID, le.ReferenceID.Bytes, batchID)
				}
				if !le.ReferenceType.Valid {
					t.Errorf("ledger entry %s reference_type not set", le.ID)
				} else if le.ReferenceType.String != "settlement_batch" {
					t.Errorf("ledger entry %s reference_type = %s, want settlement_batch", le.ID, le.ReferenceType.String)
				}
			}
		}
	}

	_ = consumer1
	_ = consumer2
}

func TestFinalizeBatch_PartialFailure(t *testing.T) {
	ctx := context.Background()

	p1 := seedProvider(ctx, t, "fb-partial-p1@example.com", "GAP1")
	p2 := seedProvider(ctx, t, "fb-partial-p2@example.com", "GAP2")

	consumer1, acct1 := seedConsumerWithAccount(ctx, t, "fb-partial-c1@example.com")
	consumer2, acct2 := seedConsumerWithAccount(ctx, t, "fb-partial-c2@example.com")

	seedDeductionLedgerEntry(ctx, t, acct1, "20.00")
	seedDeductionLedgerEntry(ctx, t, acct2, "30.00")

	reconciler := NewReconciler(testPool, testQueries)

	payouts := []ProviderPayout{
		{
			ProviderID:    p1,
			Amount:        decimal.NewFromFloat(20.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GAP1",
		},
		{
			ProviderID:    p2,
			Amount:        decimal.NewFromFloat(30.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GAP2",
		},
	}

	batchID, entries, err := reconciler.CreateBatch(ctx, payouts)
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	results := []PayoutResult{
		{
			ProviderID: p1,
			TxHash:     "tx1",
			Status:     TransactionSuccess,
		},
		{
			ProviderID: p2,
			Error:      "insufficient funds",
			Status:     TransactionFailed,
		},
	}

	err = reconciler.FinalizeBatch(ctx, batchID, results)
	if err != nil {
		t.Fatalf("FinalizeBatch: %v", err)
	}

	batch, err := testQueries.GetSettlementBatchByID(ctx, batchID)
	if err != nil {
		t.Fatalf("GetSettlementBatchByID: %v", err)
	}
	if batch.Status != repository.BatchStatusFailed {
		t.Errorf("batch status = %s, want failed", batch.Status)
	}

	entry1, err := testQueries.GetSettlementEntryByID(ctx, entries[0].ID)
	if err != nil {
		t.Fatalf("GetSettlementEntryByID: %v", err)
	}
	if entry1.Status != repository.SettlementEntryStatusCompleted {
		t.Errorf("entry1 status = %s, want completed", entry1.Status)
	}

	entry2, err := testQueries.GetSettlementEntryByID(ctx, entries[1].ID)
	if err != nil {
		t.Fatalf("GetSettlementEntryByID: %v", err)
	}
	if entry2.Status != repository.SettlementEntryStatusFailed {
		t.Errorf("entry2 status = %s, want failed", entry2.Status)
	}

	p1LedgerEntries, err := testQueries.ListLedgerEntriesByAccount(
		ctx,
		repository.ListLedgerEntriesByAccountParams{
			AccountID: acct1,
			Limit:     10,
			Offset:    0,
		},
	)
	if err != nil {
		t.Fatalf("ListLedgerEntriesByAccount: %v", err)
	}
	for _, le := range p1LedgerEntries {
		if le.EntryType == repository.EntryTypeDeduction {
			if !le.ReferenceID.Valid {
				t.Errorf("p1 ledger entry %s reference_id should be set", le.ID)
			}
		}
	}

	p2LedgerEntries, err := testQueries.ListLedgerEntriesByAccount(
		ctx,
		repository.ListLedgerEntriesByAccountParams{
			AccountID: acct2,
			Limit:     10,
			Offset:    0,
		},
	)
	if err != nil {
		t.Fatalf("ListLedgerEntriesByAccount: %v", err)
	}
	for _, le := range p2LedgerEntries {
		if le.EntryType == repository.EntryTypeDeduction {
			if le.ReferenceID.Valid {
				t.Errorf("p2 ledger entry %s reference_id should NOT be set (failed payout)", le.ID)
			}
		}
	}

	_ = consumer1
	_ = consumer2
}

func TestFinalizeBatch_Idempotent(t *testing.T) {
	ctx := context.Background()

	p1 := seedProvider(ctx, t, "fb-idem-p1@example.com", "GAIDEM")

	reconciler := NewReconciler(testPool, testQueries)

	payouts := []ProviderPayout{
		{
			ProviderID:    p1,
			Amount:        decimal.NewFromFloat(50.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GAIDEM",
		},
	}

	batchID, _, err := reconciler.CreateBatch(ctx, payouts)
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	results := []PayoutResult{
		{
			ProviderID: p1,
			TxHash:     "txidem",
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

func TestFinalizeBatch_UnknownBatch(t *testing.T) {
	ctx := context.Background()

	reconciler := NewReconciler(testPool, testQueries)
	err := reconciler.FinalizeBatch(ctx, uuid.New(), []PayoutResult{})
	if err != ErrBatchNotFound {
		t.Errorf("expected ErrBatchNotFound, got %v", err)
	}
}

func TestFinalizeBatch_AtomicityOnFailure(t *testing.T) {
	ctx := context.Background()

	p1 := seedProvider(ctx, t, "fb-atomic-p1@example.com", "GAATOM")

	consumer1, acct1 := seedConsumerWithAccount(ctx, t, "fb-atomic-c1@example.com")
	seedDeductionLedgerEntry(ctx, t, acct1, "100.00")

	reconciler := NewReconciler(testPool, testQueries)

	payouts := []ProviderPayout{
		{
			ProviderID:    p1,
			Amount:        decimal.NewFromFloat(100.00),
			Currency:      repository.CurrencyXLM,
			WalletAddress: "GAATOM",
		},
	}

	batchID, _, err := reconciler.CreateBatch(ctx, payouts)
	if err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}

	defer func() {
		_, _ = testPool.Exec(ctx, `DELETE FROM settlement_entries WHERE batch_id = $1`, batchID)
		_, _ = testPool.Exec(ctx, `DELETE FROM settlement_batches WHERE id = $1`, batchID)
	}()

	invalidProviderID := uuid.New()
	results := []PayoutResult{
		{
			ProviderID: invalidProviderID,
			TxHash:     "",
			Status:     TransactionSuccess,
		},
	}

	err = reconciler.FinalizeBatch(ctx, batchID, results)
	if err == nil {
		t.Fatal("expected error for non-existent provider entry in batch")
	}

	var batchCount int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM settlement_batches WHERE id = $1 AND status = 'pending'`, batchID).Scan(&batchCount)
	if err != nil {
		t.Fatalf("count batches: %v", err)
	}
	if batchCount != 1 {
		t.Errorf("expected batch to remain pending after rollback, got count = %d", batchCount)
	}

	var markedCount int
	err = testPool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ledger_entries WHERE account_id = $1 AND reference_id IS NOT NULL`,
		acct1,
	).Scan(&markedCount)
	if err != nil {
		t.Fatalf("count marked ledger entries: %v", err)
	}
	if markedCount != 0 {
		t.Errorf("expected 0 marked ledger entries after rollback, got %d", markedCount)
	}

	_ = consumer1
	_ = acct1
}
