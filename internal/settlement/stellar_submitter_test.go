package settlement

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"testing"

	"castellan/internal/repository/db"
	"castellan/internal/stellar"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stellar/go-stellar-sdk/clients/horizonclient"
	"github.com/stellar/go-stellar-sdk/keypair"
	hProtocol "github.com/stellar/go-stellar-sdk/protocols/horizon"
	"github.com/stellar/go-stellar-sdk/protocols/horizon/effects"
	"github.com/stellar/go-stellar-sdk/protocols/horizon/operations"
	"github.com/stellar/go-stellar-sdk/txnbuild"
)

type mockHorizonClient struct {
	accountDetailFunc     func(horizonclient.AccountRequest) (hProtocol.Account, error)
	submitTransactionFunc func(*txnbuild.Transaction) (hProtocol.Transaction, error)
	transactionDetailFunc func(string) (hProtocol.Transaction, error)
}

func (m *mockHorizonClient) AccountDetail(request horizonclient.AccountRequest) (hProtocol.Account, error) {
	if m.accountDetailFunc != nil {
		return m.accountDetailFunc(request)
	}
	return hProtocol.Account{}, errors.New("unexpected call to AccountDetail")
}

func (m *mockHorizonClient) SubmitTransaction(tx *txnbuild.Transaction) (hProtocol.Transaction, error) {
	if m.submitTransactionFunc != nil {
		return m.submitTransactionFunc(tx)
	}
	return hProtocol.Transaction{}, errors.New("unexpected call to SubmitTransaction")
}

func (m *mockHorizonClient) TransactionDetail(txHash string) (hProtocol.Transaction, error) {
	if m.transactionDetailFunc != nil {
		return m.transactionDetailFunc(txHash)
	}
	return hProtocol.Transaction{}, errors.New("unexpected call to TransactionDetail")
}

func (m *mockHorizonClient) Accounts(horizonclient.AccountsRequest) (hProtocol.AccountsPage, error) {
	return hProtocol.AccountsPage{}, errors.New("unexpected Accounts")
}

func (m *mockHorizonClient) AccountData(horizonclient.AccountRequest) (hProtocol.AccountData, error) {
	return hProtocol.AccountData{}, errors.New("unexpected AccountData")
}

func (m *mockHorizonClient) Effects(horizonclient.EffectRequest) (effects.EffectsPage, error) {
	return effects.EffectsPage{}, errors.New("unexpected Effects")
}

func (m *mockHorizonClient) Assets(horizonclient.AssetRequest) (hProtocol.AssetsPage, error) {
	return hProtocol.AssetsPage{}, errors.New("unexpected Assets")
}

func (m *mockHorizonClient) Ledgers(horizonclient.LedgerRequest) (hProtocol.LedgersPage, error) {
	return hProtocol.LedgersPage{}, errors.New("unexpected Ledgers")
}

func (m *mockHorizonClient) LedgerDetail(uint32) (hProtocol.Ledger, error) {
	return hProtocol.Ledger{}, errors.New("unexpected LedgerDetail")
}

func (m *mockHorizonClient) FeeStats() (hProtocol.FeeStats, error) {
	return hProtocol.FeeStats{}, errors.New("unexpected FeeStats")
}

func (m *mockHorizonClient) Offers(horizonclient.OfferRequest) (hProtocol.OffersPage, error) {
	return hProtocol.OffersPage{}, errors.New("unexpected Offers")
}

func (m *mockHorizonClient) OfferDetails(string) (hProtocol.Offer, error) {
	return hProtocol.Offer{}, errors.New("unexpected OfferDetails")
}

func (m *mockHorizonClient) Operations(horizonclient.OperationRequest) (operations.OperationsPage, error) {
	return operations.OperationsPage{}, errors.New("unexpected Operations")
}

func (m *mockHorizonClient) OperationDetail(string) (operations.Operation, error) {
	return nil, errors.New("unexpected OperationDetail")
}

