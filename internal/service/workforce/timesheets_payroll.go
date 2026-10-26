package workforce

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	"github.com/orange-cat-investments/oci/pkg/envelope"
)

type TimesheetFilter struct {
	EmployeeID string `json:"employee_id"`
	Status     string `json:"status"`
	FromDate   string `json:"from_date"`
	ToDate     string `json:"to_date"`
	TaskCode   string `json:"task_code"`
}

type TimesheetApprovedPayload struct {
	TimesheetID       string    `json:"timesheet_id"`
	EmployeeID        string    `json:"employee_id"`
	ShiftDate         string    `json:"shift_date"`
	HoursWorked       float64   `json:"hours_worked"`
	TaskCode          string    `json:"task_code"`
	CostCenter        string    `json:"cost_center"`
	ApprovedBy        string    `json:"approved_by"`
	ApprovedAt        time.Time `json:"approved_at"`
	FrappeTimesheetID *string   `json:"frappe_timesheet_id,omitempty"`
}

var validTaskCodes = map[string]bool{
	"habitat_maintenance": true,
	"feline_feeding":      true,
	"veterinary_assist":   true,
	"admin":               true,
	"invest_ops":          true,
}

func (s *workforceService) SubmitTimesheet(ctx context.Context, ts *frappe.Timesheet) (*frappe.Timesheet, error) {
	if ts == nil {
		return nil, errors.New("timesheet cannot be nil")
	}
	ts.Normalize()
	if ts.EmployeeID == "" {
		return nil, errors.New("employee_id is required")
	}
	if ts.StartTime == nil || ts.EndTime == nil {
		return nil, errors.New("start_time and end_time are required")
	}
	if !ts.StartTime.Before(*ts.EndTime) {
		return nil, errors.New("start_time must be before end_time")
	}
	duration := ts.EndTime.Sub(*ts.StartTime)
	if duration > 24*time.Hour {
		return nil, errors.New("shift duration cannot exceed 24 hours")
	}
	if ts.BreakMinutes < 0 {
		return nil, errors.New("break_minutes cannot be negative")
	}
	netDurationHours := duration.Hours() - float64(ts.BreakMinutes)/60.0
	if netDurationHours <= 0 {
		return nil, errors.New("net worked hours must be greater than zero after break")
	}
	if ts.HoursWorked <= 0 {
		ts.HoursWorked = math.Round(netDurationHours*100) / 100
		ts.TotalHours = ts.HoursWorked
	}
	if ts.ShiftDate == "" {
		ts.ShiftDate = ts.StartTime.Format("2006-01-02")
		ts.StartDate = ts.ShiftDate
		ts.EndDate = ts.ShiftDate
	}
	if !validTaskCodes[ts.TaskCode] {
		return nil, fmt.Errorf("invalid task_code %q: must be one of 'habitat_maintenance', 'feline_feeding', 'veterinary_assist', 'admin', 'invest_ops'", ts.TaskCode)
	}
	if ts.CostCenter == "" {
		switch ts.TaskCode {
		case "habitat_maintenance":
			ts.CostCenter = "Facilities Operations"
		case "feline_feeding", "veterinary_assist":
			ts.CostCenter = "Veterinary & Care"
		case "invest_ops":
			ts.CostCenter = "Trading & Investment Ops"
		default:
			ts.CostCenter = "General Administration"
		}
	}
	if ts.Status == "" {
		ts.Status = "submitted"
	}

	existing, err := s.ListTimesheets(ctx, TimesheetFilter{
		EmployeeID: ts.EmployeeID,
		FromDate:   ts.ShiftDate,
		ToDate:     ts.ShiftDate,
	})
	if err == nil {
		for _, e := range existing {
			if (e.TimesheetID != "" && e.TimesheetID == ts.TimesheetID) || (e.Name != "" && e.Name == ts.Name) || strings.EqualFold(e.Status, "rejected") || strings.EqualFold(e.Status, "cancelled") {
				continue
			}
			if e.StartTime != nil && e.EndTime != nil && ts.StartTime.Before(*e.EndTime) && e.StartTime.Before(*ts.EndTime) {
				return nil, fmt.Errorf("shift overlaps with existing timesheet %s (%s to %s)", e.TimesheetID, e.StartTime.Format(time.RFC3339), e.EndTime.Format(time.RFC3339))
			}
		}
	}

	now := time.Now().UTC()
	if ts.CreatedAt.IsZero() {
		ts.CreatedAt = now
	}
	ts.UpdatedAt = now

	if s.hrClient != nil {
		ts.Normalize()
		ts.Status = "Draft"
		created, err := s.hrClient.CreateTimesheet(ctx, ts)
		if err != nil {
			return nil, err
		}
		created.Normalize()
		created.Status = "submitted"
		if ts.StartTime != nil && created.StartTime == nil {
			created.StartTime = ts.StartTime
		}
		if ts.EndTime != nil && created.EndTime == nil {
			created.EndTime = ts.EndTime
		}
		if created.BreakMinutes == 0 && ts.BreakMinutes > 0 {
			created.BreakMinutes = ts.BreakMinutes
		}
		if created.Notes == nil && ts.Notes != nil {
			created.Notes = ts.Notes
		}
		return created, nil
	}

	s.memMu.Lock()
	defer s.memMu.Unlock()
	if ts.TimesheetID == "" {
		ts.TimesheetID = fmt.Sprintf("ts-%d", time.Now().UnixNano())
	}
	ts.Normalize()
	copyTS := *ts
	s.memTimesheets[ts.TimesheetID] = &copyTS
	return &copyTS, nil
}

