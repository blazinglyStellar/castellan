package seed

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func seedLedgerEntries(ctx context.Context, pool *pgxpool.Pool, consumerAccountID, providerAccountID uuid.UUID) error {
	if err := seedConsumerLedger(ctx, pool, consumerAccountID); err != nil {
		return err
	}
	return seedProviderLedger(ctx, pool, providerAccountID)
}

func seedConsumerLedger(ctx context.Context, pool *pgxpool.Pool, accountID uuid.UUID) error {
	var count int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM ledger_entries WHERE account_id = $1`, accountID).Scan(&count); err != nil {
		return fmt.Errorf("count ledger entries: %w", err)
	}
	if count > 0 {
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	entries := []struct {
		entryType    string
		amount       string
		balanceAfter string
		description  string
	}{
		{"deposit", "700.00", "700.00", "initial deposit"},
		{"deduction", "-0.05", "699.95", "weather-api: current weather"},
		{"deduction", "-0.10", "699.85", "blockchain-node: transaction submission"},
		{"deduction", "-0.03", "699.82", "ai-inference: embedding generation"},
		{"refund", "0.02", "699.84", "partial refund for overcharge"},
	}

	for _, e := range entries {
		if _, err := tx.Exec(ctx, `
			INSERT INTO ledger_entries (account_id, entry_type, amount, balance_after, currency, status, description)
			VALUES ($1, $2, $3, $4, 'XLM', 'completed', $5)
		`, accountID, e.entryType, e.amount, e.balanceAfter, e.description); err != nil {
			return fmt.Errorf("insert ledger entry %s: %w", e.entryType, err)
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE accounts SET balance = $2, updated_at = now() WHERE id = $1`,
		accountID, "699.84"); err != nil {
		return fmt.Errorf("update consumer balance: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func seedProviderLedger(ctx context.Context, pool *pgxpool.Pool, accountID uuid.UUID) error {
	var count int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM ledger_entries WHERE account_id = $1`, accountID).Scan(&count); err != nil {
		return fmt.Errorf("count ledger entries: %w", err)
	}
	if count > 0 {
		return nil
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	entries := []struct {
		entryType    string
		amount       string
		balanceAfter string
		description  string
	}{
		{"deposit", "1500.00", "1500.00", "deposit from Stellar"},
		{"settlement", "50.00", "1550.00", "settlement payout batch #1"},
	}

	for _, e := range entries {
		if _, err := tx.Exec(ctx, `
			INSERT INTO ledger_entries (account_id, entry_type, amount, balance_after, currency, status, description)
			VALUES ($1, $2, $3, $4, 'XLM', 'completed', $5)
		`, accountID, e.entryType, e.amount, e.balanceAfter, e.description); err != nil {
			return fmt.Errorf("insert ledger entry %s: %w", e.entryType, err)
		}
	}

	if _, err := tx.Exec(ctx, `UPDATE accounts SET balance = $2, updated_at = now() WHERE id = $1`,
		accountID, "1550.00"); err != nil {
		return fmt.Errorf("update provider balance: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
