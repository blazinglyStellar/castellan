package settlement

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"castellan/internal/repository/db"
	"castellan/internal/stellar"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stellar/go-stellar-sdk/clients/horizonclient"
	"github.com/stellar/go-stellar-sdk/keypair"
	"github.com/stellar/go-stellar-sdk/txnbuild"
)

var ErrTxSequenceConsumed = errors.New("transaction sequence already consumed on Stellar, status unknown")

const (
	retryMaxAttempts     = 3
	retryBaseDelay       = 200 * time.Millisecond
	retryMaxDelay        = 3 * time.Second
	hashLookupMaxResults = 50
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
	_ uuid.UUID,
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

		txHash, subErr := s.submitSinglePayout(ctx, fullKP, payout)
		if subErr != nil {
			if errors.Is(subErr, ErrTxSequenceConsumed) {
				s.log.WarnContext(
					ctx, "payout sequence consumed, marking as pending for recovery",
					slog.String("provider_id", payout.ProviderID.String()),
				)

				result.Error = subErr.Error()
				result.Status = TransactionPending
				results = append(results, result)
				continue
			}

			result.Error = subErr.Error()
			result.Status = TransactionFailed
			results = append(results, result)
			continue
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

	account, err := s.horizon.AccountDetail(horizonclient.AccountRequest{
		AccountID: s.stellarCfg.HotWalletAddress,
	})
	if err != nil {
		return "", fmt.Errorf("load source account: %w", err)
	}

	seqNum, err := account.GetSequenceNumber()
	if err != nil {
		return "", fmt.Errorf("get sequence number: %w", err)
	}

	sourceAccount := txnbuild.NewSimpleAccount(
		s.stellarCfg.HotWalletAddress,
		seqNum,
	)

	tx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &sourceAccount,
		Operations:           []txnbuild.Operation{paymentOp},
		BaseFee:              txnbuild.MinBaseFee,
		Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewInfiniteTimeout()},
		IncrementSequenceNum: true,
	})
	if err != nil {
		return "", fmt.Errorf("build transaction: %w", err)
	}

	networkPassphrase := s.stellarCfg.NetworkPassphrase()
	tx, err = tx.Sign(networkPassphrase, kp)
	if err != nil {
		return "", fmt.Errorf("sign transaction: %w", err)
	}

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

		resp, err := s.horizon.SubmitTransaction(tx)
		if err != nil {
			if isTxBadSeq(err) {
				if attempt == 0 {
					s.log.WarnContext(
						ctx, "sequence already consumed by concurrent process, rebuilding tx",
						slog.String("provider_id", payout.ProviderID.String()),
						slog.Int64("sequence", seqNum),
					)

					account, loadErr := s.horizon.AccountDetail(horizonclient.AccountRequest{
						AccountID: s.stellarCfg.HotWalletAddress,
					})
					if loadErr != nil {
						lastErr = fmt.Errorf("re-fetch account after tx_bad_seq: %w", loadErr)
						continue
					}

					newSeq, seqErr := account.GetSequenceNumber()
					if seqErr != nil {
						lastErr = fmt.Errorf("re-fetch sequence after tx_bad_seq: %w", seqErr)
						continue
					}

					newSource := txnbuild.NewSimpleAccount(s.stellarCfg.HotWalletAddress, newSeq)
					newTx, buildErr := txnbuild.NewTransaction(txnbuild.TransactionParams{
						SourceAccount:        &newSource,
						Operations:           []txnbuild.Operation{paymentOp},
						BaseFee:              txnbuild.MinBaseFee,
						Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewInfiniteTimeout()},
						IncrementSequenceNum: true,
					})
					if buildErr != nil {
						lastErr = fmt.Errorf("rebuild tx after tx_bad_seq: %w", buildErr)
						continue
					}

					newTx, signErr := newTx.Sign(networkPassphrase, kp)
					if signErr != nil {
						lastErr = fmt.Errorf("re-sign tx after tx_bad_seq: %w", signErr)
						continue
					}

					tx = newTx
					seqNum = newSeq
					lastErr = fmt.Errorf("sequence consumed by concurrent process, rebuilt tx with seq %d", newSeq)
					continue
				}

				s.log.WarnContext(
					ctx, "sequence already consumed, recovering tx hash",
					slog.String("provider_id", payout.ProviderID.String()),
					slog.Int("attempt", attempt),
					slog.Int64("sequence", seqNum),
				)

				hash, findErr := s.findTxHashBySequence(ctx, payout, seqNum)
				if findErr != nil {
					return "", fmt.Errorf("%w: %w", ErrTxSequenceConsumed, findErr)
				}

				return hash, nil
			}

			lastErr = fmt.Errorf("submit transaction: %w", err)
			continue
		}

		return resp.Hash, nil
	}

	return "", lastErr
}

func (s *StellarSubmitter) findTxHashBySequence(ctx context.Context, payout ProviderPayout, seqNum int64) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", fmt.Errorf("context cancelled during hash lookup: %w", err)
	}

	txReq := horizonclient.TransactionRequest{
		ForAccount: s.stellarCfg.HotWalletAddress,
		Order:      "desc",
		Limit:      hashLookupMaxResults,
	}

	txs, err := s.horizon.Transactions(txReq)
	if err != nil {
		return "", fmt.Errorf("query account transactions: %w", err)
	}

	for _, tx := range txs.Embedded.Records {
		if tx.AccountSequence == seqNum && tx.Successful && tx.Account == s.stellarCfg.HotWalletAddress {
			return tx.Hash, nil
		}
	}

	s.log.WarnContext(
		ctx, "no successful transaction found for sequence",
		slog.Int64("sequence", seqNum),
		slog.String("destination", payout.WalletAddress),
		slog.String("amount", payout.Amount.String()),
	)

	return "", fmt.Errorf("no successful transaction found for sequence %d", seqNum)
}

func isTxBadSeq(err error) bool {
	var hErr *horizonclient.Error
	if !errors.As(err, &hErr) {
		return false
	}

	codes, resultErr := hErr.ResultCodes()
	if resultErr != nil {
		return false
	}

	return strings.Contains(codes.TransactionCode, "tx_bad_seq")
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

func jitteredDelay(attempt int) time.Duration {
	base := float64(retryBaseDelay) * math.Pow(2, float64(attempt-1)) //nolint:mnd
	if base > float64(retryMaxDelay) {
		base = float64(retryMaxDelay)
	}

	// #nosec G404 — weak RNG acceptable for retry jitter
	jitter := rand.Float64() * base * 0.1 //nolint:mnd

	return time.Duration(base + jitter)
}
