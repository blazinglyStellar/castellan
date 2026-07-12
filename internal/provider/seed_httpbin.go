package provider

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	httpbinSystemEmail = "httpbin-system@castellan.local"
	httpbinMethodPOST  = "POST"
)

type httpbinEndpoint struct {
	Route       string
	Method      string
	Description string
}

var httpbinEndpoints = []httpbinEndpoint{
	{Route: "/get", Method: "GET", Description: "Echo back request data including headers, query parameters, and body. Perfect for verifying gateway authentication and header forwarding."},
	{Route: "/post", Method: httpbinMethodPOST, Description: "Echo back POST request data. Test how the gateway handles JSON payloads, form data, and file uploads."},
	{Route: "/patch", Method: "PATCH", Description: "Echo back PATCH request data. Verify partial update semantics through the gateway."},
	{Route: "/put", Method: "PUT", Description: "Echo back PUT request data. Test idempotent request patterns through the gateway."},
	{Route: "/delete", Method: "DELETE", Description: "Echo back DELETE request data. Verify the gateway correctly routes and tracks deletion requests."},
}

func SeedHttpbinProvider(ctx context.Context, pool *pgxpool.Pool, queries repository.Querier) error {
	ownerID, err := ensureHttpbinSystemUser(ctx, queries)
	if err != nil {
		return fmt.Errorf("ensure httpbin system user: %w", err)
	}

	if err := seedHttpbinProviderData(ctx, pool, ownerID); err != nil {
		return fmt.Errorf("seed httpbin provider: %w", err)
	}

	return nil
}

func ensureHttpbinSystemUser(ctx context.Context, queries repository.Querier) (uuid.UUID, error) {
	existing, err := queries.GetUserByEmail(ctx, httpbinSystemEmail)
	if err == nil {
		return existing.ID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("lookup system user: %w", err)
	}

	row, err := queries.UpsertUserByEmail(ctx, httpbinSystemEmail)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert system user: %w", err)
	}

	slog.InfoContext(ctx, "created httpbin system user",
		slog.String("user_id", row.ID.String()),
	)

	return row.ID, nil
}

//nolint:mnd
func seedHttpbinProviderData(ctx context.Context, pool *pgxpool.Pool, ownerID uuid.UUID) error {
	var providerID uuid.UUID
	if err := pool.QueryRow(ctx, `
		INSERT INTO providers (owner_id, name, base_url, description, status)
		VALUES ($1, 'httpbin', 'https://httpbin.org', 'Public httpbin.org test service — make test API calls with zero setup. All endpoints priced at 0.001 XLM each.', 'active')
		ON CONFLICT (name) DO UPDATE SET updated_at = NOW()
		RETURNING id
	`, ownerID).Scan(&providerID); err != nil {
		return fmt.Errorf("ensure httpbin provider: %w", err)
	}

	price := "0.001"
	rateLimit := 60

	for _, ep := range httpbinEndpoints {
		tag, err := pool.Exec(ctx, `
			INSERT INTO api_endpoints (provider_id, route, method, price_amount, currency, rate_limit, status, description)
			VALUES ($1, $2, $3, $4, 'XLM', $5, 'active', $6)
			ON CONFLICT (provider_id, route, method) DO NOTHING
		`, providerID, ep.Route, ep.Method, price, rateLimit, ep.Description)
		if err != nil {
			return fmt.Errorf("create endpoint %s %s: %w", ep.Method, ep.Route, err)
		}
		if tag.RowsAffected() > 0 {
			slog.DebugContext(ctx, "created httpbin endpoint",
				slog.String("route", ep.Route),
				slog.String("method", ep.Method),
			)
		}
	}

	slog.InfoContext(ctx, "seeded httpbin provider",
		slog.Int("endpoints", len(httpbinEndpoints)),
	)

	return nil
}