func (m *mockHorizonClient) SubmitTransactionXDR(string) (hProtocol.Transaction, error) {
	return hProtocol.Transaction{}, errors.New("unexpected SubmitTransactionXDR")
}

func (m *mockHorizonClient) SubmitFeeBumpTransactionWithOptions(*txnbuild.FeeBumpTransaction, horizonclient.SubmitTxOpts) (hProtocol.Transaction, error) {
	return hProtocol.Transaction{}, errors.New("unexpected SubmitFeeBumpTransactionWithOptions")
}

func (m *mockHorizonClient) SubmitTransactionWithOptions(*txnbuild.Transaction, horizonclient.SubmitTxOpts) (hProtocol.Transaction, error) {
	return hProtocol.Transaction{}, errors.New("unexpected SubmitTransactionWithOptions")
}

func (m *mockHorizonClient) SubmitFeeBumpTransaction(*txnbuild.FeeBumpTransaction) (hProtocol.Transaction, error) {
	return hProtocol.Transaction{}, errors.New("unexpected SubmitFeeBumpTransaction")
}

func (m *mockHorizonClient) AsyncSubmitTransactionXDR(string) (hProtocol.AsyncTransactionSubmissionResponse, error) {
	return hProtocol.AsyncTransactionSubmissionResponse{}, errors.New("unexpected AsyncSubmitTransactionXDR")
}

func (m *mockHorizonClient) AsyncSubmitFeeBumpTransactionWithOptions(*txnbuild.FeeBumpTransaction, horizonclient.SubmitTxOpts) (hProtocol.AsyncTransactionSubmissionResponse, error) {
	return hProtocol.AsyncTransactionSubmissionResponse{}, errors.New("unexpected AsyncSubmitFeeBumpTransactionWithOptions")
}

func (m *mockHorizonClient) AsyncSubmitTransactionWithOptions(*txnbuild.Transaction, horizonclient.SubmitTxOpts) (hProtocol.AsyncTransactionSubmissionResponse, error) {
	return hProtocol.AsyncTransactionSubmissionResponse{}, errors.New("unexpected AsyncSubmitTransactionWithOptions")
}

func (m *mockHorizonClient) AsyncSubmitFeeBumpTransaction(*txnbuild.FeeBumpTransaction) (hProtocol.AsyncTransactionSubmissionResponse, error) {
	return hProtocol.AsyncTransactionSubmissionResponse{}, errors.New("unexpected AsyncSubmitFeeBumpTransaction")
}

func (m *mockHorizonClient) AsyncSubmitTransaction(*txnbuild.Transaction) (hProtocol.AsyncTransactionSubmissionResponse, error) {
	return hProtocol.AsyncTransactionSubmissionResponse{}, errors.New("unexpected AsyncSubmitTransaction")
}

func (m *mockHorizonClient) Transactions(horizonclient.TransactionRequest) (hProtocol.TransactionsPage, error) {
	return hProtocol.TransactionsPage{}, errors.New("unexpected Transactions")
}

func (m *mockHorizonClient) OrderBook(horizonclient.OrderBookRequest) (hProtocol.OrderBookSummary, error) {
	return hProtocol.OrderBookSummary{}, errors.New("unexpected OrderBook")
}

func (m *mockHorizonClient) Paths(horizonclient.PathsRequest) (hProtocol.PathsPage, error) {
	return hProtocol.PathsPage{}, errors.New("unexpected Paths")
}

func (m *mockHorizonClient) Payments(horizonclient.OperationRequest) (operations.OperationsPage, error) {
	return operations.OperationsPage{}, errors.New("unexpected Payments")
}

func (m *mockHorizonClient) TradeAggregations(horizonclient.TradeAggregationRequest) (hProtocol.TradeAggregationsPage, error) {
	return hProtocol.TradeAggregationsPage{}, errors.New("unexpected TradeAggregations")
}

func (m *mockHorizonClient) Trades(horizonclient.TradeRequest) (hProtocol.TradesPage, error) {
	return hProtocol.TradesPage{}, errors.New("unexpected Trades")
}

