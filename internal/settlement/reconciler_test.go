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
