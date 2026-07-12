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
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"

	"castellan/internal/accounts"
	"castellan/internal/auth"
	"castellan/internal/database"
	"castellan/internal/deposit"
	"castellan/internal/gateway"
	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/ledger"
	"castellan/internal/provider"
	"castellan/internal/proxy"
	"castellan/internal/repository/db"
	"castellan/internal/server/middleware"
	"castellan/internal/settlement"
	"castellan/internal/stellar"
	"castellan/internal/usage"
)

type Server struct {
	port int

	db                database.Service
	proxy             *proxy.Proxy
	balance           middleware.BalanceChecker
	pricingResolver   middleware.EndpointPricingResolver
	usageRepo         middleware.UsageEventRepository
	rateLimiter       gateway.RateLimiter
	redisClient       *redis.Client
	ledger            gateway.LedgerService
	keyHandler        *auth.KeyHandler
	keyValidator      middleware.KeyValidator
	sessionValidator  middleware.SessionValidator
	authHandler       *auth.Handler
	oauthHandler      *auth.OAuthHandler
	providerHandler   *provider.Handler
	endpointHandler   *provider.EndpointHandler
	accountHandler    *accounts.Handler
	depositHandler    *deposit.Handler
	creditHandler     *deposit.CreditHandler
	watcher           *deposit.Watcher
	settlementHandler *settlement.Handler
	usageHandler      *usage.Handler

	stellarConfig stellar.Config

	windowSeconds int
}

