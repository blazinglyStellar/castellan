package deposit

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"castellan/internal/repository/db"
	"castellan/internal/stellar"

	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/stellar/go-stellar-sdk/clients/horizonclient"
	"github.com/stellar/go-stellar-sdk/protocols/horizon/operations"
)

const (
	pollInterval        = 30 * time.Second
	horizonPaymentLimit = 100
)

type Watcher struct {
	queries       repository.Querier
	cfg           stellar.Config
	creditHandler *CreditHandler
	horizon       horizonclient.ClientInterface
	memoCache     map[string]string
}

func NewWatcher(queries repository.Querier, cfg stellar.Config, creditHandler *CreditHandler) *Watcher {
	client := horizonclient.Client{
		HorizonURL: cfg.HorizonURL,
		HTTP:       horizonclient.DefaultTestNetClient.HTTP,
	}

	return &Watcher{
		queries:       queries,
		cfg:           cfg,
		creditHandler: creditHandler,
		horizon:       &client,
		memoCache:     make(map[string]string),
	}
}

func (w *Watcher) Run(ctx context.Context) error {
	slog.InfoContext(ctx, "deposit watcher starting",
		slog.String("horizon_url", w.cfg.HorizonURL),
		slog.String("network", w.cfg.Network),
	)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "deposit watcher stopped")
			return context.Canceled
		case <-ticker.C:
			if err := w.poll(ctx); err != nil {
				if !errors.Is(err, context.Canceled) {
					slog.ErrorContext(ctx, "deposit poll failed", slog.String("error", err.Error()))
				}
			}
		}
	}
}

func (w *Watcher) poll(ctx context.Context) error {
	if w.cfg.HotWalletAddress == "" {
		return nil
	}

	cursor, err := w.queries.GetWatcherCursor(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("init cursor: %w", w.queries.UpsertWatcherCursor(ctx, "now"))
		}
		return fmt.Errorf("get watcher cursor: %w", err)
	}

	if cursor.Cursor == "now" {
		return nil
	}

	payments, err := w.horizon.Payments(horizonclient.OperationRequest{
		ForAccount: w.cfg.HotWalletAddress,
		Order:      horizonclient.OrderAsc,
		Cursor:     cursor.Cursor,
		Limit:      horizonPaymentLimit,
	})
	if err != nil {
		return fmt.Errorf("horizon payments: %w", err)
	}

	for _, op := range payments.Embedded.Records {
		payment, ok := op.(operations.Payment)
		if !ok {
			continue
		}

		if payment.To != w.cfg.HotWalletAddress {
			continue
		}

		memo := w.getMemo(ctx, payment.TransactionHash)

		amount, err := decimal.NewFromString(payment.Amount)
		if err != nil {
			continue
		}

		asset := "XLM"
		if payment.Asset.Type != "native" {
			asset = payment.Code
		}

		if err := w.creditHandler.CreditDeposit(ctx, PaymentOperation{
			TxHash:      payment.TransactionHash,
			FromAddress: payment.From,
			Amount:      amount,
			Memo:        memo,
			Asset:       asset,
		}); err != nil {
			return fmt.Errorf("credit deposit: %w", err)
		}

		if err := w.queries.UpsertWatcherCursor(ctx, payment.PagingToken()); err != nil {
			return fmt.Errorf("update cursor: %w", err)
		}
	}

	return nil
}

func (w *Watcher) getMemo(ctx context.Context, txHash string) string {
	if m, ok := w.memoCache[txHash]; ok {
		return m
	}

	tx, err := w.horizon.TransactionDetail(txHash)
	if err != nil {
		slog.WarnContext(ctx, "failed to get transaction details",
			slog.String("tx_hash", txHash),
			slog.String("error", err.Error()),
		)
		return ""
	}

	w.memoCache[txHash] = tx.Memo
	return tx.Memo
}
