package seed

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func seedWatcherCursor(ctx context.Context, pool *pgxpool.Pool) error {
	tag, err := pool.Exec(ctx, `
		INSERT INTO stellar_watcher_cursor (id, cursor, updated_at)
		VALUES (1, '123456789', now())
		ON CONFLICT (id) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("upsert watcher cursor: %w", err)
	}
	if tag.RowsAffected() > 0 {
		fmt.Println("  created stellar watcher cursor")
	}

	return nil
}
