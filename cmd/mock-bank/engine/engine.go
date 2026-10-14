package engine

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/orange-cat-investments/oci/cmd/mock-bank/ledger"
)

// SettlementEngine processes ACH clearing cycles and state transitions.
type SettlementEngine struct {
	store        *ledger.LedgerStore
	dispatcher   *WebhookDispatcher
	tickInterval time.Duration
	postDelay    time.Duration
	settleDelay  time.Duration
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

// NewSettlementEngine instantiates a new settlement engine.
func NewSettlementEngine(store *ledger.LedgerStore, dispatcher *WebhookDispatcher, tickInterval time.Duration) *SettlementEngine {
	if tickInterval <= 0 {
		tickInterval = 1000 * time.Millisecond
	}
	return &SettlementEngine{
		store:        store,
		dispatcher:   dispatcher,
		tickInterval: tickInterval,
		postDelay:    2 * tickInterval,
		settleDelay:  2 * tickInterval,
		stopCh:       make(chan struct{}),
	}
}

// SetDelays overrides settlement timing delays (useful for tests).
func (e *SettlementEngine) SetDelays(postDelay, settleDelay time.Duration) {
	e.postDelay = postDelay
	e.settleDelay = settleDelay
}

// Start launches the background ACH settlement ticker.
func (e *SettlementEngine) Start(ctx context.Context) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		ticker := time.NewTicker(e.tickInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-e.stopCh:
				return
			case <-ticker.C:
				e.tick(ctx)
			}
		}
	}()
}

// Stop cleanly terminates the settlement loop.
func (e *SettlementEngine) Stop() {
	select {
	case <-e.stopCh:
		// already closed
	default:
		close(e.stopCh)
	}
	e.wg.Wait()
}

func (e *SettlementEngine) tick(ctx context.Context) {
	pending := e.store.GetPendingTransfers()
	now := time.Now().UTC()

	for _, t := range pending {
		elapsed := now.Sub(t.CreatedAt)

		switch t.Status {
		case ledger.TransferStatusPending:
			if elapsed >= e.postDelay {
				updated, err := e.store.UpdateTransferStatus(t.TransferID, ledger.TransferStatusPosted, "")
				if err == nil && e.dispatcher != nil {
					e.dispatcher.Dispatch(ctx, updated)
				}
			}

		case ledger.TransferStatusPosted:
			if elapsed >= (e.postDelay + e.settleDelay) {
				updated, err := e.store.UpdateTransferStatus(t.TransferID, ledger.TransferStatusSettled, "")
				if err == nil {
					log.Printf("[mock-bank] ACH transfer %s settled successfully ($%.2f)", updated.TransferID, updated.Amount)
					if e.dispatcher != nil {
						e.dispatcher.Dispatch(ctx, updated)
					}
				}
			}
		}
	}
}

// SimulateSettlement immediately marks a transfer settled and dispatches a webhook event.
func (e *SettlementEngine) SimulateSettlement(ctx context.Context, transferID string) (*ledger.ACHTransfer, error) {
	updated, err := e.store.UpdateTransferStatus(transferID, ledger.TransferStatusSettled, "")
	if err != nil {
		return nil, err
	}
	if e.dispatcher != nil {
		e.dispatcher.Dispatch(ctx, updated)
	}
	return updated, nil
}

// SimulateReturn immediately marks a transfer returned (e.g. R01 Insufficient Funds) and dispatches a webhook event.
func (e *SettlementEngine) SimulateReturn(ctx context.Context, transferID string, returnCode string) (*ledger.ACHTransfer, error) {
	if returnCode == "" {
		returnCode = "R01"
	}
	updated, err := e.store.UpdateTransferStatus(transferID, ledger.TransferStatusReturned, returnCode)
	if err != nil {
		return nil, err
	}
	if e.dispatcher != nil {
		e.dispatcher.Dispatch(ctx, updated)
	}
	return updated, nil
}
