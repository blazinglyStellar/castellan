package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	maxNameLen  = 255
	maxRouteLen = 2048
)

var validMethods = map[string]bool{
	"GET": true, "POST": true, "PUT": true,
	"PATCH": true, "DELETE": true, "HEAD": true, "OPTIONS": true,
}

func validateName(name string) error {
	if len(strings.TrimSpace(name)) == 0 {
		return errors.New("name is required")
	}
	if len(name) > maxNameLen {
		return errors.New("name must be at most 255 characters")
	}
	return nil
}

func validateBaseURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return errors.New("base_url must be a valid URL")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("base_url scheme must be http or https")
	}
	return nil
}

func validateRoute(route string) error {
	if len(strings.TrimSpace(route)) == 0 {
		return errors.New("route is required")
	}
	if len(route) > maxRouteLen {
		return errors.New("route must be at most 2048 characters")
	}
	if !strings.HasPrefix(route, "/") {
		return errors.New("route must start with /")
	}
	return nil
}

func validateMethod(method string) error {
	if !validMethods[method] {
		return fmt.Errorf("invalid HTTP method: %s", method)
	}
	return nil
}

func validatePriceAmount(amount pgtype.Numeric) error {
	if !amount.Valid {
		return errors.New("price amount is required")
	}
	if amount.NaN || amount.InfinityModifier != pgtype.Finite {
		return errors.New("price amount must be a finite number")
	}
	if amount.Int != nil && amount.Int.Sign() < 0 {
		return errors.New("price amount must be non-negative")
	}
	return nil
}

func validateCurrency(currency repository.Currency) error {
	if currency != repository.CurrencyXLM && currency != repository.CurrencyUSDC {
		return errors.New("currency must be XLM or USDC")
	}
	return nil
}

func validateRateLimit(rateLimit pgtype.Int4) error {
	if rateLimit.Valid && rateLimit.Int32 <= 0 {
		return errors.New("rate limit must be a positive integer")
	}
	return nil
}

func validateProviderStatus(status repository.ProviderStatus) error {
	switch status {
	case repository.ProviderStatusActive, repository.ProviderStatusInactive, repository.ProviderStatusSuspended:
		return nil
	default:
		return fmt.Errorf("invalid provider status: %s", string(status))
	}
}

func validateEndpointStatus(status repository.EndpointStatus) error {
	switch status {
	case repository.EndpointStatusActive, repository.EndpointStatusInactive, repository.EndpointStatusDraft:
		return nil
	default:
		return fmt.Errorf("invalid endpoint status: %s", string(status))
	}
}

//nolint:revive // provider.ProviderService is clear in context
type ProviderService struct {
	queries repository.Querier
}

func NewProviderService(queries repository.Querier) *ProviderService {
	return &ProviderService{queries: queries}
}

func (s *ProviderService) CreateProvider(ctx context.Context, ownerID uuid.UUID, name, baseURL string) (repository.Provider, error) {
	if err := validateName(name); err != nil {
		return repository.Provider{}, err
	}
	if err := validateBaseURL(baseURL); err != nil {
		return repository.Provider{}, err
	}

	params := repository.CreateProviderParams{
		OwnerID: ownerID,
		Name:    name,
		BaseUrl: baseURL,
		Status:  repository.ProviderStatusActive,
	}

	provider, err := s.queries.CreateProvider(ctx, params)
	if err != nil {
		return repository.Provider{}, fmt.Errorf("create provider: %w", err)
	}
	return provider, nil
}

func (s *ProviderService) GetProviderByID(ctx context.Context, providerID, ownerID uuid.UUID) (repository.Provider, error) {
	provider, err := s.queries.GetProviderByID(ctx, providerID)
	if err != nil {
		return repository.Provider{}, fmt.Errorf("provider not found: %w", err)
	}
	if provider.OwnerID != ownerID {
		return repository.Provider{}, errors.New("provider not found")
	}
	return provider, nil
}

func (s *ProviderService) ListProviders(ctx context.Context, ownerID uuid.UUID) ([]repository.Provider, error) {
	providers, err := s.queries.ListProvidersByOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	return providers, nil
}

