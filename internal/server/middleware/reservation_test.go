package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	gatewaycontext "castellan/internal/gateway/context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type call struct {
	Method      string
	ReferenceID string
}

type mockLedgerService struct {
	mu          sync.Mutex
	Calls       []call
	ReserveFunc func(context.Context, uuid.UUID, decimal.Decimal, string) error
	CommitFunc  func(context.Context, string) error
	ReleaseFunc func(context.Context, string) error
}

func (m *mockLedgerService) Reserve(ctx context.Context, consumerID uuid.UUID, amount decimal.Decimal, referenceID string) error {
	m.mu.Lock()
	m.Calls = append(m.Calls, call{Method: "Reserve", ReferenceID: referenceID})
	m.mu.Unlock()
	if m.ReserveFunc != nil {
		return m.ReserveFunc(ctx, consumerID, amount, referenceID)
	}
	return nil
}

func (m *mockLedgerService) Commit(ctx context.Context, referenceID string) error {
	m.mu.Lock()
	m.Calls = append(m.Calls, call{Method: "Commit", ReferenceID: referenceID})
	m.mu.Unlock()
	if m.CommitFunc != nil {
		return m.CommitFunc(ctx, referenceID)
	}
	return nil
}

func (m *mockLedgerService) Release(ctx context.Context, referenceID string) error {
	m.mu.Lock()
	m.Calls = append(m.Calls, call{Method: "Release", ReferenceID: referenceID})
	m.mu.Unlock()
	if m.ReleaseFunc != nil {
		return m.ReleaseFunc(ctx, referenceID)
	}
	return nil
}

func withContext(r *http.Request) *http.Request {
	return r.WithContext(
		gatewaycontext.SetPricingInfo(
			gatewaycontext.SetConsumerInfo(r.Context(), gatewaycontext.ConsumerInfo{
				ConsumerID: uuid.NewString(),
			}),
			gatewaycontext.PricingInfo{
				EndpointID:  "ep-1",
				PriceAmount: decimal.NewFromFloat(1.00),
				Currency:    gatewaycontext.CurrencyXLM,
			},
		),
	)
}

func withRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

func expectLedgerCalls(t *testing.T, ledger *mockLedgerService, expected ...string) {
	t.Helper()
	if len(ledger.Calls) != len(expected) {
		t.Fatalf("expected %d ledger calls; got %d", len(expected), len(ledger.Calls))
	}
	for i, method := range expected {
		if ledger.Calls[i].Method != method {
			t.Fatalf("expected call %d to be %s; got %s", i, method, ledger.Calls[i].Method)
		}
	}
}

func TestReservationCommitOn2xx(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ledger := &mockLedgerService{}
	recorder := httptest.NewRecorder()
	request := withContext(httptest.NewRequest(http.MethodGet, "/v1/providers/", nil))

	Reservation(ledger)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d; got %d", http.StatusOK, recorder.Code)
	}
	expectLedgerCalls(t, ledger, "Reserve", "Commit")
}

func TestReservationReleaseOn4xx(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	ledger := &mockLedgerService{}
	recorder := httptest.NewRecorder()
	request := withContext(httptest.NewRequest(http.MethodGet, "/v1/providers/", nil))

	Reservation(ledger)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d; got %d", http.StatusNotFound, recorder.Code)
	}
	expectLedgerCalls(t, ledger, "Reserve", "Release")
}

func TestReservationReleaseOn5xx(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})

	ledger := &mockLedgerService{}
	recorder := httptest.NewRecorder()
	request := withContext(httptest.NewRequest(http.MethodGet, "/v1/providers/", nil))

	Reservation(ledger)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected status %d; got %d", http.StatusBadGateway, recorder.Code)
	}
	expectLedgerCalls(t, ledger, "Reserve", "Release")
}

func TestReservationReleaseOnNoStatusWritten(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
	})

	ledger := &mockLedgerService{}
	recorder := httptest.NewRecorder()
	request := withContext(httptest.NewRequest(http.MethodGet, "/v1/providers/", nil))

	Reservation(ledger)(handler).ServeHTTP(recorder, request)

	expectLedgerCalls(t, ledger, "Reserve", "Release")
}

func TestReservationMissingConsumerContext(t *testing.T) {
	t.Parallel()

	called := false
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	})

	ledger := &mockLedgerService{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)

	Reservation(ledger)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d; got %d", http.StatusInternalServerError, recorder.Code)
	}
	if called {
		t.Fatal("expected downstream handler not to be called after missing consumer context")
	}
	expectLedgerCalls(t, ledger)
}

