package frappe

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Client handles REST communication with Frappe HR / ERPNext.
type Client struct {
	baseURL    string
	apiKey     string
	apiSecret  string
	httpClient *http.Client
}

// Config holds configuration parameters for the Frappe HR client.
type Config struct {
	BaseURL    string
	APIKey     string
	APISecret  string
	HTTPClient *http.Client
}

// Timesheet represents a Frappe HR Timesheet DocType.
type Timesheet struct {
	Name              string     `json:"name,omitempty"`
	TimesheetID       string     `json:"timesheet_id,omitempty"`
	Employee          string     `json:"employee,omitempty"`
	EmployeeID        string     `json:"employee_id,omitempty"`
	StartDate         string     `json:"start_date,omitempty"`
	ShiftDate         string     `json:"shift_date,omitempty"`
	EndDate           string     `json:"end_date,omitempty"`
	StartTime         *time.Time `json:"start_time,omitempty"`
	EndTime           *time.Time `json:"end_time,omitempty"`
	BreakMinutes      int        `json:"break_minutes,omitempty"`
	TotalHours        float64    `json:"total_hours,omitempty"`
	HoursWorked       float64    `json:"hours_worked,omitempty"`
	Billable          bool       `json:"billable,omitempty"`
	Status            string     `json:"status"` // "Draft", "Submitted", "Approved", "Cancelled", "submitted", "approved", "rejected"
	TimeLogs          []TimeLog  `json:"time_logs,omitempty"`
	TaskCode          string     `json:"task_code,omitempty"`
	CostCenter        string     `json:"cost_center,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
	ApprovedBy        *string    `json:"approved_by,omitempty"`
	ApprovedAt        *time.Time `json:"approved_at,omitempty"`
	FrappeTimesheetID *string    `json:"frappe_timesheet_id,omitempty"`
	CreatedAt         time.Time  `json:"created_at,omitempty"`
	UpdatedAt         time.Time  `json:"updated_at,omitempty"`
}

// Normalize ensures canonical mapping between Frappe HR DocType attributes and domain fields.
func (t *Timesheet) Normalize() {
	if t.TimesheetID == "" && t.Name != "" {
		t.TimesheetID = t.Name
	}
	if t.Name == "" && t.TimesheetID != "" {
		t.Name = t.TimesheetID
	}
	if t.EmployeeID == "" && t.Employee != "" {
		t.EmployeeID = t.Employee
	}
	if t.Employee == "" && t.EmployeeID != "" {
		t.Employee = t.EmployeeID
	}
	if t.ShiftDate == "" && t.StartDate != "" {
		t.ShiftDate = t.StartDate
	}
	if t.StartDate == "" && t.ShiftDate != "" {
		t.StartDate = t.ShiftDate
	}
	if t.EndDate == "" && t.ShiftDate != "" {
		t.EndDate = t.ShiftDate
	}
	if t.HoursWorked == 0 && t.TotalHours != 0 {
		t.HoursWorked = t.TotalHours
	}
	if t.TotalHours == 0 && t.HoursWorked != 0 {
		t.TotalHours = t.HoursWorked
	}
	if t.FrappeTimesheetID == nil && t.Name != "" {
		name := t.Name
		t.FrappeTimesheetID = &name
	}
	if t.TaskCode == "" && len(t.TimeLogs) > 0 {
		t.TaskCode = t.TimeLogs[0].ActivityType
	}
	if t.CostCenter == "" && len(t.TimeLogs) > 0 {
		t.CostCenter = t.TimeLogs[0].CostCenter
	}
	if !t.Billable && len(t.TimeLogs) > 0 && t.TimeLogs[0].Billable == 1 {
		t.Billable = true
	}
	if t.StartTime == nil && len(t.TimeLogs) > 0 && t.TimeLogs[0].FromTime != "" {
		if pt, err := time.Parse("2006-01-02 15:04:05", t.TimeLogs[0].FromTime); err == nil {
			t.StartTime = &pt
		} else if pt, err := time.Parse(time.RFC3339, t.TimeLogs[0].FromTime); err == nil {
			t.StartTime = &pt
		}
	}
	if t.EndTime == nil && len(t.TimeLogs) > 0 && t.TimeLogs[0].ToTime != "" {
		if pt, err := time.Parse("2006-01-02 15:04:05", t.TimeLogs[0].ToTime); err == nil {
			t.EndTime = &pt
		} else if pt, err := time.Parse(time.RFC3339, t.TimeLogs[0].ToTime); err == nil {
			t.EndTime = &pt
		}
	}
	if len(t.TimeLogs) == 0 && t.StartTime != nil && t.EndTime != nil {
		billableInt := 0
		if t.Billable {
			billableInt = 1
		}
		t.TimeLogs = []TimeLog{
			{
				ActivityType: t.TaskCode,
				FromTime:     t.StartTime.Format("2006-01-02 15:04:05"),
				ToTime:       t.EndTime.Format("2006-01-02 15:04:05"),
				Hours:        t.HoursWorked,
				Billable:     billableInt,
				CostCenter:   t.CostCenter,
			},
		}
	}
	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now().UTC()
	}
	if t.UpdatedAt.IsZero() {
		t.UpdatedAt = time.Now().UTC()
	}
}

// TimeLog represents a row in Frappe HR Timesheet Detail table.
type TimeLog struct {
	ActivityType string  `json:"activity_type"`
	FromTime     string  `json:"from_time"`
	ToTime       string  `json:"to_time"`
	Hours        float64 `json:"hours"`
	Billable     int     `json:"billable"` // 1 or 0
	CostCenter   string  `json:"cost_center,omitempty"`
}

// SalaryComponent represents an itemized earning or deduction row on a Salary Slip.
type SalaryComponent struct {
	SalaryComponent string  `json:"salary_component"`
	Type            string  `json:"type"` // "Earning" or "Deduction"
	Amount          float64 `json:"amount"`
	Description     string  `json:"description,omitempty"`
}

// SalarySlip represents a Frappe HR Salary Slip DocType.
type SalarySlip struct {
	Name            string            `json:"name,omitempty"`
	SalarySlipID    string            `json:"salary_slip_id,omitempty"`
	Employee        string            `json:"employee"`
	EmployeeID      string            `json:"employee_id,omitempty"`
	EmployeeName    string            `json:"employee_name"`
	PostingDate     string            `json:"posting_date"`
	StartDate       string            `json:"start_date"`
	EndDate         string            `json:"end_date"`
	GrossPay        float64           `json:"gross_pay"`
	TotalDeductions float64           `json:"total_deductions"`
	NetPay          float64           `json:"net_pay"`
	Currency        string            `json:"currency,omitempty"`
	Status          string            `json:"status"` // "Draft", "Submitted", "Cancelled"
	Earnings        []SalaryComponent `json:"earnings,omitempty"`
	Deductions      []SalaryComponent `json:"deductions,omitempty"`
	PayrollEntry    *string           `json:"payroll_entry,omitempty"`
}

// Normalize ensures canonical mapping between Frappe HR DocType attributes and domain fields.
func (s *SalarySlip) Normalize() {
	if s.SalarySlipID == "" && s.Name != "" {
		s.SalarySlipID = s.Name
	}
	if s.Name == "" && s.SalarySlipID != "" {
		s.Name = s.SalarySlipID
	}
	if s.EmployeeID == "" && s.Employee != "" {
		s.EmployeeID = s.Employee
	}
	if s.Employee == "" && s.EmployeeID != "" {
		s.Employee = s.EmployeeID
	}
	if s.Currency == "" {
		s.Currency = "USD"
	}
}

// PayrollEntry represents a Frappe HR Payroll Entry DocType for executing bulk payroll runs.
type PayrollEntry struct {
	Name                 string   `json:"name,omitempty"`
	PayrollEntryID       string   `json:"payroll_entry_id,omitempty"`
	PostingDate          string   `json:"posting_date"`
	Company              string   `json:"company,omitempty"`
	Currency             string   `json:"currency,omitempty"`
	PayrollFrequency     string   `json:"payroll_frequency,omitempty"` // "Monthly", "Weekly", "Biweekly"
	StartDate            string   `json:"start_date"`
	EndDate              string   `json:"end_date"`
	Department           string   `json:"department,omitempty"`
	Status               string   `json:"status"` // "Draft", "Submitted", "Cancelled"
	SalarySlipsCreated   int      `json:"salary_slips_created,omitempty"`
	SalarySlipsSubmitted int      `json:"salary_slips_submitted,omitempty"`
	TotalGrossPay        float64  `json:"total_gross_pay,omitempty"`
	TotalNetPay          float64  `json:"total_net_pay,omitempty"`
	Employees            []string `json:"employees,omitempty"`
}

func (p *PayrollEntry) Normalize() {
	if p.PayrollEntryID == "" && p.Name != "" {
		p.PayrollEntryID = p.Name
	}
	if p.Name == "" && p.PayrollEntryID != "" {
		p.Name = p.PayrollEntryID
	}
	if p.Company == "" {
		p.Company = "Orange Cat Investments LLC"
	}
	if p.Currency == "" {
		p.Currency = "USD"
	}
	if p.PayrollFrequency == "" {
		p.PayrollFrequency = "Monthly"
	}
	if p.Status == "" {
		p.Status = "Submitted"
	}
}

// Attendance represents a Frappe HR Attendance DocType.
type Attendance struct {
	Name           string `json:"name,omitempty"`
	Employee       string `json:"employee"`
	AttendanceDate string `json:"attendance_date"`
	Status         string `json:"status"` // "Present", "Absent", "Half Day", "On Leave"
	Shift          string `json:"shift,omitempty"`
	InTime         string `json:"in_time,omitempty"`
	OutTime        string `json:"out_time,omitempty"`
	LateEntry      int    `json:"late_entry,omitempty"`
	EarlyExit      int    `json:"early_exit,omitempty"`
}

// NewClient creates a new Frappe HR REST client.
func NewClient(cfg Config) *Client {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "http://frappe-hr-service.oci-internal.svc.cluster.local:8000"
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	return &Client{
		baseURL:    baseURL,
		apiKey:     cfg.APIKey,
		apiSecret:  cfg.APISecret,
		httpClient: httpClient,
	}
}

func (c *Client) do(ctx context.Context, method, path string, body any, target any) error {
	fullURL := c.baseURL + path
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" && c.apiSecret != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s:%s", c.apiKey, c.apiSecret))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("frappe api error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	if target != nil {
		// Frappe wraps doc data in {"data": ...}
		wrapper := struct {
			Data json.RawMessage `json:"data"`
		}{}
		if err := json.Unmarshal(respBytes, &wrapper); err == nil && len(wrapper.Data) > 0 {
			if err := json.Unmarshal(wrapper.Data, target); err != nil {
				return json.Unmarshal(respBytes, target)
			}
			return nil
		}
		return json.Unmarshal(respBytes, target)
	}

	return nil
}

// CreateTimesheet creates a new Timesheet in Frappe HR.
func (c *Client) CreateTimesheet(ctx context.Context, ts *Timesheet) (*Timesheet, error) {
	var result Timesheet
	err := c.do(ctx, http.MethodPost, "/api/resource/Timesheet", ts, &result)
	if err != nil {
		return nil, err
	}
	result.Normalize()
	return &result, nil
}

// GetTimesheet retrieves a timesheet by ID/name.
func (c *Client) GetTimesheet(ctx context.Context, name string) (*Timesheet, error) {
	var result Timesheet
	err := c.do(ctx, http.MethodGet, "/api/resource/Timesheet/"+name, nil, &result)
	if err != nil {
		return nil, err
	}
	result.Normalize()
	return &result, nil
}

// ApproveTimesheet submits and approves a timesheet in Frappe HR (docstatus 1).
func (c *Client) ApproveTimesheet(ctx context.Context, name string) (*Timesheet, error) {
	updatePayload := map[string]any{
		"docstatus": 1,
		"status":    "Approved",
	}
	var result Timesheet
	err := c.do(ctx, http.MethodPut, "/api/resource/Timesheet/"+name, updatePayload, &result)
	if err != nil {
		return nil, err
	}
	result.Normalize()
	return &result, nil
}

// CancelTimesheet cancels a timesheet in Frappe HR (docstatus 2).
func (c *Client) CancelTimesheet(ctx context.Context, name string) (*Timesheet, error) {
	updatePayload := map[string]any{
		"docstatus": 2,
		"status":    "Cancelled",
	}
	var result Timesheet
	err := c.do(ctx, http.MethodPut, "/api/resource/Timesheet/"+name, updatePayload, &result)
	if err != nil {
		return nil, err
	}
	result.Normalize()
	return &result, nil
}

// ListTimesheets retrieves timesheets from Frappe HR.
func (c *Client) ListTimesheets(ctx context.Context, employeeID, status string) ([]Timesheet, error) {
	path := "/api/resource/Timesheet?fields=[\"*\"]"
	if employeeID != "" && status != "" {
		path = fmt.Sprintf("/api/resource/Timesheet?fields=[\"*\"]&filters=[[\"employee\",\"=\",\"%s\"],[\"status\",\"=\",\"%s\"]]", employeeID, status)
	} else if employeeID != "" {
		path = fmt.Sprintf("/api/resource/Timesheet?fields=[\"*\"]&filters=[[\"employee\",\"=\",\"%s\"]]", employeeID)
	} else if status != "" {
		path = fmt.Sprintf("/api/resource/Timesheet?fields=[\"*\"]&filters=[[\"status\",\"=\",\"%s\"]]", status)
	}

	var list []Timesheet
	err := c.do(ctx, http.MethodGet, path, nil, &list)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Normalize()
	}
	return list, nil
}

// MarkAttendance logs daily attendance in Frappe HR.
func (c *Client) MarkAttendance(ctx context.Context, att *Attendance) (*Attendance, error) {
	var result Attendance
	err := c.do(ctx, http.MethodPost, "/api/resource/Attendance", att, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetAttendance retrieves an attendance record by ID/name.
func (c *Client) GetAttendance(ctx context.Context, name string) (*Attendance, error) {
	var result Attendance
	err := c.do(ctx, http.MethodGet, "/api/resource/Attendance/"+name, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListAttendance queries attendance records.
func (c *Client) ListAttendance(ctx context.Context, employeeID string) ([]Attendance, error) {
	path := "/api/resource/Attendance?fields=[\"*\"]"
	if employeeID != "" {
		path = fmt.Sprintf("/api/resource/Attendance?fields=[\"*\"]&filters=[[\"employee\",\"=\",\"%s\"]]", employeeID)
	}
	var list []Attendance
	err := c.do(ctx, http.MethodGet, path, nil, &list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// GetSalarySlips retrieves salary slips for an employee or all slips if employeeID is empty.
func (c *Client) GetSalarySlips(ctx context.Context, employeeID string) ([]SalarySlip, error) {
	path := "/api/resource/Salary Slip?fields=[\"*\"]"
	if employeeID != "" {
		path = fmt.Sprintf("/api/resource/Salary Slip?fields=[\"*\"]&filters=[[\"employee\",\"=\",\"%s\"]]", employeeID)
	}
	var slips []SalarySlip
	err := c.do(ctx, http.MethodGet, path, nil, &slips)
	if err != nil {
		return nil, err
	}
	for i := range slips {
		slips[i].Normalize()
	}
	return slips, nil
}

// GetSalarySlip retrieves a specific salary slip by name/ID.
func (c *Client) GetSalarySlip(ctx context.Context, name string) (*SalarySlip, error) {
	var slip SalarySlip
	err := c.do(ctx, http.MethodGet, "/api/resource/Salary Slip/"+name, nil, &slip)
	if err != nil {
		return nil, err
	}
	slip.Normalize()
	return &slip, nil
}

// CreateSalarySlip creates a single salary slip in Frappe HR.
func (c *Client) CreateSalarySlip(ctx context.Context, slip *SalarySlip) (*SalarySlip, error) {
	slip.Normalize()
	var result SalarySlip
	err := c.do(ctx, http.MethodPost, "/api/resource/Salary Slip", slip, &result)
	if err != nil {
		return nil, err
	}
	result.Normalize()
	return &result, nil
}

// CreatePayrollEntry creates and runs a Payroll Entry in Frappe HR.
func (c *Client) CreatePayrollEntry(ctx context.Context, pe *PayrollEntry) (*PayrollEntry, error) {
	pe.Normalize()
	var result PayrollEntry
	err := c.do(ctx, http.MethodPost, "/api/resource/Payroll Entry", pe, &result)
	if err != nil {
		return nil, err
	}
	result.Normalize()
	return &result, nil
}

// FinancialMetrics represents ERPNext General Ledger & Profit & Loss aggregated metrics.
type FinancialMetrics struct {
	Period            string  `json:"period"`
	Currency          string  `json:"currency"`
	TotalRevenue      float64 `json:"total_revenue"`
	OperatingExpenses float64 `json:"operating_expenses"`
	PayrollExpenses   float64 `json:"payroll_expenses"`
	NetProfit         float64 `json:"net_profit"`
	OperationalBurn   float64 `json:"operational_burn"`
	CashReserves      float64 `json:"cash_reserves"`
	RunwayMonths      float64 `json:"runway_months"`
}

// GetFinancialMetrics retrieves financial P&L, burn, and cash reserves from ERPNext.
func (c *Client) GetFinancialMetrics(ctx context.Context, period string) (*FinancialMetrics, error) {
	if period == "" {
		period = time.Now().UTC().Format("2006-01")
	}

	path := fmt.Sprintf("/api/v1/erpnext/financial-metrics?period=%s", period)
	var resp struct {
		Data *FinancialMetrics `json:"data"`
		*FinancialMetrics
	}
	err := c.do(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get financial metrics from ERPNext: %w", err)
	}
	if resp.Data != nil {
		return resp.Data, nil
	}
	if resp.FinancialMetrics != nil && resp.FinancialMetrics.TotalRevenue > 0 {
		return resp.FinancialMetrics, nil
	}
	return nil, errors.New("no financial metrics data returned from ERPNext")
}
