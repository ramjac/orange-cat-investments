package workforce

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetEmployeeByID(t *testing.T) {
	repo := NewRepository(nil)
	ctx := context.Background()

	emp, err := repo.GetEmployeeByID(ctx, "emp-feline-garfield")
	assert.NoError(t, err)
	assert.Equal(t, "Garfield", emp.FirstName)
	assert.Equal(t, "feline", emp.EmployeeType)
}

func TestCareScheduleAndEmergencyMedicalHold(t *testing.T) {
	repo := NewRepository(nil)
	ctx := context.Background()

	cs, err := repo.GetCareScheduleByFelineID(ctx, "emp-feline-garfield")
	assert.NoError(t, err)
	assert.False(t, cs.EmergencyMedicalHold)

	// Set emergency medical hold
	updatedCS, err := repo.SetEmergencyMedicalHold(ctx, "emp-feline-garfield", true)
	assert.NoError(t, err)
	assert.True(t, updatedCS.EmergencyMedicalHold)

	// Fetch again to verify persistence
	cs2, err := repo.GetCareScheduleByFelineID(ctx, "emp-feline-garfield")
	assert.NoError(t, err)
	assert.True(t, cs2.EmergencyMedicalHold)
}

func TestCreateAndListEmployees(t *testing.T) {
	repo := NewRepository(nil)
	ctx := context.Background()

	newEmp := &Employee{
		EmployeeID:   "emp-feline-barneby",
		EmployeeType: "feline",
		FirstName:    "Barneby",
		RoleTitle:    "Senior Alpha Perch Analyst",
		Department:   "Trading Analytics",
		Status:       "active",
	}
	created, err := repo.CreateEmployee(ctx, newEmp)
	assert.NoError(t, err)
	assert.Equal(t, "Barneby", created.FirstName)

	list, err := repo.ListEmployees(ctx, "feline", "active")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 2)
}

func TestUpdateEmployeeStatus(t *testing.T) {
	repo := NewRepository(nil)
	ctx := context.Background()

	t.Run("Success Path", func(t *testing.T) {
		updated, err := repo.UpdateEmployeeStatus(ctx, "emp-feline-garfield", "on_leave")
		assert.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, "on_leave", updated.Status)

		// Verify persistence
		fetched, err := repo.GetEmployeeByID(ctx, "emp-feline-garfield")
		assert.NoError(t, err)
		assert.Equal(t, "on_leave", fetched.Status)
	})

	t.Run("Errors for Non-Existent ID", func(t *testing.T) {
		updated, err := repo.UpdateEmployeeStatus(ctx, "non-existent-employee-id", "active")
		assert.Error(t, err)
		assert.Nil(t, updated)
		assert.EqualError(t, err, "employee with id non-existent-employee-id not found")
	})
}

func TestOnboardingChecklistTasks(t *testing.T) {
	repo := NewRepository(nil)
	ctx := context.Background()

	task := &OnboardingTask{
		EmployeeID:  "emp-feline-barneby",
		TaskName:    "Provision Smart Collar",
		Category:    "hardware",
		IsCompleted: false,
	}
	created, err := repo.CreateOnboardingTask(ctx, task)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.ChecklistID)

	tasks, err := repo.ListOnboardingChecklist(ctx, "emp-feline-barneby")
	assert.NoError(t, err)
	assert.Len(t, tasks, 1)

	updated, err := repo.UpdateOnboardingTask(ctx, created.ChecklistID, true)
	assert.NoError(t, err)
	assert.True(t, updated.IsCompleted)
	assert.NotNil(t, updated.CompletedAt)
}

func TestLeaveRequests(t *testing.T) {
	repo := NewRepository(nil)
	ctx := context.Background()

	t.Run("List and Filter", func(t *testing.T) {
		list, err := repo.ListLeaveRequests(ctx, "", "")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 2)

		garfieldList, err := repo.ListLeaveRequests(ctx, "emp-feline-garfield", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, garfieldList)
		assert.Equal(t, "emp-feline-garfield", garfieldList[0].EmployeeID)
		assert.Equal(t, "catnip_break", garfieldList[0].LeaveType)

		pendingList, err := repo.ListLeaveRequests(ctx, "", "pending")
		assert.NoError(t, err)
		assert.NotEmpty(t, pendingList)
		for _, req := range pendingList {
			assert.Equal(t, "pending", req.Status)
		}
	})

	t.Run("Create Get and Update Status", func(t *testing.T) {
		reason := "Extended weekend sunbeam rest"
		newReq := &LeaveRequest{
			EmployeeID: "emp-feline-garfield",
			LeaveType:  "catnip_break",
			StartDate:  "2026-11-01",
			EndDate:    "2026-11-04",
			Reason:     &reason,
		}
		created, err := repo.CreateLeaveRequest(ctx, newReq)
		assert.NoError(t, err)
		assert.NotEmpty(t, created.LeaveID)
		assert.Equal(t, "pending", created.Status)
		assert.Equal(t, "2026-11-01", created.StartDate)

		fetched, err := repo.GetLeaveRequestByID(ctx, created.LeaveID)
		assert.NoError(t, err)
		assert.Equal(t, created.LeaveID, fetched.LeaveID)
		assert.Equal(t, "Extended weekend sunbeam rest", *fetched.Reason)

		updated, err := repo.UpdateLeaveRequestStatus(ctx, created.LeaveID, "approved")
		assert.NoError(t, err)
		assert.Equal(t, "approved", updated.Status)
	})

	t.Run("Errors for Non-Existent ID", func(t *testing.T) {
		_, err := repo.GetLeaveRequestByID(ctx, "non-existent-id")
		assert.Error(t, err)
		_, err = repo.UpdateLeaveRequestStatus(ctx, "non-existent-id", "rejected")
		assert.Error(t, err)
	})
}

