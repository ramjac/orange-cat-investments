package ledger

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFlexibleFloat64_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantVal   float64
		expectErr bool
	}{
		{
			name:      "numeric float",
			input:     `123.45`,
			wantVal:   123.45,
			expectErr: false,
		},
		{
			name:      "numeric integer",
			input:     `100`,
			wantVal:   100.0,
			expectErr: false,
		},
		{
			name:      "stringified float",
			input:     `"456.78"`,
			wantVal:   456.78,
			expectErr: false,
		},
		{
			name:      "invalid string float",
			input:     `"not_a_number"`,
			wantVal:   0,
			expectErr: true,
		},
		{
			name:      "unsupported json type boolean",
			input:     `true`,
			wantVal:   0,
			expectErr: true,
		},
		{
			name:      "unsupported json type object",
			input:     `{"amount": 10}`,
			wantVal:   0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f FlexibleFloat64
			err := json.Unmarshal([]byte(tt.input), &f)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error unmarshaling %s, got nil", tt.input)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error unmarshaling %s: %v", tt.input, err)
				}
				if f.Float64() != tt.wantVal {
					t.Errorf("expected Float64() = %f, got %f", tt.wantVal, f.Float64())
				}
			}
		})
	}
}

func TestModels_InstantiationAndJSON(t *testing.T) {
	// Test BankAccount
	acc := BankAccount{
		AccountID:        "acc_123",
		InstitutionID:    "inst_chase",
		InstitutionName:  "Chase Bank",
		RoutingNumber:    "123456789",
		AccountNumber:    "****4821",
		AccountType:      "checking",
		AccountSubtype:   "checking",
		CurrentBalance:   1500.50,
		AvailableBalance: 1400.00,
		Currency:         "USD",
		OwnerID:          "cust_user_1",
		Name:             "Premier Checking",
	}

	data, err := json.Marshal(acc)
	if err != nil {
		t.Fatalf("failed to marshal BankAccount: %v", err)
	}

	var unmarshaledAcc BankAccount
	if err := json.Unmarshal(data, &unmarshaledAcc); err != nil {
		t.Fatalf("failed to unmarshal BankAccount: %v", err)
	}
	if unmarshaledAcc.AccountID != acc.AccountID || unmarshaledAcc.CurrentBalance != acc.CurrentBalance {
		t.Errorf("mismatch after json roundtrip: %+v", unmarshaledAcc)
	}

	// Test ACHTransfer & ACHTransferStatus
	now := time.Now().Truncate(time.Second)
	transfer := ACHTransfer{
		TransferID:       "tr_123",
		AccountID:        "acc_123",
		Amount:           100.00,
		Direction:        "debit",
		Type:             "ach",
		Status:           TransferStatusPending,
		ReturnCode:       "",
		Description:      "Test transfer",
		CreatedAt:        now,
		SettledAt:        nil,
		CustodyAccountID: "custody_001",
	}

	trData, err := json.Marshal(transfer)
	if err != nil {
		t.Fatalf("failed to marshal ACHTransfer: %v", err)
	}

	var unmarshaledTr ACHTransfer
	if err := json.Unmarshal(trData, &unmarshaledTr); err != nil {
		t.Fatalf("failed to unmarshal ACHTransfer: %v", err)
	}
	if unmarshaledTr.Status != TransferStatusPending || unmarshaledTr.Amount != transfer.Amount {
		t.Errorf("mismatch after ACHTransfer json roundtrip: %+v", unmarshaledTr)
	}

	// Test LinkToken
	linkToken := LinkToken{
		LinkToken:    "link-sandbox-123",
		ClientUserID: "user_456",
		CreatedAt:    now,
		ExpiresAt:    now.Add(30 * time.Minute),
	}
	linkData, err := json.Marshal(linkToken)
	if err != nil {
		t.Fatalf("failed to marshal LinkToken: %v", err)
	}

	var unmarshaledLink LinkToken
	if err := json.Unmarshal(linkData, &unmarshaledLink); err != nil {
		t.Fatalf("failed to unmarshal LinkToken: %v", err)
	}
	if unmarshaledLink.LinkToken != linkToken.LinkToken {
		t.Errorf("mismatch after LinkToken json roundtrip: %+v", unmarshaledLink)
	}

	// Test AccessToken
	accessToken := AccessToken{
		AccessToken: "access-sandbox-123",
		ItemID:      "item_123",
		OwnerID:     "user_456",
		AccountIDs:  []string{"acc_123", "acc_456"},
		CreatedAt:   now,
	}
	accessData, err := json.Marshal(accessToken)
	if err != nil {
		t.Fatalf("failed to marshal AccessToken: %v", err)
	}

	var unmarshaledAccess AccessToken
	if err := json.Unmarshal(accessData, &unmarshaledAccess); err != nil {
		t.Fatalf("failed to unmarshal AccessToken: %v", err)
	}
	if unmarshaledAccess.AccessToken != accessToken.AccessToken || len(unmarshaledAccess.AccountIDs) != 2 {
		t.Errorf("mismatch after AccessToken json roundtrip: %+v", unmarshaledAccess)
	}
}

func TestACHTransferStatusConstants(t *testing.T) {
	statuses := []ACHTransferStatus{
		TransferStatusPending,
		TransferStatusPosted,
		TransferStatusSettled,
		TransferStatusFailed,
		TransferStatusReturned,
		TransferStatusCancelled,
	}

	expected := []string{"pending", "posted", "settled", "failed", "returned", "cancelled"}

	for i, s := range statuses {
		if string(s) != expected[i] {
			t.Errorf("expected status %s, got %s", expected[i], string(s))
		}
	}
}
