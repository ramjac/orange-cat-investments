package workforce

import (
	"time"
)

func (r *mockRepository) seedDefaults() {
	now := time.Now().UTC()
	r.seedEmployees(now)
	r.seedCareSchedules(now)
	r.seedChecklists(now)
	r.seedLeaveRequests(now)
	r.seedReviewCycles(now)
	r.seedExecutiveReports(now)
}

func (r *mockRepository) seedEmployees(now time.Time) {
	r.seedFelineEmployees(now)
	r.seedHumanEmployees(now)
}

func (r *mockRepository) seedFelineEmployees(now time.Time) {
	garfieldID := "emp-feline-garfield"
	r.employees[garfieldID] = &Employee{
		EmployeeID:   garfieldID,
		EmployeeType: "feline",
		FirstName:    "Garfield",
		RoleTitle:    "Chief Observation Officer",
		Department:   "Executive Feline Suite",
		Status:       "active",
		HiredAt:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	barnebyID := "emp-feline-barneby"
	r.employees[barnebyID] = &Employee{
		EmployeeID:   barnebyID,
		EmployeeType: "feline",
		FirstName:    "Barneby",
		RoleTitle:    "Senior Alpha Perch Analyst",
		Department:   "Alpha Perch Research",
		Status:       "active",
		HiredAt:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func (r *mockRepository) seedHumanEmployees(now time.Time) {
	aliceID := "emp-human-alice"
	r.employees[aliceID] = &Employee{
		EmployeeID:   aliceID,
		EmployeeType: "human",
		FirstName:    "Alice",
		LastName:     strPtr("Vance"),
		Email:        strPtr("alice.vance@oci.local"),
		RoleTitle:    "Head of Human & Feline Resources",
		Department:   "Workforce Operations",
		Status:       "active",
		HiredAt:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	elenaID := "emp-human-elena"
	r.employees[elenaID] = &Employee{
		EmployeeID:   elenaID,
		EmployeeType: "human",
		FirstName:    "Elena",
		LastName:     strPtr("Rostova"),
		Email:        strPtr("elena.rostova@oci.local"),
		RoleTitle:    "Chief Veterinary Officer",
		Department:   "Feline Health & Welfare",
		Status:       "active",
		HiredAt:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	rickID := "emp-human-rick"
	r.employees[rickID] = &Employee{
		EmployeeID:   rickID,
		EmployeeType: "human",
		FirstName:    "Rick",
		LastName:     strPtr("Newhire"),
		Email:        strPtr("rick.newhire@oci.local"),
		RoleTitle:    "Junior Operations Associate",
		Department:   "Workforce Operations",
		Status:       "onboarding",
		HiredAt:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	eliseID := "emp-human-elise"
	r.employees[eliseID] = &Employee{
		EmployeeID:   eliseID,
		EmployeeType: "human",
		FirstName:    "Elise",
		LastName:     strPtr("Dev"),
		Email:        strPtr("elise.dev@oci.local"),
		RoleTitle:    "Software Developer",
		Department:   "Engineering & Platform Development",
		Status:       "active",
		HiredAt:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	frankID := "emp-human-frank"
	r.employees[frankID] = &Employee{
		EmployeeID:   frankID,
		EmployeeType: "human",
		FirstName:    "Frank",
		LastName:     strPtr("Operations"),
		Email:        strPtr("frank.ops@oci.local"),
		RoleTitle:    "Platform Security & K8s Infrastructure Lead",
		Department:   "Platform Security & Infrastructure",
		Status:       "active",
		HiredAt:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	bobID := "emp-human-bob"
	r.employees[bobID] = &Employee{
		EmployeeID:   bobID,
		EmployeeType: "human",
		FirstName:    "Bob",
		LastName:     strPtr("Builder"),
		Email:        strPtr("bob.builder@oci.local"),
		RoleTitle:    "Lead Facilities & Edge Telemetry Engineer",
		Department:   "Habitat Facilities & Infrastructure",
		Status:       "active",
		HiredAt:      now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func (r *mockRepository) seedCareSchedules(now time.Time) {
	garfieldID := "emp-feline-garfield"
	r.careSchedules[garfieldID] = &CareSchedule{
		ScheduleID:           "cs-garfield-01",
		FelineID:             garfieldID,
		DietaryPlan:          "High-protein salmon pate & prescription kibble",
		FeedingTimes:         []string{"08:00 AM", "12:00 PM", "06:00 PM"},
		EmergencyMedicalHold: false,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	barnebyID := "emp-feline-barneby"
	r.careSchedules[barnebyID] = &CareSchedule{
		ScheduleID:           "cs-barneby-01",
		FelineID:             barnebyID,
		DietaryPlan:          "Grain-free organic turkey pate",
		FeedingTimes:         []string{"07:30 AM", "05:30 PM"},
		EmergencyMedicalHold: false,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
}

func (r *mockRepository) seedChecklists(now time.Time) {
	aliceID := "emp-human-alice"
	rickID := "emp-human-rick"

	r.checklists["chk-rick-01"] = &OnboardingTask{
		ChecklistID: "chk-rick-01",
		EmployeeID:  rickID,
		TaskName:    "Forgejo Account Provisioning & SPIFFE ID Issuance",
		Category:    "access",
		IsCompleted: false,
		AssignedTo:  &aliceID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	r.checklists["chk-rick-02"] = &OnboardingTask{
		ChecklistID: "chk-rick-02",
		EmployeeID:  rickID,
		TaskName:    "Catnip Safety Orientation & Workstation Setup",
		Category:    "training",
		IsCompleted: false,
		AssignedTo:  &aliceID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (r *mockRepository) seedLeaveRequests(now time.Time) {
	garfieldID := "emp-feline-garfield"
	barnebyID := "emp-feline-barneby"
	elenaID := "emp-human-elena"

	garfieldLeaveID := "018e0000-0000-7000-8000-000000000010"
	r.leaveRequests[garfieldLeaveID] = &LeaveRequest{
		LeaveID:    garfieldLeaveID,
		EmployeeID: garfieldID,
		LeaveType:  "catnip_break",
		StartDate:  "2026-10-01",
		EndDate:    "2026-10-03",
		Status:     "approved",
		Reason:     strPtr("Mandatory post-alpha observation rest & organic catnip relaxation"),
		CreatedAt:  now.Add(-24 * time.Hour),
		UpdatedAt:  now.Add(-24 * time.Hour),
	}

	barnebyLeaveID := "018e0000-0000-7000-8000-000000000011"
	r.leaveRequests[barnebyLeaveID] = &LeaveRequest{
		LeaveID:    barnebyLeaveID,
		EmployeeID: barnebyID,
		LeaveType:  "catnip_break",
		StartDate:  "2026-10-05",
		EndDate:    "2026-10-06",
		Status:     "pending",
		Reason:     strPtr("Alpha perch rotation decompression and premium catnip session"),
		CreatedAt:  now.Add(-2 * time.Hour),
		UpdatedAt:  now.Add(-2 * time.Hour),
	}

	elenaLeaveID := "018e0000-0000-7000-8000-000000000012"
	r.leaveRequests[elenaLeaveID] = &LeaveRequest{
		LeaveID:    elenaLeaveID,
		EmployeeID: elenaID,
		LeaveType:  "vacation",
		StartDate:  "2026-10-15",
		EndDate:    "2026-10-20",
		Status:     "approved",
		Reason:     strPtr("Annual veterinary feline habitat conference"),
		CreatedAt:  now.Add(-48 * time.Hour),
		UpdatedAt:  now.Add(-48 * time.Hour),
	}
}

func floatPtr(f float64) *float64 {
	return &f
}
