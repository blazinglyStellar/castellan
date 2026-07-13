package seed

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func seedAccounts(ctx context.Context, pool *pgxpool.Pool, consumerID, providerOwnerID uuid.UUID) error {
	for _, entry := range []struct {
		userID  uuid.UUID
		balance string
	}{
		{consumerID, "1000.00"},
		{providerOwnerID, "1000.00"},
	} {
		tag, err := pool.Exec(ctx, `
			UPDATE users SET balance = $2, currency = 'XLM', account_updated_at = now()
			WHERE id = $1 AND balance IS NULL
		`, entry.userID, entry.balance)
		if err != nil {
			return fmt.Errorf("seed balance for user %s: %w", entry.userID, err)
		}
		if tag.RowsAffected() > 0 {
			fmt.Printf("  seeded balance %s XLM for user %s\n", entry.balance, entry.userID)
		}
	}

	return nil
}

func seedDeposits(ctx context.Context, pool *pgxpool.Pool, consumerID, providerOwnerID uuid.UUID) error {
	deposits := []struct {
		userID      uuid.UUID
		fromAddress string
		amount      string
		memo        string
		txHash      string
		status      string
	}{
		{consumerID, "GA4ACA...EXAMPLE", "500.00", "dep_consumer_1", "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6", "confirmed"},
		{consumerID, "GA4ACA...EXAMPLE", "200.00", "dep_consumer_2", "b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7", "confirmed"},
		{consumerID, "GA4ACA...EXAMPLE", "100.00", "dep_consumer_3", "c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8", "pending"},
		{providerOwnerID, "GB4PVD...PROVIDER", "1000.00", "dep_provider_1", "d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9", "confirmed"},
		{providerOwnerID, "GB4PVD...PROVIDER", "500.00", "dep_provider_2", "e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0", "confirmed"},
		{providerOwnerID, "GB4PVD...PROVIDER", "300.00", "dep_provider_3", "f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1", "failed"},
	}

	for _, d := range deposits {
		tag, err := pool.Exec(ctx, `
			INSERT INTO deposits (user_id, from_address, amount, currency, memo, tx_hash, status)
			VALUES ($1, $2, $3, 'XLM', $4, $5, $6)
			ON CONFLICT (tx_hash) DO NOTHING
		`, d.userID, d.fromAddress, d.amount, d.memo, d.txHash, d.status)
		if err != nil {
			return fmt.Errorf("insert deposit %s: %w", d.txHash, err)
		}
		if tag.RowsAffected() > 0 {
			fmt.Printf("  created deposit %s for user %s (%s)\n", d.txHash[:8], d.userID.String()[:8], d.status)
		}
	}

	return nil
}
