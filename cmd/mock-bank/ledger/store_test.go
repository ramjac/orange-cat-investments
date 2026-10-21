package ledger

import (
	"sync"
	"testing"
)

func TestLedgerStore_SeedAndAccounts(t *testing.T) {
	store := NewLedgerStore()

	acc, err := store.GetAccount("acc_arthur_chase_chk")
	if err != nil {
		t.Fatalf("expected Arthur account, got error: %v", err)
	}
	if acc.CurrentBalance != 48500.00 || acc.AvailableBalance != 48250.00 {
		t.Errorf("unexpected Arthur balance: %+v", acc)
	}

	arthurAccounts := store.GetAccountsByOwner("cust-longterm-arthur")
	if len(arthurAccounts) != 2 {
		t.Errorf("expected 2 accounts for Arthur, got %d", len(arthurAccounts))
	}

	tokAccounts, err := store.GetAccountsByAccessToken("access-sandbox-arthur")
	if err != nil {
		t.Fatalf("unexpected error fetching by token: %v", err)
	}
	if len(tokAccounts) != 2 {
		t.Errorf("expected 2 accounts via token, got %d", len(tokAccounts))
	}

	byIDs := store.GetAccountsByIDs([]string{"acc_arthur_chase_chk", "acc_chloe_svcu_chk", "non_existent"})
	if len(byIDs) != 2 {
		t.Errorf("expected 2 accounts fetched by IDs, got %d", len(byIDs))
	}
}

func TestLedgerStore_CreateTransferDebitSuccessAndSettlement(t *testing.T) {
	store := NewLedgerStore()

	// Initial available: 48250.00, current: 48500.00
	transfer, err := store.CreateTransfer("acc_arthur_chase_chk", 500.00, "debit", "Monthly ACH Deposit")
	if err != nil {
		t.Fatalf("expected successful transfer creation, got: %v", err)
	}

	if transfer.Status != TransferStatusPending {
		t.Errorf("expected pending status, got %s", transfer.Status)
	}

	// Verify hold was placed on available balance immediately
	acc, _ := store.GetAccount("acc_arthur_chase_chk")
	if acc.AvailableBalance != 47750.00 {
		t.Errorf("expected available balance to decrease to 47750, got %f", acc.AvailableBalance)
	}
	if acc.CurrentBalance != 48500.00 {
		t.Errorf("expected current balance to remain 48500 during pending hold, got %f", acc.CurrentBalance)
	}

	// Settle the transfer
	settled, err := store.UpdateTransferStatus(transfer.TransferID, TransferStatusSettled, "")
	if err != nil {
		t.Fatalf("failed to settle transfer: %v", err)
	}
	if settled.Status != TransferStatusSettled || settled.SettledAt == nil {
		t.Errorf("expected settled transfer with SettledAt, got: %+v", settled)
	}

	// Verify final balances
	acc, _ = store.GetAccount("acc_arthur_chase_chk")
	if acc.CurrentBalance != 48000.00 {
		t.Errorf("expected current balance 48000, got %f", acc.CurrentBalance)
	}
	if acc.AvailableBalance != 47750.00 {
		t.Errorf("expected available balance 47750, got %f", acc.AvailableBalance)
	}

	// Custody account should have received the funds
	custody, _ := store.GetAccount(DefaultCustodyAccountID)
	if custody.CurrentBalance != 10000500.00 {
		t.Errorf("expected custody balance 10000500, got %f", custody.CurrentBalance)
	}
}

func TestLedgerStore_InsufficientFunds(t *testing.T) {
	store := NewLedgerStore()

	// Chloe has available $22,400. Try to transfer $30,000.
	transfer, err := store.CreateTransfer("acc_chloe_svcu_chk", 30000.00, "debit", "Excessive Transfer")
	if err == nil {
		t.Fatal("expected error ErrInsufficientFunds, got nil")
	}
	if transfer == nil || transfer.ReturnCode != "R01" || transfer.Status != TransferStatusFailed {
		t.Errorf("expected failed transfer with R01 code, got: %+v", transfer)
	}

	// Verify Chloe's balance wasn't changed
	acc, _ := store.GetAccount("acc_chloe_svcu_chk")
	if acc.AvailableBalance != 22400.00 {
		t.Errorf("expected Chloe available balance unchanged, got %f", acc.AvailableBalance)
	}
}

func TestLedgerStore_TransferReturnReleasesHold(t *testing.T) {
	store := NewLedgerStore()

	transfer, err := store.CreateTransfer("acc_arthur_chase_chk", 1000.00, "debit", "Test Return")
	if err != nil {
		t.Fatalf("failed to create transfer: %v", err)
	}

	acc, _ := store.GetAccount("acc_arthur_chase_chk")
	if acc.AvailableBalance != 47250.00 {
		t.Fatalf("expected available balance 47250.00, got %f", acc.AvailableBalance)
	}

	// Transfer returned by clearing house (e.g. simulated customer dispute or stop payment)
	returned, err := store.UpdateTransferStatus(transfer.TransferID, TransferStatusReturned, "R08")
	if err != nil {
		t.Fatalf("failed to return transfer: %v", err)
	}
	if returned.ReturnCode != "R08" {
		t.Errorf("expected return code R08, got %s", returned.ReturnCode)
	}

	acc, _ = store.GetAccount("acc_arthur_chase_chk")
	if acc.AvailableBalance != 48250.00 {
		t.Errorf("expected available balance restored to 48250.00, got %f", acc.AvailableBalance)
	}
	if acc.CurrentBalance != 48500.00 {
		t.Errorf("expected current balance remained 48500.00, got %f", acc.CurrentBalance)
	}
}

func TestLedgerStore_ConcurrentTransfers(t *testing.T) {
	store := NewLedgerStore()

	var wg sync.WaitGroup
	numWorkers := 20
	transfersPerWorker := 10

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < transfersPerWorker; j++ {
				tr, err := store.CreateTransfer("acc_arthur_chase_sav", 10.00, "debit", "Concurrent Debit")
				if err == nil {
					_, _ = store.UpdateTransferStatus(tr.TransferID, TransferStatusSettled, "")
				}
			}
		}()
	}

	wg.Wait()

	acc, err := store.GetAccount("acc_arthur_chase_sav")
	if err != nil {
		t.Fatalf("failed to get account: %v", err)
	}
	expectedBalance := 125000.00 - float64(numWorkers*transfersPerWorker*10)
	if acc.CurrentBalance != expectedBalance {
		t.Errorf("expected balance %f, got %f", expectedBalance, acc.CurrentBalance)
	}
}
