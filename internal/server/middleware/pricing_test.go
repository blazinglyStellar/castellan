package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	gatewaycontext "castellan/internal/gateway/context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

type mockPricingResolver struct {
	pricing *gatewaycontext.PricingInfo
	err     error
}

func (m *mockPricingResolver) ResolvePricing(_ context.Context, _ uuid.UUID, _, _ string) (*gatewaycontext.PricingInfo, error) {
	return m.pricing, m.err
}

func TestPricingResolver_HappyPath(t *testing.T) {
	providerID := uuid.New()
	endpointID := uuid.New()

	mock := &mockPricingResolver{
		pricing: &gatewaycontext.PricingInfo{
			EndpointID:  endpointID.String(),
			ProviderID:  providerID.String(),
			PriceAmount: decimal.NewFromFloat(10),
			Currency:    gatewaycontext.CurrencyXLM,
		},
	}

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		pricing := gatewaycontext.GetPricingInfo(r.Context())
		if pricing.EndpointID != endpointID.String() {
			t.Errorf("expected endpoint %s, got %s", endpointID.String(), pricing.EndpointID)
		}
		if pricing.ProviderID != providerID.String() {
			t.Errorf("expected provider %s, got %s", providerID.String(), pricing.ProviderID)
		}
		if !pricing.PriceAmount.Equal(decimal.NewFromFloat(10)) {
			t.Errorf("expected price 10, got %s", pricing.PriceAmount)
		}
		if pricing.Currency != gatewaycontext.CurrencyXLM {
			t.Errorf("expected XLM, got %s", pricing.Currency)
		}
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/gateway/"+providerID.String()+"/v1/chat", nil)

	PricingResolver(mock)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200; got %d", recorder.Code)
	}
	if !called {
		t.Fatal("expected downstream handler to be called")
	}
}

func TestPricingResolver_InvalidPath(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"no prefix", "/invalid/path"},
		{"empty after prefix", "/api/gateway/"},
		{"root", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				t.Fatal("downstream handler should not be called")
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, tt.path, nil)

			PricingResolver(&mockPricingResolver{})(handler).ServeHTTP(recorder, request)

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected 400; got %d", recorder.Code)
			}
		})
	}
}

func TestPricingResolver_InvalidProviderID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("downstream handler should not be called")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/gateway/not-a-uuid/echo", nil)

	PricingResolver(&mockPricingResolver{})(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400; got %d", recorder.Code)
	}
}

func TestPricingResolver_EndpointNotFound(t *testing.T) {
	mock := &mockPricingResolver{
		err: pgx.ErrNoRows,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("downstream handler should not be called")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/gateway/"+uuid.NewString()+"/v1/chat", nil)

	PricingResolver(mock)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404; got %d", recorder.Code)
	}
}

func TestPricingResolver_ResolverError(t *testing.T) {
	mock := &mockPricingResolver{
		err: errors.New("database connection failed"),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("downstream handler should not be called")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/gateway/"+uuid.NewString()+"/v1/chat", nil)

	PricingResolver(mock)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500; got %d", recorder.Code)
	}
}

func TestPricingResolver_NilPricingWithoutError(t *testing.T) {
	mock := &mockPricingResolver{
		pricing: nil,
		err:     nil,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("downstream handler should not be called")
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/gateway/"+uuid.NewString()+"/v1/chat", nil)

	PricingResolver(mock)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500; got %d", recorder.Code)
	}
}
