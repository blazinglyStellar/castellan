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
			INSERT INTO accounts (owner_id, balance, currency)
			VALUES ($1, $2, 'XLM')
			ON CONFLICT (owner_id) DO NOTHING
		`, entry.userID, entry.balance)
		if err != nil {
			return fmt.Errorf("create account for %s: %w", entry.userID, err)
		}
		if tag.RowsAffected() > 0 {
			fmt.Printf("  created account for user %s with balance %s XLM\n", entry.userID, entry.balance)
		}
	}

	return nil
}

func seedDeposits(ctx context.Context, pool *pgxpool.Pool, consumerID, providerOwnerID uuid.UUID) error {
	consumerAccountID, providerAccountID, err := getAccountIDs(ctx, pool, consumerID, providerOwnerID)
	if err != nil {
		return fmt.Errorf("get account ids: %w", err)
	}

	deposits := []struct {
		accountID   uuid.UUID
		fromAddress string
		amount      string
		memo        string
		txHash      string
		status      string
	}{
		{consumerAccountID, "GA4ACA...EXAMPLE", "500.00", "dep_consumer_1", "a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6", "confirmed"},
		{consumerAccountID, "GA4ACA...EXAMPLE", "200.00", "dep_consumer_2", "b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7", "confirmed"},
		{consumerAccountID, "GA4ACA...EXAMPLE", "100.00", "dep_consumer_3", "c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8", "pending"},
		{providerAccountID, "GB4PVD...PROVIDER", "1000.00", "dep_provider_1", "d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9", "confirmed"},
		{providerAccountID, "GB4PVD...PROVIDER", "500.00", "dep_provider_2", "e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0", "confirmed"},
		{providerAccountID, "GB4PVD...PROVIDER", "300.00", "dep_provider_3", "f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1", "failed"},
	}

	for _, d := range deposits {
		tag, err := pool.Exec(ctx, `
			INSERT INTO deposits (account_id, from_address, amount, currency, memo, tx_hash, status)
			VALUES ($1, $2, $3, 'XLM', $4, $5, $6)
			ON CONFLICT (tx_hash) DO NOTHING
		`, d.accountID, d.fromAddress, d.amount, d.memo, d.txHash, d.status)
		if err != nil {
			return fmt.Errorf("insert deposit %s: %w", d.txHash, err)
		}
		if tag.RowsAffected() > 0 {
			fmt.Printf("  created deposit %s for account %s (%s)\n", d.txHash[:8], d.accountID.String()[:8], d.status)
		}
	}

	return nil
}

func getAccountIDs(ctx context.Context, pool *pgxpool.Pool, consumerID, providerOwnerID uuid.UUID) (consumerAccountID uuid.UUID, providerAccountID uuid.UUID, err error) {
	rows, err := pool.Query(ctx, `SELECT id, owner_id FROM accounts WHERE owner_id = ANY($1)`, []uuid.UUID{consumerID, providerOwnerID})
	if err != nil {
		return consumerAccountID, providerAccountID, fmt.Errorf("query accounts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id, ownerID uuid.UUID
		if scanErr := rows.Scan(&id, &ownerID); scanErr != nil {
			return consumerAccountID, providerAccountID, fmt.Errorf("scan account: %w", scanErr)
		}
		switch ownerID {
		case consumerID:
			consumerAccountID = id
		case providerOwnerID:
			providerAccountID = id
		}
	}
	err = rows.Err()
	if err != nil {
		return consumerAccountID, providerAccountID, fmt.Errorf("rows iteration: %w", err)
	}

	return consumerAccountID, providerAccountID, nil
}
