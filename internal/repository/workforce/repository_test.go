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

	t.Run("Create with Auto-Generated ID and Optional Fields", func(t *testing.T) {
		vetID := "emp-human-elena"
		prescriptions := "Amoxicillin 50mg daily"
		rec := &VHRRecord{
			FelineID:          "emp-feline-garfield",
			VeterinarianID:    &vetID,
			VisitDate:         "2026-11-01",
			WeightKG:          5.4,
			DentalScore:       9,
			VaccinationStatus: "up_to_date",
			Prescriptions:     &prescriptions,
			ClinicalNotes:     "Excellent wellness checkup and routine teeth cleaning",
		}

		created, err := repo.CreateVHRRecord(ctx, rec)
		require.NoError(t, err)
		assert.NotEmpty(t, created.RecordID)
		assert.Equal(t, "emp-feline-garfield", created.FelineID)
		assert.NotNil(t, created.VeterinarianID)
		assert.Equal(t, vetID, *created.VeterinarianID)
		assert.Equal(t, "2026-11-01", created.VisitDate)
		assert.Equal(t, 5.4, created.WeightKG)
		assert.Equal(t, 9, created.DentalScore)
		assert.Equal(t, "up_to_date", created.VaccinationStatus)
		assert.NotNil(t, created.Prescriptions)
		assert.Equal(t, prescriptions, *created.Prescriptions)
		assert.Equal(t, "Excellent wellness checkup and routine teeth cleaning", created.ClinicalNotes)
		assert.False(t, created.CreatedAt.IsZero())
		assert.False(t, created.UpdatedAt.IsZero())
	})

	t.Run("Create with Explicit Custom ID and Nil Optional Fields", func(t *testing.T) {
		customID := "vhr-custom-999"
		rec := &VHRRecord{
			RecordID:          customID,
			FelineID:          "emp-feline-barneby",
			VisitDate:         "2026-11-02",
			WeightKG:          4.8,
			DentalScore:       8,
			VaccinationStatus: "pending_booster",
			ClinicalNotes:     "Routine health check, booster needed",
		}

		created, err := repo.CreateVHRRecord(ctx, rec)
		require.NoError(t, err)
		assert.Equal(t, customID, created.RecordID)
		assert.Equal(t, "emp-feline-barneby", created.FelineID)
		assert.Nil(t, created.VeterinarianID)
		assert.Nil(t, created.Prescriptions)
		assert.False(t, created.CreatedAt.IsZero())
		assert.False(t, created.UpdatedAt.IsZero())
	})

	t.Run("List and Filter VHR Records", func(t *testing.T) {
		r := NewRepository(nil)
		vetID := "emp-human-elena"
		prescriptions := "Amoxicillin 50mg"

		// Create record 1 for Garfield (2026-11-01)
		rec1 := &VHRRecord{
			FelineID:          "emp-feline-garfield",
			VeterinarianID:    &vetID,
			VisitDate:         "2026-11-01",
			WeightKG:          5.4,
			DentalScore:       9,
			VaccinationStatus: "up_to_date",
			ClinicalNotes:     "Garfield Visit 1",
		}
		c1, err := r.CreateVHRRecord(ctx, rec1)
		require.NoError(t, err)

		// Create record 2 for Garfield (2026-11-10) - later date with prescription
		rec2 := &VHRRecord{
			FelineID:          "emp-feline-garfield",
			VeterinarianID:    &vetID,
			VisitDate:         "2026-11-10",
			WeightKG:          5.5,
			DentalScore:       9,
			VaccinationStatus: "up_to_date",
			Prescriptions:     &prescriptions,
			ClinicalNotes:     "Garfield Visit 2",
		}
		c2, err := r.CreateVHRRecord(ctx, rec2)
		require.NoError(t, err)

		// Create record 3 for Barneby (2026-11-05)
		rec3 := &VHRRecord{
			FelineID:          "emp-feline-barneby",
			VeterinarianID:    &vetID,
			VisitDate:         "2026-11-05",
			WeightKG:          4.9,
			DentalScore:       8,
			VaccinationStatus: "up_to_date",
			ClinicalNotes:     "Barneby Visit 1",
		}
		c3, err := r.CreateVHRRecord(ctx, rec3)
		require.NoError(t, err)

		// Filter for Garfield
		garfieldRecords, err := r.ListVHRRecords(ctx, "emp-feline-garfield")
		require.NoError(t, err)
		assert.Len(t, garfieldRecords, 2)
		// Should be sorted by VisitDate DESC (c2: 2026-11-10 before c1: 2026-11-01)
		assert.Equal(t, c2.RecordID, garfieldRecords[0].RecordID)
		assert.Equal(t, "2026-11-10", garfieldRecords[0].VisitDate)
		assert.Equal(t, 5.5, garfieldRecords[0].WeightKG)
		assert.Equal(t, &prescriptions, garfieldRecords[0].Prescriptions)

		assert.Equal(t, c1.RecordID, garfieldRecords[1].RecordID)
		assert.Equal(t, "2026-11-01", garfieldRecords[1].VisitDate)
		assert.Equal(t, 5.4, garfieldRecords[1].WeightKG)

		// Filter for Barneby
		barnebyRecords, err := r.ListVHRRecords(ctx, "emp-feline-barneby")
		require.NoError(t, err)
		assert.Len(t, barnebyRecords, 1)
		assert.Equal(t, c3.RecordID, barnebyRecords[0].RecordID)
		assert.Equal(t, 4.9, barnebyRecords[0].WeightKG)

		// Empty feline ID should list all records sorted by VisitDate DESC
		allRecords, err := r.ListVHRRecords(ctx, "")
		require.NoError(t, err)
		assert.Len(t, allRecords, 3)
		assert.Equal(t, "2026-11-10", allRecords[0].VisitDate)
		assert.Equal(t, "2026-11-05", allRecords[1].VisitDate)
		assert.Equal(t, "2026-11-01", allRecords[2].VisitDate)

		// Query for a feline with no records
		noRecords, err := r.ListVHRRecords(ctx, "emp-feline-nonexistent")
		require.NoError(t, err)
		assert.NotNil(t, noRecords)
		assert.Empty(t, noRecords)
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

func TestWorkforcePgxRepository_NilDB(t *testing.T) {
	repo := &pgxRepository{db: nil}
	ctx := context.Background()

	_, err := repo.GetEmployeeByID(ctx, "any")
	assert.Error(t, err)

	_, err = repo.ListEmployees(ctx, "", "")
	assert.Error(t, err)

	_, err = repo.CreateEmployee(ctx, &Employee{})
	assert.Error(t, err)

	_, err = repo.UpdateEmployeeStatus(ctx, "any", "active")
	assert.Error(t, err)

	_, err = repo.CreateOnboardingTask(ctx, &OnboardingTask{})
	assert.Error(t, err)

	_, err = repo.ListOnboardingChecklist(ctx, "any")
	assert.Error(t, err)

	_, err = repo.UpdateOnboardingTask(ctx, "any", true)
	assert.Error(t, err)

	_, err = repo.GetCareScheduleByFelineID(ctx, "any")
	assert.Error(t, err)

	_, err = repo.UpsertCareSchedule(ctx, &CareSchedule{})
	assert.Error(t, err)

	_, err = repo.SetEmergencyMedicalHold(ctx, "any", true)
	assert.Error(t, err)

	_, err = repo.CreateLeaveRequest(ctx, &LeaveRequest{})
	assert.Error(t, err)

	_, err = repo.GetLeaveRequestByID(ctx, "any")
	assert.Error(t, err)

	_, err = repo.ListLeaveRequests(ctx, "", "")
	assert.Error(t, err)

	_, err = repo.UpdateLeaveRequestStatus(ctx, "any", "approved")
	assert.Error(t, err)

	_, err = repo.CreateReviewCycle(ctx, &ReviewCycle{})
	assert.Error(t, err)

	_, err = repo.GetReviewCycleByID(ctx, "any")
	assert.Error(t, err)

	_, err = repo.ListReviewCycles(ctx, "", "", "")
	assert.Error(t, err)

	_, err = repo.UpdateReviewCycle(ctx, &ReviewCycle{})
	assert.Error(t, err)

	_, err = repo.CreateVHRRecord(ctx, &VHRRecord{})
	assert.Error(t, err)

	_, err = repo.ListVHRRecords(ctx, "any")
	assert.Error(t, err)

	_, err = repo.CreateWorkplaceIncident(ctx, &WorkplaceIncident{})
	assert.Error(t, err)

	_, err = repo.ListWorkplaceIncidents(ctx, "")
	assert.Error(t, err)

	_, err = repo.UpdateWorkplaceIncidentStatus(ctx, "any", "resolved", "notes")
	assert.Error(t, err)
}

