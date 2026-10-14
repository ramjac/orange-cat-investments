package ledger

import (
	"time"
)

// SeedDefaultAccounts populates initial accounts and holdings on startup.
func SeedDefaultAccounts(store *Store, initialPrices map[string]float64) {
	now := time.Now().UTC()

	// 1. Arthur Pendelton / OCI Alpha Growth Fund
	arthurPortfolioID := "018f3a9a-2222-7000-8000-000000000002"
	arthurAccount := &Account{
		ID:            arthurPortfolioID,
		AccountNumber: "OCI-COR-001",
		Status:        "ACTIVE",
		Currency:      "USD",
		Cash:          32100.25,
		AccountType:   "individual",
		CustomerName:  "Arthur Pendelton",
		CreatedAt:     now.Add(-180 * 24 * time.Hour),
	}
	_ = store.CreateAccount(arthurAccount)

	arthurHoldings := []struct {
		symbol   string
		qty      float64
		avgPrice float64
	}{
		{"AAPL", 120.0, 178.50},
		{"NVDA", 85.0, 110.20},
		{"MSFT", 90.0, 380.00},
		{"TSLA", 70.0, 190.00},
	}

	for _, h := range arthurHoldings {
		price := initialPrices[h.symbol]
		if price <= 0 {
			price = h.avgPrice
		}
		cost := h.qty * h.avgPrice
		pos := &Position{
			AssetID:        "ast-" + h.symbol,
			Symbol:         h.symbol,
			Exchange:       "NASDAQ",
			AssetClass:     "us_equity",
			Qty:            h.qty,
			AvgEntryPrice:  h.avgPrice,
			Side:           "long",
			CostBasis:      cost,
			CurrentPrice:   price,
			LastdayPrice:   price * 0.985,
			MarketValue:    h.qty * price,
			UnrealizedPL:   (h.qty * price) - cost,
			UnrealizedPLPC: ((h.qty * price) - cost) / cost,
		}
		store.SetPosition(arthurPortfolioID, pos)
	}

	// 2. Chloe Spark / High-Velocity Momentum Fund
	chloePortfolioID := "018f3a9a-2222-7000-8000-000000000003"
	chloeAccount := &Account{
		ID:            chloePortfolioID,
		AccountNumber: "OCI-MOM-002",
		Status:        "ACTIVE",
		Currency:      "USD",
		Cash:          15400.00,
		AccountType:   "individual",
		CustomerName:  "Chloe Spark",
		CreatedAt:     now.Add(-90 * 24 * time.Hour),
	}
	_ = store.CreateAccount(chloeAccount)

	chloeHoldings := []struct {
		symbol   string
		qty      float64
		avgPrice float64
	}{
		{"NVDA", 250.0, 124.00},
		{"TSLA", 180.0, 210.00},
	}

	for _, h := range chloeHoldings {
		price := initialPrices[h.symbol]
		if price <= 0 {
			price = h.avgPrice
		}
		cost := h.qty * h.avgPrice
		pos := &Position{
			AssetID:        "ast-" + h.symbol,
			Symbol:         h.symbol,
			Exchange:       "NASDAQ",
			AssetClass:     "us_equity",
			Qty:            h.qty,
			AvgEntryPrice:  h.avgPrice,
			Side:           "long",
			CostBasis:      cost,
			CurrentPrice:   price,
			LastdayPrice:   price * 0.99,
			MarketValue:    h.qty * price,
			UnrealizedPL:   (h.qty * price) - cost,
			UnrealizedPLPC: ((h.qty * price) - cost) / cost,
		}
		store.SetPosition(chloePortfolioID, pos)
	}

	// Recalculate portfolio valuations
	store.UpdateAccountValuations(initialPrices)
}
