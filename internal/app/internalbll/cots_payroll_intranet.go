package internalbll

import (
	"context"
	"time"

	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) ListTimesheets(ctx context.Context, in *internalbllv1.ListTimesheetsRequest) (*internalbllv1.ListTimesheetsResponse, error) {
	list, err := s.wfSvc.ListTimesheets(ctx, wfService.TimesheetFilter{
		EmployeeID: in.EmployeeID,
		Status:     in.Status,
		FromDate:   in.FromDate,
		ToDate:     in.ToDate,
		TaskCode:   in.TaskCode,
	})
	if err != nil {
		return nil, err
	}
	return &internalbllv1.ListTimesheetsResponse{Timesheets: list}, nil
}

func (s *Server) SubmitTimesheet(ctx context.Context, in *internalbllv1.SubmitTimesheetRequest) (*internalbllv1.SubmitTimesheetResponse, error) {
	if in.Timesheet == nil {
		return nil, status.Error(codes.InvalidArgument, "timesheet is required")
	}
	created, err := s.wfSvc.SubmitTimesheet(ctx, in.Timesheet)
	if err != nil {
		return nil, err
	}
	return &internalbllv1.SubmitTimesheetResponse{Timesheet: created}, nil
}

func (s *Server) GetTimesheet(ctx context.Context, in *internalbllv1.GetTimesheetRequest) (*internalbllv1.GetTimesheetResponse, error) {
	if in.TimesheetID == "" {
		return nil, status.Error(codes.InvalidArgument, "timesheet_id is required")
	}
	ts, err := s.wfSvc.GetTimesheet(ctx, in.TimesheetID)
	if err != nil {
		return nil, err
	}
	return &internalbllv1.GetTimesheetResponse{Timesheet: ts}, nil
}

func (s *Server) ApproveTimesheet(ctx context.Context, in *internalbllv1.ApproveTimesheetRequest) (*internalbllv1.ApproveTimesheetResponse, error) {
	if in.TimesheetID == "" {
		return nil, status.Error(codes.InvalidArgument, "timesheet_id is required")
	}
	ts, err := s.wfSvc.ApproveTimesheet(ctx, in.TimesheetID, in.ApproverID)
	if err != nil {
		return nil, err
	}
	return &internalbllv1.ApproveTimesheetResponse{Timesheet: ts}, nil
}

func (s *Server) RejectTimesheet(ctx context.Context, in *internalbllv1.RejectTimesheetRequest) (*internalbllv1.RejectTimesheetResponse, error) {
	if in.TimesheetID == "" {
		return nil, status.Error(codes.InvalidArgument, "timesheet_id is required")
	}
	ts, err := s.wfSvc.RejectTimesheet(ctx, in.TimesheetID, in.ReviewerID, in.Reason)
	if err != nil {
		return nil, err
	}
	return &internalbllv1.RejectTimesheetResponse{Timesheet: ts}, nil
}

func (s *Server) ListSalarySlips(ctx context.Context, in *internalbllv1.ListSalarySlipsRequest) (*internalbllv1.ListSalarySlipsResponse, error) {
	slips, err := s.wfSvc.ListSalarySlips(ctx, in.EmployeeID)
	if err != nil {
		return nil, err
	}
	return &internalbllv1.ListSalarySlipsResponse{SalarySlips: slips}, nil
}

func (s *Server) GetSalarySlip(ctx context.Context, in *internalbllv1.GetSalarySlipRequest) (*internalbllv1.GetSalarySlipResponse, error) {
	if in.SlipID == "" {
		return nil, status.Error(codes.InvalidArgument, "slip_id is required")
	}
	slip, err := s.wfSvc.GetSalarySlip(ctx, in.SlipID)
	if err != nil {
		return nil, err
	}
	return &internalbllv1.GetSalarySlipResponse{SalarySlip: slip}, nil
}