func NewServer() (*http.Server, error) {
	port, _ := strconv.Atoi(os.Getenv("PORT"))

	stellarCfg := stellar.ConfigFromEnv()
	if stellarCfg.HotWalletAddress == "" {
		return nil, errors.New("STELLAR_HOT_WALLET_ADDRESS is required")
	}

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

	windowSeconds := 60
	if ws := os.Getenv("RATE_LIMIT_WINDOW_SECONDS"); ws != "" {
		if v, err := strconv.Atoi(ws); err != nil {
			slog.Warn("invalid RATE_LIMIT_WINDOW_SECONDS, using default",
				slog.String("value", ws),
				slog.String("error", err.Error()),
			)
		} else if v <= 0 {
			slog.Warn("RATE_LIMIT_WINDOW_SECONDS must be positive, using default",
				slog.Int("value", v),
			)
		} else {
			windowSeconds = v
		}
	}

	rdb, err := connectRedis()
	if err != nil {
		if closeErr := databaseService.Close(); closeErr != nil {
			slog.Warn(
				"failed to close database after redis error",
				slog.String("error", closeErr.Error()),
			)
		}
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	rateLimiter := gateway.NewRedisRateLimiter(rdb, windowSeconds)

	balancer := middleware.BalanceCheckerFunc(func(ctx context.Context, ownerID uuid.UUID) (decimal.Decimal, error) {
		user, err := queries.GetUserByID(ctx, ownerID)
		if err != nil {
			return decimal.Zero, fmt.Errorf("get user: %w", err)
		}

		f64, err := user.Balance.Float64Value()
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

	pricingResolver := middleware.EndpointPricingResolverFunc(func(ctx context.Context, providerName string, route, method string) (*middleware.EndpointResolution, error) {
		provider, err := queries.GetProviderByName(ctx, providerName)
		if err != nil {
			return nil, fmt.Errorf("resolve provider by name: %w", err)
		}

		endpoint, err := queries.GetEndpointByProviderRouteMethod(ctx, repository.GetEndpointByProviderRouteMethodParams{
			ProviderID: provider.ID,
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

		rateLimit := 0
		if endpoint.RateLimit.Valid {
			rateLimit = int(endpoint.RateLimit.Int32)
		}

		return &middleware.EndpointResolution{
			PricingInfo: &gatewaycontext.PricingInfo{
				EndpointID:  endpoint.ID.String(),
				ProviderID:  endpoint.ProviderID.String(),
				PriceAmount: priceAmount,
				Currency:    gatewaycontext.Currency(endpoint.Currency),
			},
			RateLimit: rateLimit,
		}, nil
	})

	pxy := proxy.NewReverseProxy(resolver, slog.Default(), proxy.ConfigFromEnv())

	keySvc := auth.NewKeyService(queries)

	keyValidator := middleware.KeyValidatorFunc(func(ctx context.Context, keyHash string) (repository.ApiKey, error) {
		return queries.GetKeyByHash(ctx, keyHash)
	})

	sessionSvc := auth.NewSessionService(queries)

	sessionValidator := middleware.SessionValidatorFunc(func(ctx context.Context, rawToken string) (*repository.SessionToken, error) {
		return sessionSvc.ValidateSession(ctx, rawToken)
	})

	dashboardURL := os.Getenv("DASHBOARD_URL")
	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		apiBaseURL = "http://localhost:8080"
	}
	auth.InitGoth(dashboardURL, apiBaseURL)

	providerSvc := provider.NewProviderService(queries)
	endpointSvc := provider.NewEndpointService(queries)
	accountSvc := accounts.NewService(queries)
	depositSvc := deposit.NewService(queries, stellarCfg)
	creditHandler := deposit.NewCreditHandler(databaseService.Pool(), stellarCfg)
	watcher := deposit.NewWatcher(queries, stellarCfg, creditHandler)
	watcherCtx, watcherCancel := context.WithCancel(context.Background())
	go func() {
		if err := watcher.Run(watcherCtx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Warn("watcher exited with error", slog.String("error", err.Error()))
		}
	}()

	settlementReconciler := settlement.NewReconciler(databaseService.Pool(), queries)
	minThreshold := settlementThresholdFromEnv()
	settlementHandler := settlement.NewHandler(settlementReconciler, settlementReconciler, minThreshold)
	usageSvc := usage.NewService(queries)
	usageHandler := usage.NewHandler(usageSvc)

	srv := &Server{
		port:              port,
		db:                databaseService,
		proxy:             pxy,
		balance:           balancer,
		pricingResolver:   pricingResolver,
		usageRepo:         usageRepo,
		rateLimiter:       rateLimiter,
		redisClient:       rdb,
		ledger:            ledger.NewPostgresLedger(databaseService.Pool()),
		keyHandler:        auth.NewKeyHandler(keySvc),
		keyValidator:      keyValidator,
		sessionValidator:  sessionValidator,
		authHandler:       auth.NewHandler(sessionSvc, queries),
		oauthHandler:      auth.NewOAuthHandler(databaseService.Pool(), queries, sessionSvc, dashboardURL),
		providerHandler:   provider.NewProviderHandler(providerSvc),
		endpointHandler:   provider.NewEndpointHandler(endpointSvc),
		accountHandler:    accounts.NewHandler(accountSvc, stellarCfg.HorizonURL),
		depositHandler:    deposit.NewHandler(depositSvc),
		creditHandler:     creditHandler,
		watcher:           watcher,
		settlementHandler: settlementHandler,
		usageHandler:      usageHandler,
		stellarConfig:     stellarCfg,
		windowSeconds:     windowSeconds,
	}

	httpServer := &http.Server{
		Addr:         net.JoinHostPort("", strconv.Itoa(srv.port)),
		Handler:      srv.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	httpServer.RegisterOnShutdown(func() {
		watcherCancel()
	})

	return httpServer, nil
}

func connectRedis() (*redis.Client, error) {
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}

	rdb := redis.NewClient(opts)

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		if closeErr := rdb.Close(); closeErr != nil {
			slog.Warn("redis close after ping failure",
				slog.Any("close_err", closeErr),
			)
		}
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	slog.Info("connected to redis",
		slog.String("addr", opts.Addr),
	)

	return rdb, nil
}

func settlementThresholdFromEnv() decimal.Decimal {
	v := os.Getenv("SETTLEMENT_MIN_THRESHOLD")
	if v == "" {
		return decimal.Zero
	}

	d, err := decimal.NewFromString(v)
	if err != nil {
		slog.Warn("invalid SETTLEMENT_MIN_THRESHOLD, using default",
			slog.String("value", v),
			slog.String("error", err.Error()),
		)

		return decimal.Zero
	}

	if d.IsNegative() {
		slog.Warn("SETTLEMENT_MIN_THRESHOLD must be non-negative, using default",
			slog.String("value", v),
		)

		return decimal.Zero
	}

	return d
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
