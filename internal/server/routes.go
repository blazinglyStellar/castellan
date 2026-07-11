package server

import (
	"encoding/json"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"castellan/internal/gateway/context"
	"castellan/internal/server/middleware"
)

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.HelloWorldHandler)
	mux.HandleFunc("/health", s.healthHandler)
	mux.Handle("GET /auth/{provider}/login", http.HandlerFunc(s.oauthHandler.Login))
	mux.Handle("GET /auth/{provider}/callback", http.HandlerFunc(s.oauthHandler.Callback))
	mux.Handle("GET /api/v1/auth/me", s.authMiddleware(http.HandlerFunc(s.authHandler.Me)))
	mux.Handle("POST /api/v1/auth/logout", s.authMiddleware(http.HandlerFunc(s.authHandler.Logout)))
	mux.Handle("GET /api/v1/auth/logout", http.HandlerFunc(s.authHandler.LogoutRedirect))
	mux.Handle("GET /api/v1/keys", s.authMiddleware(http.HandlerFunc(s.keyHandler.ListKeys)))
	mux.Handle("POST /api/v1/keys", s.authMiddleware(http.HandlerFunc(s.keyHandler.CreateKey)))
	mux.Handle("GET /api/v1/discover", s.authMiddleware(http.HandlerFunc(s.providerHandler.ListPublicProviders)))
	mux.Handle("POST /api/v1/keys/{id}/revoke", s.authMiddleware(http.HandlerFunc(s.keyHandler.RevokeKey)))
	mux.Handle("PATCH /api/v1/keys/{id}", s.authMiddleware(http.HandlerFunc(s.keyHandler.UpdateKey)))
	mux.Handle("POST /api/v1/keys/{id}/rotate", s.authMiddleware(http.HandlerFunc(s.keyHandler.RotateKey)))
	mux.Handle("POST /api/v1/providers", s.authMiddleware(http.HandlerFunc(s.providerHandler.CreateProvider)))
	mux.Handle("GET /api/v1/providers", s.authMiddleware(http.HandlerFunc(s.providerHandler.ListProviders)))
	mux.Handle("GET /api/v1/providers/{id}", s.authMiddleware(http.HandlerFunc(s.providerHandler.GetProvider)))
	mux.Handle("PATCH /api/v1/providers/{id}", s.authMiddleware(http.HandlerFunc(s.providerHandler.UpdateProvider)))
	mux.Handle("PATCH /api/v1/providers/{id}/status", s.authMiddleware(http.HandlerFunc(s.providerHandler.UpdateProviderStatus)))
	mux.Handle("DELETE /api/v1/providers/{id}", s.authMiddleware(http.HandlerFunc(s.providerHandler.DeleteProvider)))
	mux.Handle("POST /api/v1/providers/{providerId}/endpoints", s.authMiddleware(http.HandlerFunc(s.endpointHandler.CreateEndpoint)))
	mux.Handle("GET /api/v1/providers/{providerId}/endpoints", s.authMiddleware(http.HandlerFunc(s.endpointHandler.ListEndpoints)))
	mux.Handle("GET /api/v1/providers/{providerId}/endpoints/public", s.authMiddleware(http.HandlerFunc(s.endpointHandler.ListPublicEndpoints)))
	mux.Handle("GET /api/v1/endpoints/{id}", s.authMiddleware(http.HandlerFunc(s.endpointHandler.GetEndpoint)))
	mux.Handle("PATCH /api/v1/endpoints/{id}", s.authMiddleware(http.HandlerFunc(s.endpointHandler.UpdateEndpoint)))
	mux.Handle("PATCH /api/v1/endpoints/{id}/status", s.authMiddleware(http.HandlerFunc(s.endpointHandler.UpdateEndpointStatus)))
	mux.Handle("DELETE /api/v1/endpoints/{id}", s.authMiddleware(http.HandlerFunc(s.endpointHandler.DeleteEndpoint)))
	mux.Handle("GET /api/v1/accounts/me", s.authMiddleware(http.HandlerFunc(s.accountHandler.GetAccount)))
	mux.Handle("GET /api/v1/accounts/me/entries", s.authMiddleware(http.HandlerFunc(s.accountHandler.ListEntries)))
	mux.Handle("GET /api/v1/accounts/me/entries/{id}", s.authMiddleware(http.HandlerFunc(s.accountHandler.GetEntry)))
	mux.Handle("GET /api/v1/deposits/intent", s.authMiddleware(http.HandlerFunc(s.depositHandler.DepositIntent)))
	mux.Handle("GET /api/v1/deposits", s.authMiddleware(http.HandlerFunc(s.depositHandler.ListDeposits)))
	mux.Handle("GET /api/v1/settlements", s.authMiddleware(http.HandlerFunc(s.settlementHandler.ListSettlements)))
	mux.Handle("GET /api/v1/settlements/summary", s.authMiddleware(http.HandlerFunc(s.settlementHandler.HandleSummary)))
	mux.Handle("GET /api/v1/settlements/threshold", s.authMiddleware(http.HandlerFunc(s.settlementHandler.HandleThreshold)))
	mux.Handle("GET /api/v1/me", s.authMiddleware(http.HandlerFunc(s.authHandler.DashboardMe)))
	mux.Handle("GET /api/v1/balance", s.authMiddleware(http.HandlerFunc(s.accountHandler.GetBalance)))
	mux.Handle("GET /api/v1/usage", s.authMiddleware(http.HandlerFunc(s.usageHandler.ListUsage)))
	mux.Handle("GET /api/v1/earnings", s.authMiddleware(http.HandlerFunc(s.usageHandler.GetEarnings)))
	mux.HandleFunc("GET /openapi.yaml", openapiHandler)
	mux.HandleFunc("GET /docs", docsHandler)
	s.GatewayRoutes(mux)

	var handler http.Handler = mux

	handler = middleware.Recovery(slog.Default())(handler)
	handler = middleware.RequestLogger(slog.Default())(handler)
	handler = middleware.RequestID()(handler)

	return s.corsMiddleware(handler)
}

