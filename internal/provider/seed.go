package provider

import (
	"context"
	"fmt"
	"log/slog"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type seedEndpoint struct {
	Route       string
	Method      string
	PriceAmount string
	RateLimit   int
}

type seedProvider struct {
	Name      string
	BaseURL   string
	Endpoints []seedEndpoint
}

//nolint:mnd
var seedData = []seedProvider{
	{
		Name:    "weather-api",
		BaseURL: "https://api.weather.example.com",
		Endpoints: []seedEndpoint{
			{Route: "/v1/weather/current", Method: "GET", PriceAmount: "0.01", RateLimit: 100},
			{Route: "/v1/weather/forecast", Method: "GET", PriceAmount: "0.02", RateLimit: 50},
		},
	},
	{
		Name:    "ai-inference",
		BaseURL: "https://inference.ai.example.com",
		Endpoints: []seedEndpoint{
			{Route: "/v1/chat/completions", Method: "POST", PriceAmount: "0.05", RateLimit: 30},
			{Route: "/v1/embeddings", Method: "POST", PriceAmount: "0.03", RateLimit: 60},
		},
	},
	{
		Name:    "blockchain-node",
		BaseURL: "https://node.blockchain.example.com",
		Endpoints: []seedEndpoint{
			{Route: "/v1/transactions", Method: "POST", PriceAmount: "0.10", RateLimit: 10},
		},
	},
}

func SeedProviders(ctx context.Context, queries repository.Querier, userID uuid.UUID) error {
	existing, err := queries.ListProvidersByOwner(ctx, userID)
	if err != nil {
		return fmt.Errorf("list existing providers: %w", err)
	}

	existingNames := make(map[string]bool, len(existing))
	for _, p := range existing {
		existingNames[p.Name] = true
	}

	for _, sp := range seedData {
		if existingNames[sp.Name] {
			slog.DebugContext(ctx, "seed: provider already exists, skipping",
				slog.String("name", sp.Name),
			)
			continue
		}

		provider, err := queries.CreateProvider(ctx, repository.CreateProviderParams{
			OwnerID:     userID,
			Name:        sp.Name,
			BaseUrl:     sp.BaseURL,
			Description: "",
			Status:      repository.ProviderStatusActive,
		})
		if err != nil {
			return fmt.Errorf("create provider %s: %w", sp.Name, err)
		}

		for _, ep := range sp.Endpoints {
			var price pgtype.Numeric
			if err := price.Scan(ep.PriceAmount); err != nil {
				return fmt.Errorf("parse price %s: %w", ep.PriceAmount, err)
			}

			var rateLimit pgtype.Int4
			if ep.RateLimit > 0 {
				rateLimit = pgtype.Int4{Int32: int32(ep.RateLimit), Valid: true} // #nosec G115 — seed data rate limits fit in int32
			}

			if _, err = queries.CreateEndpoint(ctx, repository.CreateEndpointParams{
				ProviderID:  provider.ID,
				Route:       ep.Route,
				Method:      ep.Method,
				PriceAmount: price,
				Currency:    repository.CurrencyXLM,
				RateLimit:   rateLimit,
				Status:      repository.EndpointStatusActive,
			}); err != nil {
				return fmt.Errorf("create endpoint %s %s: %w", ep.Method, ep.Route, err)
			}
		}

		slog.InfoContext(ctx, "seed: created provider with endpoints",
			slog.String("name", sp.Name),
			slog.Int("endpoints", len(sp.Endpoints)),
		)
	}

	return nil
}
