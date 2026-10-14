package ledger

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// FlexibleFloat64 allows unmarshaling JSON values that may arrive as numbers or strings.
type FlexibleFloat64 float64

// UnmarshalJSON implements json.Unmarshaler.
func (f *FlexibleFloat64) UnmarshalJSON(b []byte) error {
	var num float64
	if err := json.Unmarshal(b, &num); err == nil {
		*f = FlexibleFloat64(num)
		return nil
	}

	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		val, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return fmt.Errorf("cannot parse %q as float64: %w", str, err)
		}
		*f = FlexibleFloat64(val)
		return nil
	}

	return fmt.Errorf("cannot unmarshal %s into FlexibleFloat64", string(b))
}

// Float64 returns standard float64 value.
func (f FlexibleFloat64) Float64() float64 {
	return float64(f)
}

// BankAccount represents an external customer checking or savings account.
type BankAccount struct {
	AccountID       string  `json:"account_id"`
	InstitutionID   string  `json:"institution_id"`
	InstitutionName string  `json:"institution_name"`
	RoutingNumber   string  `json:"routing_number"`
	AccountNumber   string  `json:"account_number"` // Masked: ****4821
	AccountType     string  `json:"account_type"`   // checking, savings
	AccountSubtype  string  `json:"account_subtype"` // e.g. checking, savings
	CurrentBalance  float64 `json:"current_balance"`
	AvailableBalance float64 `json:"available_balance"`
	Currency        string  `json:"currency"`       // USD
	OwnerID         string  `json:"owner_id"`        // Persona customer_id (e.g. cust-longterm-arthur)
	Name            string  `json:"name"`            // Friendly name: "Chase Premier Checking"
}

// ACHTransferStatus defines the clearing status of an ACH transfer.
type ACHTransferStatus string

const (
	TransferStatusPending   ACHTransferStatus = "pending"
	TransferStatusPosted    ACHTransferStatus = "posted"
	TransferStatusSettled   ACHTransferStatus = "settled"
	TransferStatusFailed    ACHTransferStatus = "failed"
	TransferStatusReturned  ACHTransferStatus = "returned"
	TransferStatusCancelled ACHTransferStatus = "cancelled"
)

// ACHTransfer represents an automated clearing house transaction between a bank account and OCI custody.
type ACHTransfer struct {
	TransferID        string            `json:"transfer_id"`
	AccountID         string            `json:"account_id"`
	Amount            float64           `json:"amount"`
	Direction         string            `json:"direction"`          // debit (pull funds to OCI), credit (push funds to bank)
	Type              string            `json:"type"`               // ach
	Status            ACHTransferStatus `json:"status"`             // pending, posted, settled, returned, failed
	ReturnCode        string            `json:"return_code,omitempty"` // NACHA return code, e.g. R01 (Insufficient Funds)
	Description       string            `json:"description"`
	CreatedAt         time.Time         `json:"created_at"`
	SettledAt         *time.Time        `json:"settled_at,omitempty"`
	CustodyAccountID  string            `json:"custody_account_id"`
}

// LinkToken represents a Plaid-style short-lived token to initialize account linking.
type LinkToken struct {
	LinkToken    string    `json:"link_token"`
	ClientUserID string    `json:"client_user_id"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// AccessToken represents a linked institution token granting access to account information.
type AccessToken struct {
	AccessToken string    `json:"access_token"`
	ItemID      string    `json:"item_id"`
	OwnerID     string    `json:"owner_id"`
	AccountIDs  []string  `json:"account_ids"`
	CreatedAt   time.Time `json:"created_at"`
}
