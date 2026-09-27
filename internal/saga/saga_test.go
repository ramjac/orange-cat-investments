package saga

import (
	"context"
	"testing"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
	"github.com/stretchr/testify/assert"
)

func TestOnboardingSagaAndOffboardingEngine(t *testing.T) {
	repo := workforce.NewRepository(nil)
	wfSvc := wfService.NewService(repo)

	onboardSaga := NewOnboardingSaga(wfSvc, nil, nil)
	offboardEngine := NewOffboardingEngine(wfSvc, nil, nil)

	ctx := context.Background()

	t.Run("Execute Feline Onboarding Saga", func(t *testing.T) {
		req := OnboardingRequest{
			EmployeeID:   "emp-feline-felix",
			EmployeeType: "feline",
			FirstName:    "Felix",
			RoleTitle:    "Junior Observation Specialist",
			Department:   "Habitat Operations",
			DietaryPlan:  "Salmon Supreme and Crunchies",
			FeedingTimes: []string{"07:30 AM", "06:00 PM"},
		}

		res, err := onboardSaga.Execute(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, "emp-feline-felix", res.EmployeeID)
		assert.Equal(t, "onboarding", res.Status)
		assert.NotEmpty(t, res.ChecklistTasks)
		assert.NotNil(t, res.CareSchedule)
		assert.Equal(t, "Salmon Supreme and Crunchies", res.CareSchedule.DietaryPlan)
	})

	t.Run("Execute Feline Offboarding Engine", func(t *testing.T) {
		req := OffboardingRequest{
			EmployeeID:   "emp-feline-felix",
			Reason:       "Medical Leave / Retirement",
			TargetStatus: "retired",
		}

		res, err := offboardEngine.Execute(ctx, req)
		assert.NoError(t, err)
		assert.Equal(t, "emp-feline-felix", res.EmployeeID)
		assert.Equal(t, "retired", res.Status)
		assert.True(t, res.IdentityRevoked)
		assert.True(t, res.TradingHoldState)

		// Verify emergency medical hold is active for retired feline
		cs, err := wfSvc.GetCareSchedule(ctx, "emp-feline-felix")
		assert.NoError(t, err)
		assert.True(t, cs.EmergencyMedicalHold)
	})

	t.Run("Execute Human Onboarding Saga", func(t *testing.T) {
		req := OnboardingRequest{
			EmployeeType: "human",
			FirstName:    "Alice",
			LastName:     "Vance",
			Email:        "alice.vance@oci.local",
			RoleTitle:    "Head of HR",
			Department:   "Workforce Operations",
		}

		res, err := onboardSaga.Execute(ctx, req)
		assert.NoError(t, err)
		assert.NotEmpty(t, res.EmployeeID)
		assert.Equal(t, "onboarding", res.Status)
		assert.Nil(t, res.CareSchedule)
	})
}
