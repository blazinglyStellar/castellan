package settlement

import (
	"context"
	"errors"
	"testing"
	"time"

	"castellan/internal/repository/db"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type mockAggregator struct {
	aggregateFunc func(context.Context) ([]ProviderPayout, error)
}

func (m *mockAggregator) Aggregate(ctx context.Context) ([]ProviderPayout, error) {
	if m.aggregateFunc != nil {
		return m.aggregateFunc(ctx)
	}

	return nil, errors.New("unexpected call to Aggregate")
}

type mockSubmitter struct {
	submitPayoutsFunc func(context.Context, uuid.UUID, []ProviderPayout) ([]PayoutResult, error)
	monitorTxFunc     func(context.Context, string) (TransactionStatus, error)
}

func (m *mockSubmitter) SubmitPayouts(ctx context.Context, batchID uuid.UUID, payouts []ProviderPayout) ([]PayoutResult, error) {
	if m.submitPayoutsFunc != nil {
		return m.submitPayoutsFunc(ctx, batchID, payouts)
	}

	return nil, errors.New("unexpected call to SubmitPayouts")
}

func (m *mockSubmitter) MonitorTransaction(ctx context.Context, txHash string) (TransactionStatus, error) {
	if m.monitorTxFunc != nil {
		return m.monitorTxFunc(ctx, txHash)
	}

	return TransactionFailed, errors.New("unexpected call to MonitorTransaction")
}

type mockReconciler struct {
	recoverFailedFunc func(context.Context, TransactionMonitor) error
	createBatchFunc   func(context.Context, []ProviderPayout) (uuid.UUID, []repository.SettlementEntry, error)
	finalizeBatchFunc func(context.Context, uuid.UUID, []PayoutResult) error
}

func (m *mockReconciler) RecoverFailed(ctx context.Context, monitor TransactionMonitor) error {
	if m.recoverFailedFunc != nil {
		return m.recoverFailedFunc(ctx, monitor)
	}

	return errors.New("unexpected call to RecoverFailed")
}

func (m *mockReconciler) CreateBatch(ctx context.Context, payouts []ProviderPayout) (uuid.UUID, []repository.SettlementEntry, error) {
	if m.createBatchFunc != nil {
		return m.createBatchFunc(ctx, payouts)
	}

	return uuid.Nil, nil, errors.New("unexpected call to CreateBatch")
}

func (m *mockReconciler) FinalizeBatch(ctx context.Context, batchID uuid.UUID, results []PayoutResult) error {
	if m.finalizeBatchFunc != nil {
		return m.finalizeBatchFunc(ctx, batchID, results)
	}

	return errors.New("unexpected call to FinalizeBatch")
}

func newTestCycle(agg payoutAggregator, sub payoutSubmitter, rec cycleReconciler) *settlementCycle {
	return &settlementCycle{
		aggregator: agg,
		submitter:  sub,
		reconciler: rec,
	}
}

func TestFullCycle_Success(t *testing.T) {
	t.Parallel()

	batchID := uuid.New()
	payouts := []ProviderPayout{
		{ProviderID: uuid.New(), Amount: decimal.NewFromInt(100), Currency: repository.CurrencyXLM, WalletAddress: "GAONE"},
		{ProviderID: uuid.New(), Amount: decimal.NewFromInt(200), Currency: repository.CurrencyXLM, WalletAddress: "GATWO"},
	}

	agg := &mockAggregator{
		aggregateFunc: func(_ context.Context) ([]ProviderPayout, error) {
			return payouts, nil
		},
	}

	sub := &mockSubmitter{
		submitPayoutsFunc: func(_ context.Context, id uuid.UUID, p []ProviderPayout) ([]PayoutResult, error) {
			if id != batchID {
				t.Errorf("batchID = %v, want %v", id, batchID)
			}

			results := make([]PayoutResult, len(p))
			for i := range p {
				results[i] = PayoutResult{ProviderID: p[i].ProviderID, TxHash: "txhash", Status: TransactionSuccess}
			}

			return results, nil
		},
	}

	rec := &mockReconciler{
		recoverFailedFunc: func(_ context.Context, _ TransactionMonitor) error {
			return nil
		},
		createBatchFunc: func(_ context.Context, p []ProviderPayout) (uuid.UUID, []repository.SettlementEntry, error) {
			if len(p) != 2 {
				t.Errorf("payouts count = %d, want 2", len(p))
			}

			return batchID, nil, nil
		},
		finalizeBatchFunc: func(_ context.Context, id uuid.UUID, results []PayoutResult) error {
			if id != batchID {
				t.Errorf("batchID = %v, want %v", id, batchID)
			}
			if len(results) != 2 {
				t.Errorf("results count = %d, want 2", len(results))
			}

			return nil
		},
	}

	cycle := newTestCycle(agg, sub, rec)
	err := cycle.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestNoopCycle_NoPayouts(t *testing.T) {
	t.Parallel()

	agg := &mockAggregator{
		aggregateFunc: func(_ context.Context) ([]ProviderPayout, error) {
			return nil, nil
		},
	}

	sub := &mockSubmitter{}
	rec := &mockReconciler{
		recoverFailedFunc: func(_ context.Context, _ TransactionMonitor) error {
			return nil
		},
	}

	cycle := newTestCycle(agg, sub, rec)
	err := cycle.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestThresholdBelowMinimum_SkipsCycle(t *testing.T) {
	t.Parallel()

	payouts := []ProviderPayout{
		{ProviderID: uuid.New(), Amount: decimal.NewFromInt(50), Currency: repository.CurrencyXLM, WalletAddress: "GATHOLD"},
	}

	agg := &mockAggregator{
		aggregateFunc: func(_ context.Context) ([]ProviderPayout, error) {
			return payouts, nil
		},
	}

	sub := &mockSubmitter{}
	rec := &mockReconciler{
		recoverFailedFunc: func(_ context.Context, _ TransactionMonitor) error {
			return nil
		},
	}

	cycle := newTestCycle(agg, sub, rec)
	cycle.minThreshold = decimal.NewFromInt(100)

	err := cycle.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestThresholdAboveMinimum_Proceeds(t *testing.T) {
	t.Parallel()

	batchID := uuid.New()
	payouts := []ProviderPayout{
		{ProviderID: uuid.New(), Amount: decimal.NewFromInt(200), Currency: repository.CurrencyXLM, WalletAddress: "GAPROCD"},
	}

	agg := &mockAggregator{
		aggregateFunc: func(_ context.Context) ([]ProviderPayout, error) {
			return payouts, nil
		},
	}

	sub := &mockSubmitter{
		submitPayoutsFunc: func(_ context.Context, _ uuid.UUID, _ []ProviderPayout) ([]PayoutResult, error) {
			return []PayoutResult{{ProviderID: payouts[0].ProviderID, TxHash: "tx", Status: TransactionSuccess}}, nil
		},
	}

	rec := &mockReconciler{
		recoverFailedFunc: func(_ context.Context, _ TransactionMonitor) error {
			return nil
		},
		createBatchFunc: func(_ context.Context, _ []ProviderPayout) (uuid.UUID, []repository.SettlementEntry, error) {
			return batchID, nil, nil
		},
		finalizeBatchFunc: func(_ context.Context, _ uuid.UUID, _ []PayoutResult) error {
			return nil
		},
	}

	cycle := newTestCycle(agg, sub, rec)
	cycle.minThreshold = decimal.NewFromInt(100)

	err := cycle.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestCreateBatchFailure_Aborts(t *testing.T) {
	t.Parallel()

	payouts := []ProviderPayout{
		{ProviderID: uuid.New(), Amount: decimal.NewFromInt(50), Currency: repository.CurrencyXLM, WalletAddress: "GAABORT"},
	}

	agg := &mockAggregator{
		aggregateFunc: func(_ context.Context) ([]ProviderPayout, error) {
			return payouts, nil
		},
	}

	sub := &mockSubmitter{}
	rec := &mockReconciler{
		recoverFailedFunc: func(_ context.Context, _ TransactionMonitor) error {
			return nil
		},
		createBatchFunc: func(_ context.Context, _ []ProviderPayout) (uuid.UUID, []repository.SettlementEntry, error) {
			return uuid.Nil, nil, errors.New("db connection lost")
		},
	}

	cycle := newTestCycle(agg, sub, rec)
	err := cycle.Run(context.Background())
	if err == nil {
		t.Fatal("expected error from CreateBatch failure, got nil")
	}
}

func TestPartialStellarFailure(t *testing.T) {
	t.Parallel()

	batchID := uuid.New()
	successProvider := uuid.New()
	failProvider := uuid.New()
	payouts := []ProviderPayout{
		{ProviderID: successProvider, Amount: decimal.NewFromInt(100), Currency: repository.CurrencyXLM, WalletAddress: "GAOK"},
		{ProviderID: failProvider, Amount: decimal.NewFromInt(50), Currency: repository.CurrencyXLM, WalletAddress: "GAFAIL"},
	}

	agg := &mockAggregator{
		aggregateFunc: func(_ context.Context) ([]ProviderPayout, error) {
			return payouts, nil
		},
	}

	sub := &mockSubmitter{
		submitPayoutsFunc: func(_ context.Context, _ uuid.UUID, p []ProviderPayout) ([]PayoutResult, error) {
			results := []PayoutResult{
				{ProviderID: p[0].ProviderID, TxHash: "tx_ok", Status: TransactionSuccess},
				{ProviderID: p[1].ProviderID, Error: "stellar timeout", Status: TransactionFailed},
			}

			return results, nil
		},
	}

	var capturedResults []PayoutResult
	rec := &mockReconciler{
		recoverFailedFunc: func(_ context.Context, _ TransactionMonitor) error {
			return nil
		},
		createBatchFunc: func(_ context.Context, _ []ProviderPayout) (uuid.UUID, []repository.SettlementEntry, error) {
			return batchID, nil, nil
		},
		finalizeBatchFunc: func(_ context.Context, _ uuid.UUID, results []PayoutResult) error {
			capturedResults = results
			return nil
		},
	}

	cycle := newTestCycle(agg, sub, rec)
	err := cycle.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if len(capturedResults) != 2 {
		t.Fatalf("FinalizeBatch received %d results, want 2", len(capturedResults))
	}

	for _, r := range capturedResults {
		switch r.ProviderID {
		case successProvider:
			if r.Status != TransactionSuccess {
				t.Errorf("provider %s status = %q, want success", r.ProviderID, r.Status)
			}
		case failProvider:
			if r.Status != TransactionFailed {
				t.Errorf("provider %s status = %q, want failed", r.ProviderID, r.Status)
			}
		default:
			t.Errorf("unexpected provider %s in results", r.ProviderID)
		}
	}
}

func TestRecoverFailed_Error_NonFatal(t *testing.T) {
	t.Parallel()

	payouts := []ProviderPayout{
		{ProviderID: uuid.New(), Amount: decimal.NewFromInt(75), Currency: repository.CurrencyXLM, WalletAddress: "GARECOV"},
	}

	agg := &mockAggregator{
		aggregateFunc: func(_ context.Context) ([]ProviderPayout, error) {
			return payouts, nil
		},
	}

	sub := &mockSubmitter{
		submitPayoutsFunc: func(_ context.Context, _ uuid.UUID, _ []ProviderPayout) ([]PayoutResult, error) {
			return []PayoutResult{{ProviderID: payouts[0].ProviderID, TxHash: "tx", Status: TransactionSuccess}}, nil
		},
	}

	rec := &mockReconciler{
		recoverFailedFunc: func(_ context.Context, _ TransactionMonitor) error {
			return errors.New("horizon unreachable")
		},
		createBatchFunc: func(_ context.Context, _ []ProviderPayout) (uuid.UUID, []repository.SettlementEntry, error) {
			return uuid.New(), nil, nil
		},
		finalizeBatchFunc: func(_ context.Context, _ uuid.UUID, _ []PayoutResult) error {
			return nil
		},
	}

	cycle := newTestCycle(agg, sub, rec)
	err := cycle.Run(context.Background())
	if err != nil {
		t.Fatalf("Run should tolerate RecoverFailed error, got: %v", err)
	}
}

func TestContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	agg := &mockAggregator{
		aggregateFunc: func(_ context.Context) ([]ProviderPayout, error) {
			cancel()

			return nil, context.Canceled
		},
	}

	sub := &mockSubmitter{}
	rec := &mockReconciler{
		recoverFailedFunc: func(_ context.Context, _ TransactionMonitor) error {
			return nil
		},
	}

	cycle := newTestCycle(agg, sub, rec)
	err := cycle.Run(ctx)
	if err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestNilAggregatorPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil Aggregator")
		}
	}()

	NewCycle(nil, &StellarSubmitter{}, &Reconciler{}, decimal.Zero)
}

func TestNilSubmitterPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil StellarSubmitter")
		}
	}()

	NewCycle(&Aggregator{}, nil, &Reconciler{}, decimal.Zero)
}

