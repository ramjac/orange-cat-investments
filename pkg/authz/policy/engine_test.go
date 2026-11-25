package policy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkforcePolicy_TableDriven(t *testing.T) {
	ctx := context.Background()
	eng, err := NewEngine(ctx)
	require.NoError(t, err)

	tests := []struct {
		name      string
		in        WorkforceInput
		wantAllow bool
		wantMsg   string
	}{
		{
			name: "veterinary can toggle medical hold",
			in: WorkforceInput{
				Action:   "toggle_medical_hold",
				Actor:    Actor{ID: "emp-human-elena", Role: "veterinary"},
				Resource: WorkforceResource{FelineID: "emp-feline-garfield"},
			},
			wantAllow: true,
		},
		{
			name: "admin can toggle medical hold",
			in: WorkforceInput{
				Action:   "toggle_medical_hold",
				Actor:    Actor{ID: "emp-feline-garfield", Role: "admin"},
				Resource: WorkforceResource{FelineID: "emp-feline-garfield"},
			},
			wantAllow: true,
		},
		{
			name: "staff cannot toggle medical hold",
			in: WorkforceInput{
				Action:   "toggle_medical_hold",
				Actor:    Actor{ID: "emp-human-rick", Role: "staff"},
				Resource: WorkforceResource{FelineID: "emp-feline-garfield"},
			},
			wantAllow: false,
			wantMsg:   "not authorized to modify feline emergency medical holds",
		},
		{
			name: "empty role denied fail-closed",
			in: WorkforceInput{
				Action:   "toggle_medical_hold",
				Actor:    Actor{ID: "anonymous", Role: ""},
				Resource: WorkforceResource{FelineID: "emp-feline-garfield"},
			},
			wantAllow: false,
			wantMsg:   "authenticated actor role is required",
		},
		{
			name: "hr can offboard employee",
			in: WorkforceInput{
				Action:   "offboard_employee",
				Actor:    Actor{ID: "emp-human-alice", Role: "hr"},
				Resource: WorkforceResource{EmployeeID: "emp-human-rick"},
			},
			wantAllow: true,
		},
		{
			name: "engineering cannot offboard employee",
			in: WorkforceInput{
				Action:   "offboard_employee",
				Actor:    Actor{ID: "emp-human-elise", Role: "engineering"},
				Resource: WorkforceResource{EmployeeID: "emp-human-rick"},
			},
			wantAllow: false,
			wantMsg:   "not authorized to offboard employees",
		},
		{
			name: "hr can approve another employee leave",
			in: WorkforceInput{
				Action:   "approve_leave",
				Actor:    Actor{ID: "emp-human-alice", Role: "hr"},
				Resource: WorkforceResource{EmployeeID: "emp-human-rick"},
			},
			wantAllow: true,
		},
		{
			name: "hr cannot self-approve own leave",
			in: WorkforceInput{
				Action:   "approve_leave",
				Actor:    Actor{ID: "emp-human-alice", Role: "hr"},
				Resource: WorkforceResource{EmployeeID: "emp-human-alice"},
			},
			wantAllow: false,
			wantMsg:   "cannot self-approve",
		},
		{
			name: "staff cannot approve leave",
			in: WorkforceInput{
				Action:   "approve_leave",
				Actor:    Actor{ID: "emp-human-rick", Role: "staff"},
				Resource: WorkforceResource{EmployeeID: "emp-human-elise"},
			},
			wantAllow: false,
			wantMsg:   "not authorized to approve or reject leave",
		},
		{
			name: "veterinary can submit vet audit",
			in: WorkforceInput{
				Action:   "submit_vet_audit",
				Actor:    Actor{ID: "emp-human-elena", Role: "veterinary"},
				Resource: WorkforceResource{EmployeeID: "emp-feline-garfield"},
			},
			wantAllow: true,
		},
		{
			name: "facilities cannot submit vet audit",
			in: WorkforceInput{
				Action:   "submit_vet_audit",
				Actor:    Actor{ID: "emp-human-bob", Role: "facilities"},
				Resource: WorkforceResource{EmployeeID: "emp-feline-garfield"},
			},
			wantAllow: false,
			wantMsg:   "not authorized to submit veterinary wellness audits",
		},
		{
			name: "manager cannot self-assess own review",
			in: WorkforceInput{
				Action:   "submit_manager_assessment",
				Actor:    Actor{ID: "emp-human-alice", Role: "hr"},
				Resource: WorkforceResource{EmployeeID: "emp-human-alice"},
			},
			wantAllow: false,
			wantMsg:   "cannot submit a manager assessment for their own review",
		},
		{
			name: "hr can manage review cycle",
			in: WorkforceInput{
				Action:   "manage_review_cycle",
				Actor:    Actor{ID: "emp-human-alice", Role: "hr"},
				Resource: WorkforceResource{EmployeeID: "emp-human-rick"},
			},
			wantAllow: true,
		},
		{
			name: "unknown action denied by default",
			in: WorkforceInput{
				Action: "delete_database",
				Actor:  Actor{ID: "emp-human-alice", Role: "admin"},
			},
			wantAllow: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dec, err := eng.EvaluateWorkforce(ctx, tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.wantAllow, dec.Allow, "reasons: %v", dec.Reasons)
			if tc.wantMsg != "" {
				assert.Contains(t, dec.ReasonSummary(), tc.wantMsg)
			}
		})
	}
}