func (s *ProviderService) PartialUpdateProvider(ctx context.Context, providerID, ownerID uuid.UUID, name, baseURL *string) (repository.Provider, error) {
	current, err := s.GetProviderByID(ctx, providerID, ownerID)
	if err != nil {
		return repository.Provider{}, err
	}

	resolvedName := current.Name
	if name != nil {
		resolvedName = *name
	}
	resolvedBaseURL := current.BaseUrl
	if baseURL != nil {
		resolvedBaseURL = *baseURL
	}

	return s.UpdateProvider(ctx, providerID, ownerID, resolvedName, resolvedBaseURL)
}

func (s *ProviderService) UpdateProvider(ctx context.Context, providerID, ownerID uuid.UUID, name, baseURL string) (repository.Provider, error) {
	if _, err := s.GetProviderByID(ctx, providerID, ownerID); err != nil {
		return repository.Provider{}, err
	}
	if err := validateName(name); err != nil {
		return repository.Provider{}, err
	}
	if err := validateBaseURL(baseURL); err != nil {
		return repository.Provider{}, err
	}

	params := repository.UpdateProviderParams{
		ID:      providerID,
		Name:    name,
		BaseUrl: baseURL,
	}

	provider, err := s.queries.UpdateProvider(ctx, params)
	if err != nil {
		return repository.Provider{}, fmt.Errorf("update provider: %w", err)
	}
	return provider, nil
}

func (s *ProviderService) UpdateProviderStatus(ctx context.Context, providerID, ownerID uuid.UUID, status repository.ProviderStatus) (repository.Provider, error) {
	if _, err := s.GetProviderByID(ctx, providerID, ownerID); err != nil {
		return repository.Provider{}, err
	}
	if err := validateProviderStatus(status); err != nil {
		return repository.Provider{}, err
	}

	params := repository.UpdateProviderStatusParams{
		ID:     providerID,
		Status: status,
	}

	provider, err := s.queries.UpdateProviderStatus(ctx, params)
	if err != nil {
		return repository.Provider{}, fmt.Errorf("update provider status: %w", err)
	}
	return provider, nil
}

func (s *ProviderService) DeleteProvider(ctx context.Context, providerID, ownerID uuid.UUID) (repository.Provider, error) {
	if _, err := s.GetProviderByID(ctx, providerID, ownerID); err != nil {
		return repository.Provider{}, err
	}

	provider, err := s.queries.DeleteProvider(ctx, providerID)
	if err != nil {
		return repository.Provider{}, fmt.Errorf("delete provider: %w", err)
	}
	return provider, nil
}

type CreateEndpointInput struct {
	OwnerID     uuid.UUID
	ProviderID  uuid.UUID
	Route       string
	Method      string
	PriceAmount pgtype.Numeric
	Currency    repository.Currency
	RateLimit   pgtype.Int4
	Status      repository.EndpointStatus
}

type UpdateEndpointInput struct {
	OwnerID     uuid.UUID
	EndpointID  uuid.UUID
	Route       string
	Method      string
	PriceAmount pgtype.Numeric
	Currency    repository.Currency
	RateLimit   pgtype.Int4
}

type EndpointService struct {
	queries repository.Querier
}

func NewEndpointService(queries repository.Querier) *EndpointService {
	return &EndpointService{queries: queries}
}

func (s *EndpointService) CreateEndpoint(
	ctx context.Context,
	input CreateEndpointInput,
) (repository.ApiEndpoint, error) {
	provider, err := s.queries.GetProviderByID(ctx, input.ProviderID)
	if err != nil {
		return repository.ApiEndpoint{}, fmt.Errorf("endpoint not found: %w", err)
	}
	if provider.OwnerID != input.OwnerID {
		return repository.ApiEndpoint{}, errors.New("endpoint not found")
	}
	if err := validateRoute(input.Route); err != nil {
		return repository.ApiEndpoint{}, err
	}
	if err := validateMethod(input.Method); err != nil {
		return repository.ApiEndpoint{}, err
	}
	if err := validatePriceAmount(input.PriceAmount); err != nil {
		return repository.ApiEndpoint{}, err
	}
	if err := validateCurrency(input.Currency); err != nil {
		return repository.ApiEndpoint{}, err
	}
	if err := validateRateLimit(input.RateLimit); err != nil {
		return repository.ApiEndpoint{}, err
	}
	if err := validateEndpointStatus(input.Status); err != nil {
		return repository.ApiEndpoint{}, err
	}

	params := repository.CreateEndpointParams{
		ProviderID:  input.ProviderID,
		Route:       input.Route,
		Method:      input.Method,
		PriceAmount: input.PriceAmount,
		Currency:    input.Currency,
		RateLimit:   input.RateLimit,
		Status:      input.Status,
	}

	endpoint, err := s.queries.CreateEndpoint(ctx, params)
	if err != nil {
		return repository.ApiEndpoint{}, fmt.Errorf("create endpoint: %w", err)
	}
	return endpoint, nil
}