func TestReviewCycles(t *testing.T) {
	repo := NewRepository(nil)
	ctx := context.Background()

	t.Run("List and Filter", func(t *testing.T) {
		list, err := repo.ListReviewCycles(ctx, "", "", "")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 3)

		garfieldList, err := repo.ListReviewCycles(ctx, "emp-feline-garfield", "", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, garfieldList)
		assert.Equal(t, "emp-feline-garfield", garfieldList[0].EmployeeID)
		assert.Equal(t, "feline_health_assessment", garfieldList[0].ReviewType)

		completedList, err := repo.ListReviewCycles(ctx, "", "completed", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, completedList)
		for _, rc := range completedList {
			assert.Equal(t, "completed", rc.Status)
		}

		perfList, err := repo.ListReviewCycles(ctx, "", "", "performance")
		assert.NoError(t, err)
		assert.NotEmpty(t, perfList)
		for _, rc := range perfList {
			assert.Equal(t, "performance", rc.ReviewType)
		}
	})

	t.Run("Create Get and Update", func(t *testing.T) {
		notes := "Quarterly agility and sunbeam orientation evaluation"
		reviewer := "emp-human-elena"
		newRC := &ReviewCycle{
			EmployeeID:   "emp-feline-barneby",
			ReviewType:   "feline_health_assessment",
			ScheduledFor: "2026-11-15",
			Status:       "scheduled",
			ReviewerID:   &reviewer,
			Notes:        &notes,
		}
		created, err := repo.CreateReviewCycle(ctx, newRC)
		assert.NoError(t, err)
		assert.NotEmpty(t, created.ReviewID)
		assert.Equal(t, "scheduled", created.Status)
		assert.Equal(t, "2026-11-15", created.ScheduledFor)

		fetched, err := repo.GetReviewCycleByID(ctx, created.ReviewID)
		assert.NoError(t, err)
		assert.Equal(t, created.ReviewID, fetched.ReviewID)
		assert.Equal(t, notes, *fetched.Notes)

		score := 4.88
		compNotes := "Outstanding perch balance and high alpha drive"
		now := time.Now().UTC()
		updateRC := &ReviewCycle{
			ReviewID:    created.ReviewID,
			Status:      "completed",
			Score:       &score,
			Notes:       &compNotes,
			CompletedAt: &now,
		}
		updated, err := repo.UpdateReviewCycle(ctx, updateRC)
		assert.NoError(t, err)
		assert.Equal(t, "completed", updated.Status)
		assert.Equal(t, score, *updated.Score)
		assert.Equal(t, compNotes, *updated.Notes)
		assert.NotNil(t, updated.CompletedAt)
	})

	t.Run("Errors for Non-Existent ID", func(t *testing.T) {
		_, err := repo.GetReviewCycleByID(ctx, "non-existent-id")
		assert.Error(t, err)
		_, err = repo.UpdateReviewCycle(ctx, &ReviewCycle{ReviewID: "non-existent-id"})
		assert.Error(t, err)
	})
}

