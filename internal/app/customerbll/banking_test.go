package customerbll

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/orange-cat-investments/oci/pkg/bank/plaid"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestListBankAccounts_InvalidArgument(t *testing.T) {
	s := NewServer(Config{})

	tests := []struct {
		name string
		req  *customerbllv1.ListBankAccountsRequest
	}{
		{
			name: "nil request",
			req:  nil,
		},
		{
			name: "empty customer_id",
			req:  &customerbllv1.ListBankAccountsRequest{CustomerID: ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := s.ListBankAccounts(context.Background(), tt.req)
			require.Error(t, err)
			assert.Nil(t, resp)

			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, codes.InvalidArgument, st.Code())
			assert.Contains(t, st.Message(), "customer_id is required")
		})
	}
}

func TestListBankAccounts_WithBankClientSuccess(t *testing.T) {
	mockAccounts := []plaid.Account{
		{
			AccountID:        "acc-101",
			InstitutionName:  "Catnip Financial",
			AccountNumber:    "****5555",
			Name:             "Main Checking",
			AccountType:      "depository",
			AccountSubtype:   "checking",
			AvailableBalance: 2500.50,
			CurrentBalance:   3000.00,
		},
		{
			AccountID:        "acc-102",
			InstitutionName:  "Catnip Financial",
			AccountNumber:    "****8888",
			Name:             "High Yield Savings",
			AccountType:      "depository",
			AccountSubtype:   "savings",
			AvailableBalance: 15000.00,
			CurrentBalance:   15000.00,
		},
	}

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/accounts", r.URL.Path)
		assert.Equal(t, "cust-12345", r.URL.Query().Get("owner_id"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"accounts": mockAccounts,
		})
	}))
	defer mockServer.Close()

	bankClient := plaid.NewClient(plaid.Config{
		BaseURL: mockServer.URL,
	})

	s := NewServer(Config{
		BankClient: bankClient,
	})

	req := &customerbllv1.ListBankAccountsRequest{CustomerID: "cust-12345"}
	resp, err := s.ListBankAccounts(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Accounts, 2)

	assert.Equal(t, "acc-101", resp.Accounts[0].AccountID)
	assert.Equal(t, "Catnip Financial", resp.Accounts[0].InstitutionName)
	assert.Equal(t, "****5555", resp.Accounts[0].AccountNumber)
	assert.Equal(t, "Main Checking", resp.Accounts[0].Name)
	assert.Equal(t, "depository", resp.Accounts[0].Type)
	assert.Equal(t, "checking", resp.Accounts[0].Subtype)
	assert.Equal(t, 2500.50, resp.Accounts[0].AvailableBalance)
	assert.Equal(t, 3000.00, resp.Accounts[0].CurrentBalance)

	assert.Equal(t, "acc-102", resp.Accounts[1].AccountID)
	assert.Equal(t, "High Yield Savings", resp.Accounts[1].Name)
}

func TestListBankAccounts_WithBankClientEmptyOrErrorFallback(t *testing.T) {
	t.Run("bank client returns empty accounts list", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"accounts": []plaid.Account{},
			})
		}))
		defer mockServer.Close()

		bankClient := plaid.NewClient(plaid.Config{
			BaseURL: mockServer.URL,
		})
		s := NewServer(Config{BankClient: bankClient})

		resp, err := s.ListBankAccounts(context.Background(), &customerbllv1.ListBankAccountsRequest{
			CustomerID: "cust-empty",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Accounts, 1)

		assert.Equal(t, "acc_cust-empty_primary", resp.Accounts[0].AccountID)
		assert.Equal(t, "Primary Linked Institution", resp.Accounts[0].InstitutionName)
		assert.Equal(t, "****1001", resp.Accounts[0].AccountNumber)
		assert.Equal(t, "Primary Checking Account", resp.Accounts[0].Name)
		assert.Equal(t, 10000.00, resp.Accounts[0].AvailableBalance)
		assert.Equal(t, 10000.00, resp.Accounts[0].CurrentBalance)
	})

	t.Run("bank client returns HTTP error", func(t *testing.T) {
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer mockServer.Close()

		bankClient := plaid.NewClient(plaid.Config{
			BaseURL: mockServer.URL,
		})
		s := NewServer(Config{BankClient: bankClient})

		resp, err := s.ListBankAccounts(context.Background(), &customerbllv1.ListBankAccountsRequest{
			CustomerID: "cust-err",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Accounts, 1)

		assert.Equal(t, "acc_cust-err_primary", resp.Accounts[0].AccountID)
		assert.Equal(t, "Primary Linked Institution", resp.Accounts[0].InstitutionName)
	})
}

func TestListBankAccounts_NilBankClientFallback(t *testing.T) {
	s := NewServer(Config{
		BankClient: nil,
	})

	resp, err := s.ListBankAccounts(context.Background(), &customerbllv1.ListBankAccountsRequest{
		CustomerID: "cust-nobank",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Accounts, 1)

	assert.Equal(t, "acc_cust-nobank_primary", resp.Accounts[0].AccountID)
	assert.Equal(t, "Primary Linked Institution", resp.Accounts[0].InstitutionName)
	assert.Equal(t, "****1001", resp.Accounts[0].AccountNumber)
	assert.Equal(t, "Primary Checking Account", resp.Accounts[0].Name)
	assert.Equal(t, 10000.00, resp.Accounts[0].AvailableBalance)
	assert.Equal(t, 10000.00, resp.Accounts[0].CurrentBalance)
}
