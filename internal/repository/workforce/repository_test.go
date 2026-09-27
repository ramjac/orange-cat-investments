package workforce

import (
	"context"
	"testing"

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
}
