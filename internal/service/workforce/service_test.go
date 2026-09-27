package workforce

import (
	"context"
	"testing"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	"github.com/stretchr/testify/assert"
)

func TestWorkforceService(t *testing.T) {
	repo := workforce.NewRepository(nil)
	svc := NewService(repo)
	ctx := context.Background()

	t.Run("Get Employee", func(t *testing.T) {
		emp, err := svc.GetEmployee(ctx, "emp-feline-garfield")
		assert.NoError(t, err)
		assert.Equal(t, "Garfield", emp.FirstName)
	})

	t.Run("Onboard Feline Employee", func(t *testing.T) {
		feline := &workforce.Employee{
			FirstName:    "Sylvester",
			EmployeeType: "feline",
			RoleTitle:    "Junior Perch Inspector",
			Department:   "Habitat Ops",
		}
		cs := &workforce.CareSchedule{
			DietaryPlan:  "Tuna delight",
			FeedingTimes: []string{"09:00 AM", "05:00 PM"},
		}

		created, err := svc.OnboardEmployee(ctx, feline, cs)
		assert.NoError(t, err)
		assert.NotEmpty(t, created.EmployeeID)
		assert.Equal(t, "onboarding", created.Status)

		// Verify onboarding tasks were generated
		tasks, err := svc.ListOnboardingChecklist(ctx, created.EmployeeID)
		assert.NoError(t, err)
		assert.Len(t, tasks, 4)

		// Verify care schedule was stored
		storedCS, err := svc.GetCareSchedule(ctx, created.EmployeeID)
		assert.NoError(t, err)
		assert.Equal(t, "Tuna delight", storedCS.DietaryPlan)
	})

	t.Run("Emergency Medical Hold Toggle", func(t *testing.T) {
		cs, err := svc.SetEmergencyMedicalHold(ctx, "emp-feline-garfield", true)
		assert.NoError(t, err)
		assert.True(t, cs.EmergencyMedicalHold)

		cs2, err := svc.SetEmergencyMedicalHold(ctx, "emp-feline-garfield", false)
		assert.NoError(t, err)
		assert.False(t, cs2.EmergencyMedicalHold)
	})

	t.Run("Leave Requests Service Workflow & Validations", func(t *testing.T) {
		reason := "Premium catnip break & laser chasing recharge"
		req, err := svc.CreateLeaveRequest(ctx, "emp-feline-garfield", "catnip_break", "2026-10-10", "2026-10-12", &reason)
		assert.NoError(t, err)
		assert.NotEmpty(t, req.LeaveID)
		assert.Equal(t, "catnip_break", req.LeaveType)
		assert.Equal(t, "pending", req.Status)

		// Get Leave Request
		fetched, err := svc.GetLeaveRequest(ctx, req.LeaveID)
		assert.NoError(t, err)
		assert.Equal(t, req.LeaveID, fetched.LeaveID)

		// List Leave Requests
		list, err := svc.ListLeaveRequests(ctx, "emp-feline-garfield", "pending")
		assert.NoError(t, err)
		assert.NotEmpty(t, list)

		// Approve Leave Request
		approved, err := svc.UpdateLeaveRequestStatus(ctx, req.LeaveID, "approved")
		assert.NoError(t, err)
		assert.Equal(t, "approved", approved.Status)

		// Validation: Missing employee_id
		_, err = svc.CreateLeaveRequest(ctx, "", "vacation", "2026-10-10", "2026-10-12", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "employee_id is required")

		// Validation: Invalid leave_type
		_, err = svc.CreateLeaveRequest(ctx, "emp-feline-garfield", "unsupported_type", "2026-10-10", "2026-10-12", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid leave_type")

		// Validation: Invalid date format
		_, err = svc.CreateLeaveRequest(ctx, "emp-feline-garfield", "catnip_break", "10-10-2026", "2026-10-12", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "YYYY-MM-DD")

		// Validation: End date before start date
		_, err = svc.CreateLeaveRequest(ctx, "emp-feline-garfield", "catnip_break", "2026-10-15", "2026-10-12", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "end_date must be on or after start_date")

		// Validation: Invalid status update
		_, err = svc.UpdateLeaveRequestStatus(ctx, req.LeaveID, "invalid_status")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})

	t.Run("Review Cycles Service Workflow & Validations", func(t *testing.T) {
		notes := "Annual feline agility, sunbeam tracking, and alpha response readiness"
		reviewer := "emp-human-elena"
		rc, err := svc.CreateReviewCycle(ctx, "emp-feline-garfield", "feline_health_assessment", "2026-11-20", &reviewer, &notes)
		assert.NoError(t, err)
		assert.NotEmpty(t, rc.ReviewID)
		assert.Equal(t, "feline_health_assessment", rc.ReviewType)
		assert.Equal(t, "scheduled", rc.Status)
		assert.Equal(t, "2026-11-20", rc.ScheduledFor)

		// Get Review Cycle
		fetched, err := svc.GetReviewCycle(ctx, rc.ReviewID)
		assert.NoError(t, err)
		assert.Equal(t, rc.ReviewID, fetched.ReviewID)

		// List Review Cycles
		list, err := svc.ListReviewCycles(ctx, "emp-feline-garfield", "scheduled", "feline_health_assessment")
		assert.NoError(t, err)
		assert.NotEmpty(t, list)

		// Update Review Cycle (Evaluation)
		score := 4.92
		newStatus := "completed"
		evalNotes := "Phenomenal alpha perch surveillance metrics and optimal vitals"
		updated, err := svc.UpdateReviewCycle(ctx, rc.ReviewID, &newStatus, nil, &score, &evalNotes, nil)
		assert.NoError(t, err)
		assert.Equal(t, "completed", updated.Status)
		assert.Equal(t, score, *updated.Score)
		assert.Equal(t, evalNotes, *updated.Notes)
		assert.NotNil(t, updated.CompletedAt)

		// Validation: Missing employee_id
		_, err = svc.CreateReviewCycle(ctx, "", "performance", "2026-11-20", nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "employee_id is required")

		// Validation: Invalid review_type
		_, err = svc.CreateReviewCycle(ctx, "emp-feline-garfield", "invalid_type", "2026-11-20", nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid review_type")

		// Validation: Invalid scheduled_for date format
		_, err = svc.CreateReviewCycle(ctx, "emp-feline-garfield", "performance", "11-20-2026", nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "YYYY-MM-DD")

		// Validation: Score out of range (> 5.0)
		badScore := 6.5
		_, err = svc.UpdateReviewCycle(ctx, rc.ReviewID, nil, nil, &badScore, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "score must be between 0.00 and 5.00")

		// Validation: Negative score
		negScore := -1.0
		_, err = svc.UpdateReviewCycle(ctx, rc.ReviewID, nil, nil, &negScore, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "score must be between 0.00 and 5.00")

		// Validation: Invalid status update
		badStatus := "flying_high"
		_, err = svc.UpdateReviewCycle(ctx, rc.ReviewID, &badStatus, nil, nil, nil, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid status")
	})
}