func (s *EndpointService) GetEndpointByID(ctx context.Context, endpointID, ownerID uuid.UUID) (repository.ApiEndpoint, error) {
	endpoint, err := s.queries.GetEndpointByID(ctx, endpointID)
	if err != nil {
		return repository.ApiEndpoint{}, fmt.Errorf("endpoint not found: %w", err)
	}
	provider, err := s.queries.GetProviderByID(ctx, endpoint.ProviderID)
	if err != nil {
		return repository.ApiEndpoint{}, fmt.Errorf("endpoint not found: %w", err)
	}
	if provider.OwnerID != ownerID {
		return repository.ApiEndpoint{}, errors.New("endpoint not found")
	}
	return endpoint, nil
}

func (s *EndpointService) ListEndpoints(ctx context.Context, providerID, ownerID uuid.UUID) ([]repository.ApiEndpoint, error) {
	provider, err := s.queries.GetProviderByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("endpoint not found: %w", err)
	}
	if provider.OwnerID != ownerID {
		return nil, errors.New("endpoint not found")
	}

	params := repository.ListEndpointsByProviderParams{
		ProviderID: providerID,
		Status:     nil,
	}
	endpoints, err := s.queries.ListEndpointsByProvider(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list endpoints: %w", err)
	}
	return endpoints, nil
}

func (s *EndpointService) UpdateEndpoint(
	ctx context.Context,
	input UpdateEndpointInput,
) (repository.ApiEndpoint, error) {
	if _, err := s.GetEndpointByID(ctx, input.EndpointID, input.OwnerID); err != nil {
		return repository.ApiEndpoint{}, err
	}
	if err := validateRoute(input.Route); err != nil {
		return repository.ApiEndpoint{}, err
	}
	if err := validateMethod(input.Method); err != nil {
		return repository.ApiEndpoint{}, err
	}
	if err := validatePriceAmount(input.PriceAmount); err != nil {
		return repository.ApiEndpoint{}, err
	}
	if err := validateCurrency(input.Currency); err != nil {
		return repository.ApiEndpoint{}, err
	}
	if err := validateRateLimit(input.RateLimit); err != nil {
		return repository.ApiEndpoint{}, err
	}

	params := repository.UpdateEndpointParams{
		ID:          input.EndpointID,
		Route:       input.Route,
		Method:      input.Method,
		PriceAmount: input.PriceAmount,
		Currency:    input.Currency,
		RateLimit:   input.RateLimit,
	}

	endpoint, err := s.queries.UpdateEndpoint(ctx, params)
	if err != nil {
		return repository.ApiEndpoint{}, fmt.Errorf("update endpoint: %w", err)
	}
	return endpoint, nil
}

func (s *EndpointService) UpdateEndpointStatus(ctx context.Context, endpointID, ownerID uuid.UUID, status repository.EndpointStatus) (repository.ApiEndpoint, error) {
	if _, err := s.GetEndpointByID(ctx, endpointID, ownerID); err != nil {
		return repository.ApiEndpoint{}, err
	}
	if err := validateEndpointStatus(status); err != nil {
		return repository.ApiEndpoint{}, err
	}

	params := repository.UpdateEndpointStatusParams{
		ID:     endpointID,
		Status: status,
	}

	endpoint, err := s.queries.UpdateEndpointStatus(ctx, params)
	if err != nil {
		return repository.ApiEndpoint{}, fmt.Errorf("update endpoint status: %w", err)
	}
	return endpoint, nil
}

func (s *EndpointService) DeleteEndpoint(ctx context.Context, endpointID, ownerID uuid.UUID) (repository.ApiEndpoint, error) {
	if _, err := s.GetEndpointByID(ctx, endpointID, ownerID); err != nil {
		return repository.ApiEndpoint{}, err
	}

	endpoint, err := s.queries.DeleteEndpoint(ctx, endpointID)
	if err != nil {
		return repository.ApiEndpoint{}, fmt.Errorf("delete endpoint: %w", err)
	}
	return endpoint, nil
}
