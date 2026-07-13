//revive:disable:package-directory-mismatch
package gatewaycontext

import (
	stdcontext "context"
	"fmt"

	"github.com/shopspring/decimal"
)

type pricingInfoKey struct{}

// Currency identifies the billing currency for an endpoint price.
type Currency string

const (
	// CurrencyXLM represents the Stellar lumen.
	CurrencyXLM Currency = "XLM"
	// CurrencyUSDC represents the Stellar USDC stablecoin.
	CurrencyUSDC Currency = "USDC"
)

// IsValid reports whether the currency is accepted by the gateway.
func (c Currency) IsValid() bool {
	switch c {
	case CurrencyXLM, CurrencyUSDC:
		return true
	default:
		return false
	}
}

// Validate returns an error when the currency is not accepted by the gateway.
func (c Currency) Validate() error {
	if c.IsValid() {
		return nil
	}

	return fmt.Errorf("unsupported currency %q", c)
}

// PricingInfo describes the endpoint pricing applied to a gateway request.
type PricingInfo struct {
	EndpointID  string
	ProviderID  string
	PriceAmount decimal.Decimal
	Currency    Currency
}

// SetPricingInfo stores pricing information in the request context.
func SetPricingInfo(ctx stdcontext.Context, info PricingInfo) stdcontext.Context {
	return stdcontext.WithValue(ctx, pricingInfoKey{}, info)
}

// GetPricingInfo reads pricing information from the request context.
func GetPricingInfo(ctx stdcontext.Context) PricingInfo {
	info, ok := ctx.Value(pricingInfoKey{}).(PricingInfo)
	if !ok {
		return PricingInfo{}
	}

	return info
}
