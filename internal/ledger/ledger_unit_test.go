package ledger

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type mockRepository struct {
	ReserveBalanceFunc     func(ctx context.Context, ownerID uuid.UUID, amount decimal.Decimal, requestID string) (*ReserveBalanceResult, error)
	ConfirmReservationFunc func(ctx context.Context, requestID string) error
	ReleaseReservationFunc func(ctx context.Context, requestID string) error
}

func (m *mockRepository) ReserveBalance(ctx context.Context, ownerID uuid.UUID, amount decimal.Decimal, requestID string) (*ReserveBalanceResult, error) {
	return m.ReserveBalanceFunc(ctx, ownerID, amount, requestID)
}

func (m *mockRepository) ConfirmReservation(ctx context.Context, requestID string) error {
	return m.ConfirmReservationFunc(ctx, requestID)
}

func (m *mockRepository) ReleaseReservation(ctx context.Context, requestID string) error {
	return m.ReleaseReservationFunc(ctx, requestID)
}

func TestReserve_Success(t *testing.T) {
	expectedRef := uuid.New().String()

	mock := &mockRepository{
		ReserveBalanceFunc: func(_ context.Context, _ uuid.UUID, _ decimal.Decimal, requestID string) (*ReserveBalanceResult, error) {
			if requestID != expectedRef {
				t.Fatalf("expected requestID %s, got %s", expectedRef, requestID)
			}
			return &ReserveBalanceResult{}, nil
		},
	}

	ledger := NewPostgresLedgerWithRepo(mock)
	err := ledger.Reserve(context.Background(), uuid.New(), decimal.NewFromFloat(10), expectedRef)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestReserve_InsufficientBalance(t *testing.T) {
	mock := &mockRepository{
		ReserveBalanceFunc: func(_ context.Context, _ uuid.UUID, _ decimal.Decimal, _ string) (*ReserveBalanceResult, error) {
			return nil, ErrInsufficientBalance
		},
	}

	ledger := NewPostgresLedgerWithRepo(mock)
	err := ledger.Reserve(context.Background(), uuid.New(), decimal.NewFromFloat(100), uuid.New().String())
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("expected ErrInsufficientBalance, got %v", err)
	}
}

func TestCommit_Success(t *testing.T) {
	expectedRef := uuid.New().String()

	mock := &mockRepository{
		ConfirmReservationFunc: func(_ context.Context, requestID string) error {
			if requestID != expectedRef {
				t.Fatalf("expected requestID %s, got %s", expectedRef, requestID)
			}
			return nil
		},
	}

	ledger := NewPostgresLedgerWithRepo(mock)
	err := ledger.Commit(context.Background(), expectedRef)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestCommit_UnknownReference(t *testing.T) {
	mock := &mockRepository{
		ConfirmReservationFunc: func(_ context.Context, _ string) error {
			return ErrReservationNotFound
		},
	}

	ledger := NewPostgresLedgerWithRepo(mock)
	err := ledger.Commit(context.Background(), uuid.New().String())
	if !errors.Is(err, ErrReservationNotFound) {
		t.Fatalf("expected ErrReservationNotFound, got %v", err)
	}
}

func TestRelease_Success(t *testing.T) {
	expectedRef := uuid.New().String()

	mock := &mockRepository{
		ReleaseReservationFunc: func(_ context.Context, requestID string) error {
			if requestID != expectedRef {
				t.Fatalf("expected requestID %s, got %s", expectedRef, requestID)
			}
			return nil
		},
	}

	ledger := NewPostgresLedgerWithRepo(mock)
	err := ledger.Release(context.Background(), expectedRef)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
