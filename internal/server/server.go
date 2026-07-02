package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	_ "github.com/joho/godotenv/autoload"
	"github.com/shopspring/decimal"

	"castellan/internal/accounts"
	"castellan/internal/auth"
	"castellan/internal/database"
	"castellan/internal/gateway"
	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/ledger"
	"castellan/internal/provider"
	"castellan/internal/proxy"
	"castellan/internal/repository/db"
	"castellan/internal/server/middleware"
)

// Server wires together database, resolver, proxy, ledger, and middleware dependencies.
type Server struct {
	port int

	db               database.Service
	proxy            *proxy.Proxy
	balance          middleware.BalanceChecker
	pricingResolver  middleware.EndpointPricingResolver
	usageRepo        middleware.UsageEventRepository
	ledger           gateway.LedgerService
	keyHandler       *auth.KeyHandler
	keyValidator     middleware.KeyValidator
	sessionValidator middleware.SessionValidator
	sessionHandler   *auth.SessionHandler
	providerHandler  *provider.Handler
	endpointHandler  *provider.EndpointHandler
	accountHandler   *accounts.Handler
}

// NewServer creates an http.Server with all dependencies wired: database pool,
// sqlc queries, provider resolver, reverse proxy, and the full middleware chain.
func NewServer() (*http.Server, error) {
	port, _ := strconv.Atoi(os.Getenv("PORT"))

	databaseService, err := database.New()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	queries := repository.New(databaseService.Pool())
	resolver, err := provider.NewDBResolver(queries)
	if err != nil {
		if closeErr := databaseService.Close(); closeErr != nil {
			slog.Warn(
				"failed to close database after resolver error",
				slog.String("error", closeErr.Error()),
			)
		}

		return nil, fmt.Errorf("failed to create provider resolver: %w", err)
	}

	balancer := middleware.BalanceCheckerFunc(func(ctx context.Context, ownerID uuid.UUID) (decimal.Decimal, error) {
		if _, err := queries.GetOrCreateAccount(ctx, ownerID); err != nil {
			return decimal.Zero, fmt.Errorf("get or create account: %w", err)
		}

		balance, err := queries.GetAccountBalance(ctx, ownerID)
		if err != nil {
			return decimal.Zero, fmt.Errorf("balance unavailable: %w", err)
		}

		f64, err := balance.Float64Value()
		if err != nil {
			return decimal.Zero, fmt.Errorf("failed to convert balance: %w", err)
		}
		if !f64.Valid {
			return decimal.Zero, errors.New("balance is null")
		}

		return decimal.NewFromFloat(f64.Float64), nil
	})

	usageRepo := middleware.UsageEventRepositoryFunc(func(ctx context.Context, arg repository.CreateUsageEventParams) (repository.CreateUsageEventRow, error) {
		return queries.CreateUsageEvent(ctx, arg)
	})

	pricingResolver := middleware.EndpointPricingResolverFunc(func(ctx context.Context, providerID uuid.UUID, route, method string) (*gatewaycontext.PricingInfo, error) {
		endpoint, err := queries.GetEndpointByProviderRouteMethod(ctx, repository.GetEndpointByProviderRouteMethodParams{
			ProviderID: providerID,
			Route:      route,
			Method:     method,
		})
		if err != nil {
			return nil, fmt.Errorf("resolve endpoint pricing: %w", err)
		}

		priceAmount, err := numericToDecimal(endpoint.PriceAmount)
		if err != nil {
			return nil, fmt.Errorf("invalid price amount: %w", err)
		}

		return &gatewaycontext.PricingInfo{
			EndpointID:  endpoint.ID.String(),
			ProviderID:  endpoint.ProviderID.String(),
			PriceAmount: priceAmount,
			Currency:    gatewaycontext.Currency(endpoint.Currency),
		}, nil
	})

	pxy := proxy.NewReverseProxy(resolver, slog.Default(), proxy.ConfigFromEnv())

	keySvc := auth.NewKeyService(queries)

	keyValidator := middleware.KeyValidatorFunc(func(ctx context.Context, keyHash string) (repository.ApiKey, error) {
		return queries.GetKeyByHash(ctx, keyHash)
	})

	sessionSvc := auth.NewSessionService(queries)

	sessionValidator := middleware.SessionValidatorFunc(func(ctx context.Context, rawToken string) (*repository.SessionToken, error) {
		return sessionSvc.ValidateSessionToken(ctx, rawToken)
	})

	providerSvc := provider.NewProviderService(queries)
	endpointSvc := provider.NewEndpointService(queries)
	accountSvc := accounts.NewService(queries)

	srv := &Server{
		port:             port,
		db:               databaseService,
		proxy:            pxy,
		balance:          balancer,
		pricingResolver:  pricingResolver,
		usageRepo:        usageRepo,
		ledger:           ledger.NewPostgresLedger(databaseService.Pool()),
		keyHandler:       auth.NewKeyHandler(keySvc),
		keyValidator:     keyValidator,
		sessionValidator: sessionValidator,
		sessionHandler:   auth.NewSessionHandler(sessionSvc),
		providerHandler:  provider.NewProviderHandler(providerSvc),
		endpointHandler:  provider.NewEndpointHandler(endpointSvc),
		accountHandler:   accounts.NewHandler(accountSvc),
	}

	httpServer := &http.Server{
		Addr:         net.JoinHostPort("", strconv.Itoa(srv.port)),
		Handler:      srv.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return httpServer, nil
}

func numericToDecimal(n pgtype.Numeric) (decimal.Decimal, error) {
	if !n.Valid {
		return decimal.Zero, nil
	}
	if n.NaN {
		return decimal.Zero, errors.New("numeric is NaN")
	}
	return decimal.NewFromBigInt(n.Int, n.Exp), nil
}
