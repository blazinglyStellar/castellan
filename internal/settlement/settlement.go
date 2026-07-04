package settlement

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"
)

type Cycle interface {
	Run(ctx context.Context) error
}

func NumericToDecimal(n pgtype.Numeric) (decimal.Decimal, error) {
	if !n.Valid {
		return decimal.Zero, errors.New("numeric is null")
	}

	return decimal.NewFromBigInt(n.Int, n.Exp), nil
}

func DecimalToNumeric(d decimal.Decimal) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if err := n.Scan(d.String()); err != nil {
		return n, fmt.Errorf("scan decimal to numeric: %w", err)
	}

	return n, nil
}
