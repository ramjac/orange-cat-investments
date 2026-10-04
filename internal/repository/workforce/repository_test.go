package workforce

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorkforceRepository(t *testing.T) {
	repo := NewRepository(nil)
	ctx := context.Background()

	t.Run("Get Seeded Garfield", func(t *testing.T) {
		emp, err := repo.GetEmployeeByID(ctx, "emp-feline-garfield")
		assert.NoError(t, err)
		assert.Equal(t, "Garfield", emp.FirstName)
		assert.Equal(t, "feline", emp.EmployeeType)
	})

	t.Run("Care Schedule and Emergency Medical Hold", func(t *testing.T) {
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
	})

	t.Run("Create and List Employees", func(t *testing.T) {
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
	})

	t.Run("Onboarding Checklist Tasks", func(t *testing.T) {
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
	})

	t.Run("Leave Requests Operations", func(t *testing.T) {
		// List seeded leave requests
		list, err := repo.ListLeaveRequests(ctx, "", "")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 2)

		// Filter by employee
		garfieldList, err := repo.ListLeaveRequests(ctx, "emp-feline-garfield", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, garfieldList)
		assert.Equal(t, "emp-feline-garfield", garfieldList[0].EmployeeID)
		assert.Equal(t, "catnip_break", garfieldList[0].LeaveType)

		// Filter by status
		pendingList, err := repo.ListLeaveRequests(ctx, "", "pending")
		assert.NoError(t, err)
		assert.NotEmpty(t, pendingList)
		for _, req := range pendingList {
			assert.Equal(t, "pending", req.Status)
		}

		// Create new leave request
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

		// Get by ID
		fetched, err := repo.GetLeaveRequestByID(ctx, created.LeaveID)
		assert.NoError(t, err)
		assert.Equal(t, created.LeaveID, fetched.LeaveID)
		assert.Equal(t, "Extended weekend sunbeam rest", *fetched.Reason)

		// Update status to approved
		updated, err := repo.UpdateLeaveRequestStatus(ctx, created.LeaveID, "approved")
		assert.NoError(t, err)
		assert.Equal(t, "approved", updated.Status)

		// Non-existent ID error handling
		_, err = repo.GetLeaveRequestByID(ctx, "non-existent-id")
		assert.Error(t, err)
		_, err = repo.UpdateLeaveRequestStatus(ctx, "non-existent-id", "rejected")
		assert.Error(t, err)
	})

	t.Run("Review Cycles Operations", func(t *testing.T) {
		// List seeded review cycles
		list, err := repo.ListReviewCycles(ctx, "", "", "")
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(list), 3)

		// Filter by employee
		garfieldList, err := repo.ListReviewCycles(ctx, "emp-feline-garfield", "", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, garfieldList)
		assert.Equal(t, "emp-feline-garfield", garfieldList[0].EmployeeID)
		assert.Equal(t, "feline_health_assessment", garfieldList[0].ReviewType)

		// Filter by status
		completedList, err := repo.ListReviewCycles(ctx, "", "completed", "")
		assert.NoError(t, err)
		assert.NotEmpty(t, completedList)
		for _, rc := range completedList {
			assert.Equal(t, "completed", rc.Status)
		}

		// Filter by review type
		perfList, err := repo.ListReviewCycles(ctx, "", "", "performance")
		assert.NoError(t, err)
		assert.NotEmpty(t, perfList)
		for _, rc := range perfList {
			assert.Equal(t, "performance", rc.ReviewType)
		}

		// Create review cycle
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

		// Get by ID
		fetched, err := repo.GetReviewCycleByID(ctx, created.ReviewID)
		assert.NoError(t, err)
		assert.Equal(t, created.ReviewID, fetched.ReviewID)
		assert.Equal(t, notes, *fetched.Notes)

		// Update review cycle (record score and complete)
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

		// Non-existent ID error handling
		_, err = repo.GetReviewCycleByID(ctx, "non-existent-id")
		assert.Error(t, err)
		_, err = repo.UpdateReviewCycle(ctx, &ReviewCycle{ReviewID: "non-existent-id"})
		assert.Error(t, err)
	})

	t.Run("Workplace Incidents Operations", func(t *testing.T) {
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
	})
}