func TestNilReconcilerPanics(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil Reconciler")
		}
	}()

	NewCycle(&Aggregator{}, &StellarSubmitter{}, nil, decimal.Zero)
}

func TestSettlementRunner_RunsImmediately(t *testing.T) {
	t.Parallel()

	var runCount int
	mockCycle := &mockCycle{
		runFunc: func(_ context.Context) error {
			runCount++

			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	runner := NewRunner(mockCycle, 1*time.Hour)

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_ = runner.Run(ctx)

	if runCount != 1 {
		t.Errorf("expected 1 immediate run, got %d", runCount)
	}
}

func TestSettlementRunner_SkipsOverlappingTick(t *testing.T) {
	t.Parallel()

	var runCount int
	runStarted := make(chan struct{})
	runBlocked := make(chan struct{})
	mockCycle := &mockCycle{
		runFunc: func(_ context.Context) error {
			runCount++
			close(runStarted)
			<-runBlocked

			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	runner := NewRunner(mockCycle, 50*time.Millisecond)

	go func() {
		<-runStarted
		time.Sleep(200 * time.Millisecond)
		cancel()
	}()

	go func() {
		time.Sleep(300 * time.Millisecond)
		close(runBlocked)
	}()

	_ = runner.Run(ctx)

	if runCount != 1 {
		t.Errorf("expected 1 run (subsequent ticks skipped), got %d", runCount)
	}
}

func TestSettlementRunner_ContextCancellation(t *testing.T) {
	t.Parallel()

	mockCycle := &mockCycle{
		runFunc: func(_ context.Context) error {
			return nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	runner := NewRunner(mockCycle, 1*time.Hour)

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := runner.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

type mockCycle struct {
	runFunc func(context.Context) error
}

func (m *mockCycle) Run(ctx context.Context) error {
	if m.runFunc != nil {
		return m.runFunc(ctx)
	}

	return nil
}
