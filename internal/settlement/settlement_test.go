//go:build integration

package settlement

import (
	"context"
	"testing"
)

func cleanupSettlementData(ctx context.Context, t *testing.T) {
	t.Helper()

	_, err := testPool.Exec(ctx, `TRUNCATE TABLE
		settlement_entries,
		settlement_batches,
		ledger_entries,
		usage_events,
		providers,
		users
	RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("cleanup settlement data: %v", err)
	}
}