func (m *mockHorizonClient) Fund(string) (hProtocol.Transaction, error) {
	return hProtocol.Transaction{}, errors.New("unexpected Fund")
}

func (m *mockHorizonClient) StreamTransactions(context.Context, horizonclient.TransactionRequest, horizonclient.TransactionHandler) error {
	return errors.New("unexpected StreamTransactions")
}

func (m *mockHorizonClient) StreamTrades(context.Context, horizonclient.TradeRequest, horizonclient.TradeHandler) error {
	return errors.New("unexpected StreamTrades")
}

func (m *mockHorizonClient) StreamEffects(context.Context, horizonclient.EffectRequest, horizonclient.EffectHandler) error {
	return errors.New("unexpected StreamEffects")
}

func (m *mockHorizonClient) StreamOperations(context.Context, horizonclient.OperationRequest, horizonclient.OperationHandler) error {
	return errors.New("unexpected StreamOperations")
}

func (m *mockHorizonClient) StreamPayments(context.Context, horizonclient.OperationRequest, horizonclient.OperationHandler) error {
	return errors.New("unexpected StreamPayments")
}

func (m *mockHorizonClient) StreamOffers(context.Context, horizonclient.OfferRequest, horizonclient.OfferHandler) error {
	return errors.New("unexpected StreamOffers")
}

func (m *mockHorizonClient) StreamLedgers(context.Context, horizonclient.LedgerRequest, horizonclient.LedgerHandler) error {
	return errors.New("unexpected StreamLedgers")
}

func (m *mockHorizonClient) StreamOrderBooks(context.Context, horizonclient.OrderBookRequest, horizonclient.OrderBookHandler) error {
	return errors.New("unexpected StreamOrderBooks")
}

func (m *mockHorizonClient) Root() (hProtocol.Root, error) {
	return hProtocol.Root{}, errors.New("unexpected Root")
}

func (m *mockHorizonClient) NextAccountsPage(hProtocol.AccountsPage) (hProtocol.AccountsPage, error) {
	return hProtocol.AccountsPage{}, errors.New("unexpected NextAccountsPage")
}

func (m *mockHorizonClient) NextAssetsPage(hProtocol.AssetsPage) (hProtocol.AssetsPage, error) {
	return hProtocol.AssetsPage{}, errors.New("unexpected NextAssetsPage")
}

func (m *mockHorizonClient) PrevAssetsPage(hProtocol.AssetsPage) (hProtocol.AssetsPage, error) {
	return hProtocol.AssetsPage{}, errors.New("unexpected PrevAssetsPage")
}

func (m *mockHorizonClient) NextLedgersPage(hProtocol.LedgersPage) (hProtocol.LedgersPage, error) {
	return hProtocol.LedgersPage{}, errors.New("unexpected NextLedgersPage")
}

func (m *mockHorizonClient) PrevLedgersPage(hProtocol.LedgersPage) (hProtocol.LedgersPage, error) {
	return hProtocol.LedgersPage{}, errors.New("unexpected PrevLedgersPage")
}

func (m *mockHorizonClient) NextEffectsPage(effects.EffectsPage) (effects.EffectsPage, error) {
	return effects.EffectsPage{}, errors.New("unexpected NextEffectsPage")
}

func (m *mockHorizonClient) PrevEffectsPage(effects.EffectsPage) (effects.EffectsPage, error) {
	return effects.EffectsPage{}, errors.New("unexpected PrevEffectsPage")
}

func (m *mockHorizonClient) NextTransactionsPage(hProtocol.TransactionsPage) (hProtocol.TransactionsPage, error) {
	return hProtocol.TransactionsPage{}, errors.New("unexpected NextTransactionsPage")
}

func (m *mockHorizonClient) PrevTransactionsPage(hProtocol.TransactionsPage) (hProtocol.TransactionsPage, error) {
	return hProtocol.TransactionsPage{}, errors.New("unexpected PrevTransactionsPage")
}

