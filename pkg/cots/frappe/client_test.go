package frappe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFrappeClient_CreateAndApproveTimesheet(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Verify authorization header format
		auth := r.Header.Get("Authorization")
		if auth != "token test-key:test-secret" {
			t.Errorf("unexpected authorization header: %s", auth)
		}

		if r.Method == http.MethodPost && r.URL.Path == "/api/resource/Timesheet" {
			var body Timesheet
			_ = json.NewDecoder(r.Body).Decode(&body)
			body.Name = "TS-2026-0001"
			body.Status = "Draft"
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"data": body})
			return
		}

		if r.Method == http.MethodPut && r.URL.Path == "/api/resource/Timesheet/TS-2026-0001" {
			res := Timesheet{
				Name:       "TS-2026-0001",
				Employee:   "emp-human-bob",
				Status:     "Approved",
				TotalHours: 8.0,
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"data": res})
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/api/resource/Salary Slip" {
			slips := []SalarySlip{
				{
					Name:         "SAL-SLIP-001",
					Employee:     "emp-human-bob",
					EmployeeName: "Bob Builder",
					GrossPay:     5000.0,
					NetPay:       4200.0,
					Status:       "Submitted",
				},
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"data": slips})
			return
		}

		if r.Method == http.MethodGet && r.URL.Path == "/api/resource/Salary Slip/SAL-SLIP-001" {
			slip := SalarySlip{
				Name:            "SAL-SLIP-001",
				Employee:        "emp-human-bob",
				EmployeeName:    "Bob Builder",
				PostingDate:     "2026-10-15",
				GrossPay:        5000.0,
				TotalDeductions: 800.0,
				NetPay:          4200.0,
				Status:          "Submitted",
			}
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"data": slip})
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/api/resource/Payroll Entry" {
			var pe PayrollEntry
			_ = json.NewDecoder(r.Body).Decode(&pe)
			pe.Name = "PAY-2026-10"
			pe.Status = "Submitted"
			pe.SalarySlipsCreated = 3
			pe.SalarySlipsSubmitted = 3
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{"data": pe})
			return
		}

		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	client := NewClient(Config{
		BaseURL:   mockServer.URL,
		APIKey:    "test-key",
		APISecret: "test-secret",
	})

	ctx := context.Background()

	// 1. Create Timesheet
	ts, err := client.CreateTimesheet(ctx, &Timesheet{
		Employee:   "emp-human-bob",
		StartDate:  "2026-10-15",
		EndDate:    "2026-10-15",
		TotalHours: 8.0,
	})
	if err != nil {
		t.Fatalf("failed to create timesheet: %v", err)
	}
	if ts.Name != "TS-2026-0001" {
		t.Errorf("expected timesheet name TS-2026-0001, got %s", ts.Name)
	}

	// 2. Approve Timesheet
	approved, err := client.ApproveTimesheet(ctx, ts.Name)
	if err != nil {
		t.Fatalf("failed to approve timesheet: %v", err)
	}
	if approved.Status != "Approved" {
		t.Errorf("expected status Approved, got %s", approved.Status)
	}

	// 3. Get Salary Slips
	slips, err := client.GetSalarySlips(ctx, "emp-human-bob")
	if err != nil {
		t.Fatalf("failed to get salary slips: %v", err)
	}
	if len(slips) != 1 || slips[0].NetPay != 4200.0 {
		t.Errorf("unexpected salary slips: %+v", slips)
	}

	// 4. Get Salary Slip by ID
	singleSlip, err := client.GetSalarySlip(ctx, "SAL-SLIP-001")
	if err != nil {
		t.Fatalf("failed to get single salary slip: %v", err)
	}
	if singleSlip.Name != "SAL-SLIP-001" || singleSlip.NetPay != 4200.0 {
		t.Errorf("unexpected single salary slip: %+v", singleSlip)
	}

	// 5. Create Payroll Entry
	pe, err := client.CreatePayrollEntry(ctx, &PayrollEntry{
		StartDate: "2026-10-01",
		EndDate:   "2026-10-31",
	})
	if err != nil {
		t.Fatalf("failed to create payroll entry: %v", err)
	}
	if pe.Name != "PAY-2026-10" || pe.SalarySlipsCreated != 3 {
		t.Errorf("unexpected payroll entry result: %+v", pe)
	}
}

func TestFrappeClient_GetFinancialMetrics(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/erpnext/financial-metrics" {
			metrics := FinancialMetrics{
				Period:            "2026-10",
				Currency:          "USD",
				TotalRevenue:      1500000.0,
				OperatingExpenses: 300000.0,
				PayrollExpenses:   400000.0,
				NetProfit:         800000.0,
				OperationalBurn:   700000.0,
				CashReserves:      10000000.0,
				RunwayMonths:      14.28,
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": metrics})
			return
		}
		http.NotFound(w, r)
	}))
	defer mockServer.Close()

	client := NewClient(Config{
		BaseURL:   mockServer.URL,
		APIKey:    "test-key",
		APISecret: "test-secret",
	})

	ctx := context.Background()
	metrics, err := client.GetFinancialMetrics(ctx, "2026-10")
	if err != nil {
		t.Fatalf("failed to get financial metrics: %v", err)
	}
	if metrics.TotalRevenue != 1500000.0 || metrics.NetProfit != 800000.0 {
		t.Errorf("unexpected metrics: %+v", metrics)
	}

	// Test error when endpoint is unreachable
	errorClient := NewClient(Config{
		BaseURL: "http://invalid-endpoint.local",
	})
	_, err = errorClient.GetFinancialMetrics(ctx, "2026-10")
	if err == nil {
		t.Fatalf("expected error from unreachable endpoint, got nil")
	}
}
