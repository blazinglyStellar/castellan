package seed

import (
	"context"
	"fmt"
	"log/slog"

	"castellan/internal/provider"
	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func Run(ctx context.Context, pool *pgxpool.Pool, queries repository.Querier) error {
	consumerID, providerOwnerID, err := seedUsers(ctx, pool)
	if err != nil {
		return fmt.Errorf("seed users: %w", err)
	}

	slog.InfoContext(ctx, "seed using consumer",
		slog.String("user_id", consumerID.String()),
	)
	slog.InfoContext(ctx, "seed using provider owner",
		slog.String("user_id", providerOwnerID.String()),
	)

	if _, err := pool.Exec(ctx, `
		UPDATE users SET payout_stellar_address = 'GC4PVDXJ7THAFS6435YQH6Q5SQEKFHI7ZL7Y7Z6Y6J' WHERE id = $1 AND payout_stellar_address IS NULL
	`, providerOwnerID); err != nil {
		return fmt.Errorf("seed payout address: %w", err)
	}

	if err := provider.SeedProviders(ctx, queries, providerOwnerID); err != nil {
		return fmt.Errorf("seed providers: %w", err)
	}

	eps, err := queryEndpoints(ctx, pool, providerOwnerID)
	if err != nil {
		return fmt.Errorf("query endpoints: %w", err)
	}

	if err := seedAccounts(ctx, pool, consumerID, providerOwnerID); err != nil {
		return fmt.Errorf("seed accounts: %w", err)
	}

	if err := seedDeposits(ctx, pool, consumerID, providerOwnerID); err != nil {
		return fmt.Errorf("seed deposits: %w", err)
	}

	if err := seedUsageEvents(ctx, pool, consumerID, eps); err != nil {
		return fmt.Errorf("seed usage events: %w", err)
	}

	if err := seedLedgerEntries(ctx, pool, consumerID, providerOwnerID); err != nil {
		return fmt.Errorf("seed ledger entries: %w", err)
	}

	if err := seedSettlements(ctx, pool, providerOwnerID); err != nil {
		return fmt.Errorf("seed settlements: %w", err)
	}

	if err := seedWatcherCursor(ctx, pool); err != nil {
		return fmt.Errorf("seed watcher cursor: %w", err)
	}

	return nil
}

func seedUsers(ctx context.Context, pool *pgxpool.Pool) (consumerID uuid.UUID, providerOwnerID uuid.UUID, err error) {
	rows, qErr := pool.Query(ctx, `SELECT id FROM users ORDER BY created_at LIMIT 2`)
	if qErr != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("query users: %w", qErr)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return uuid.Nil, uuid.Nil, fmt.Errorf("scan user: %w", scanErr)
		}
		ids = append(ids, id)
	}
	if qErr = rows.Err(); qErr != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("rows iteration: %w", qErr)
	}

	switch len(ids) {
	case 0:
		emails := []string{"amustee11@gmail.com", "seed-provider@castellan.local"}
		memos := []string{"seed_consumer", "seed_provider"}
		for i, email := range emails {
			var id uuid.UUID
			if insErr := pool.QueryRow(ctx, `
				INSERT INTO users (email, deposit_memo)
				VALUES ($1, $2)
				ON CONFLICT (email) DO NOTHING
				RETURNING id
			`, email, memos[i]).Scan(&id); insErr != nil {
				return uuid.Nil, uuid.Nil, fmt.Errorf("insert user %s: %w", email, insErr)
			}
			switch i {
			case 0:
				consumerID = id
			case 1:
				providerOwnerID = id
			}
		}
	case 1:
		consumerID = ids[0]
		providerOwnerID = ids[0]
	default:
		consumerID = ids[0]
		providerOwnerID = ids[1]
	}

	return consumerID, providerOwnerID, nil
}

type endpointInfo struct {
	ProviderID  uuid.UUID
	EndpointID  uuid.UUID
	PriceAmount string
}

func queryEndpoints(ctx context.Context, pool *pgxpool.Pool, ownerID uuid.UUID) ([]endpointInfo, error) {
	rows, err := pool.Query(ctx, `
		SELECT p.id, e.id, e.price_amount
		FROM providers p
		JOIN api_endpoints e ON e.provider_id = p.id
		WHERE p.owner_id = $1
		ORDER BY p.name, e.route
	`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("query endpoints: %w", err)
	}
	defer rows.Close()

	var eps []endpointInfo
	for rows.Next() {
		var ep endpointInfo
		if err := rows.Scan(&ep.ProviderID, &ep.EndpointID, &ep.PriceAmount); err != nil {
			return nil, fmt.Errorf("scan endpoint: %w", err)
		}
		eps = append(eps, ep)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return eps, nil
}