func (m *mockHorizonClient) NextOperationsPage(operations.OperationsPage) (operations.OperationsPage, error) {
	return operations.OperationsPage{}, errors.New("unexpected NextOperationsPage")
}

func (m *mockHorizonClient) PrevOperationsPage(operations.OperationsPage) (operations.OperationsPage, error) {
	return operations.OperationsPage{}, errors.New("unexpected PrevOperationsPage")
}

func (m *mockHorizonClient) NextPaymentsPage(operations.OperationsPage) (operations.OperationsPage, error) {
	return operations.OperationsPage{}, errors.New("unexpected NextPaymentsPage")
}

func (m *mockHorizonClient) PrevPaymentsPage(operations.OperationsPage) (operations.OperationsPage, error) {
	return operations.OperationsPage{}, errors.New("unexpected PrevPaymentsPage")
}

func (m *mockHorizonClient) NextOffersPage(hProtocol.OffersPage) (hProtocol.OffersPage, error) {
	return hProtocol.OffersPage{}, errors.New("unexpected NextOffersPage")
}

func (m *mockHorizonClient) PrevOffersPage(hProtocol.OffersPage) (hProtocol.OffersPage, error) {
	return hProtocol.OffersPage{}, errors.New("unexpected PrevOffersPage")
}

func (m *mockHorizonClient) NextTradesPage(hProtocol.TradesPage) (hProtocol.TradesPage, error) {
	return hProtocol.TradesPage{}, errors.New("unexpected NextTradesPage")
}

func (m *mockHorizonClient) PrevTradesPage(hProtocol.TradesPage) (hProtocol.TradesPage, error) {
	return hProtocol.TradesPage{}, errors.New("unexpected PrevTradesPage")
}

func (m *mockHorizonClient) HomeDomainForAccount(string) (string, error) {
	return "", errors.New("unexpected HomeDomainForAccount")
}

func (m *mockHorizonClient) NextTradeAggregationsPage(hProtocol.TradeAggregationsPage) (hProtocol.TradeAggregationsPage, error) {
	return hProtocol.TradeAggregationsPage{}, errors.New("unexpected NextTradeAggregationsPage")
}

func (m *mockHorizonClient) PrevTradeAggregationsPage(hProtocol.TradeAggregationsPage) (hProtocol.TradeAggregationsPage, error) {
	return hProtocol.TradeAggregationsPage{}, errors.New("unexpected PrevTradeAggregationsPage")
}

func (m *mockHorizonClient) LiquidityPoolDetail(horizonclient.LiquidityPoolRequest) (hProtocol.LiquidityPool, error) {
	return hProtocol.LiquidityPool{}, errors.New("unexpected LiquidityPoolDetail")
}

func (m *mockHorizonClient) LiquidityPools(horizonclient.LiquidityPoolsRequest) (hProtocol.LiquidityPoolsPage, error) {
	return hProtocol.LiquidityPoolsPage{}, errors.New("unexpected LiquidityPools")
}

func (m *mockHorizonClient) NextLiquidityPoolsPage(hProtocol.LiquidityPoolsPage) (hProtocol.LiquidityPoolsPage, error) {
	return hProtocol.LiquidityPoolsPage{}, errors.New("unexpected NextLiquidityPoolsPage")
}

func (m *mockHorizonClient) PrevLiquidityPoolsPage(hProtocol.LiquidityPoolsPage) (hProtocol.LiquidityPoolsPage, error) {
	return hProtocol.LiquidityPoolsPage{}, errors.New("unexpected PrevLiquidityPoolsPage")
}

var testSubmitterCfg = stellar.Config{
	HotWalletAddress: "GAJGJ3LNNS223JOUZPYCDT6SHQKETKXZNGWPBNMBIAPULBPZBSSC4LWU",
	WalletSecretKey:  "SDOIUDYACC4BZH6ZKY7YT4EGI3YAWPEFYZEMIR5HOD5GUTFK7KGX3G2G",
	Network:          "testnet",
	HorizonURL:       "https://horizon-testnet.stellar.org",
}

