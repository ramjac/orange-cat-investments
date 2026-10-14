package ledger

import "time"

// Seed populates the in-memory ledger with accounts and balances for Arthur, Chloe, and OCI Omnibus.
func (s *LedgerStore) Seed() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.accounts = make(map[string]*BankAccount)
	s.transfers = make(map[string]*ACHTransfer)
	s.linkTokens = make(map[string]*LinkToken)
	s.accessTokens = make(map[string]*AccessToken)
	s.custodyAccountID = DefaultCustodyAccountID

	// 1. Arthur Pendelton Accounts
	arthurChk := &BankAccount{
		AccountID:        "acc_arthur_chase_chk",
		InstitutionID:    "ins_chase",
		InstitutionName:  "JPMorgan Chase",
		RoutingNumber:    "021000021",
		AccountNumber:    "****4821",
		AccountType:      "depository",
		AccountSubtype:   "checking",
		CurrentBalance:   48500.00,
		AvailableBalance: 48250.00,
		Currency:         "USD",
		OwnerID:          "cust-longterm-arthur",
		Name:             "Chase Checking (****4821)",
	}
	arthurSav := &BankAccount{
		AccountID:        "acc_arthur_chase_sav",
		InstitutionID:    "ins_chase",
		InstitutionName:  "JPMorgan Chase",
		RoutingNumber:    "021000021",
		AccountNumber:    "****9104",
		AccountType:      "depository",
		AccountSubtype:   "savings",
		CurrentBalance:   125000.00,
		AvailableBalance: 125000.00,
		Currency:         "USD",
		OwnerID:          "cust-longterm-arthur",
		Name:             "Chase Sapphire Savings (****9104)",
	}

	// 2. Chloe Spark Accounts
	chloeChk := &BankAccount{
		AccountID:        "acc_chloe_svcu_chk",
		InstitutionID:    "ins_svcu",
		InstitutionName:  "Silicon Valley Tech CU",
		RoutingNumber:    "321171184",
		AccountNumber:    "****7732",
		AccountType:      "depository",
		AccountSubtype:   "checking",
		CurrentBalance:   22800.00,
		AvailableBalance: 22400.00,
		Currency:         "USD",
		OwnerID:          "cust-active-chloe",
		Name:             "High-Yield Tech Checking (****7732)",
	}

	// 3. OCI Master Brokerage Clearing / Custody Omnibus
	ociCustody := &BankAccount{
		AccountID:        DefaultCustodyAccountID,
		InstitutionID:    "ins_bnymellon",
		InstitutionName:  "BNY Mellon",
		RoutingNumber:    "011000138",
		AccountNumber:    "****9900",
		AccountType:      "depository",
		AccountSubtype:   "fbo_omnibus",
		CurrentBalance:   10000000.00,
		AvailableBalance: 10000000.00,
		Currency:         "USD",
		OwnerID:          "org-oci-custody",
		Name:             "OCI Client Cash Omnibus (****9900)",
	}

	s.accounts[arthurChk.AccountID] = arthurChk
	s.accounts[arthurSav.AccountID] = arthurSav
	s.accounts[chloeChk.AccountID] = chloeChk
	s.accounts[ociCustody.AccountID] = ociCustody

	// Pre-seed standard test Access Tokens
	now := time.Now().UTC()
	s.accessTokens["access-sandbox-arthur"] = &AccessToken{
		AccessToken: "access-sandbox-arthur",
		ItemID:      "item-sandbox-arthur",
		OwnerID:     "cust-longterm-arthur",
		AccountIDs:  []string{arthurChk.AccountID, arthurSav.AccountID},
		CreatedAt:   now,
	}
	s.accessTokens["access-sandbox-chloe"] = &AccessToken{
		AccessToken: "access-sandbox-chloe",
		ItemID:      "item-sandbox-chloe",
		OwnerID:     "cust-active-chloe",
		AccountIDs:  []string{chloeChk.AccountID},
		CreatedAt:   now,
	}
}
