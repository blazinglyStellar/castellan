//go:build integration

package provider

import (
	"context"
	"testing"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
)

func TestDBResolver_ResolveBaseURL_ActiveProvider(t *testing.T) {
	queries := repository.New(testPool)
	resolver, err := NewDBResolver(queries)
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}
	ctx := context.Background()

	userID := uuid.New()
	_, err = testPool.Exec(ctx, `INSERT INTO users (id, email) VALUES ($1, $2)`, userID, "active-test@example.com")
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	providerID := uuid.New()
	expectedURL := "https://api.active-provider.com"
	_, err = testPool.Exec(ctx,
		`INSERT INTO providers (id, owner_id, name, base_url, status) VALUES ($1, $2, $3, $4, 'active')`,
		providerID, userID, "active-provider", expectedURL)
	if err != nil {
		t.Fatalf("failed to seed provider: %v", err)
	}

	got, err := resolver.ResolveBaseURL(ctx, providerID.String())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expectedURL {
		t.Fatalf("expected %q, got %q", expectedURL, got)
	}
}

func TestDBResolver_ResolveBaseURL_InactiveProvider(t *testing.T) {
	queries := repository.New(testPool)
	resolver, err := NewDBResolver(queries)
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}
	ctx := context.Background()

	userID := uuid.New()
	_, err = testPool.Exec(ctx, `INSERT INTO users (id, email) VALUES ($1, $2)`, userID, "inactive-test@example.com")
	if err != nil {
		t.Fatalf("failed to seed user: %v", err)
	}

	providerID := uuid.New()
	_, err = testPool.Exec(ctx,
		`INSERT INTO providers (id, owner_id, name, base_url, status) VALUES ($1, $2, $3, $4, 'inactive')`,
		providerID, userID, "inactive-provider", "https://api.inactive-provider.com")
	if err != nil {
		t.Fatalf("failed to seed provider: %v", err)
	}

	_, err = resolver.ResolveBaseURL(ctx, providerID.String())
	if err == nil {
		t.Fatal("expected error for inactive provider, got nil")
	}
}
