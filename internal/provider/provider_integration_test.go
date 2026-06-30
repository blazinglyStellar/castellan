//go:build integration

package provider

import (
	"context"
	"testing"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func createTestUser(t *testing.T, ctx context.Context, email string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := testPool.Exec(ctx, `INSERT INTO users (id, email) VALUES ($1, $2)`, id, email)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return id
}

func TestProviderLifecycle(t *testing.T) {
	ctx := context.Background()
	queries := repository.New(testPool)
	ps := NewProviderService(queries)

	ownerID := createTestUser(t, ctx, "lifecycle@test.com")

	provider, err := ps.CreateProvider(ctx, ownerID, "Test Provider", "https://api.test.com")
	if err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}
	if provider.ID == uuid.Nil {
		t.Fatal("expected non-nil provider ID")
	}
	if provider.Name != "Test Provider" {
		t.Errorf("expected name %q, got %q", "Test Provider", provider.Name)
	}
	if provider.Status != repository.ProviderStatusActive {
		t.Errorf("expected status %q, got %q", repository.ProviderStatusActive, provider.Status)
	}

	got, err := ps.GetProviderByID(ctx, provider.ID, ownerID)
	if err != nil {
		t.Fatalf("GetProviderByID failed: %v", err)
	}
	if got.ID != provider.ID {
		t.Errorf("expected provider ID %v, got %v", provider.ID, got.ID)
	}

	list, err := ps.ListProviders(ctx, ownerID)
	if err != nil {
		t.Fatalf("ListProviders failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(list))
	}

	updated, err := ps.UpdateProvider(ctx, provider.ID, ownerID, "Updated Provider", "https://api.updated.com")
	if err != nil {
		t.Fatalf("UpdateProvider failed: %v", err)
	}
	if updated.Name != "Updated Provider" {
		t.Errorf("expected name %q, got %q", "Updated Provider", updated.Name)
	}

	statusChanged, err := ps.UpdateProviderStatus(ctx, provider.ID, ownerID, repository.ProviderStatusInactive)
	if err != nil {
		t.Fatalf("UpdateProviderStatus failed: %v", err)
	}
	if statusChanged.Status != repository.ProviderStatusInactive {
		t.Errorf("expected status %q, got %q", repository.ProviderStatusInactive, statusChanged.Status)
	}

	deleted, err := ps.DeleteProvider(ctx, provider.ID, ownerID)
	if err != nil {
		t.Fatalf("DeleteProvider failed: %v", err)
	}
	if deleted.ID != provider.ID {
		t.Errorf("expected deleted provider ID %v, got %v", provider.ID, deleted.ID)
	}

	_, err = ps.GetProviderByID(ctx, provider.ID, ownerID)
	if err == nil {
		t.Fatal("expected error getting deleted provider")
	}
}

func TestProviderCascadeDeleteEndpoints(t *testing.T) {
	ctx := context.Background()
	queries := repository.New(testPool)
	ps := NewProviderService(queries)
	es := NewEndpointService(queries)

	ownerID := createTestUser(t, ctx, "cascade@test.com")

	provider, err := ps.CreateProvider(ctx, ownerID, "Cascade Provider", "https://api.cascade.com")
	if err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}

	ep1, err := es.CreateEndpoint(ctx, CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  provider.ID,
		Route:       "/ep1",
		Method:      "GET",
		PriceAmount: numericFromInt64(10),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatalf("CreateEndpoint ep1 failed: %v", err)
	}

	ep2, err := es.CreateEndpoint(ctx, CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  provider.ID,
		Route:       "/ep2",
		Method:      "POST",
		PriceAmount: numericFromInt64(20),
		Currency:    repository.CurrencyUSDC,
		Status:      repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatalf("CreateEndpoint ep2 failed: %v", err)
	}

	_, err = ps.DeleteProvider(ctx, provider.ID, ownerID)
	if err != nil {
		t.Fatalf("DeleteProvider failed: %v", err)
	}

	var count int
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM api_endpoints WHERE provider_id = $1`, provider.ID).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query endpoints: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 endpoints after provider delete, got %d", count)
	}

	_, err = es.GetEndpointByID(ctx, ep1.ID, ownerID)
	if err == nil {
		t.Error("expected error getting endpoint after provider delete")
	}
	_, err = es.GetEndpointByID(ctx, ep2.ID, ownerID)
	if err == nil {
		t.Error("expected error getting endpoint after provider delete")
	}
}

func TestProviderOwnershipIsolation(t *testing.T) {
	ctx := context.Background()
	queries := repository.New(testPool)
	ps := NewProviderService(queries)

	userA := createTestUser(t, ctx, "owner-a@test.com")
	userB := createTestUser(t, ctx, "owner-b@test.com")

	provider, err := ps.CreateProvider(ctx, userA, "A's Provider", "https://api.a.com")
	if err != nil {
		t.Fatalf("CreateProvider for user A failed: %v", err)
	}

	list, err := ps.ListProviders(ctx, userB)
	if err != nil {
		t.Fatalf("ListProviders for user B failed: %v", err)
	}
	if len(list) != 0 {
		t.Errorf("expected 0 providers for user B, got %d", len(list))
	}

	_, err = ps.GetProviderByID(ctx, provider.ID, userB)
	if err == nil {
		t.Error("expected error for user B getting A's provider")
	}

	_, err = ps.UpdateProvider(ctx, provider.ID, userB, "Hacked", "https://evil.com")
	if err == nil {
		t.Error("expected error for user B updating A's provider")
	}

	_, err = ps.DeleteProvider(ctx, provider.ID, userB)
	if err == nil {
		t.Error("expected error for user B deleting A's provider")
	}
}

func TestEndpointLifecycle(t *testing.T) {
	ctx := context.Background()
	queries := repository.New(testPool)
	ps := NewProviderService(queries)
	es := NewEndpointService(queries)

	ownerID := createTestUser(t, ctx, "ep-lifecycle@test.com")

	provider, err := ps.CreateProvider(ctx, ownerID, "EP Provider", "https://api.ep.com")
	if err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}

	endpoint, err := es.CreateEndpoint(ctx, CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  provider.ID,
		Route:       "/v1/test",
		Method:      "GET",
		PriceAmount: numericFromInt64(5),
		Currency:    repository.CurrencyXLM,
		RateLimit:   pgtype.Int4{Int32: 100, Valid: true},
		Status:      repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatalf("CreateEndpoint failed: %v", err)
	}
	if endpoint.ID == uuid.Nil {
		t.Fatal("expected non-nil endpoint ID")
	}

	got, err := es.GetEndpointByID(ctx, endpoint.ID, ownerID)
	if err != nil {
		t.Fatalf("GetEndpointByID failed: %v", err)
	}
	if got.ID != endpoint.ID {
		t.Errorf("expected endpoint ID %v, got %v", endpoint.ID, got.ID)
	}

	list, err := es.ListEndpoints(ctx, provider.ID, ownerID, nil)
	if err != nil {
		t.Fatalf("ListEndpoints failed: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 endpoint, got %d", len(list))
	}

	updated, err := es.UpdateEndpoint(ctx, UpdateEndpointInput{
		OwnerID:     ownerID,
		EndpointID:  endpoint.ID,
		Route:       "/v1/updated",
		Method:      "POST",
		PriceAmount: numericFromInt64(10),
		Currency:    repository.CurrencyUSDC,
		RateLimit:   pgtype.Int4{Int32: 50, Valid: true},
	})
	if err != nil {
		t.Fatalf("UpdateEndpoint failed: %v", err)
	}
	if updated.Route != "/v1/updated" {
		t.Errorf("expected route %q, got %q", "/v1/updated", updated.Route)
	}

	statusChanged, err := es.UpdateEndpointStatus(ctx, endpoint.ID, ownerID, repository.EndpointStatusInactive)
	if err != nil {
		t.Fatalf("UpdateEndpointStatus failed: %v", err)
	}
	if statusChanged.Status != repository.EndpointStatusInactive {
		t.Errorf("expected status %q, got %q", repository.EndpointStatusInactive, statusChanged.Status)
	}

	deleted, err := es.DeleteEndpoint(ctx, endpoint.ID, ownerID)
	if err != nil {
		t.Fatalf("DeleteEndpoint failed: %v", err)
	}
	if deleted.ID != endpoint.ID {
		t.Errorf("expected deleted endpoint ID %v, got %v", endpoint.ID, deleted.ID)
	}

	_, err = es.GetEndpointByID(ctx, endpoint.ID, ownerID)
	if err == nil {
		t.Fatal("expected error getting deleted endpoint")
	}
}

func TestEndpointUniqueness_SameProvider(t *testing.T) {
	ctx := context.Background()
	queries := repository.New(testPool)
	ps := NewProviderService(queries)
	es := NewEndpointService(queries)

	ownerID := createTestUser(t, ctx, "unique@test.com")

	provider, err := ps.CreateProvider(ctx, ownerID, "Unique Provider", "https://api.unique.com")
	if err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}

	_, err = es.CreateEndpoint(ctx, CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  provider.ID,
		Route:       "/same",
		Method:      "GET",
		PriceAmount: numericFromInt64(5),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatalf("first CreateEndpoint failed: %v", err)
	}

	_, err = es.CreateEndpoint(ctx, CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  provider.ID,
		Route:       "/same",
		Method:      "GET",
		PriceAmount: numericFromInt64(10),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	})
	if err == nil {
		t.Fatal("expected error for duplicate endpoint")
	}
}

func TestEndpointUniqueness_DifferentProvider(t *testing.T) {
	ctx := context.Background()
	queries := repository.New(testPool)
	ps := NewProviderService(queries)
	es := NewEndpointService(queries)

	ownerID := createTestUser(t, ctx, "cross@test.com")

	p1, err := ps.CreateProvider(ctx, ownerID, "Provider 1", "https://api.p1.com")
	if err != nil {
		t.Fatalf("CreateProvider 1 failed: %v", err)
	}

	p2, err := ps.CreateProvider(ctx, ownerID, "Provider 2", "https://api.p2.com")
	if err != nil {
		t.Fatalf("CreateProvider 2 failed: %v", err)
	}

	_, err = es.CreateEndpoint(ctx, CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  p1.ID,
		Route:       "/same",
		Method:      "GET",
		PriceAmount: numericFromInt64(5),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatalf("CreateEndpoint on p1 failed: %v", err)
	}

	_, err = es.CreateEndpoint(ctx, CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  p2.ID,
		Route:       "/same",
		Method:      "GET",
		PriceAmount: numericFromInt64(5),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatalf("CreateEndpoint on p2 should be allowed, got: %v", err)
	}
}

func TestEndpoint_UpdateToDuplicateRouteMethod(t *testing.T) {
	ctx := context.Background()
	queries := repository.New(testPool)
	ps := NewProviderService(queries)
	es := NewEndpointService(queries)

	ownerID := createTestUser(t, ctx, "update-conflict@test.com")

	provider, err := ps.CreateProvider(ctx, ownerID, "Conflict Provider", "https://api.conflict.com")
	if err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}

	ep1, err := es.CreateEndpoint(ctx, CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  provider.ID,
		Route:       "/ep1",
		Method:      "GET",
		PriceAmount: numericFromInt64(10),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatalf("CreateEndpoint ep1 failed: %v", err)
	}

	_, err = es.CreateEndpoint(ctx, CreateEndpointInput{
		OwnerID:     ownerID,
		ProviderID:  provider.ID,
		Route:       "/ep2",
		Method:      "GET",
		PriceAmount: numericFromInt64(20),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatalf("CreateEndpoint ep2 failed: %v", err)
	}

	_, err = es.UpdateEndpoint(ctx, UpdateEndpointInput{
		OwnerID:     ownerID,
		EndpointID:  ep1.ID,
		Route:       "/ep2",
		Method:      "GET",
		PriceAmount: numericFromInt64(99),
		Currency:    repository.CurrencyXLM,
	})
	if err == nil {
		t.Fatal("expected error for duplicate on update")
	}
}

func TestEndpointOwnershipEnforcement(t *testing.T) {
	ctx := context.Background()
	queries := repository.New(testPool)
	ps := NewProviderService(queries)
	es := NewEndpointService(queries)

	userA := createTestUser(t, ctx, "ep-owner-a@test.com")
	userB := createTestUser(t, ctx, "ep-owner-b@test.com")

	provider, err := ps.CreateProvider(ctx, userA, "A's Provider", "https://api.a.com")
	if err != nil {
		t.Fatalf("CreateProvider failed: %v", err)
	}

	ep, err := es.CreateEndpoint(ctx, CreateEndpointInput{
		OwnerID:     userA,
		ProviderID:  provider.ID,
		Route:       "/a-route",
		Method:      "GET",
		PriceAmount: numericFromInt64(5),
		Currency:    repository.CurrencyXLM,
		Status:      repository.EndpointStatusActive,
	})
	if err != nil {
		t.Fatalf("CreateEndpoint failed: %v", err)
	}

	_, err = es.GetEndpointByID(ctx, ep.ID, userB)
	if err == nil {
		t.Error("expected error for user B getting A's endpoint")
	}

	_, err = es.UpdateEndpoint(ctx, UpdateEndpointInput{
		OwnerID:     userB,
		EndpointID:  ep.ID,
		Route:       "/hacked",
		Method:      "GET",
		PriceAmount: numericFromInt64(5),
		Currency:    repository.CurrencyXLM,
	})
	if err == nil {
		t.Error("expected error for user B updating A's endpoint")
	}

	_, err = es.DeleteEndpoint(ctx, ep.ID, userB)
	if err == nil {
		t.Error("expected error for user B deleting A's endpoint")
	}
}