func newTestSubmitter(horizon horizonclient.ClientInterface, q repository.Querier) *StellarSubmitter {
	return &StellarSubmitter{
		queries:    q,
		stellarCfg: testSubmitterCfg,
		horizon:    horizon,
		log:        slog.New(slog.DiscardHandler),
	}
}

func testPayout() ProviderPayout {
	return ProviderPayout{
		ProviderID:    uuid.New(),
		Amount:        decimal.NewFromInt(10),
		Currency:      repository.CurrencyXLM,
		WalletAddress: "GASCW3JHYTXFIDK7UJ6MZY4P5YDFEH6CK4ISSOEPYDAD7NA27YNGUVAT",
	}
}

type mockSubmitterQuerier struct {
	repository.Querier
	insertSettlementEntryFunc func(context.Context, repository.InsertSettlementEntryParams) (repository.SettlementEntry, error)
}

func (m *mockSubmitterQuerier) InsertSettlementEntry(ctx context.Context, arg repository.InsertSettlementEntryParams) (repository.SettlementEntry, error) {
	if m.insertSettlementEntryFunc != nil {
		return m.insertSettlementEntryFunc(ctx, arg)
	}
	return repository.SettlementEntry{}, errors.New("unexpected call to InsertSettlementEntry")
}

func TestSubmitSinglePayout_BuildsCorrectTx(t *testing.T) {
	t.Parallel()

	horizon := &mockHorizonClient{
		accountDetailFunc: func(_ horizonclient.AccountRequest) (hProtocol.Account, error) {
			return hProtocol.Account{
				AccountID: testSubmitterCfg.HotWalletAddress,
				Sequence:  1234,
			}, nil
		},
		submitTransactionFunc: func(tx *txnbuild.Transaction) (hProtocol.Transaction, error) {
			ops := tx.Operations()
			if len(ops) != 1 {
				t.Fatalf("expected 1 operation, got %d", len(ops))
			}

			payment, ok := ops[0].(*txnbuild.Payment)
			if !ok {
				t.Fatalf("expected Payment operation, got %T", ops[0])
			}

			if payment.Destination != "GASCW3JHYTXFIDK7UJ6MZY4P5YDFEH6CK4ISSOEPYDAD7NA27YNGUVAT" {
				t.Errorf("destination = %q, want GASCW3...UVAT", payment.Destination)
			}

			if payment.Amount != "10" {
				t.Errorf("amount = %q, want 10", payment.Amount)
			}

			return hProtocol.Transaction{
				Hash: "abc123def456",
			}, nil
		},
	}

	kp, err := keypair.Parse(testSubmitterCfg.WalletSecretKey)
	if err != nil {
		t.Fatalf("parse keypair: %v", err)
	}
	fullKP, ok := kp.(*keypair.Full)
	if !ok {
		t.Fatal("keypair is not a Full keypair")
	}

	submitter := newTestSubmitter(horizon, nil)
	txHash, err := submitter.submitSinglePayout(context.Background(), fullKP, testPayout())
	if err != nil {
		t.Fatalf("submitSinglePayout: %v", err)
	}

	if txHash != "abc123def456" {
		t.Errorf("txHash = %q, want abc123def456", txHash)
	}
}

func TestSubmitSinglePayout_SignsWithHotWallet(t *testing.T) {
	t.Parallel()

	var capturedTx *txnbuild.Transaction
	horizon := &mockHorizonClient{
		accountDetailFunc: func(_ horizonclient.AccountRequest) (hProtocol.Account, error) {
			return hProtocol.Account{
				AccountID: testSubmitterCfg.HotWalletAddress,
				Sequence:  5678,
			}, nil
		},
		submitTransactionFunc: func(tx *txnbuild.Transaction) (hProtocol.Transaction, error) {
			capturedTx = tx
			return hProtocol.Transaction{Hash: "signed-tx-hash"}, nil
		},
	}

	kp, err := keypair.Parse(testSubmitterCfg.WalletSecretKey)
	if err != nil {
		t.Fatalf("parse keypair: %v", err)
	}
	fullKP, ok := kp.(*keypair.Full)
	if !ok {
		t.Fatal("keypair is not a Full keypair")
	}

	submitter := newTestSubmitter(horizon, nil)
	_, err = submitter.submitSinglePayout(context.Background(), fullKP, testPayout())
	if err != nil {
		t.Fatalf("submitSinglePayout: %v", err)
	}

	if capturedTx == nil {
		t.Fatal("transaction was not submitted")
	}

	fromAddr := capturedTx.SourceAccount().AccountID

	if fromAddr != testSubmitterCfg.HotWalletAddress {
		t.Errorf("source account = %q, want %q", fromAddr, testSubmitterCfg.HotWalletAddress)
	}
}

