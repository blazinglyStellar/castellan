package server

import (
	"encoding/json"
	"log"
	"log/slog"
	"net/http"

	"castellan/internal/server/middleware"
)

// RegisterRoutes builds the top-level mux with health, hello, and gateway routes,
// then wraps it with recovery, request logging, request ID, and CORS middleware.
func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.HelloWorldHandler)
	mux.HandleFunc("/health", s.healthHandler)
	mux.Handle("GET /api/v1/keys", http.HandlerFunc(s.keyHandler.ListKeys))
	mux.Handle("POST /api/v1/keys", http.HandlerFunc(s.keyHandler.CreateKey))
	mux.Handle("POST /api/v1/keys/{id}/revoke", http.HandlerFunc(s.keyHandler.RevokeKey))
	mux.Handle("POST /api/v1/keys/{id}/rotate", http.HandlerFunc(s.keyHandler.RotateKey))
	mux.Handle("POST /api/v1/sessions", http.HandlerFunc(s.sessionHandler.CreateSession))
	mux.Handle("GET /api/v1/sessions", http.HandlerFunc(s.sessionHandler.ListSessions))
	mux.Handle("POST /api/v1/sessions/{id}/revoke", http.HandlerFunc(s.sessionHandler.RevokeSession))
	mux.Handle("POST /api/v1/providers", http.HandlerFunc(s.providerHandler.CreateProvider))
	mux.Handle("GET /api/v1/providers", http.HandlerFunc(s.providerHandler.ListProviders))
	mux.Handle("GET /api/v1/providers/{id}", http.HandlerFunc(s.providerHandler.GetProvider))
	mux.Handle("PATCH /api/v1/providers/{id}", http.HandlerFunc(s.providerHandler.UpdateProvider))
	mux.Handle("PATCH /api/v1/providers/{id}/status", http.HandlerFunc(s.providerHandler.UpdateProviderStatus))
	mux.Handle("DELETE /api/v1/providers/{id}", http.HandlerFunc(s.providerHandler.DeleteProvider))
	mux.Handle("POST /api/v1/providers/{providerId}/endpoints", http.HandlerFunc(s.endpointHandler.CreateEndpoint))
	mux.Handle("GET /api/v1/providers/{providerId}/endpoints", http.HandlerFunc(s.endpointHandler.ListEndpoints))
	mux.Handle("GET /api/v1/endpoints/{id}", http.HandlerFunc(s.endpointHandler.GetEndpoint))
	mux.Handle("PATCH /api/v1/endpoints/{id}", http.HandlerFunc(s.endpointHandler.UpdateEndpoint))
	mux.Handle("PATCH /api/v1/endpoints/{id}/status", http.HandlerFunc(s.endpointHandler.UpdateEndpointStatus))
	mux.Handle("DELETE /api/v1/endpoints/{id}", http.HandlerFunc(s.endpointHandler.DeleteEndpoint))
	mux.Handle("GET /api/v1/accounts/me", http.HandlerFunc(s.accountHandler.GetAccount))
	mux.Handle("GET /api/v1/accounts/me/entries", http.HandlerFunc(s.accountHandler.ListEntries))
	mux.Handle("GET /api/v1/accounts/me/entries/{id}", http.HandlerFunc(s.accountHandler.GetEntry))
	mux.Handle("GET /api/v1/deposits/intent", s.authMiddleware(http.HandlerFunc(s.depositHandler.DepositIntent)))
	mux.Handle("GET /api/v1/deposits", s.authMiddleware(http.HandlerFunc(s.depositHandler.ListDeposits)))
	mux.Handle("GET /api/v1/settlements", s.authMiddleware(http.HandlerFunc(s.settlementHandler.ListSettlements)))
	s.GatewayRoutes(mux)

	var handler http.Handler = mux

	handler = middleware.Recovery(slog.Default())(handler)
	handler = middleware.RequestLogger(slog.Default())(handler)
	handler = middleware.RequestID()(handler)

	return s.corsMiddleware(handler)
}

// GatewayRoutes registers POST /api/gateway/ with the middleware chain:
// AuthCheck → PricingResolver → RateLimitCheck → BalanceCheck → MaxBodySize → Reservation → UsageCapture → Proxy.
func (s *Server) GatewayRoutes(mux *http.ServeMux) {
	handler := http.Handler(s.proxy)
	handler = middleware.UsageCapture(s.usageRepo, slog.Default())(handler)
	handler = middleware.Reservation(s.ledger)(handler)
	handler = middleware.MaxBodySize(middleware.MaxBodySizeFromEnv())(handler)
	handler = middleware.BalanceCheck(s.balance)(handler)
	handler = middleware.RateLimitCheck(s.rateLimiter)(handler)
	handler = middleware.PricingResolver(s.pricingResolver, s.windowSeconds)(handler)
	handler = middleware.AuthCheck(s.keyValidator, s.sessionValidator)(handler)

	mux.Handle("POST /api/gateway/", handler)
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return middleware.AuthCheck(s.keyValidator, s.sessionValidator)(next)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")
		w.Header().Set("Access-Control-Allow-Credentials", "false")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		next.ServeHTTP(w, r)
	})
}

// HelloWorldHandler responds with {"message": "Hello World"} at GET /.
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
