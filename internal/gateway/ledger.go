//revive:disable:package-directory-mismatch
package gateway

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type LedgerService interface {
	// Reserve creates a temporary hold on the consumer's account for the given
	// amount. The hold is identified by referenceID for later commit or release.
	Reserve(ctx context.Context, consumerID uuid.UUID, amount decimal.Decimal, referenceID string) error
	// Commit converts a prior reservation into a permanent deduction.
	Commit(ctx context.Context, referenceID string) error
	// Release cancels a prior reservation, returning the hold to the account.
	Release(ctx context.Context, referenceID string) error
}

// LedgerServiceFunc is an adapter that lets a set of functions be used as a
// LedgerService. Each field defaults to a no-op when nil.
type LedgerServiceFunc struct {
	ReserveFunc func(context.Context, uuid.UUID, decimal.Decimal, string) error
	CommitFunc  func(context.Context, string) error
	ReleaseFunc func(context.Context, string) error
}

func (f *LedgerServiceFunc) Reserve(ctx context.Context, consumerID uuid.UUID, amount decimal.Decimal, referenceID string) error {
	if f.ReserveFunc == nil {
		return nil
	}
	return f.ReserveFunc(ctx, consumerID, amount, referenceID)
}

func (f *LedgerServiceFunc) Commit(ctx context.Context, referenceID string) error {
	if f.CommitFunc == nil {
		return nil
	}
	return f.CommitFunc(ctx, referenceID)
}

func (f *LedgerServiceFunc) Release(ctx context.Context, referenceID string) error {
	if f.ReleaseFunc == nil {
		return nil
	}
	return f.ReleaseFunc(ctx, referenceID)
}

type NoopLedger struct{}

func (NoopLedger) Reserve(_ context.Context, _ uuid.UUID, _ decimal.Decimal, _ string) error {
	return nil
}

func (NoopLedger) Commit(_ context.Context, _ string) error {
	return nil
}

func (NoopLedger) Release(_ context.Context, _ string) error {
	return nil
}
