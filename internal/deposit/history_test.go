package deposit

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/repository/db"
	"castellan/internal/stellar"
)

func TestListDeposits_Unauthenticated(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/deposits", nil)

	h.ListDeposits(w, r)

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

type listDepositsMockQuerier struct {
	repository.Querier
	account    *repository.Account
	deposits   []repository.Deposit
	accountErr error
	depositErr error
}

func (m *listDepositsMockQuerier) GetAccountByOwnerID(_ context.Context, _ uuid.UUID) (repository.Account, error) {
	if m.accountErr != nil {
		return repository.Account{}, m.accountErr
	}
	if m.account == nil {
		return repository.Account{}, pgx.ErrNoRows
	}
	return *m.account, nil
}

func (m *listDepositsMockQuerier) ListDepositsByAccount(_ context.Context, _ uuid.UUID) ([]repository.Deposit, error) {
	if m.depositErr != nil {
		return nil, m.depositErr
	}
	return m.deposits, nil
}

func TestListDeposits_Empty(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	accountID := uuid.New()

	mq := &listDepositsMockQuerier{
		account: &repository.Account{
			ID:        accountID,
			OwnerID:   userID,
			Balance:   pgtype.Numeric{Valid: true, Int: new(big.Int), Exp: 0},
			Currency:  repository.CurrencyXLM,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		deposits: []repository.Deposit{},
	}

	svc := &Service{queries: mq, cfg: stellar.DefaultConfig()}
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/deposits", nil)
	ctx := gatewaycontext.SetConsumerInfo(r.Context(), gatewaycontext.ConsumerInfo{
		ConsumerID:      userID.String(),
		IsAuthenticated: true,
	})
	r = r.WithContext(ctx)

	h.ListDeposits(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var deposits []Response
	if err := json.NewDecoder(w.Body).Decode(&deposits); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(deposits) != 0 {
		t.Errorf("len(deposits) = %d, want 0", len(deposits))
	}
}

func TestListDeposits_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	accountID := uuid.New()
	now := time.Now()
	confirmedAt := now.Add(30 * time.Second)

	deposit := repository.Deposit{
		ID:          uuid.New(),
		AccountID:   accountID,
		FromAddress: "GABC12345",
		Amount:      pgtype.Numeric{Valid: true, Int: big.NewInt(5000), Exp: 0},
		Currency:    repository.CurrencyXLM,
		Memo:        pgtype.Text{String: "memo-1", Valid: true},
		TxHash:      "a3f8c2d1e9b7a4f6",
		Status:      repository.DepositStatusConfirmed,
		CreatedAt:   now,
		ConfirmedAt: pgtype.Timestamptz{Time: confirmedAt, Valid: true},
	}

	mq := &listDepositsMockQuerier{
		account: &repository.Account{
			ID:        accountID,
			OwnerID:   userID,
			Balance:   pgtype.Numeric{Valid: true, Int: new(big.Int), Exp: 0},
			Currency:  repository.CurrencyXLM,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		deposits: []repository.Deposit{deposit},
	}

	svc := &Service{queries: mq, cfg: stellar.DefaultConfig()}
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/deposits", nil)
	ctx := gatewaycontext.SetConsumerInfo(r.Context(), gatewaycontext.ConsumerInfo{
		ConsumerID:      userID.String(),
		IsAuthenticated: true,
	})
	r = r.WithContext(ctx)

	h.ListDeposits(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var deposits []Response
	if err := json.NewDecoder(w.Body).Decode(&deposits); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(deposits) != 1 {
		t.Fatalf("len(deposits) = %d, want 1", len(deposits))
	}

	d := deposits[0]
	if d.ID != deposit.ID.String() {
		t.Errorf("id = %q, want %q", d.ID, deposit.ID.String())
	}
	if d.Amount != "5000" {
		t.Errorf("amount = %q, want %q", d.Amount, "5000")
	}
	if d.Currency != string(deposit.Currency) {
		t.Errorf("currency = %q, want %q", d.Currency, string(deposit.Currency))
	}
	if d.TxHash != deposit.TxHash {
		t.Errorf("tx_hash = %q, want %q", d.TxHash, deposit.TxHash)
	}
	if d.Status != string(deposit.Status) {
		t.Errorf("status = %q, want %q", d.Status, string(deposit.Status))
	}
	if d.FromAddress != deposit.FromAddress {
		t.Errorf("from_address = %q, want %q", d.FromAddress, deposit.FromAddress)
	}
	if d.ConfirmedAt == nil {
		t.Fatal("confirmed_at is nil, want non-nil")
	}
	if *d.ConfirmedAt != confirmedAt.Format(time.RFC3339) {
		t.Errorf("confirmed_at = %q, want %q", *d.ConfirmedAt, confirmedAt.Format(time.RFC3339))
	}
}

func TestListDeposits_InvalidConsumerID(t *testing.T) {
	t.Parallel()

	h := &Handler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/deposits", nil)
	ctx := gatewaycontext.SetConsumerInfo(r.Context(), gatewaycontext.ConsumerInfo{
		ConsumerID:      "not-a-uuid",
		IsAuthenticated: true,
	})
	r = r.WithContext(ctx)

	h.ListDeposits(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestListDeposits_NoAccount(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	mq := &listDepositsMockQuerier{
		accountErr: pgx.ErrNoRows,
	}

	svc := &Service{queries: mq, cfg: stellar.DefaultConfig()}
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/deposits", nil)
	ctx := gatewaycontext.SetConsumerInfo(r.Context(), gatewaycontext.ConsumerInfo{
		ConsumerID:      userID.String(),
		IsAuthenticated: true,
	})
	r = r.WithContext(ctx)

	h.ListDeposits(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var deposits []Response
	if err := json.NewDecoder(w.Body).Decode(&deposits); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(deposits) != 0 {
		t.Errorf("len(deposits) = %d, want 0", len(deposits))
	}
}

func TestListDeposits_AccountQueryError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	mq := &listDepositsMockQuerier{
		accountErr: errors.New("db error"),
	}

	svc := &Service{queries: mq, cfg: stellar.DefaultConfig()}
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/deposits", nil)
	ctx := gatewaycontext.SetConsumerInfo(r.Context(), gatewaycontext.ConsumerInfo{
		ConsumerID:      userID.String(),
		IsAuthenticated: true,
	})
	r = r.WithContext(ctx)

	h.ListDeposits(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestListDeposits_ListError(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	accountID := uuid.New()

	mq := &listDepositsMockQuerier{
		account: &repository.Account{
			ID:        accountID,
			OwnerID:   userID,
			Balance:   pgtype.Numeric{Valid: true, Int: new(big.Int), Exp: 0},
			Currency:  repository.CurrencyXLM,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		depositErr: errors.New("db list error"),
	}

	svc := &Service{queries: mq, cfg: stellar.DefaultConfig()}
	h := NewHandler(svc)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/deposits", nil)
	ctx := gatewaycontext.SetConsumerInfo(r.Context(), gatewaycontext.ConsumerInfo{
		ConsumerID:      userID.String(),
		IsAuthenticated: true,
	})
	r = r.WithContext(ctx)

	h.ListDeposits(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}