func (s *workforceService) GetTimesheet(ctx context.Context, id string) (*frappe.Timesheet, error) {
	if id == "" {
		return nil, errors.New("timesheet id cannot be empty")
	}
	if s.hrClient != nil {
		ts, err := s.hrClient.GetTimesheet(ctx, id)
		if err != nil {
			return nil, err
		}
		ts.Normalize()
		return ts, nil
	}

	s.memMu.RLock()
	defer s.memMu.RUnlock()
	ts, ok := s.memTimesheets[id]
	if !ok {
		return nil, fmt.Errorf("timesheet with id %q not found", id)
	}
	copyTS := *ts
	return &copyTS, nil
}

func (s *workforceService) ListTimesheets(ctx context.Context, filter TimesheetFilter) ([]*frappe.Timesheet, error) {
	var list []*frappe.Timesheet

	if s.hrClient != nil {
		frappeList, err := s.hrClient.ListTimesheets(ctx, filter.EmployeeID, "")
		if err != nil {
			return nil, err
		}
		for i := range frappeList {
			frappeList[i].Normalize()
			list = append(list, &frappeList[i])
		}
	} else {
		s.memMu.RLock()
		for _, ts := range s.memTimesheets {
			copyTS := *ts
			list = append(list, &copyTS)
		}
		s.memMu.RUnlock()
	}

	var filtered []*frappe.Timesheet
	for _, ts := range list {
		if filter.EmployeeID != "" && ts.EmployeeID != filter.EmployeeID && ts.Employee != filter.EmployeeID {
			continue
		}
		if filter.Status != "" {
			match := strings.EqualFold(ts.Status, filter.Status)
			if !match {
				if strings.EqualFold(filter.Status, "submitted") && strings.EqualFold(ts.Status, "draft") {
					match = true
				} else if strings.EqualFold(filter.Status, "draft") && strings.EqualFold(ts.Status, "submitted") {
					match = true
				} else if strings.EqualFold(filter.Status, "rejected") && strings.EqualFold(ts.Status, "cancelled") {
					match = true
				}
			}
			if !match {
				continue
			}
		}
		if filter.FromDate != "" && ts.ShiftDate < filter.FromDate {
			continue
		}
		if filter.ToDate != "" && ts.ShiftDate > filter.ToDate {
			continue
		}
		if filter.TaskCode != "" && ts.TaskCode != filter.TaskCode {
			continue
		}
		filtered = append(filtered, ts)
	}

	return filtered, nil
}