func TestTradingPolicy_TableDriven(t *testing.T) {
	ctx := context.Background()
	eng, err := NewEngine(ctx)
	require.NoError(t, err)

	tests := []struct {
		name      string
		in        TradingInput
		wantAllow bool
		wantMsg   string
	}{
		{
			name: "valid buy order within limits and no medical hold",
			in: TradingInput{
				Actor:   Actor{ID: "cust-active-chloe", Role: "customer"},
				Account: AccountState{Status: "active", BuyingPowerUSD: 50000},
				Feline:  FelineState{FelineID: "emp-feline-garfield", EmergencyMedicalHold: false},
				Order:   OrderRequest{Symbol: "NVDA", Side: "buy", Quantity: 10, OrderType: "market", EstimatedNotionalUSD: 1250},
			},
			wantAllow: true,
		},
		{
			name: "denied when feline emergency medical hold is active",
			in: TradingInput{
				Actor:   Actor{ID: "cust-active-chloe", Role: "customer"},
				Account: AccountState{Status: "active", BuyingPowerUSD: 50000},
				Feline:  FelineState{FelineID: "emp-feline-garfield", EmergencyMedicalHold: true},
				Order:   OrderRequest{Symbol: "NVDA", Side: "buy", Quantity: 10, OrderType: "market", EstimatedNotionalUSD: 1250},
			},
			wantAllow: false,
			wantMsg:   "emergency medical hold active",
		},
		{
			name: "allowed during medical hold for admin risk override",
			in: TradingInput{
				Actor:   Actor{ID: "emp-admin", Role: "admin"},
				Account: AccountState{Status: "active", BuyingPowerUSD: 50000},
				Feline:  FelineState{FelineID: "emp-feline-garfield", EmergencyMedicalHold: true},
				Order:   OrderRequest{Symbol: "NVDA", Side: "sell", Quantity: 10, OrderType: "market", EstimatedNotionalUSD: 1250},
			},
			wantAllow: true,
		},
		{
			name: "denied when account is closed",
			in: TradingInput{
				Actor:   Actor{ID: "cust-active-chloe", Role: "customer"},
				Account: AccountState{Status: "closed", BuyingPowerUSD: 50000},
				Feline:  FelineState{FelineID: "emp-feline-garfield", EmergencyMedicalHold: false},
				Order:   OrderRequest{Symbol: "NVDA", Side: "buy", Quantity: 5, OrderType: "market", EstimatedNotionalUSD: 600},
			},
			wantAllow: false,
			wantMsg:   `account status is "closed"`,
		},
		{
			name: "denied when order notional exceeds max limit",
			in: TradingInput{
				Actor:   Actor{ID: "cust-active-chloe", Role: "customer"},
				Account: AccountState{Status: "active", BuyingPowerUSD: 1000000},
				Feline:  FelineState{FelineID: "emp-feline-garfield", EmergencyMedicalHold: false},
				Order:   OrderRequest{Symbol: "NVDA", Side: "buy", Quantity: 5000, OrderType: "limit", EstimatedNotionalUSD: 500000},
				Limits:  TradingLimits{MaxOrderNotionalUSD: 250000},
			},
			wantAllow: false,
			wantMsg:   "exceeds single-order policy limit",
		},
		{
			name: "denied when buy exceeds buying power",
			in: TradingInput{
				Actor:   Actor{ID: "cust-active-chloe", Role: "customer"},
				Account: AccountState{Status: "active", BuyingPowerUSD: 1000},
				Feline:  FelineState{FelineID: "emp-feline-garfield", EmergencyMedicalHold: false},
				Order:   OrderRequest{Symbol: "NVDA", Side: "buy", Quantity: 20, OrderType: "limit", EstimatedNotionalUSD: 2500},
			},
			wantAllow: false,
			wantMsg:   "insufficient buying power",
		},
		{
			name: "denied when quantity <= 0",
			in: TradingInput{
				Actor:   Actor{ID: "cust-active-chloe", Role: "customer"},
				Account: AccountState{Status: "active"},
				Feline:  FelineState{FelineID: "emp-feline-garfield", EmergencyMedicalHold: false},
				Order:   OrderRequest{Symbol: "NVDA", Side: "buy", Quantity: 0, OrderType: "market"},
			},
			wantAllow: false,
			wantMsg:   "order quantity must be strictly positive",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dec, err := eng.EvaluateTrading(ctx, tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.wantAllow, dec.Allow, "reasons: %v", dec.Reasons)
			if tc.wantMsg != "" {
				assert.Contains(t, dec.ReasonSummary(), tc.wantMsg)
			}
		})
	}
}
