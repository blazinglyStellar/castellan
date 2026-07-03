package stellar

import (
	"log/slog"
	"os"

	"github.com/shopspring/decimal"
)

const (
	defaultHorizonURL    = "https://horizon-testnet.stellar.org"
	defaultNetwork       = "testnet"
	defaultMinDepositAmt = "5"
)

type Config struct {
	HorizonURL       string
	Network          string
	HotWalletAddress string
	WalletSecretKey  string
	MinDepositAmount decimal.Decimal
}

func DefaultConfig() Config {
	minDep := decimal.RequireFromString(defaultMinDepositAmt)
	return Config{
		HorizonURL:       defaultHorizonURL,
		Network:          defaultNetwork,
		MinDepositAmount: minDep,
	}
}

func ConfigFromEnv() Config {
	cfg := DefaultConfig()

	if v := os.Getenv("STELLAR_HORIZON"); v != "" {
		cfg.HorizonURL = v
	}

	if v := os.Getenv("STELLAR_NETWORK"); v != "" {
		cfg.Network = v
	}

	cfg.HotWalletAddress = os.Getenv("STELLAR_HOT_WALLET_ADDRESS")

	cfg.WalletSecretKey = os.Getenv("WALLET_SECRET_KEY")

	if v := os.Getenv("STELLAR_DEPOSIT_MIN_AMOUNT"); v != "" {
		d, parseErr := decimal.NewFromString(v)
		switch {
		case parseErr != nil:
			slog.Warn(
				"invalid STELLAR_DEPOSIT_MIN_AMOUNT, using default",
				slog.String("value", v),
				slog.String("error", parseErr.Error()),
			)
		case d.IsNegative():
			slog.Warn(
				"STELLAR_DEPOSIT_MIN_AMOUNT must be non-negative, using default",
				slog.String("value", v),
			)
		default:
			cfg.MinDepositAmount = d
		}
	}

	return cfg
}
