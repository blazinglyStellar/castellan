package provider

import (
	"context"
	"errors"
	"fmt"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
)

// DBResolver resolves a provider's upstream base URL from the PostgreSQL database.
type DBResolver struct {
	queries *repository.Queries
}

// Resolver abstracts provider base URL resolution (for testability).
type Resolver interface {
	ResolveBaseURL(ctx context.Context, id string) (string, error)
}

// NewDBResolver returns a DBResolver backed by the given sqlc Queries.
func NewDBResolver(queries *repository.Queries) (*DBResolver, error) {
	if queries == nil {
		return nil, errors.New("queries cannot be nil")
	}
	return &DBResolver{queries: queries}, nil
}

// ResolveBaseURL looks up the provider's base URL by UUID, filtering for active status.
func (r *DBResolver) ResolveBaseURL(ctx context.Context, id string) (string, error) {
	providerUUID, err := uuid.Parse(id)
	if err != nil {
		return "", fmt.Errorf("failed to parse uuid: %s", err.Error())
	}

	baseURL, err := r.queries.GetProviderBaseURL(ctx, providerUUID)
	if err != nil {
		return "", fmt.Errorf("failed to resolve provider: %w", err)
	}

	return baseURL, nil
}
