package ledger

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

type PostgresLedger struct {
	repo *Repository
}

func NewPostgresLedger(pool *pgxpool.Pool) *PostgresLedger {
	return &PostgresLedger{repo: NewRepository(pool)}
}

func (l *PostgresLedger) Reserve(ctx context.Context, consumerID uuid.UUID, amount decimal.Decimal, referenceID string) error {
	_, err := l.repo.ReserveBalance(ctx, consumerID, amount, referenceID)
	return err
}

func (l *PostgresLedger) Commit(ctx context.Context, referenceID string) error {
	return l.repo.ConfirmReservation(ctx, referenceID)
}

func (l *PostgresLedger) Release(ctx context.Context, referenceID string) error {
	return l.repo.ReleaseReservation(ctx, referenceID)
}
