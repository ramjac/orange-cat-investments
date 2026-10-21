package ledger

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrAccountNotFound    = errors.New("bank account not found")
	ErrInsufficientFunds  = errors.New("insufficient funds (R01)")
	ErrInvalidAmount      = errors.New("transfer amount must be greater than zero")
	ErrTransferNotFound   = errors.New("transfer not found")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrCustodyNotFound    = errors.New("custody account not found")
)

const (
	DefaultCustodyAccountID = "acc_oci_custody_001"
)

// LedgerStore holds ephemeral in-memory state for bank accounts and ACH transfers.
type LedgerStore struct {
	mu               sync.RWMutex
	accounts         map[string]*BankAccount
	transfers        map[string]*ACHTransfer
	linkTokens       map[string]*LinkToken
	accessTokens     map[string]*AccessToken
	custodyAccountID string
}

// NewLedgerStore creates a new thread-safe in-memory ledger store.
func NewLedgerStore() *LedgerStore {
	store := &LedgerStore{
		accounts:         make(map[string]*BankAccount),
		transfers:        make(map[string]*ACHTransfer),
		linkTokens:       make(map[string]*LinkToken),
		accessTokens:     make(map[string]*AccessToken),
		custodyAccountID: DefaultCustodyAccountID,
	}
	store.Seed()
	return store
}

// GetAccount retrieves a bank account by ID.
func (s *LedgerStore) GetAccount(accountID string) (*BankAccount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	acc, exists := s.accounts[accountID]
	if !exists {
		return nil, ErrAccountNotFound
	}
	cp := *acc
	return &cp, nil
}

// GetAccountsByIDs retrieves bank accounts matching a slice of account IDs.
func (s *LedgerStore) GetAccountsByIDs(accountIDs []string) []*BankAccount {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*BankAccount, 0, len(accountIDs))
	for _, id := range accountIDs {
		if acc, ok := s.accounts[id]; ok {
			cp := *acc
			result = append(result, &cp)
		}
	}
	return result
}

// GetAccountsByOwner retrieves all bank accounts owned by a given persona/customer ID.
func (s *LedgerStore) GetAccountsByOwner(ownerID string) []*BankAccount {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*BankAccount
	for _, acc := range s.accounts {
		if acc.OwnerID == ownerID {
			cp := *acc
			result = append(result, &cp)
		}
	}
	return result
}

// GetAccountsByAccessToken retrieves accounts associated with an authorized access token.
func (s *LedgerStore) GetAccountsByAccessToken(accessToken string) ([]*BankAccount, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tok, exists := s.accessTokens[accessToken]
	if !exists {
		return nil, ErrInvalidToken
	}

	var result []*BankAccount
	for _, accID := range tok.AccountIDs {
		if acc, ok := s.accounts[accID]; ok {
			cp := *acc
			result = append(result, &cp)
		}
	}
	return result, nil
}

// CreateLinkToken generates an ephemeral Plaid-style link session token.
func (s *LedgerStore) CreateLinkToken(clientUserID string) *LinkToken {
	s.mu.Lock()
	defer s.mu.Unlock()

	tokID := fmt.Sprintf("link-sandbox-%d", time.Now().UnixNano())
	now := time.Now().UTC()
	tok := &LinkToken{
		LinkToken:    tokID,
		ClientUserID: clientUserID,
		CreatedAt:    now,
		ExpiresAt:    now.Add(4 * time.Hour),
	}
	s.linkTokens[tokID] = tok
	return tok
}

// ExchangePublicToken exchanges a public link token for an access token.
func (s *LedgerStore) ExchangePublicToken(publicToken, clientUserID string) (*AccessToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var matchedAccountIDs []string
	for _, acc := range s.accounts {
		if acc.OwnerID == clientUserID {
			matchedAccountIDs = append(matchedAccountIDs, acc.AccountID)
		}
	}

	tokenID := fmt.Sprintf("access-sandbox-%d", time.Now().UnixNano())
	itemID := fmt.Sprintf("item-sandbox-%d", time.Now().UnixNano())
	tok := &AccessToken{
		AccessToken: tokenID,
		ItemID:      itemID,
		OwnerID:     clientUserID,
		AccountIDs:  matchedAccountIDs,
		CreatedAt:   time.Now().UTC(),
	}
	s.accessTokens[tokenID] = tok
	return tok, nil
}