func (s *workforceService) ApproveTimesheet(ctx context.Context, timesheetID, approverID string) (*frappe.Timesheet, error) {
	if timesheetID == "" {
		return nil, errors.New("timesheet id cannot be empty")
	}
	if approverID == "" {
		return nil, errors.New("approver id cannot be empty")
	}

	ts, err := s.GetTimesheet(ctx, timesheetID)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(ts.Status, "approved") {
		return ts, nil
	}

	now := time.Now().UTC()
	var updated *frappe.Timesheet

	if s.hrClient != nil {
		appr, err := s.hrClient.ApproveTimesheet(ctx, timesheetID)
		if err != nil {
			return nil, err
		}
		appr.Normalize()
		appr.ApprovedBy = &approverID
		appr.ApprovedAt = &now
		appr.Status = "approved"
		appr.UpdatedAt = now
		if ts.StartTime != nil && appr.StartTime == nil {
			appr.StartTime = ts.StartTime
		}
		if ts.EndTime != nil && appr.EndTime == nil {
			appr.EndTime = ts.EndTime
		}

		_, _ = s.hrClient.MarkAttendance(ctx, &frappe.Attendance{
			Employee:       appr.Employee,
			AttendanceDate: appr.ShiftDate,
			Status:         "Present",
		})

		updated = appr
	} else {
		s.memMu.Lock()
		ts.Status = "approved"
		ts.ApprovedBy = &approverID
		ts.ApprovedAt = &now
		ts.UpdatedAt = now
		s.memTimesheets[timesheetID] = ts
		copyTS := *ts
		updated = &copyTS
		s.memMu.Unlock()
	}

	if s.publisher != nil {
		evt := envelope.NewEventEnvelope(
			fmt.Sprintf("evt-ts-%d", time.Now().UnixNano()),
			"workforce.timesheet_approved",
			"",
			TimesheetApprovedPayload{
				TimesheetID:       updated.TimesheetID,
				EmployeeID:        updated.EmployeeID,
				ShiftDate:         updated.ShiftDate,
				HoursWorked:       updated.HoursWorked,
				TaskCode:          updated.TaskCode,
				CostCenter:        updated.CostCenter,
				ApprovedBy:        approverID,
				ApprovedAt:        now,
				FrappeTimesheetID: updated.FrappeTimesheetID,
			},
		)
		_ = s.publisher.Publish("events.workforce.timesheet_approved.v1", evt)
	}

	return updated, nil
}

func (s *workforceService) RejectTimesheet(ctx context.Context, timesheetID, reviewerID, reason string) (*frappe.Timesheet, error) {
	if timesheetID == "" {
		return nil, errors.New("timesheet id cannot be empty")
	}
	if reviewerID == "" {
		return nil, errors.New("reviewer id cannot be empty")
	}
	if reason == "" {
		return nil, errors.New("rejection reason is required")
	}

	ts, err := s.GetTimesheet(ctx, timesheetID)
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(ts.Status, "approved") {
		return nil, errors.New("cannot reject an already approved timesheet")
	}

	now := time.Now().UTC()
	var updated *frappe.Timesheet

	if s.hrClient != nil {
		canceled, err := s.hrClient.CancelTimesheet(ctx, timesheetID)
		if err != nil {
			return nil, err
		}
		canceled.Normalize()
		canceled.Status = "rejected"
		canceled.ApprovedBy = &reviewerID
		canceled.ApprovedAt = &now
		canceled.Notes = &reason
		canceled.UpdatedAt = now
		updated = canceled
	} else {
		s.memMu.Lock()
		ts.Status = "rejected"
		ts.ApprovedBy = &reviewerID
		ts.ApprovedAt = &now
		ts.Notes = &reason
		ts.UpdatedAt = now
		s.memTimesheets[timesheetID] = ts
		copyTS := *ts
		updated = &copyTS
		s.memMu.Unlock()
	}

	return updated, nil
}

