package settlement

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"time"

	"castellan/internal/repository/db"
	"castellan/internal/stellar"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stellar/go-stellar-sdk/clients/horizonclient"
	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stellar/go-stellar-sdk/txnbuild"
)

const (
	retryMaxAttempts = 3
	retryBaseDelay   = 200 * time.Millisecond
	retryMaxDelay    = 3 * time.Second
)

type StellarSubmitter struct {
	pool       *pgxpool.Pool
	queries    repository.Querier
	stellarCfg stellar.Config
	horizon    horizonclient.ClientInterface
	log        *slog.Logger
}

func NewStellarSubmitter(
	pool *pgxpool.Pool,
	queries repository.Querier,
	stellarCfg stellar.Config,
) *StellarSubmitter {
	client := horizonclient.Client{
		HorizonURL: stellarCfg.HorizonURL,
		HTTP:       horizonclient.DefaultTestNetClient.HTTP,
	}

	return &StellarSubmitter{
		pool:       pool,
		queries:    queries,
		stellarCfg: stellarCfg,
		horizon:    &client,
		log:        slog.With(slog.String("component", "stellar_submitter")),
	}
}

func (s *StellarSubmitter) SubmitPayouts(
	ctx context.Context,
	batchID uuid.UUID,
	payouts []ProviderPayout,
) ([]PayoutResult, error) {
	kp, err := keypair.Parse(s.stellarCfg.WalletSecretKey)
	if err != nil {
		return nil, fmt.Errorf("parse hot wallet keypair: %w", err)
	}

	fullKP, ok := kp.(*keypair.Full)
	if !ok {
		return nil, errors.New("WALLET_SECRET_KEY is not a secret key")
	}

	results := make([]PayoutResult, 0, len(payouts))

	for _, payout := range payouts {
		result := PayoutResult{ProviderID: payout.ProviderID}

		amountNumeric, err := DecimalToNumeric(payout.Amount)
		if err != nil {
			result.Error = fmt.Sprintf("convert amount: %v", err)
			result.Status = TransactionFailed
			results = append(results, result)
			continue
		}

		entry, err := s.queries.InsertSettlementEntry(ctx, repository.InsertSettlementEntryParams{
			BatchID:       batchID,
			ProviderID:    payout.ProviderID,
			Amount:        amountNumeric,
			Currency:      payout.Currency,
			WalletAddress: payout.WalletAddress,
			Status:        repository.SettlementEntryStatusPending,
		})
		if err != nil {
			result.Error = fmt.Sprintf("insert settlement entry: %v", err)
			result.Status = TransactionFailed
			results = append(results, result)
			continue
		}

		txHash, subErr := s.submitSinglePayout(ctx, fullKP, payout)
		if subErr != nil {
			if updateErr := s.setEntryStatus(ctx, entry.ID, repository.SettlementEntryStatusFailed, ""); updateErr != nil {
				s.log.ErrorContext(
					ctx, "failed to mark entry as failed",
					slog.String("entry_id", entry.ID.String()),
					slog.String("error", updateErr.Error()),
				)
			}

			result.Error = subErr.Error()
			result.Status = TransactionFailed
			results = append(results, result)
			continue
		}

		if updateErr := s.setEntryStatus(ctx, entry.ID, repository.SettlementEntryStatusCompleted, txHash); updateErr != nil {
			s.log.ErrorContext(
				ctx, "failed to mark entry as completed",
				slog.String("entry_id", entry.ID.String()),
				slog.String("tx_hash", txHash),
				slog.String("error", updateErr.Error()),
			)
		}

		result.TxHash = txHash
		result.Status = TransactionSuccess
		results = append(results, result)
	}

	return results, nil
}

