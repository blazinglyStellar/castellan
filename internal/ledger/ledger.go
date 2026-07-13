package ledger

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// Repository defines the data-access contract for the ledger domain.
type Repository interface {
	ReserveBalance(ctx context.Context, ownerID uuid.UUID, amount decimal.Decimal, requestID string) (*ReserveBalanceResult, error)
	ConfirmReservation(ctx context.Context, requestID string) error
	ReleaseReservation(ctx context.Context, requestID string) error
}

type PostgresLedger struct {
	repo Repository
}

func NewPostgresLedger(pool *pgxpool.Pool) *PostgresLedger {
	return &PostgresLedger{repo: NewPostgresRepository(pool)}
}

func NewPostgresLedgerWithRepo(repo Repository) *PostgresLedger {
	return &PostgresLedger{repo: repo}
}

func (l *PostgresLedger) Reserve(ctx context.Context, consumerID uuid.UUID, amount decimal.Decimal, referenceID string) error {
	_, err := l.repo.ReserveBalance(ctx, consumerID, amount, referenceID)
	if err != nil {
		return fmt.Errorf("reserve: %w", err)
	}
	return nil
}

func (l *PostgresLedger) Commit(ctx context.Context, referenceID string) error {
	if err := l.repo.ConfirmReservation(ctx, referenceID); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

func (l *PostgresLedger) Release(ctx context.Context, referenceID string) error {
	if err := l.repo.ReleaseReservation(ctx, referenceID); err != nil {
		return fmt.Errorf("release: %w", err)
	}
	return nil
}
