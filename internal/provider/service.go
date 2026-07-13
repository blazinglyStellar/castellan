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

var ErrDuplicateEndpoint = errors.New("endpoint with this route and method already exists on this provider")

var ErrEndpointNotFound = errors.New("endpoint not found")

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

func (s *ProviderService) CreateProvider(ctx context.Context, ownerID uuid.UUID, name, baseURL, description string) (repository.Provider, error) {
	if err := validateName(name); err != nil {
		return repository.Provider{}, err
	}
	if err := validateBaseURL(baseURL); err != nil {
		return repository.Provider{}, err
	}

	params := repository.CreateProviderParams{
		OwnerID:     ownerID,
		Name:        name,
		BaseUrl:     baseURL,
		Description: description,
		Status:      repository.ProviderStatusActive,
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

func (s *ProviderService) ListProviders(ctx context.Context, ownerID uuid.UUID) ([]OwnerProvider, error) {
	providers, err := s.queries.ListProvidersByOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}

	stats, err := s.queries.GetProviderStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get provider stats: %w", err)
	}

	statsByID := make(map[uuid.UUID]repository.GetProviderStatsRow, len(stats))
	for _, st := range stats {
		statsByID[st.ID] = st
	}

	result := make([]OwnerProvider, len(providers))
	for i, p := range providers {
		result[i] = OwnerProvider{Provider: p}
		if st, ok := statsByID[p.ID]; ok {
			result[i].EndpointCount = st.EndpointCount
			result[i].TotalCalls = st.TotalCalls
			result[i].ActiveConsumers = st.ActiveConsumers
		}
	}

	return result, nil
}

type OwnerProvider struct {
	repository.Provider
	EndpointCount   int64 `json:"endpoint_count"`
	TotalCalls      int64 `json:"total_calls"`
	ActiveConsumers int64 `json:"active_consumers"`
}

type PublicProvider struct {
	repository.Provider
	TotalCalls      int64 `json:"total_calls"`
	ActiveConsumers int64 `json:"active_consumers"`
}

func (s *ProviderService) ListPublicProviders(ctx context.Context) ([]PublicProvider, error) {
	providers, err := s.queries.ListAllProviders(ctx, repository.NullProviderStatus{
		ProviderStatus: repository.ProviderStatusActive,
		Valid:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("list public providers: %w", err)
	}

	stats, err := s.queries.GetProviderStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("get provider stats: %w", err)
	}

	statsByID := make(map[uuid.UUID]repository.GetProviderStatsRow, len(stats))
	for _, st := range stats {
		statsByID[st.ID] = st
	}

	result := make([]PublicProvider, len(providers))
	for i, p := range providers {
		result[i] = PublicProvider{Provider: p}
		if st, ok := statsByID[p.ID]; ok {
			result[i].TotalCalls = st.TotalCalls
			result[i].ActiveConsumers = st.ActiveConsumers
		}
	}

	return result, nil
}

type partialUpdateProviderInput struct {
	Name        *string
	BaseURL     *string
	Description *string
}

func (s *ProviderService) PartialUpdateProvider(ctx context.Context, providerID, ownerID uuid.UUID, input partialUpdateProviderInput) (repository.Provider, error) {
	current, err := s.GetProviderByID(ctx, providerID, ownerID)
	if err != nil {
		return repository.Provider{}, err
	}

	resolvedName := current.Name
	if input.Name != nil {
		resolvedName = *input.Name
	}
	resolvedBaseURL := current.BaseUrl
	if input.BaseURL != nil {
		resolvedBaseURL = *input.BaseURL
	}
	resolvedDescription := current.Description
	if input.Description != nil {
		resolvedDescription = *input.Description
	}

	return s.UpdateProvider(ctx, providerID, ownerID, resolvedName, resolvedBaseURL, resolvedDescription)
}

func (s *ProviderService) UpdateProvider(ctx context.Context, providerID, ownerID uuid.UUID, name, baseURL, description string) (repository.Provider, error) {
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
		ID:          providerID,
		Name:        name,
		BaseUrl:     baseURL,
		Description: description,
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
	Description string
}

type UpdateEndpointInput struct {
	OwnerID     uuid.UUID
	EndpointID  uuid.UUID
	Route       string
	Method      string
	PriceAmount pgtype.Numeric
	Currency    repository.Currency
	RateLimit   pgtype.Int4
	Description string
}