func TestReservationMissingPricingContext(t *testing.T) {
	t.Parallel()

	called := false
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	})

	ledger := &mockLedgerService{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)
	request = request.WithContext(
		gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID: uuid.NewString(),
		}),
	)

	Reservation(ledger)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d; got %d", http.StatusInternalServerError, recorder.Code)
	}
	if called {
		t.Fatal("expected downstream handler not to be called after missing pricing context")
	}
	expectLedgerCalls(t, ledger)
}

func TestReservationInvalidConsumerUUID(t *testing.T) {
	t.Parallel()

	called := false
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	})

	ledger := &mockLedgerService{}
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

	Reservation(ledger)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d; got %d", http.StatusInternalServerError, recorder.Code)
	}
	if called {
		t.Fatal("expected downstream handler not to be called after invalid consumer UUID")
	}
	expectLedgerCalls(t, ledger)
}

func TestReservationReserveError(t *testing.T) {
	t.Parallel()

	called := false
	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		called = true
	})

	ledger := &mockLedgerService{
		ReserveFunc: func(_ context.Context, _ uuid.UUID, _ decimal.Decimal, _ string) error {
			return context.DeadlineExceeded
		},
	}
	recorder := httptest.NewRecorder()
	request := withContext(httptest.NewRequest(http.MethodGet, "/v1/providers/", nil))

	Reservation(ledger)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d; got %d", http.StatusServiceUnavailable, recorder.Code)
	}
	if called {
		t.Fatal("expected downstream handler not to be called after reserve error")
	}

	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected valid JSON response body: %v", err)
	}
	if body["error"] != "reservation failed" {
		t.Fatalf("expected error %q; got %q", "reservation failed", body["error"])
	}
}

func TestReservationUsesRequestID(t *testing.T) {
	t.Parallel()

	const customTraceID = "my-custom-trace-123"

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ledger := &mockLedgerService{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)
	request.Header.Set("X-Request-Id", customTraceID)
	request = request.WithContext(
		withRequestID(
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
			customTraceID,
		),
	)

	Reservation(ledger)(handler).ServeHTTP(recorder, request)

	if len(ledger.Calls) < 2 {
		t.Fatalf("expected at least 2 ledger calls; got %d", len(ledger.Calls))
	}

	for _, c := range ledger.Calls {
		if c.ReferenceID != customTraceID {
			t.Fatalf("expected referenceID %q for call %s; got %q", customTraceID, c.Method, c.ReferenceID)
		}
	}
}

func TestReservationDoesNotLogRawAmount(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	original := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(original) })

	// Success path: no error logs emitted, no amount leaks.
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ledger := &mockLedgerService{}
	recorder := httptest.NewRecorder()
	request := withContext(httptest.NewRequest(http.MethodGet, "/v1/providers/", nil))

	Reservation(ledger)(handler).ServeHTTP(recorder, request)

	logOutput := buf.String()
	if logOutput != "" {
		t.Fatal("expected no log output on success path")
	}

	buf.Reset()

	// Error path: reservation fails — no amount should leak.
	handler2 := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ledger2 := &mockLedgerService{
		ReserveFunc: func(_ context.Context, _ uuid.UUID, _ decimal.Decimal, _ string) error {
			return context.DeadlineExceeded
		},
	}
	recorder2 := httptest.NewRecorder()
	request2 := withContext(httptest.NewRequest(http.MethodGet, "/v1/providers/", nil))

	Reservation(ledger2)(handler2).ServeHTTP(recorder2, request2)

	logOutput = buf.String()
	if strings.Contains(logOutput, "1.00") {
		t.Fatal("raw amount must not appear in log output on error path")
	}
	if !strings.Contains(logOutput, "consumer_id") {
		t.Fatal("expected consumer_id in log output on error path")
	}
}

func TestReservationCommitErrorNonFatal(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	ledger := &mockLedgerService{
		CommitFunc: func(_ context.Context, _ string) error {
			return context.DeadlineExceeded
		},
	}
	recorder := httptest.NewRecorder()
	request := withContext(httptest.NewRequest(http.MethodGet, "/v1/providers/", nil))

	Reservation(ledger)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d; got %d", http.StatusOK, recorder.Code)
	}
}

func TestReservationReleaseErrorNonFatal(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	ledger := &mockLedgerService{
		ReleaseFunc: func(_ context.Context, _ string) error {
			return context.DeadlineExceeded
		},
	}
	recorder := httptest.NewRecorder()
	request := withContext(httptest.NewRequest(http.MethodGet, "/v1/providers/", nil))

	Reservation(ledger)(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d; got %d", http.StatusInternalServerError, recorder.Code)
	}
}
