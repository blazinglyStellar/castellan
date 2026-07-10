package middleware

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gatewaycontext "castellan/internal/gateway/context"
	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type mockUsageEventRepository struct {
	createUsageCalls []repository.CreateUsageEventParams
	createErr        error
}

func (m *mockUsageEventRepository) CreateUsageEvent(_ context.Context, arg repository.CreateUsageEventParams) (repository.CreateUsageEventRow, error) {
	m.createUsageCalls = append(m.createUsageCalls, arg)
	return repository.CreateUsageEventRow{}, m.createErr
}

func TestUsageCaptureDoesNotLogRawAmount(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	consumerID := uuid.New()
	providerID := uuid.New()
	endpointID := uuid.New()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)
	request = request.WithContext(
		gatewaycontext.SetUpstreamMetrics(
			gatewaycontext.SetPricingInfo(
				gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
					ConsumerID: consumerID.String(),
				}),
				gatewaycontext.PricingInfo{
					EndpointID:  endpointID.String(),
					ProviderID:  providerID.String(),
					PriceAmount: decimal.NewFromFloat(1.50),
					Currency:    gatewaycontext.CurrencyXLM,
				},
			),
			gatewaycontext.UpstreamMetrics{
				StatusCode:   200,
				LatencyMs:    42,
				ResponseSize: 1024,
			},
		),
	)
	request = request.WithContext(
		context.WithValue(request.Context(), requestIDContextKey, uuid.New().String()),
	)

	mock := &mockUsageEventRepository{}
	UsageCapture(mock, logger)(handler).ServeHTTP(recorder, request)

	logOutput := buf.String()
	if strings.Contains(logOutput, "1.50") {
		t.Fatal("raw amount must not appear in log output")
	}
	if strings.Contains(logOutput, "request_cost") {
		t.Fatal("request_cost must not appear in log output on success path")
	}
}

func TestUsageCaptureSuccess(t *testing.T) {
	consumerID := uuid.New()
	providerID := uuid.New()
	endpointID := uuid.New()
	requestID := uuid.New().String()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)
	request = request.WithContext(
		gatewaycontext.SetUpstreamMetrics(
			gatewaycontext.SetPricingInfo(
				gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
					ConsumerID: consumerID.String(),
				}),
				gatewaycontext.PricingInfo{
					EndpointID:  endpointID.String(),
					ProviderID:  providerID.String(),
					PriceAmount: decimal.NewFromFloat(1.50),
					Currency:    gatewaycontext.CurrencyXLM,
				},
			),
			gatewaycontext.UpstreamMetrics{
				StatusCode:   200,
				LatencyMs:    42,
				ResponseSize: 1024,
			},
		),
	)
	request = request.WithContext(
		context.WithValue(request.Context(), requestIDContextKey, requestID),
	)

	mock := &mockUsageEventRepository{}
	UsageCapture(mock, slog.Default())(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d; got %d", http.StatusOK, recorder.Code)
	}
	if len(mock.createUsageCalls) != 1 {
		t.Fatalf("expected 1 create call; got %d", len(mock.createUsageCalls))
	}

	params := mock.createUsageCalls[0]
	if params.ConsumerID != consumerID {
		t.Fatalf("expected consumer_id %v; got %v", consumerID, params.ConsumerID)
	}
	if params.ProviderID != providerID {
		t.Fatalf("expected provider_id %v; got %v", providerID, params.ProviderID)
	}
	if params.EndpointID != endpointID {
		t.Fatalf("expected endpoint_id %v; got %v", endpointID, params.EndpointID)
	}
	if params.RequestID != requestID {
		t.Fatalf("expected request_id %s; got %s", requestID, params.RequestID)
	}
	if params.Status != repository.UsageStatusCompleted {
		t.Fatalf("expected status %q; got %q", repository.UsageStatusCompleted, params.Status)
	}
	if !params.StatusCode.Valid || params.StatusCode.Int32 != 200 {
		t.Fatalf("expected status_code 200; got %+v", params.StatusCode)
	}
	if !params.LatencyMs.Valid || params.LatencyMs.Int32 != 42 {
		t.Fatalf("expected latency_ms 42; got %+v", params.LatencyMs)
	}
	if !params.ResponseSize.Valid || params.ResponseSize.Int32 != 1024 {
		t.Fatalf("expected response_size 1024; got %+v", params.ResponseSize)
	}
}

