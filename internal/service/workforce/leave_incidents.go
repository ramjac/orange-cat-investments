package workforce

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
)

type CreateLeaveRequestParams struct {
	EmployeeID string
	LeaveType  string
	StartDate  string
	EndDate    string
	Reason     *string
}

func (s *workforceService) CreateLeaveRequest(ctx context.Context, params CreateLeaveRequestParams) (*workforce.LeaveRequest, error) {
	if params.EmployeeID == "" {
		return nil, errors.New("employee_id is required")
	}
	switch params.LeaveType {
	case "vacation", "sick", "catnip_break", "sabbatical":
		// valid
	default:
		return nil, errors.New("invalid leave_type: must be vacation, sick, catnip_break, or sabbatical")
	}

	if params.StartDate == "" || params.EndDate == "" {
		return nil, errors.New("start_date and end_date are required")
	}

	const dateLayout = "2006-01-02"
	start, err := time.Parse(dateLayout, params.StartDate)
	if err != nil {
		return nil, errors.New("start_date must be in YYYY-MM-DD format")
	}
	end, err := time.Parse(dateLayout, params.EndDate)
	if err != nil {
		return nil, errors.New("end_date must be in YYYY-MM-DD format")
	}
	if end.Before(start) {
		return nil, errors.New("end_date must be on or after start_date")
	}

	req := &workforce.LeaveRequest{
		EmployeeID: params.EmployeeID,
		LeaveType:  params.LeaveType,
		StartDate:  params.StartDate,
		EndDate:    params.EndDate,
		Status:     "pending",
		Reason:     params.Reason,
	}

	return s.repo.CreateLeaveRequest(ctx, req)
}

func (s *workforceService) GetLeaveRequest(ctx context.Context, id string) (*workforce.LeaveRequest, error) {
	if id == "" {
		return nil, errors.New("leave request id cannot be empty")
	}
	return s.repo.GetLeaveRequestByID(ctx, id)
}

func (s *workforceService) ListLeaveRequests(ctx context.Context, employeeID, status string) ([]*workforce.LeaveRequest, error) {
	if status != "" {
		switch status {
		case "pending", "approved", "rejected", "cancelled":
			// valid
		default:
			return nil, errors.New("invalid status: must be pending, approved, rejected, or cancelled")
		}
	}
	return s.repo.ListLeaveRequests(ctx, employeeID, status)
}

func (s *workforceService) UpdateLeaveRequestStatus(ctx context.Context, id, status string) (*workforce.LeaveRequest, error) {
	if id == "" {
		return nil, errors.New("leave request id cannot be empty")
	}
	switch status {
	case "pending", "approved", "rejected", "cancelled":
		// valid
	default:
		return nil, errors.New("invalid status: must be pending, approved, rejected, or cancelled")
	}
	return s.repo.UpdateLeaveRequestStatus(ctx, id, status)
}

func (s *workforceService) CreateWorkplaceIncident(ctx context.Context, inc *workforce.WorkplaceIncident) (*workforce.WorkplaceIncident, error) {
	if inc == nil || inc.Title == "" || inc.Description == "" {
		return nil, errors.New("title and description are required for incident logging")
	}
	if inc.Category == "" {
		inc.Category = "habitat_disruption"
	}
	return s.repo.CreateWorkplaceIncident(ctx, inc)
}

func (s *workforceService) ListWorkplaceIncidents(ctx context.Context, status string) ([]*workforce.WorkplaceIncident, error) {
	return s.repo.ListWorkplaceIncidents(ctx, status)
}

func (s *workforceService) UpdateWorkplaceIncidentStatus(ctx context.Context, id, status, resolutionNotes string) (*workforce.WorkplaceIncident, error) {
	if id == "" || status == "" {
		return nil, errors.New("incident id and status are required")
	}
	switch status {
	case "open", "under_review", "resolved":
		// valid status
	default:
		return nil, fmt.Errorf("invalid incident status %q: status must be one of 'open', 'under_review', 'resolved'", status)
	}
	return s.repo.UpdateWorkplaceIncidentStatus(ctx, id, status, resolutionNotes)
}