func (s *Server) ProcessPayrollRun(ctx context.Context, in *internalbllv1.ProcessPayrollRunRequest) (*internalbllv1.ProcessPayrollRunResponse, error) {
	if in.PayrollEntry == nil {
		return nil, status.Error(codes.InvalidArgument, "payroll_entry is required")
	}

	// If no explicit employee list is provided, dynamically resolve active human employees
	// from workforce-api via gRPC so payroll runs do not fail when workforce service has no direct database repository.
	if len(in.PayrollEntry.Employees) == 0 && s.wfClient != nil {
		if empResp, err := s.wfClient.ListEmployees(ctx, &workforcev1.ListEmployeesRequest{
			EmployeeType: "human",
			Status:       "active",
		}); err == nil && empResp != nil {
			for _, emp := range empResp.Employees {
				if emp != nil && emp.EmployeeID != "" {
					in.PayrollEntry.Employees = append(in.PayrollEntry.Employees, emp.EmployeeID)
				}
			}
		}
	}

	created, err := s.wfSvc.ProcessPayrollRun(ctx, in.PayrollEntry)
	if err != nil {
		return nil, err
	}
	return &internalbllv1.ProcessPayrollRunResponse{PayrollEntry: created}, nil
}

func (s *Server) ListGroupFolders(ctx context.Context, _ *internalbllv1.ListGroupFoldersRequest) (*internalbllv1.ListGroupFoldersResponse, error) {
	folders, err := s.wfSvc.ListGroupFolders(ctx)
	if err != nil {
		return nil, err
	}
	return &internalbllv1.ListGroupFoldersResponse{GroupFolders: folders}, nil
}

func (s *Server) CreateGroupFolder(ctx context.Context, in *internalbllv1.CreateGroupFolderRequest) (*internalbllv1.CreateGroupFolderResponse, error) {
	folder, err := s.wfSvc.CreateGroupFolder(ctx, in.MountPoint, in.Groups, in.Quota)
	if err != nil {
		return nil, err
	}
	return &internalbllv1.CreateGroupFolderResponse{GroupFolder: folder}, nil
}

func (s *Server) ListDocuments(ctx context.Context, in *internalbllv1.ListDocumentsRequest) (*internalbllv1.ListDocumentsResponse, error) {
	docs, err := s.wfSvc.ListDocuments(ctx, in.Department, in.Category, in.Query)
	if err != nil {
		return nil, err
	}
	return &internalbllv1.ListDocumentsResponse{Documents: docs}, nil
}

func (s *Server) GetDocument(ctx context.Context, in *internalbllv1.GetDocumentRequest) (*internalbllv1.GetDocumentResponse, error) {
	if in.DocumentID == "" {
		return nil, status.Error(codes.InvalidArgument, "document_id is required")
	}
	doc, err := s.wfSvc.GetDocument(ctx, in.DocumentID)
	if err != nil {
		return nil, err
	}
	return &internalbllv1.GetDocumentResponse{Document: doc}, nil
}

func (s *Server) PublishDocument(ctx context.Context, in *internalbllv1.PublishDocumentRequest) (*internalbllv1.PublishDocumentResponse, error) {
	doc, err := s.wfSvc.PublishDocument(ctx, in.Document)
	if err != nil {
		return nil, err
	}
	return &internalbllv1.PublishDocumentResponse{Document: doc}, nil
}

func (s *Server) ListCalendarEvents(ctx context.Context, in *internalbllv1.ListCalendarEventsRequest) (*internalbllv1.ListCalendarEventsResponse, error) {
	var fromDate, toDate time.Time
	if in.FromDate != "" {
		fromDate, _ = time.Parse(time.RFC3339, in.FromDate)
	}
	if in.ToDate != "" {
		toDate, _ = time.Parse(time.RFC3339, in.ToDate)
	}
	events, err := s.wfSvc.ListCalendarEvents(ctx, in.Calendar, in.Department, fromDate, toDate)
	if err != nil {
		return nil, err
	}
	return &internalbllv1.ListCalendarEventsResponse{Events: events}, nil
}

func (s *Server) CreateCalendarEvent(ctx context.Context, in *internalbllv1.CreateCalendarEventRequest) (*internalbllv1.CreateCalendarEventResponse, error) {
	ev, err := s.wfSvc.CreateCalendarEvent(ctx, in.Event)
	if err != nil {
		return nil, err
	}
	return &internalbllv1.CreateCalendarEventResponse{Event: ev}, nil
}
