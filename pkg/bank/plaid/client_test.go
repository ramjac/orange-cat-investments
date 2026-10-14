package plaid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/cmd/mock-bank/engine"
	"github.com/orange-cat-investments/oci/cmd/mock-bank/handler"
	"github.com/orange-cat-investments/oci/cmd/mock-bank/ledger"
)

func setupTestBankServer() (*httptest.Server, *Client) {
	store := ledger.NewLedgerStore()
	eng := engine.NewSettlementEngine(store, nil, 50*time.Millisecond)
	h := handler.NewHandler(store, eng)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	client := NewClient(Config{
		BaseURL:  ts.URL,
		ClientID: "test_client",
		Secret:   "test_secret",
	})
	return ts, client
}

func TestClient_LinkAndAccounts(t *testing.T) {
	ts, client := setupTestBankServer()
	defer ts.Close()

	ctx := context.Background()

	// 1. Create link token
	linkToken, err := client.CreateLinkToken(ctx, "cust-longterm-arthur")
	if err != nil {
		t.Fatalf("failed to create link token: %v", err)
	}
	if linkToken == "" {
		t.Fatal("expected non-empty link token")
	}

	// 2. Exchange public token
	accessToken, err := client.ExchangePublicToken(ctx, "public_token_sample", "cust-longterm-arthur")
	if err != nil {
		t.Fatalf("failed to exchange public token: %v", err)
	}
	if accessToken == "" {
		t.Fatal("expected non-empty access token")
	}

	// 3. Get accounts by owner
	accounts, err := client.GetAccounts(ctx, "cust-longterm-arthur")
	if err != nil {
		t.Fatalf("failed to get accounts: %v", err)
	}
	if len(accounts) != 2 {
		t.Fatalf("expected 2 accounts for Arthur, got %d", len(accounts))
	}

	// 4. Get balances
	balances, err := client.GetBalances(ctx, accessToken, nil)
	if err != nil {
		t.Fatalf("failed to get balances: %v", err)
	}
	if len(balances) != 2 {
		t.Fatalf("expected 2 accounts with balance, got %d", len(balances))
	}
}

func TestClient_TransferLifecycle(t *testing.T) {
	ts, client := setupTestBankServer()
	defer ts.Close()

	ctx := context.Background()

	// 1. Authorize transfer
	auth, err := client.AuthorizeTransfer(ctx, "acc_arthur_chase_chk", 500.00, "debit")
	if err != nil {
		t.Fatalf("failed to authorize transfer: %v", err)
	}
	if auth.Decision != "approved" {
		t.Errorf("expected approved decision, got %s", auth.Decision)
	}

	// 2. Create transfer
	transfer, err := client.CreateTransfer(ctx, CreateTransferRequest{
		AccountID:   "acc_arthur_chase_chk",
		Amount:      500.00,
		Direction:   "debit",
		Description: "Monthly Recurring ACH",
	})
	if err != nil {
		t.Fatalf("failed to create transfer: %v", err)
	}
	if transfer.Status != "pending" || transfer.Amount != 500.00 {
		t.Errorf("unexpected transfer status: %+v", transfer)
	}

	// 3. Get transfer
	fetched, err := client.GetTransfer(ctx, transfer.TransferID)
	if err != nil {
		t.Fatalf("failed to fetch transfer: %v", err)
	}
	if fetched.TransferID != transfer.TransferID {
		t.Errorf("transfer ID mismatch: %s vs %s", fetched.TransferID, transfer.TransferID)
	}

	// 4. Test NSF transfer (declined)
	nsfAuth, err := client.AuthorizeTransfer(ctx, "acc_chloe_svcu_chk", 100000.00, "debit")
	if err != nil {
		t.Fatalf("unexpected error on authorization check: %v", err)
	}
	if nsfAuth.Decision != "declined" {
		t.Errorf("expected declined decision on NSF, got %s", nsfAuth.Decision)
	}
}
