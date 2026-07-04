package settlement

import (
	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TransactionStatus string

const (
	TransactionPending TransactionStatus = "pending"
	TransactionSuccess TransactionStatus = "success"
	TransactionFailed  TransactionStatus = "failed"
)

type ProviderPayout struct {
	ProviderID    uuid.UUID           `json:"provider_id"`
	Amount        decimal.Decimal     `json:"amount"`
	Currency      repository.Currency `json:"currency"`
	WalletAddress string              `json:"wallet_address"`
}

type PayoutResult struct {
	ProviderID uuid.UUID         `json:"provider_id"`
	TxHash     string            `json:"tx_hash,omitempty"`
	Error      string            `json:"error,omitempty"`
	Status     TransactionStatus `json:"status"`
}
