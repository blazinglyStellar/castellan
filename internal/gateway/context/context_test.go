//revive:disable:package-directory-mismatch
package gatewaycontext

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/shopspring/decimal"
)

func TestGatewayContextValuesPassBetweenHandlers(t *testing.T) {
	expectedConsumer := ConsumerInfo{
		ConsumerID:      "consumer-1",
		APIKeyID:        "api-key-1",
		IsAuthenticated: true,
	}
	expectedPricing := PricingInfo{
		EndpointID:  "endpoint-1",
		ProviderID:  "provider-1",
		PriceAmount: decimal.RequireFromString("12.34"),
		Currency:    CurrencyUSDC,
	}
	expectedMetrics := UpstreamMetrics{
		StatusCode:   http.StatusAccepted,
		LatencyMs:    42,
		ResponseSize: 2048,
	}

	reader := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := GetConsumerInfo(r.Context()); got != expectedConsumer {
			t.Fatalf("expected consumer info %#v, got %#v", expectedConsumer, got)
		}

		if got := GetPricingInfo(r.Context()); !got.PriceAmount.Equal(expectedPricing.PriceAmount) ||
			got.EndpointID != expectedPricing.EndpointID ||
			got.ProviderID != expectedPricing.ProviderID ||
			got.Currency != expectedPricing.Currency {
			t.Fatalf("expected pricing info %#v, got %#v", expectedPricing, got)
		}

		if got := GetUpstreamMetrics(r.Context()); got != expectedMetrics {
			t.Fatalf("expected upstream metrics %#v, got %#v", expectedMetrics, got)
		}

		w.WriteHeader(http.StatusNoContent)
	})

	setter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := SetConsumerInfo(r.Context(), expectedConsumer)
		ctx = SetPricingInfo(ctx, expectedPricing)
		ctx = SetUpstreamMetrics(ctx, expectedMetrics)

		reader.ServeHTTP(w, r.WithContext(ctx))
	})

	recorder := httptest.NewRecorder()
	setter.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestGatewayContextUnsetValuesReturnZeroValues(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	if got := GetConsumerInfo(request.Context()); got != (ConsumerInfo{}) {
		t.Fatalf("expected zero consumer info, got %#v", got)
	}

	pricing := GetPricingInfo(request.Context())
	if pricing.EndpointID != "" ||
		pricing.ProviderID != "" ||
		!pricing.PriceAmount.Equal(decimal.Decimal{}) ||
		pricing.Currency != "" {
		t.Fatalf("expected zero pricing info, got %#v", pricing)
	}

	if got := GetUpstreamMetrics(request.Context()); got != (UpstreamMetrics{}) {
		t.Fatalf("expected zero upstream metrics, got %#v", got)
	}
}

func TestCurrencyValidation(t *testing.T) {
	for _, currency := range []Currency{CurrencyXLM, CurrencyUSDC} {
		if !currency.IsValid() {
			t.Fatalf("expected currency %q to be valid", currency)
		}

		if err := currency.Validate(); err != nil {
			t.Fatalf("expected currency %q to validate: %v", currency, err)
		}
	}

	invalidCurrency := Currency("EUR")
	if invalidCurrency.IsValid() {
		t.Fatalf("expected currency %q to be invalid", invalidCurrency)
	}

	if err := invalidCurrency.Validate(); err == nil {
		t.Fatal("expected invalid currency to return an error")
	}
}