func TestVHRRecords(t *testing.T) {
	repo := NewRepository(nil)
	ctx := context.Background()

	vetID := "emp-human-elena"
	prescriptions := "Amoxicillin 50mg"

	rec1 := &VHRRecord{
		FelineID:          "emp-feline-garfield",
		VeterinarianID:    &vetID,
		VisitDate:         "2026-10-15",
		WeightKG:          5.2,
		DentalScore:       8,
		VaccinationStatus: "up_to_date",
		ClinicalNotes:     "Routine health checkup and weight monitoring",
	}

	rec2 := &VHRRecord{
		FelineID:          "emp-feline-garfield",
		VeterinarianID:    &vetID,
		VisitDate:         "2026-11-01",
		WeightKG:          5.4,
		DentalScore:       9,
		VaccinationStatus: "up_to_date",
		Prescriptions:     &prescriptions,
		ClinicalNotes:     "Follow-up wellness checkup and dental cleaning",
	}

	rec3 := &VHRRecord{
		FelineID:          "emp-feline-barneby",
		VeterinarianID:    &vetID,
		VisitDate:         "2026-10-20",
		WeightKG:          4.8,
		DentalScore:       9,
		VaccinationStatus: "up_to_date",
		ClinicalNotes:     "Alpha perch readiness & mobility assessment",
	}

	created1, err := repo.CreateVHRRecord(ctx, rec1)
	require.NoError(t, err)
	assert.NotEmpty(t, created1.RecordID)
	assert.Equal(t, "emp-feline-garfield", created1.FelineID)

	created2, err := repo.CreateVHRRecord(ctx, rec2)
	require.NoError(t, err)
	assert.NotEmpty(t, created2.RecordID)
	assert.Equal(t, "emp-feline-garfield", created2.FelineID)

	created3, err := repo.CreateVHRRecord(ctx, rec3)
	require.NoError(t, err)
	assert.NotEmpty(t, created3.RecordID)
	assert.Equal(t, "emp-feline-barneby", created3.FelineID)

	t.Run("Filter by feline ID and check ordering", func(t *testing.T) {
		records, err := repo.ListVHRRecords(ctx, "emp-feline-garfield")
		require.NoError(t, err)
		require.Len(t, records, 2)

		// Must be ordered by VisitDate DESC ("2026-11-01" before "2026-10-15")
		assert.Equal(t, created2.RecordID, records[0].RecordID)
		assert.Equal(t, "2026-11-01", records[0].VisitDate)
		assert.Equal(t, 5.4, records[0].WeightKG)
		assert.Equal(t, &prescriptions, records[0].Prescriptions)

		assert.Equal(t, created1.RecordID, records[1].RecordID)
		assert.Equal(t, "2026-10-15", records[1].VisitDate)
		assert.Equal(t, 5.2, records[1].WeightKG)
	})

	t.Run("List all VHR records when felineID is empty", func(t *testing.T) {
		records, err := repo.ListVHRRecords(ctx, "")
		require.NoError(t, err)
		require.Len(t, records, 3)

		// Must be ordered by VisitDate DESC ("2026-11-01", "2026-10-20", "2026-10-15")
		assert.Equal(t, "2026-11-01", records[0].VisitDate)
		assert.Equal(t, "2026-10-20", records[1].VisitDate)
		assert.Equal(t, "2026-10-15", records[2].VisitDate)
	})

	t.Run("Return empty list for non-existent feline ID", func(t *testing.T) {
		records, err := repo.ListVHRRecords(ctx, "non-existent-feline-id")
		require.NoError(t, err)
		assert.NotNil(t, records)
		assert.Empty(t, records)
	})
}

func TestWorkplaceIncidents(t *testing.T) {
	repo := NewRepository(nil)
	ctx := context.Background()

	felineID := "emp-feline-garfield"
	humanID := "emp-human-alice"

	// Create Workplace Incident
	newInc := &WorkplaceIncident{
		Title:            "Laser Pointer Interference",
		Category:         "safety",
		InvolvedFelineID: &felineID,
		InvolvedHumanID:  &humanID,
		Severity:         "medium",
		Description:      "Uncalibrated laser beam temporarily disrupted feline observation focus",
	}
	created, err := repo.CreateWorkplaceIncident(ctx, newInc)
	assert.NoError(t, err)
	assert.NotEmpty(t, created.IncidentID)
	assert.Equal(t, "open", created.Status)
	assert.Equal(t, "medium", created.Severity)
	assert.False(t, created.ReportedAt.IsZero())

	// List Workplace Incidents
	incidents, err := repo.ListWorkplaceIncidents(ctx, "")
	assert.NoError(t, err)
	assert.NotEmpty(t, incidents)

	openIncidents, err := repo.ListWorkplaceIncidents(ctx, "open")
	assert.NoError(t, err)
	assert.NotEmpty(t, openIncidents)
	for _, inc := range openIncidents {
		assert.Equal(t, "open", inc.Status)
	}

	// Update Workplace Incident Status to under_review
	reviewNotes := "Safety committee investigating optical dispersion pattern"
	underReview, err := repo.UpdateWorkplaceIncidentStatus(ctx, created.IncidentID, "under_review", reviewNotes)
	assert.NoError(t, err)
	assert.Equal(t, "under_review", underReview.Status)
	assert.NotNil(t, underReview.ResolutionNotes)
	assert.Equal(t, reviewNotes, *underReview.ResolutionNotes)
	assert.Nil(t, underReview.ResolvedAt)

	// Update Workplace Incident Status to resolved
	resolveNotes := "Laser pointer recalibrated and safety barrier installed"
	resolved, err := repo.UpdateWorkplaceIncidentStatus(ctx, created.IncidentID, "resolved", resolveNotes)
	assert.NoError(t, err)
	assert.Equal(t, "resolved", resolved.Status)
	assert.NotNil(t, resolved.ResolutionNotes)
	assert.Equal(t, resolveNotes, *resolved.ResolutionNotes)
	assert.NotNil(t, resolved.ResolvedAt)

	// Error handling: invalid status
	_, err = repo.UpdateWorkplaceIncidentStatus(ctx, created.IncidentID, "invalid_status", "Notes")
	assert.Error(t, err)

	// Error handling: non-existent ID
	_, err = repo.UpdateWorkplaceIncidentStatus(ctx, "non-existent-inc-id", "resolved", "Notes")
	assert.Error(t, err)
}
