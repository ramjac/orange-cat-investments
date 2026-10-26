package workforce

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
)

func (s *workforceService) ListSalarySlips(ctx context.Context, employeeID string) ([]frappe.SalarySlip, error) {
	if s.hrClient != nil {
		slips, err := s.hrClient.GetSalarySlips(ctx, employeeID)
		if err != nil {
			return nil, err
		}
		for i := range slips {
			slips[i].Normalize()
		}
		return slips, nil
	}

	s.memMu.RLock()
	defer s.memMu.RUnlock()
	var list []frappe.SalarySlip
	for _, slip := range s.memSalarySlips {
		if employeeID != "" && slip.EmployeeID != employeeID && slip.Employee != employeeID {
			continue
		}
		copySlip := *slip
		list = append(list, copySlip)
	}
	return list, nil
}

func (s *workforceService) GetSalarySlip(ctx context.Context, id string) (*frappe.SalarySlip, error) {
	if id == "" {
		return nil, errors.New("salary slip id cannot be empty")
	}
	if s.hrClient != nil {
		slip, err := s.hrClient.GetSalarySlip(ctx, id)
		if err != nil {
			return nil, err
		}
		slip.Normalize()
		return slip, nil
	}

	s.memMu.RLock()
	defer s.memMu.RUnlock()
	slip, ok := s.memSalarySlips[id]
	if !ok {
		return nil, fmt.Errorf("salary slip with id %q not found", id)
	}
	copySlip := *slip
	return &copySlip, nil
}

func (s *workforceService) ProcessPayrollRun(ctx context.Context, pe *frappe.PayrollEntry) (*frappe.PayrollEntry, error) {
	if pe == nil {
		return nil, errors.New("payroll entry cannot be nil")
	}
	pe.Normalize()
	if pe.StartDate == "" || pe.EndDate == "" {
		return nil, errors.New("start_date and end_date are required for payroll run")
	}

	if s.hrClient != nil {
		return s.hrClient.CreatePayrollEntry(ctx, pe)
	}

	s.memMu.Lock()
	defer s.memMu.Unlock()
	if pe.PayrollEntryID == "" {
		pe.PayrollEntryID = fmt.Sprintf("PAY-RUN-%d", time.Now().UnixNano())
		pe.Name = pe.PayrollEntryID
	}
	pe.Status = "Submitted"
	if pe.PostingDate == "" {
		pe.PostingDate = time.Now().UTC().Format("2006-01-02")
	}

	if len(pe.Employees) == 0 && s.repo != nil {
		if emps, err := s.repo.ListEmployees(ctx, "human", "active"); err == nil && len(emps) > 0 {
			for _, emp := range emps {
				pe.Employees = append(pe.Employees, emp.EmployeeID)
			}
		}
	}
	pe.SalarySlipsCreated = len(pe.Employees)
	pe.SalarySlipsSubmitted = len(pe.Employees)
	var totalGross, totalNet float64

	for idx, empID := range pe.Employees {
		slipID := fmt.Sprintf("SAL-SLIP-%s-%03d", pe.PayrollEntryID, idx+1)
		gross := 5000.00
		deductions := 1200.00
		net := gross - deductions
		totalGross += gross
		totalNet += net

		slip := &frappe.SalarySlip{
			Name:            slipID,
			SalarySlipID:    slipID,
			Employee:        empID,
			EmployeeID:      empID,
			EmployeeName:    empID,
			PostingDate:     pe.PostingDate,
			StartDate:       pe.StartDate,
			EndDate:         pe.EndDate,
			GrossPay:        gross,
			TotalDeductions: deductions,
			NetPay:          net,
			Currency:        pe.Currency,
			Status:          "Submitted",
			PayrollEntry:    &pe.PayrollEntryID,
			Earnings: []frappe.SalaryComponent{
				{SalaryComponent: "Base Salary", Type: "Earning", Amount: gross},
			},
			Deductions: []frappe.SalaryComponent{
				{SalaryComponent: "Federal Tax", Type: "Deduction", Amount: 800.00},
				{SalaryComponent: "Social Security", Type: "Deduction", Amount: 400.00},
			},
		}
		slip.Normalize()
		s.memSalarySlips[slipID] = slip
	}
	pe.TotalGrossPay = totalGross
	pe.TotalNetPay = totalNet

	copyPE := *pe
	s.memPayrollEntries[pe.PayrollEntryID] = &copyPE
	return &copyPE, nil
}
