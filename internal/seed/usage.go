//nolint:mnd // seed data constants and math/rand are acceptable for development seeding
package seed

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func seedUsageEvents(ctx context.Context, pool *pgxpool.Pool, consumerID uuid.UUID, eps []endpointInfo) error {
	if len(eps) == 0 {
		return nil
	}

	var count int64
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM usage_events WHERE consumer_id = $1`, consumerID).Scan(&count); err != nil {
		return fmt.Errorf("count usage events: %w", err)
	}
	if count > 0 {
		return nil
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano())) // #nosec G404
	now := time.Now()

	statuses := []struct {
		status     string
		statusCode int
		weight     int
	}{
		{"completed", 200, 50},
		{"completed", 201, 5},
		{"completed", 204, 3},
		{"failed", 400, 4},
		{"failed", 401, 2},
		{"failed", 429, 3},
		{"failed", 500, 3},
		{"pending", 0, 10},
	}

	totalWeight := 0
	for _, s := range statuses {
		totalWeight += s.weight
	}

	pickStatus := func() (string, int) {
		n := rng.Intn(totalWeight)
		for _, s := range statuses {
			n -= s.weight
			if n < 0 {
				return s.status, s.statusCode
			}
		}
		return "completed", 200
	}

	// 16 events over ~30 days
	numEvents := 16
	for i := range numEvents {
		offset := time.Duration(numEvents-1-i) * (30 * 24 * time.Hour / time.Duration(numEvents-1))
		ts := now.Add(-offset).Add(-time.Duration(rng.Intn(3600)) * time.Second)

		jitter := rng.Intn(6) - 3
		ts = ts.Add(time.Duration(jitter) * time.Hour)
		if ts.After(now) {
			ts = now.Add(-time.Minute)
		}

		ep := eps[rng.Intn(len(eps))]
		status, statusCode := pickStatus()

		latency := rng.Intn(800) + 50
		responseSize := rng.Intn(4000) + 200

		requestID := uuid.New().String()

		tag, err := pool.Exec(ctx, `
			INSERT INTO usage_events (
				consumer_id, provider_id, endpoint_id,
				request_cost, currency, status_code, latency_ms, response_size,
				request_id, status, created_at
			) VALUES ($1, $2, $3, $4, 'XLM', $5, $6, $7, $8, $9, $10)
			ON CONFLICT (request_id) DO NOTHING
		`, consumerID, ep.ProviderID, ep.EndpointID,
			ep.PriceAmount, statusCode, latency, responseSize,
			requestID, status, ts)
		if err != nil {
			return fmt.Errorf("insert usage event %d: %w", i, err)
		}
		if tag.RowsAffected() > 0 {
			fmt.Printf("  created usage event %d: %s on provider %s (%.2f XLM, %s)\n",
				i, ep.EndpointID.String()[:8], ep.ProviderID.String()[:8],
				float64(int(parseFloat(ep.PriceAmount)*100))/100, status)
		}
	}

	return nil
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