func (s *StellarSubmitter) submitSinglePayout(
	ctx context.Context,
	kp *keypair.Full,
	payout ProviderPayout,
) (string, error) {
	paymentOp := &txnbuild.Payment{
		Destination: payout.WalletAddress,
		Amount:      payout.Amount.String(),
		Asset:       txnbuild.NativeAsset{},
	}

	var txHash string
	var lastErr error

	for attempt := range retryMaxAttempts {
		if attempt > 0 {
			delay := jitteredDelay(attempt)
			s.log.InfoContext(
				ctx, "retrying payout submission",
				slog.String("provider_id", payout.ProviderID.String()),
				slog.Int("attempt", attempt+1),
				slog.String("delay", delay.String()),
			)

			select {
			case <-ctx.Done():
				return "", fmt.Errorf("request cancelled during retry delay: %w", ctx.Err())
			case <-time.After(delay):
			}
		}

		account, err := s.horizon.AccountDetail(horizonclient.AccountRequest{
			AccountID: s.stellarCfg.HotWalletAddress,
		})
		if err != nil {
			lastErr = fmt.Errorf("load source account: %w", err)
			continue
		}

		seqNum, err := account.GetSequenceNumber()
		if err != nil {
			lastErr = fmt.Errorf("get sequence number: %w", err)
			continue
		}

		sourceAccount := txnbuild.NewSimpleAccount(
			s.stellarCfg.HotWalletAddress,
			seqNum,
		)

		tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
			SourceAccount: &sourceAccount,
			Operations:    []txnbuild.Operation{paymentOp},
			BaseFee:       txnbuild.MinBaseFee,
			Preconditions: txnbuild.Preconditions{TimeBounds: txnbuild.NewInfiniteTimeout()},
		})
		if err != nil {
			lastErr = fmt.Errorf("build transaction: %w", err)
			continue
		}

		networkPassphrase := s.stellarCfg.NetworkPassphrase()
		tx, err = tx.Sign(networkPassphrase, kp)
		if err != nil {
			lastErr = fmt.Errorf("sign transaction: %w", err)
			continue
		}

		resp, err := s.horizon.SubmitTransaction(tx)
		if err != nil {
			lastErr = fmt.Errorf("submit transaction: %w", err)
			continue
		}

		txHash = resp.Hash
		lastErr = nil
		break
	}

	if lastErr != nil {
		return "", lastErr
	}

	return txHash, nil
}

func (s *StellarSubmitter) MonitorTransaction(
	ctx context.Context, //nolint:revive
	txHash string,
) (TransactionStatus, error) {
	tx, err := s.horizon.TransactionDetail(txHash)
	if err != nil {
		var horizonErr *horizonclient.Error
		if errors.As(err, &horizonErr) && horizonErr.Response.StatusCode == http.StatusNotFound {
			return TransactionFailed, nil
		}

		return TransactionFailed, fmt.Errorf("query transaction: %w", err)
	}

	if tx.Successful {
		return TransactionSuccess, nil
	}

	return TransactionFailed, nil
}

func (s *StellarSubmitter) setEntryStatus(
	ctx context.Context,
	entryID uuid.UUID,
	status repository.SettlementEntryStatus,
	txHash string,
) error {
	if s.pool == nil {
		return nil
	}

	if txHash != "" {
		_, err := s.pool.Exec(
			ctx,
			`UPDATE settlement_entries SET status = $1, tx_hash = $2 WHERE id = $3`,
			status, txHash, entryID,
		)
		if err != nil {
			return fmt.Errorf("update entry status and tx_hash: %w", err)
		}

		return nil
	}

	_, err := s.pool.Exec(
		ctx,
		`UPDATE settlement_entries SET status = $1 WHERE id = $2`,
		status, entryID,
	)
	if err != nil {
		return fmt.Errorf("update entry status: %w", err)
	}

	return nil
}

func jitteredDelay(attempt int) time.Duration {
	base := float64(retryBaseDelay) * math.Pow(2, float64(attempt-1)) //nolint:mnd
	if base > float64(retryMaxDelay) {
		base = float64(retryMaxDelay)
	}

	// #nosec G404 — weak RNG acceptable for retry jitter
	jitter := rand.Float64() * base * 0.1 //nolint:mnd

	return time.Duration(base + jitter)
}
