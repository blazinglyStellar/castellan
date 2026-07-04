package settlement

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"castellan/internal/repository/db"
)

type Aggregator struct {
	queries repository.Querier
}

func NewAggregator(queries repository.Querier) *Aggregator {
	return &Aggregator{queries: queries}
}

func (a *Aggregator) Aggregate(ctx context.Context) ([]ProviderPayout, error) {
	rows, err := a.queries.GetUnsettledProviderEarnings(ctx)
	if err != nil {
		return nil, fmt.Errorf("get unsettled provider earnings: %w", err)
	}

	payouts := make([]ProviderPayout, 0, len(rows))
	for _, row := range rows {
		if !row.PayoutStellarAddress.Valid || row.PayoutStellarAddress.String == "" {
			continue
		}

		amount, err := NumericToDecimal(row.TotalAmount)
		if err != nil {
			return nil, fmt.Errorf("parse amount for provider %s: %w", row.ProviderID, err)
		}

		if amount.LessThanOrEqual(decimal.Zero) {
			continue
		}

		payouts = append(payouts, ProviderPayout{
			ProviderID:    row.ProviderID,
			Amount:        amount,
			Currency:      row.Currency,
			WalletAddress: row.PayoutStellarAddress.String,
		})
	}

	return payouts, nil
}