func TestSubmitSinglePayout_RetryOnTransientError(t *testing.T) {
	t.Parallel()

	var submitCount int
	horizon := &mockHorizonClient{
		accountDetailFunc: func(_ horizonclient.AccountRequest) (hProtocol.Account, error) {
			return hProtocol.Account{
				AccountID: testSubmitterCfg.HotWalletAddress,
				Sequence:  1234,
			}, nil
		},
		submitTransactionFunc: func(tx *txnbuild.Transaction) (hProtocol.Transaction, error) {
			submitCount++
			if submitCount < retryMaxAttempts {
				return hProtocol.Transaction{}, errors.New("horizon timeout")
			}
			return hProtocol.Transaction{Hash: "retry-success-hash"}, nil
		},
	}

	kp, err := keypair.Parse(testSubmitterCfg.WalletSecretKey)
	if err != nil {
		t.Fatalf("parse keypair: %v", err)
	}
	fullKP, ok := kp.(*keypair.Full)
	if !ok {
		t.Fatal("keypair is not a Full keypair")
	}

	submitter := newTestSubmitter(horizon, nil)
	txHash, err := submitter.submitSinglePayout(context.Background(), fullKP, testPayout())
	if err != nil {
		t.Fatalf("submitSinglePayout: %v", err)
	}

	if txHash != "retry-success-hash" {
		t.Errorf("txHash = %q, want retry-success-hash", txHash)
	}
}

func TestSubmitSinglePayout_RetryExhausted(t *testing.T) {
	t.Parallel()

	var submitCount int
	horizon := &mockHorizonClient{
		accountDetailFunc: func(_ horizonclient.AccountRequest) (hProtocol.Account, error) {
			return hProtocol.Account{
				AccountID: testSubmitterCfg.HotWalletAddress,
				Sequence:  1234,
			}, nil
		},
		submitTransactionFunc: func(tx *txnbuild.Transaction) (hProtocol.Transaction, error) {
			submitCount++
			return hProtocol.Transaction{}, errors.New("persistent horizon error")
		},
	}

	kp, err := keypair.Parse(testSubmitterCfg.WalletSecretKey)
	if err != nil {
		t.Fatalf("parse keypair: %v", err)
	}
	fullKP, ok := kp.(*keypair.Full)
	if !ok {
		t.Fatal("keypair is not a Full keypair")
	}

	submitter := newTestSubmitter(horizon, nil)
	_, err = submitter.submitSinglePayout(context.Background(), fullKP, testPayout())
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}

	if submitCount != retryMaxAttempts {
		t.Errorf("expected %d attempts, got %d", retryMaxAttempts, submitCount)
	}
}

