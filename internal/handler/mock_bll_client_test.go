package handler

import (
	"sync"

	"github.com/orange-cat-investments/oci/pkg/cots/frappe"
	"github.com/orange-cat-investments/oci/pkg/cots/nextcloud"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
)

type mockInternalBLLClient struct {
	internalbllv1.InternalBLLServiceClient
	mu            sync.Mutex
	employees     map[string]*workforcev1.Employee
	careSchedules map[string]*workforcev1.CareSchedule
	leaveRequests map[string]*workforcev1.LeaveRequest
	reviewCycles  map[string]*workforcev1.ReviewCycle
	timesheets    map[string]*frappe.Timesheet
	salarySlips   map[string]*frappe.SalarySlip
	documents     map[string]*nextcloud.Document
	folders       []nextcloud.GroupFolder
	events        []nextcloud.CalendarEvent
	reports       map[string]*workforcev1.ExecutiveReport
}

func newMockInternalBLLClient() *mockInternalBLLClient {
	m := &mockInternalBLLClient{
		employees:     make(map[string]*workforcev1.Employee),
		careSchedules: make(map[string]*workforcev1.CareSchedule),
		leaveRequests: make(map[string]*workforcev1.LeaveRequest),
		reviewCycles:  make(map[string]*workforcev1.ReviewCycle),
		timesheets:    make(map[string]*frappe.Timesheet),
		salarySlips:   make(map[string]*frappe.SalarySlip),
		documents:     make(map[string]*nextcloud.Document),
		reports:       make(map[string]*workforcev1.ExecutiveReport),
	}

	// Seed Garfield care schedule
	m.careSchedules["emp-feline-garfield"] = &workforcev1.CareSchedule{
		FelineID:             "emp-feline-garfield",
		DietaryPlan:          "Lasagna & Tuna Supreme",
		FeedingTimes:         []string{"08:00 AM", "12:00 PM", "06:00 PM"},
		EmergencyMedicalHold: false,
	}

	// Seed employees
	m.employees["emp-feline-garfield"] = &workforcev1.Employee{
		EmployeeID:   "emp-feline-garfield",
		FirstName:    "Garfield",
		EmployeeType: "feline",
		RoleTitle:    "Chief Observation Officer",
		Department:   "Executive Feline Suite",
		Status:       "active",
	}

	// Seed leave requests
	m.leaveRequests["leave-1"] = &workforcev1.LeaveRequest{
		LeaveID:    "018e0000-0000-7000-8000-000000000010",
		EmployeeID: "emp-feline-garfield",
		LeaveType:  "catnip_break",
		StartDate:  "2026-10-01",
		EndDate:    "2026-10-03",
		Status:     "approved",
		Reason:     "Mandatory post-alpha observation rest & organic catnip relaxation",
	}
	m.leaveRequests["leave-2"] = &workforcev1.LeaveRequest{
		LeaveID:    "018e0000-0000-7000-8000-000000000011",
		EmployeeID: "emp-feline-barneby",
		LeaveType:  "catnip_break",
		StartDate:  "2026-10-05",
		EndDate:    "2026-10-06",
		Status:     "pending",
		Reason:     "Alpha perch rotation decompression and premium catnip session",
	}

	// Seed review cycles
	elenaID := "emp-human-elena"
	m.reviewCycles["rev-1"] = &workforcev1.ReviewCycle{
		ReviewID:           "018e0000-0000-7000-8000-000000000020",
		EmployeeID:         "emp-feline-garfield",
		ReviewType:         "feline_health_assessment",
		ReviewPeriod:       "2026-Q3",
		Phase:              "completed",
		ScheduledFor:       "2026-09-25",
		Status:             "completed",
		ReviewerID:         elenaID,
		Score:              4.95,
		Notes:              "Superb whisker symmetry, resting purr acoustics 92dB",
		SelfAppraisal:      "Maintained executive posture on alpha perch",
		SelfScore:          5.00,
		WellnessAuditScore: 4.95,
	}
	m.reviewCycles["rev-2"] = &workforcev1.ReviewCycle{
		ReviewID:     "018e0000-0000-7000-8000-000000000021",
		EmployeeID:   "emp-feline-barneby",
		ReviewType:   "feline_health_assessment",
		ReviewPeriod: "2026-Q3",
		Phase:        "manager_review",
		ScheduledFor: "2026-10-01",
		Status:       "in_progress",
	}
	m.reviewCycles["rev-3"] = &workforcev1.ReviewCycle{
		ReviewID:     "018e0000-0000-7000-8000-000000000022",
		EmployeeID:   "emp-human-alice",
		ReviewType:   "performance",
		ReviewPeriod: "2026-Q3",
		Phase:        "self_appraisal",
		ScheduledFor: "2026-10-15",
		Status:       "scheduled",
	}

	// Seed default salary slip
	m.salarySlips["slip-bob-1"] = &frappe.SalarySlip{
		SalarySlipID: "slip-bob-1",
		EmployeeID:   "emp-human-bob",
		PostingDate:  "2026-10-31",
		GrossPay:     5000.00,
		NetPay:       4200.00,
		Status:       "submitted",
	}

	// Seed 4 hub folders
	m.folders = []nextcloud.GroupFolder{
		{ID: 1, MountPoint: "Executive Board Documents", Quota: "100 GB", Groups: []string{"exec", "admin"}},
		{ID: 2, MountPoint: "Facilities & Habitats", Quota: "50 GB", Groups: []string{"facilities"}},
		{ID: 3, MountPoint: "Veterinary Medicine", Quota: "50 GB", Groups: []string{"veterinary"}},
		{ID: 4, MountPoint: "Quantitative Trading Strategy", Quota: "250 GB", Groups: []string{"investment"}},
	}

	// Seed doc-001
	m.documents["doc-001"] = &nextcloud.Document{
		DocumentID: "doc-001",
		Title:      "Employee Onboarding Guide",
		Department: "Facilities & Habitat",
		Category:   "SOP",
		Author:     "emp-human-bob",
		WebDAVPath: "/remote.php/dav/files/admin/cleanroom_sop.md",
	}

	// Seed calendar event
	m.events = append(m.events, nextcloud.CalendarEvent{
		EventID:    "ev-1",
		Calendar:   "Shift Rotations",
		Title:      "Morning Perch Inspection",
		Department: "Facilities Operations",
	})

	// Seed 2 monthly reports
	m.reports["rep-2026-08"] = &workforcev1.ExecutiveReport{
		ReportID:   "rep-2026-08",
		Period:     "2026-08",
		Status:     "archived",
		CSVContent: "Total Revenue,1500000\nMRR,120000",
		PDFContent: []byte("%PDF-1.4 sample report"),
	}
	m.reports["rep-2026-09"] = &workforcev1.ExecutiveReport{
		ReportID:   "rep-2026-09",
		Period:     "2026-09",
		Status:     "archived",
		CSVContent: "Total Revenue,1600000\nMRR,125000",
		PDFContent: []byte("%PDF-1.4 sample report"),
	}

	return m
}
