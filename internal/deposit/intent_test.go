package deposit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/repository/db"
	"castellan/internal/stellar"
)

func TestDepositIntent_Unauthenticated(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/deposits/intent", nil)

	h.DepositIntent(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body[errKey] != "authentication required" {
		t.Errorf("error message = %q, want %q", body[errKey], "authentication required")
	}
}

type mockQuerier struct {
	repository.Querier
	memo string
}

func (m *mockQuerier) EnsureUserDepositMemo(_ context.Context, _ uuid.UUID) (pgtype.Text, error) {
	return pgtype.Text{String: m.memo, Valid: true}, nil
}

func TestDepositIntent_Authenticated(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	memo := uuid.New().String()
	cfg := stellar.Config{
		HotWalletAddress: "GABCDEF12345",
		MinDepositAmount: decimal.NewFromInt(5),
	}

	svc := &Service{
		queries: &mockQuerier{memo: memo},
		cfg:     cfg,
	}
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/deposits/intent", nil)
	ctx := gatewaycontext.SetConsumerInfo(r.Context(), gatewaycontext.ConsumerInfo{
		ConsumerID:      userID.String(),
		APIKeyID:        uuid.New().String(),
		IsAuthenticated: true,
	})
	r = r.WithContext(ctx)

	h.DepositIntent(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp IntentResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if resp.Memo != memo {
		t.Errorf("memo = %q, want %q", resp.Memo, memo)
	}
	if resp.Destination != cfg.HotWalletAddress {
		t.Errorf("destination = %q, want %q", resp.Destination, cfg.HotWalletAddress)
	}
	if resp.MinAmount != cfg.MinDepositAmount.String() {
		t.Errorf("minimum_amount = %q, want %q", resp.MinAmount, cfg.MinDepositAmount.String())
	}
	if resp.Asset != "XLM" {
		t.Errorf("asset = %q, want %q", resp.Asset, "XLM")
	}
	if resp.SEP7URI == "" {
		t.Error("sep7_uri is empty")
	}
	if !strings.HasPrefix(resp.QRCode, "data:image/png;base64,") {
		t.Error("qr_code does not start with data:image/png;base64,")
	}
}

func TestDepositIntent_MemoIsStable(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	memo := uuid.New().String()
	svc := &Service{
		queries: &mockQuerier{memo: memo},
		cfg: stellar.Config{
			HotWalletAddress: "GABCDEF12345",
			MinDepositAmount: decimal.NewFromInt(5),
		},
	}
	h := NewHandler(svc)

	for i := range 3 {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/api/v1/deposits/intent", nil)
		ctx := gatewaycontext.SetConsumerInfo(r.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID:      userID.String(),
			APIKeyID:        uuid.New().String(),
			IsAuthenticated: true,
		})
		r = r.WithContext(ctx)

		h.DepositIntent(w, r)

		if w.Code != http.StatusOK {
			t.Fatalf("iteration %d: status = %d, want %d", i, w.Code, http.StatusOK)
		}

		var resp IntentResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("iteration %d: decode: %v", i, err)
		}
		if resp.Memo != memo {
			t.Errorf("iteration %d: memo = %q, want %q", i, resp.Memo, memo)
		}
	}
}

func TestDepositIntent_MemoGenerationFails(t *testing.T) {
	t.Parallel()

	svc := &Service{
		queries: &errorQuerier{},
		cfg: stellar.Config{
			HotWalletAddress: "GABCDEF12345",
			MinDepositAmount: decimal.NewFromInt(5),
		},
	}
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/deposits/intent", nil)
	ctx := gatewaycontext.SetConsumerInfo(r.Context(), gatewaycontext.ConsumerInfo{
		ConsumerID:      uuid.New().String(),
		APIKeyID:        uuid.New().String(),
		IsAuthenticated: true,
	})
	r = r.WithContext(ctx)

	h.DepositIntent(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

type errorQuerier struct {
	repository.Querier
}

func (e *errorQuerier) EnsureUserDepositMemo(_ context.Context, _ uuid.UUID) (pgtype.Text, error) {
	return pgtype.Text{}, errors.New("db error")
}

func TestEnsureDepositMemo_NullMemo(t *testing.T) {
	t.Parallel()

	svc := &Service{
		queries: &nullMemoQuerier{},
		cfg:     stellar.DefaultConfig(),
	}

	_, err := svc.EnsureDepositMemo(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected error for null memo, got nil")
	}
}

type nullMemoQuerier struct {
	repository.Querier
}

func (n *nullMemoQuerier) EnsureUserDepositMemo(_ context.Context, _ uuid.UUID) (pgtype.Text, error) {
	return pgtype.Text{Valid: false}, nil
}

func TestDepositIntent_InvalidConsumerID(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/deposits/intent", nil)
	ctx := gatewaycontext.SetConsumerInfo(r.Context(), gatewaycontext.ConsumerInfo{
		ConsumerID:      "not-a-uuid",
		APIKeyID:        uuid.New().String(),
		IsAuthenticated: true,
	})
	r = r.WithContext(ctx)

	h.DepositIntent(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