func TestUsageCaptureMissingConsumerID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)

	mock := &mockUsageEventRepository{}
	UsageCapture(mock, slog.Default())(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d; got %d", http.StatusOK, recorder.Code)
	}
	if len(mock.createUsageCalls) != 0 {
		t.Fatalf("expected 0 create calls; got %d", len(mock.createUsageCalls))
	}
}

func TestUsageCaptureMissingPricingInfo(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)
	request = request.WithContext(
		gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
			ConsumerID: uuid.NewString(),
		}),
	)

	mock := &mockUsageEventRepository{}
	UsageCapture(mock, slog.Default())(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d; got %d", http.StatusOK, recorder.Code)
	}
	if len(mock.createUsageCalls) != 0 {
		t.Fatalf("expected 0 create calls; got %d", len(mock.createUsageCalls))
	}
}

func TestUsageCaptureInvalidUUID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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
				EndpointID:  uuid.NewString(),
				ProviderID:  uuid.NewString(),
				PriceAmount: decimal.NewFromFloat(1.00),
				Currency:    gatewaycontext.CurrencyXLM,
			},
		),
	)

	mock := &mockUsageEventRepository{}
	UsageCapture(mock, slog.Default())(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d; got %d", http.StatusOK, recorder.Code)
	}
	if len(mock.createUsageCalls) != 0 {
		t.Fatalf("expected 0 create calls; got %d", len(mock.createUsageCalls))
	}
}

func TestUsageCaptureRepoError(t *testing.T) {
	consumerID := uuid.New()
	providerID := uuid.New()
	endpointID := uuid.New()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)
	request = request.WithContext(
		gatewaycontext.SetUpstreamMetrics(
			gatewaycontext.SetPricingInfo(
				gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
					ConsumerID: consumerID.String(),
				}),
				gatewaycontext.PricingInfo{
					EndpointID:  endpointID.String(),
					ProviderID:  providerID.String(),
					PriceAmount: decimal.NewFromFloat(1.00),
					Currency:    gatewaycontext.CurrencyXLM,
				},
			),
			gatewaycontext.UpstreamMetrics{
				StatusCode:   200,
				LatencyMs:    10,
				ResponseSize: 512,
			},
		),
	)

	mock := &mockUsageEventRepository{createErr: errors.New("repo error")}
	UsageCapture(mock, slog.Default())(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d; got %d", http.StatusOK, recorder.Code)
	}
	if len(mock.createUsageCalls) != 1 {
		t.Fatalf("expected 1 create call; got %d", len(mock.createUsageCalls))
	}
}

func TestUsageCaptureWithZeroMetrics(t *testing.T) {
	consumerID := uuid.New()
	providerID := uuid.New()
	endpointID := uuid.New()

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/providers/", nil)
	request = request.WithContext(
		gatewaycontext.SetUpstreamMetrics(
			gatewaycontext.SetPricingInfo(
				gatewaycontext.SetConsumerInfo(request.Context(), gatewaycontext.ConsumerInfo{
					ConsumerID: consumerID.String(),
				}),
				gatewaycontext.PricingInfo{
					EndpointID:  endpointID.String(),
					ProviderID:  providerID.String(),
					PriceAmount: decimal.NewFromFloat(5.00),
					Currency:    gatewaycontext.CurrencyXLM,
				},
			),
			gatewaycontext.UpstreamMetrics{},
		),
	)

	mock := &mockUsageEventRepository{}
	UsageCapture(mock, slog.Default())(handler).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d; got %d", http.StatusOK, recorder.Code)
	}
	if len(mock.createUsageCalls) != 1 {
		t.Fatalf("expected 1 create call; got %d", len(mock.createUsageCalls))
	}

	params := mock.createUsageCalls[0]
	if params.StatusCode.Valid {
		t.Fatal("expected status_code to be invalid (zero metrics)")
	}
	if params.LatencyMs.Valid {
		t.Fatal("expected latency_ms to be invalid (zero metrics)")
	}
	if params.ResponseSize.Valid {
		t.Fatal("expected response_size to be invalid (zero metrics)")
	}
}