// CreateTransfer initiates an ACH transfer (debit pull into OCI or credit push to bank account).
func (s *LedgerStore) CreateTransfer(accountID string, amount float64, direction, description string) (*ACHTransfer, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	acc, exists := s.accounts[accountID]
	if !exists {
		return nil, ErrAccountNotFound
	}

	custody, cExists := s.accounts[s.custodyAccountID]
	if !cExists {
		return nil, ErrCustodyNotFound
	}

	now := time.Now().UTC()
	transferID := fmt.Sprintf("tr_%d", now.UnixNano())

	if direction == "debit" {
		// Pulling funds from customer bank into OCI
		if acc.AvailableBalance < amount {
			failedTransfer := &ACHTransfer{
				TransferID:       transferID,
				AccountID:        accountID,
				Amount:           amount,
				Direction:        direction,
				Type:             "ach",
				Status:           TransferStatusFailed,
				ReturnCode:       "R01",
				Description:      description,
				CreatedAt:        now,
				CustodyAccountID: custody.AccountID,
			}
			s.transfers[transferID] = failedTransfer
			return failedTransfer, ErrInsufficientFunds
		}

		// Hold funds from available balance immediately
		acc.AvailableBalance -= amount
	}

	transfer := &ACHTransfer{
		TransferID:       transferID,
		AccountID:        accountID,
		Amount:           amount,
		Direction:        direction,
		Type:             "ach",
		Status:           TransferStatusPending,
		Description:      description,
		CreatedAt:        now,
		CustodyAccountID: custody.AccountID,
	}
	s.transfers[transferID] = transfer

	cp := *transfer
	return &cp, nil
}

// GetTransfer retrieves a transfer by its unique ID.
func (s *LedgerStore) GetTransfer(transferID string) (*ACHTransfer, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, exists := s.transfers[transferID]
	if !exists {
		return nil, ErrTransferNotFound
	}
	cp := *t
	return &cp, nil
}

// ListTransfers lists transfers filtered by accountID or returns all if empty.
func (s *LedgerStore) ListTransfers(accountID string) []*ACHTransfer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*ACHTransfer
	for _, t := range s.transfers {
		if accountID == "" || t.AccountID == accountID {
			cp := *t
			result = append(result, &cp)
		}
	}
	return result
}

// GetPendingTransfers returns all transfers currently in pending or posted status.
func (s *LedgerStore) GetPendingTransfers() []*ACHTransfer {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*ACHTransfer
	for _, t := range s.transfers {
		if t.Status == TransferStatusPending || t.Status == TransferStatusPosted {
			cp := *t
			result = append(result, &cp)
		}
	}
	return result
}

// UpdateTransferStatus updates the status and handles clearing balances.
func (s *LedgerStore) UpdateTransferStatus(transferID string, targetStatus ACHTransferStatus, returnCode string) (*ACHTransfer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	transfer, exists := s.transfers[transferID]
	if !exists {
		return nil, ErrTransferNotFound
	}

	acc, accExists := s.accounts[transfer.AccountID]
	if !accExists {
		return nil, ErrAccountNotFound
	}

	custody, cExists := s.accounts[s.custodyAccountID]
	if !cExists {
		return nil, ErrCustodyNotFound
	}

	oldStatus := transfer.Status
	transfer.Status = targetStatus

	now := time.Now().UTC()

	switch targetStatus {
	case TransferStatusSettled:
		transfer.SettledAt = &now
		transfer.ReturnCode = ""
		if transfer.Direction == "debit" {
			// Complete debit: clear CurrentBalance hold
			acc.CurrentBalance -= transfer.Amount
			custody.CurrentBalance += transfer.Amount
			custody.AvailableBalance += transfer.Amount
		} else if transfer.Direction == "credit" {
			// Complete credit to customer
			acc.CurrentBalance += transfer.Amount
			acc.AvailableBalance += transfer.Amount
			custody.CurrentBalance -= transfer.Amount
			custody.AvailableBalance -= transfer.Amount
		}

	case TransferStatusReturned, TransferStatusFailed, TransferStatusCancelled:
		transfer.ReturnCode = returnCode
		// If debit was held in pending/posted state, release the available balance hold
		if transfer.Direction == "debit" && (oldStatus == TransferStatusPending || oldStatus == TransferStatusPosted) {
			acc.AvailableBalance += transfer.Amount
		}
	}

	cp := *transfer
	return &cp, nil
}
