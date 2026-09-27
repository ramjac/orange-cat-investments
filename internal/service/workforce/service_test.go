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
}
