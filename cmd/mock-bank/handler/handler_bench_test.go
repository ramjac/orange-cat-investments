package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/orange-cat-investments/oci/cmd/mock-bank/engine"
	"github.com/orange-cat-investments/oci/cmd/mock-bank/ledger"
)

func BenchmarkGetBalances_AccountIDs(b *testing.B) {
	store := ledger.NewLedgerStore()
	eng := engine.NewSettlementEngine(store, nil, 100*time.Millisecond)
	h := NewHandler(store, eng)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	// Create extra accounts in store for benchmarking
	var accountIDs []string
	for i := 0; i < 50; i++ {
		accID := fmt.Sprintf("acc_bench_%d", i)
		accountIDs = append(accountIDs, accID)
	}

	reqBody, err := json.Marshal(map[string]interface{}{
		"account_ids": accountIDs,
	})
	if err != nil {
		b.Fatalf("failed to marshal request: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/accounts/balance/get", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
	}
}

func BenchmarkStore_GetAccount_vs_Batch(b *testing.B) {
	store := ledger.NewLedgerStore()

	var accountIDs []string
	for i := 0; i < 50; i++ {
		accID := fmt.Sprintf("acc_bench_%d", i)
		accountIDs = append(accountIDs, accID)
	}

	b.Run("Loop_GetAccount", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var accounts []*ledger.BankAccount
			for _, id := range accountIDs {
				if acc, err := store.GetAccount(id); err == nil {
					accounts = append(accounts, acc)
				}
			}
		}
	})

	b.Run("Batch_GetAccountsByIDs", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = store.GetAccountsByIDs(accountIDs)
		}
	})
}