func (s *Server) GatewayRoutes(mux *http.ServeMux) {
	handler := http.Handler(s.proxy)
	handler = middleware.Reservation(s.ledger)(handler)
	handler = middleware.UsageCapture(s.usageRepo, slog.Default())(handler)
	handler = middleware.MaxBodySize(middleware.MaxBodySizeFromEnv())(handler)
	handler = middleware.BalanceCheck(s.balance)(handler)
	handler = middleware.RateLimitCheck(s.rateLimiter)(handler)
	handler = middleware.PricingResolver(s.pricingResolver, s.windowSeconds)(handler)
	handler = middleware.AuthCheck(s.keyValidator, s.sessionValidator)(handler)

	next := handler
	handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(gatewaycontext.SetRequestStart(r.Context(), time.Now()))
		next.ServeHTTP(w, r)
	})

	mux.Handle("/api/gateway/", handler)
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return middleware.AuthCheck(s.keyValidator, s.sessionValidator)(next)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	dashboardURL := os.Getenv("DASHBOARD_URL")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && origin != dashboardURL {
			w.Header().Set("Vary", "Origin")
			http.Error(w, "origin not allowed", http.StatusForbidden)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", dashboardURL)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) HelloWorldHandler(w http.ResponseWriter, _ *http.Request) {
	resp := map[string]string{"message": "Hello World"}
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write(jsonResp); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}

func (s *Server) healthHandler(w http.ResponseWriter, _ *http.Request) {
	resp, err := json.Marshal(s.db.Health())
	if err != nil {
		http.Error(w, "Failed to marshal health check response", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	if _, err := w.Write(resp); err != nil {
		log.Printf("Failed to write response: %v", err)
	}
}
