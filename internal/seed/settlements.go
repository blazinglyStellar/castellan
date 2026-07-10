package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func seedSettlements(ctx context.Context, pool *pgxpool.Pool, providerOwnerID uuid.UUID) error {
	var count int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM settlement_batches`).Scan(&count); err != nil {
		return fmt.Errorf("count settlement batches: %w", err)
	}
	if count > 0 {
		return nil
	}

	providerIDs, err := queryProviderIDs(ctx, pool, providerOwnerID)
	if err != nil {
		return fmt.Errorf("query provider ids: %w", err)
	}
	if len(providerIDs) == 0 {
		return nil
	}

	now := time.Now()

	batches := []struct {
		status     string
		totalAmt   string
		entryCount int32
		txHash     string
		completed  time.Time
	}{
		{"completed", "25.00", 2, "sb_tx_completed_001", now.Add(-7 * 24 * time.Hour)},
		{"completed", "18.50", 1, "sb_tx_completed_002", now.Add(-3 * 24 * time.Hour)},
		{"pending", "12.30", 1, "", time.Time{}},
	}

	walletAddr := "GC4PVD...SETTLEMENT"

	for _, b := range batches {
		actualCount := min(int(b.entryCount), len(providerIDs))
		var batchID uuid.UUID
		var completedAt *time.Time
		if !b.completed.IsZero() {
			completedAt = &b.completed
		}
		err := pool.QueryRow(ctx, `
			INSERT INTO settlement_batches (status, total_amount, currency, entry_count, tx_hash, completed_at)
			VALUES ($1, $2, 'XLM', $3, NULLIF($4, ''), $5)
			RETURNING id
		`, b.status, b.totalAmt, actualCount, b.txHash, completedAt).Scan(&batchID)
		if err != nil {
			return fmt.Errorf("insert settlement batch %s: %w", b.status, err)
		}
		fmt.Printf("  created settlement batch %s (%s, %s XLM)\n",
			batchID.String()[:8], b.status, b.totalAmt)

		for j := range actualCount {
			pid := providerIDs[j]
			perProvider := fmt.Sprintf("%.2f", parseFloat(b.totalAmt)/float64(actualCount))
			if _, err := pool.Exec(ctx, `
				INSERT INTO settlement_entries (batch_id, provider_id, amount, currency, wallet_address, status)
				VALUES ($1, $2, $3, 'XLM', $4, 'completed')
			`, batchID, pid, perProvider, walletAddr); err != nil {
				return fmt.Errorf("insert settlement entry for batch %s provider %s: %w",
					batchID.String()[:8], pid.String()[:8], err)
			}
		}
	}

	return nil
}

func queryProviderIDs(ctx context.Context, pool *pgxpool.Pool, ownerID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := pool.Query(ctx, `SELECT id FROM providers WHERE owner_id = $1 ORDER BY name`, ownerID)
	if err != nil {
		return nil, fmt.Errorf("query providers: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan provider: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return ids, nil
}
