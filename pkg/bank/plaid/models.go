package plaid

import "time"

// Account represents a bank account returned by the Bank API.
type Account struct {
	AccountID        string  `json:"account_id"`
	InstitutionID    string  `json:"institution_id"`
	InstitutionName  string  `json:"institution_name"`
	RoutingNumber    string  `json:"routing_number"`
	AccountNumber    string  `json:"account_number"`
	AccountType      string  `json:"account_type"`
	AccountSubtype   string  `json:"account_subtype"`
	CurrentBalance   float64 `json:"current_balance"`
	AvailableBalance float64 `json:"available_balance"`
	Currency         string  `json:"currency"`
	OwnerID          string  `json:"owner_id"`
	Name             string  `json:"name"`
}

// BalanceDetails represents available and current balances for an account.
type BalanceDetails struct {
	Available       float64 `json:"available"`
	Current         float64 `json:"current"`
	IsoCurrencyCode string  `json:"iso_currency_code"`
}

// AccountWithBalance represents Plaid balance endpoint response structure.
type AccountWithBalance struct {
	AccountID string         `json:"account_id"`
	Name      string         `json:"name"`
	Balances  BalanceDetails `json:"balances"`
	Mask      string         `json:"mask"`
	Type      string         `json:"type"`
}

// Transfer represents an ACH transfer object.
type Transfer struct {
	TransferID       string     `json:"transfer_id"`
	AccountID        string     `json:"account_id"`
	Amount           float64    `json:"amount"`
	Direction        string     `json:"direction"` // debit or credit
	Type             string     `json:"type"`
	Status           string     `json:"status"` // pending, posted, settled, returned, failed
	ReturnCode       string     `json:"return_code,omitempty"`
	Description      string     `json:"description"`
	CreatedAt        time.Time  `json:"created_at"`
	SettledAt        *time.Time `json:"settled_at,omitempty"`
	CustodyAccountID string     `json:"custody_account_id"`
}

// CreateTransferRequest is the input payload for initiating an ACH transfer.
type CreateTransferRequest struct {
	AccountID   string  `json:"account_id"`
	Amount      float64 `json:"amount"`
	Direction   string  `json:"direction,omitempty"` // default: debit
	Description string  `json:"description,omitempty"`
}

// DecisionRationale details why an ACH authorization was declined.
type DecisionRationale struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// TransferAuthorization represents an ACH pre-authorization response.
type TransferAuthorization struct {
	ID                string             `json:"id"`
	Decision          string             `json:"decision"` // approved or declined
	DecisionRationale *DecisionRationale `json:"decision_rationale,omitempty"`
}

// WebhookEvent represents an incoming event from the Bank microservice.
type WebhookEvent struct {
	WebhookType string    `json:"webhook_type"`
	WebhookCode string    `json:"webhook_code"`
	TransferID  string    `json:"transfer_id"`
	AccountID   string    `json:"account_id"`
	EventType   string    `json:"event_type"` // pending, posted, settled, returned, failed
	Amount      float64   `json:"amount"`
	ReturnCode  string    `json:"return_code,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}
