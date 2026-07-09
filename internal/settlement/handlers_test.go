package settlement

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type mockLister struct {
	batches    []repository.SettlementBatch
	entriesMap map[uuid.UUID][]repository.ListSettlementEntriesByBatchWithProviderRow
	err        error
}

func (m *mockLister) GetSettlementHistoryByOwner(
	_ context.Context,
	_ uuid.UUID,
	_ time.Time,
	_ uuid.UUID,
	_ int32,
	_ string,
) ([]repository.SettlementBatch, map[uuid.UUID][]repository.ListSettlementEntriesByBatchWithProviderRow, error) {
	if m.err != nil {
		return nil, nil, m.err
	}
	return m.batches, m.entriesMap, nil
}

func authenticatedRequest(target string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, target, nil)
	req = req.WithContext(
		gatewaycontext.SetConsumerInfo(req.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      uuid.New().String(),
			IsAuthenticated: true,
		}),
	)
	return req
}

func unauthenticatedRequest(target string) *http.Request {
	return httptest.NewRequest(http.MethodGet, target, nil)
}

func makeNumeric(t *testing.T, s string) pgtype.Numeric {
	t.Helper()
	var n pgtype.Numeric
	if err := n.Scan(s); err != nil {
		t.Fatalf("scan numeric %s: %v", s, err)
	}
	return n
}

func TestListSettlements_Success(t *testing.T) {
	batchID := uuid.New()
	entryID := uuid.New()
	providerID := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	batch := repository.SettlementBatch{
		ID:          batchID,
		Status:      repository.BatchStatusCompleted,
		TotalAmount: makeNumeric(t, "150.0000000000"),
		Currency:    repository.CurrencyXLM,
		EntryCount:  1,
		CreatedAt:   now,
		CompletedAt: pgtype.Timestamptz{Time: now.Add(time.Minute), Valid: true},
	}

	entry := repository.ListSettlementEntriesByBatchWithProviderRow{
		ID:            entryID,
		BatchID:       batchID,
		ProviderID:    providerID,
		Amount:        makeNumeric(t, "50.0000000000"),
		Currency:      repository.CurrencyXLM,
		WalletAddress: "GABCD1234",
		Status:        repository.SettlementEntryStatusCompleted,
		CreatedAt:     now,
		ProviderName:  "Test Provider",
	}

	mock := &mockLister{
		batches:    []repository.SettlementBatch{batch},
		entriesMap: map[uuid.UUID][]repository.ListSettlementEntriesByBatchWithProviderRow{batchID: {entry}},
	}

	h := &Handler{lister: mock}
	rec := httptest.NewRecorder()
	h.ListSettlements(rec, authenticatedRequest("/api/v1/settlements?limit=10"))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp settlementListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(resp.Data))
	}

	batchResp := resp.Data[0]
	if batchResp.ID != batchID.String() {
		t.Errorf("expected batch id %s, got %s", batchID.String(), batchResp.ID)
	}
	if batchResp.Status != string(repository.BatchStatusCompleted) {
		t.Errorf("expected status completed, got %s", batchResp.Status)
	}
	if batchResp.TotalAmount != "150" {
		t.Errorf("expected total_amount 150, got %s", batchResp.TotalAmount)
	}
	if batchResp.Currency != string(repository.CurrencyXLM) {
		t.Errorf("expected currency XLM, got %s", batchResp.Currency)
	}
	if batchResp.EntryCount != 1 {
		t.Errorf("expected entry_count 1, got %d", batchResp.EntryCount)
	}
	if batchResp.CompletedAt == nil {
		t.Error("expected non-nil completed_at")
	}

	if len(batchResp.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(batchResp.Entries))
	}

	entryResp := batchResp.Entries[0]
	if entryResp.ID != entryID.String() {
		t.Errorf("expected entry id %s, got %s", entryID.String(), entryResp.ID)
	}
	if entryResp.ProviderID != providerID.String() {
		t.Errorf("expected provider id %s, got %s", providerID.String(), entryResp.ProviderID)
	}
	if entryResp.Amount != "50" {
		t.Errorf("expected amount 50, got %s", entryResp.Amount)
	}
	if entryResp.WalletAddress != "GABCD1234" {
		t.Errorf("expected wallet GABCD1234, got %s", entryResp.WalletAddress)
	}
	if entryResp.Status != string(repository.SettlementEntryStatusCompleted) {
		t.Errorf("expected status completed, got %s", entryResp.Status)
	}
}

func TestListSettlements_EmptyList(t *testing.T) {
	mock := &mockLister{
		batches:    []repository.SettlementBatch{},
		entriesMap: map[uuid.UUID][]repository.ListSettlementEntriesByBatchWithProviderRow{},
	}

	h := &Handler{lister: mock}
	rec := httptest.NewRecorder()
	h.ListSettlements(rec, authenticatedRequest("/api/v1/settlements"))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp settlementListResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data) != 0 {
		t.Errorf("expected 0 batches, got %d", len(resp.Data))
	}
}

func TestListSettlements_InvalidParams(t *testing.T) {
	h := &Handler{lister: &mockLister{}}

	tests := []struct {
		name   string
		query  string
		expect int
	}{
		{"non-numeric limit", "limit=abc", http.StatusBadRequest},
		{"negative limit", "limit=-1", http.StatusBadRequest},
		{"limit exceeds max", "limit=101", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ListSettlements(rec, authenticatedRequest("/api/v1/settlements?"+tt.query))

			if rec.Code != tt.expect {
				t.Errorf("expected %d, got %d. Body: %s", tt.expect, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestListSettlements_Unauthenticated(t *testing.T) {
	h := &Handler{lister: &mockLister{}}
	rec := httptest.NewRecorder()
	h.ListSettlements(rec, unauthenticatedRequest("/api/v1/settlements"))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestListSettlements_DefaultPagination(t *testing.T) {
	mock := &mockLister{
		batches:    []repository.SettlementBatch{},
		entriesMap: map[uuid.UUID][]repository.ListSettlementEntriesByBatchWithProviderRow{},
	}

	h := &Handler{lister: mock}
	rec := httptest.NewRecorder()
	h.ListSettlements(rec, authenticatedRequest("/api/v1/settlements"))

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}