func TestSubmitPayouts_FailureDoesNotAbortBatch(t *testing.T) {
	t.Parallel()

	payout1 := testPayout()
	payout2 := testPayout()
	payout3 := testPayout()

	var attemptCount int
	horizon := &mockHorizonClient{
		accountDetailFunc: func(_ horizonclient.AccountRequest) (hProtocol.Account, error) {
			return hProtocol.Account{
				AccountID: testSubmitterCfg.HotWalletAddress,
				Sequence:  9999,
			}, nil
		},
		submitTransactionFunc: func(tx *txnbuild.Transaction) (hProtocol.Transaction, error) {
			attemptCount++
			if attemptCount >= 2 && attemptCount <= 4 {
				return hProtocol.Transaction{}, errors.New("submission failed")
			}
			return hProtocol.Transaction{
				Hash: fmt.Sprintf("hash-%d", attemptCount),
			}, nil
		},
	}

	q := &mockSubmitterQuerier{
		insertSettlementEntryFunc: func(_ context.Context, _ repository.InsertSettlementEntryParams) (repository.SettlementEntry, error) {
			return repository.SettlementEntry{ID: uuid.New()}, nil
		},
	}

	submitter := newTestSubmitter(horizon, q)
	results, err := submitter.SubmitPayouts(context.Background(), uuid.New(), []ProviderPayout{payout1, payout2, payout3})
	if err != nil {
		t.Fatalf("SubmitPayouts: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	if results[0].Status != TransactionSuccess {
		t.Errorf("payout1 status = %q, want %q", results[0].Status, TransactionSuccess)
	}
	if results[1].Status != TransactionFailed {
		t.Errorf("payout2 status = %q, want %q", results[1].Status, TransactionFailed)
	}
	if results[2].Status != TransactionSuccess {
		t.Errorf("payout3 status = %q, want %q", results[2].Status, TransactionSuccess)
	}

	if attemptCount < 5 {
		t.Errorf("expected at least 5 total submission attempts (1+3+1), got %d", attemptCount)
	}
}

func TestMonitorTransaction_Success(t *testing.T) {
	t.Parallel()

	horizon := &mockHorizonClient{
		transactionDetailFunc: func(hash string) (hProtocol.Transaction, error) {
			if hash != "known-tx-hash" {
				t.Errorf("hash = %q, want known-tx-hash", hash)
			}
			return hProtocol.Transaction{Successful: true}, nil
		},
	}

	submitter := newTestSubmitter(horizon, nil)
	status, err := submitter.MonitorTransaction(context.Background(), "known-tx-hash")
	if err != nil {
		t.Fatalf("MonitorTransaction: %v", err)
	}

	if status != TransactionSuccess {
		t.Errorf("status = %q, want %q", status, TransactionSuccess)
	}
}

func TestMonitorTransaction_Failed(t *testing.T) {
	t.Parallel()

	horizon := &mockHorizonClient{
		transactionDetailFunc: func(hash string) (hProtocol.Transaction, error) {
			return hProtocol.Transaction{Successful: false}, nil
		},
	}

	submitter := newTestSubmitter(horizon, nil)
	status, err := submitter.MonitorTransaction(context.Background(), "failed-tx-hash")
	if err != nil {
		t.Fatalf("MonitorTransaction: %v", err)
	}

	if status != TransactionFailed {
		t.Errorf("status = %q, want %q", status, TransactionFailed)
	}
}

func TestMonitorTransaction_NotFound(t *testing.T) {
	t.Parallel()

	horizon := &mockHorizonClient{
		transactionDetailFunc: func(hash string) (hProtocol.Transaction, error) {
			return hProtocol.Transaction{}, &horizonclient.Error{
				Response: &http.Response{StatusCode: http.StatusNotFound},
			}
		},
	}

	submitter := newTestSubmitter(horizon, nil)
	status, err := submitter.MonitorTransaction(context.Background(), "unknown-tx")
	if err != nil {
		t.Fatalf("MonitorTransaction: %v", err)
	}

	if status != TransactionFailed {
		t.Errorf("status = %q, want %q", status, TransactionFailed)
	}
}

func TestMonitorTransaction_HorizonError(t *testing.T) {
	t.Parallel()

	horizon := &mockHorizonClient{
		transactionDetailFunc: func(hash string) (hProtocol.Transaction, error) {
			return hProtocol.Transaction{}, errors.New("network error")
		},
	}

	submitter := newTestSubmitter(horizon, nil)
	_, err := submitter.MonitorTransaction(context.Background(), "unknown")
	if err == nil {
		t.Fatal("expected error for network failure, got nil")
	}
}
