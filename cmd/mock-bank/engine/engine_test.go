package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/cmd/mock-bank/ledger"
)

func TestSettlementEngine_ProgressiveClearing(t *testing.T) {
	var mu sync.Mutex
	receivedEvents := make([]WebhookEvent, 0)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var evt WebhookEvent
		if err := json.NewDecoder(r.Body).Decode(&evt); err == nil {
			mu.Lock()
			receivedEvents = append(receivedEvents, evt)
			mu.Unlock()
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	store := ledger.NewLedgerStore()
	dispatcher := NewWebhookDispatcher(ts.URL)
	engine := NewSettlementEngine(store, dispatcher, 20*time.Millisecond)
	engine.SetDelays(40*time.Millisecond, 40*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	engine.Start(ctx)
	defer engine.Stop()

	// Create debit transfer
	transfer, err := store.CreateTransfer("acc_arthur_chase_chk", 150.00, "debit", "Test Clearing")
	if err != nil {
		t.Fatalf("failed to create transfer: %v", err)
	}

	// Wait for settlement to process (approx 120ms total)
	deadline := time.Now().Add(2 * time.Second)
	var finalStatus ledger.ACHTransferStatus
	for time.Now().Before(deadline) {
		tr, _ := store.GetTransfer(transfer.TransferID)
		if tr != nil && tr.Status == ledger.TransferStatusSettled {
			finalStatus = tr.Status
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	if finalStatus != ledger.TransferStatusSettled {
		t.Fatalf("expected transfer to settle, got %s", finalStatus)
	}

	// Wait briefly for webhook delivery
	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	count := len(receivedEvents)
	mu.Unlock()

	if count == 0 {
		t.Errorf("expected webhook events to be dispatched, got %d", count)
	}
}

func TestSettlementEngine_SimulateReturn(t *testing.T) {
	store := ledger.NewLedgerStore()
	engine := NewSettlementEngine(store, nil, 100*time.Millisecond)

	transfer, err := store.CreateTransfer("acc_arthur_chase_chk", 300.00, "debit", "Test Return")
	if err != nil {
		t.Fatalf("failed to create transfer: %v", err)
	}

	ret, err := engine.SimulateReturn(context.Background(), transfer.TransferID, "R01")
	if err != nil {
		t.Fatalf("failed to simulate return: %v", err)
	}
	if ret.Status != ledger.TransferStatusReturned || ret.ReturnCode != "R01" {
		t.Errorf("unexpected return state: %+v", ret)
	}

	acc, _ := store.GetAccount("acc_arthur_chase_chk")
	if acc.AvailableBalance != 48250.00 {
		t.Errorf("expected available balance restored to 48250.00, got %f", acc.AvailableBalance)
	}
}
