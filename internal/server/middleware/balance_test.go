package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gatewaycontext "castellan/internal/gateway/context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type mockBalanceChecker struct {
	balance decimal.Decimal
	err     error
}

func (m *mockBalanceChecker) GetAccountBalance(_ context.Context, _ uuid.UUID) (decimal.Decimal, error) {
	return m.balance, m.err
}

func TestBalanceCheckPassThrough(t *testing.T) {
	tests := []struct {
		name    string
		balance decimal.Decimal
		price   decimal.Decimal
	}{
		{name: "sufficient balance", balance: decimal.NewFromFloat(10.00), price: decimal.NewFromFloat(1.00)},
		{name: "exact balance match", balance: decimal.NewFromFloat(5.00), price: decimal.NewFromFloat(5.00)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				called = true
				w.WriteHeader(http.StatusOK)
			})

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)
			request = request.WithContext(
				gatewaycontext.SetPricingInfo(
					gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
						ConsumerID: uuid.NewString(),
					}),
					gatewaycontext.PricingInfo{
						EndpointID:  "ep-1",
						PriceAmount: tt.price,
						Currency:    gatewaycontext.CurrencyXLM,
					},
				),
			)

			mock := &mockBalanceChecker{balance: tt.balance}
			BalanceCheck(mock)(handler).ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("expected status %d; got %d", http.StatusOK, recorder.Code)
			}
			if !called {
				t.Fatal("expected downstream handler to be called")
			}
		})
	}
}

func TestBalanceCheckInsufficientBalance(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)
	request = request.WithContext(
		gatewaycontext.SetPricingInfo(
			gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
				ConsumerID: uuid.NewString(),
			}),
			gatewaycontext.PricingInfo{
				EndpointID:  "ep-1",
				PriceAmount: decimal.NewFromFloat(10.00),
				Currency:    gatewaycontext.CurrencyXLM,
			},
		),
	)

	mock := &mockBalanceChecker{balance: decimal.NewFromFloat(1.00)}
	BalanceCheck(mock)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusPaymentRequired {
		t.Fatalf("expected status %d; got %d", http.StatusPaymentRequired, recorder.Code)
	}
	if called {
		t.Fatal("expected downstream handler not to be called after insufficient balance")
	}

	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON response body: %v", err)
	}
	if body["error"] != "insufficient_balance" {
		t.Fatalf("expected error %q; got %q", "insufficient_balance", body["error"])
	}
	if body["balance"] != "1.00" {
		t.Fatalf("expected balance %q; got %q", "1.00", body["balance"])
	}
	if body["required"] != "10.00" {
		t.Fatalf("expected required %q; got %q", "10.00", body["required"])
	}
}

func TestBalanceCheckMissingConsumerContext(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)

	mock := &mockBalanceChecker{balance: decimal.NewFromFloat(10.00)}
	BalanceCheck(mock)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d; got %d", http.StatusInternalServerError, recorder.Code)
	}
	if called {
		t.Fatal("expected downstream handler not to be called after missing consumer context")
	}
}

func TestBalanceCheckMissingPricingContext(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)
	request = request.WithContext(
		gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID: uuid.NewString(),
		}),
	)

	mock := &mockBalanceChecker{balance: decimal.NewFromFloat(10.00)}
	BalanceCheck(mock)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d; got %d", http.StatusInternalServerError, recorder.Code)
	}
	if called {
		t.Fatal("expected downstream handler not to be called after missing pricing context")
	}
}

func TestBalanceCheckDoesNotLogBalanceAmount(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	original := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(original) })

	// Error path: balance query fails — no balance amount should leak.
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)
	request = request.WithContext(
		gatewaycontext.SetPricingInfo(
			gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
				ConsumerID: uuid.NewString(),
			}),
			gatewaycontext.PricingInfo{
				EndpointID:  "ep-1",
				PriceAmount: decimal.NewFromFloat(1.00),
				Currency:    gatewaycontext.CurrencyXLM,
			},
		),
	)

	mock := &mockBalanceChecker{err: context.DeadlineExceeded}
	BalanceCheck(mock)(handler).ServeHTTP(recorder, request)

	logOutput := buf.String()

	if strings.Contains(logOutput, "1.00") {
		t.Fatal("balance amount must not appear in log output on error path")
	}
	if !strings.Contains(logOutput, "consumer_id") {
		t.Fatal("expected consumer_id in log output")
	}
}

func TestBalanceCheckInvalidConsumerUUID(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)
	request = request.WithContext(
		gatewaycontext.SetPricingInfo(
			gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
				ConsumerID: "not-a-uuid",
			}),
			gatewaycontext.PricingInfo{
				EndpointID:  "ep-1",
				PriceAmount: decimal.NewFromFloat(1.00),
				Currency:    gatewaycontext.CurrencyXLM,
			},
		),
	)

	mock := &mockBalanceChecker{balance: decimal.NewFromFloat(10.00)}
	BalanceCheck(mock)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d; got %d", http.StatusInternalServerError, recorder.Code)
	}
	if called {
		t.Fatal("expected downstream handler not to be called after invalid consumer UUID")
	}
}

func TestBalanceCheckBalanceQueryError(t *testing.T) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)
	request = request.WithContext(
		gatewaycontext.SetPricingInfo(
			gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
				ConsumerID: uuid.NewString(),
			}),
			gatewaycontext.PricingInfo{
				EndpointID:  "ep-1",
				PriceAmount: decimal.NewFromFloat(1.00),
				Currency:    gatewaycontext.CurrencyXLM,
			},
		),
	)

	mock := &mockBalanceChecker{err: context.DeadlineExceeded}
	BalanceCheck(mock)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d; got %d", http.StatusInternalServerError, recorder.Code)
	}
	if called {
		t.Fatal("expected downstream handler not to be called after balance query error")
	}
}