type PartialUpdateEndpointInput struct {
	Route       *string
	Method      *string
	PriceAmount *pgtype.Numeric
	Currency    *repository.Currency
	RateLimit   *pgtype.Int4
	Description *string
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
		return repository.ApiEndpoint{}, fmt.Errorf("%w: %w", ErrEndpointNotFound, err)
	}
	if provider.OwnerID != input.OwnerID {
		return repository.ApiEndpoint{}, ErrEndpointNotFound
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

	dupParams := repository.GetEndpointByProviderRouteMethodParams{
		ProviderID: input.ProviderID,
		Route:      input.Route,
		Method:     input.Method,
	}
	if _, err := s.queries.GetEndpointByProviderRouteMethod(ctx, dupParams); err == nil {
		return repository.ApiEndpoint{}, ErrDuplicateEndpoint
	}

	params := repository.CreateEndpointParams{
		ProviderID:  input.ProviderID,
		Route:       input.Route,
		Method:      input.Method,
		PriceAmount: input.PriceAmount,
		Currency:    input.Currency,
		RateLimit:   input.RateLimit,
		Status:      input.Status,
		Description: input.Description,
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

func (s *EndpointService) ListEndpoints(ctx context.Context, providerID, ownerID uuid.UUID, statusFilter *repository.EndpointStatus) ([]repository.ApiEndpoint, error) {
	provider, err := s.queries.GetProviderByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEndpointNotFound, err)
	}
	if provider.OwnerID != ownerID {
		return nil, ErrEndpointNotFound
	}

	var status repository.NullEndpointStatus
	if statusFilter != nil {
		status = repository.NullEndpointStatus{
			EndpointStatus: *statusFilter,
			Valid:          true,
		}
	}
	params := repository.ListEndpointsByProviderParams{
		ProviderID: providerID,
		Status:     status,
	}
	endpoints, err := s.queries.ListEndpointsByProvider(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list endpoints: %w", err)
	}
	return endpoints, nil
}

func (s *EndpointService) ListPublicEndpoints(ctx context.Context, providerID uuid.UUID) ([]repository.ApiEndpoint, error) {
	provider, err := s.queries.GetProviderByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEndpointNotFound, err)
	}
	if provider.Status != repository.ProviderStatusActive {
		return nil, errors.New("provider not found")
	}

	params := repository.ListEndpointsByProviderParams{
		ProviderID: providerID,
		Status: repository.NullEndpointStatus{
			EndpointStatus: repository.EndpointStatusActive,
			Valid:          true,
		},
	}
	endpoints, err := s.queries.ListEndpointsByProvider(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("list public endpoints: %w", err)
	}
	return endpoints, nil
}

func (s *EndpointService) UpdateEndpoint(
	ctx context.Context,
	input UpdateEndpointInput,
) (repository.ApiEndpoint, error) {
	current, err := s.GetEndpointByID(ctx, input.EndpointID, input.OwnerID)
	if err != nil {
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

	dupParams := repository.GetEndpointByProviderRouteMethodParams{
		ProviderID: current.ProviderID,
		Route:      input.Route,
		Method:     input.Method,
	}
	if existing, err := s.queries.GetEndpointByProviderRouteMethod(ctx, dupParams); err == nil && existing.ID != input.EndpointID {
		return repository.ApiEndpoint{}, ErrDuplicateEndpoint
	}

	params := repository.UpdateEndpointParams{
		ID:          input.EndpointID,
		Route:       input.Route,
		Method:      input.Method,
		PriceAmount: input.PriceAmount,
		Currency:    input.Currency,
		RateLimit:   input.RateLimit,
		Description: input.Description,
	}

	endpoint, err := s.queries.UpdateEndpoint(ctx, params)
	if err != nil {
		return repository.ApiEndpoint{}, fmt.Errorf("update endpoint: %w", err)
	}
	return endpoint, nil
}

func (s *EndpointService) PartialUpdateEndpoint(
	ctx context.Context,
	endpointID, ownerID uuid.UUID,
	input PartialUpdateEndpointInput,
) (repository.ApiEndpoint, error) {
	current, err := s.GetEndpointByID(ctx, endpointID, ownerID)
	if err != nil {
		return repository.ApiEndpoint{}, err
	}

	resolvedRoute := current.Route
	if input.Route != nil {
		resolvedRoute = *input.Route
	}
	resolvedMethod := current.Method
	if input.Method != nil {
		resolvedMethod = *input.Method
	}
	resolvedPrice := current.PriceAmount
	if input.PriceAmount != nil {
		resolvedPrice = *input.PriceAmount
	}
	resolvedCurrency := current.Currency
	if input.Currency != nil {
		resolvedCurrency = *input.Currency
	}
	resolvedRateLimit := current.RateLimit
	if input.RateLimit != nil {
		resolvedRateLimit = *input.RateLimit
	}
	resolvedDescription := current.Description
	if input.Description != nil {
		resolvedDescription = *input.Description
	}

	fullInput := UpdateEndpointInput{
		OwnerID:     ownerID,
		EndpointID:  endpointID,
		Route:       resolvedRoute,
		Method:      resolvedMethod,
		PriceAmount: resolvedPrice,
		Currency:    resolvedCurrency,
		RateLimit:   resolvedRateLimit,
		Description: resolvedDescription,
	}

	return s.UpdateEndpoint(ctx, fullInput)
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
