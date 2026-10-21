package frappe

import (
	"context"
	"time"

	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
)

// Client wraps frappe.Client with simulation conveniences such as
// simulated financial metrics fallbacks for offline testing.
type Client struct {
	*frappe.Client
}

// Config re-exports frappe.Config.
type Config = frappe.Config

// FinancialMetrics re-exports frappe.FinancialMetrics.
type FinancialMetrics = frappe.FinancialMetrics

// NewClient initializes a simulation Frappe client.
func NewClient(cfg Config) *Client {
	return &Client{
		Client: frappe.NewClient(cfg),
	}
}

// GetFinancialMetrics retrieves financial metrics, falling back to simulated estimates
// if the live ERPNext instance is not reachable.
func (c *Client) GetFinancialMetrics(ctx context.Context, period string) (*FinancialMetrics, error) {
	if period == "" {
		period = time.Now().UTC().Format("2006-01")
	}
	metrics, err := c.Client.GetFinancialMetrics(ctx, period)
	if err == nil && metrics != nil {
		return metrics, nil
	}

	// Fallback simulated metrics for test and simulation personas
	totalRev := 1250000.00
	operatingExp := 340000.00
	payrollTotal := 420000.00
	cashReserves := 9850000.00
	burn := operatingExp + payrollTotal
	netProfit := totalRev - burn
	runway := 0.0
	if burn > 0 {
		runway = cashReserves / burn
	}

	return &FinancialMetrics{
		Period:            period,
		Currency:          "USD",
		TotalRevenue:      totalRev,
		OperatingExpenses: operatingExp,
		PayrollExpenses:   payrollTotal,
		NetProfit:         netProfit,
		OperationalBurn:   burn,
		CashReserves:      cashReserves,
		RunwayMonths:      runway,
	}, nil
}
