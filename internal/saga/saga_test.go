package saga

import (
	"context"
	"testing"

	"github.com/orange-cat-investments/oci/internal/repository/workforce"
	wfService "github.com/orange-cat-investments/oci/internal/service/workforce"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		require.NoError(t, err)
		assert.Equal(t, "emp-feline-felix", res.EmployeeID)
		assert.Equal(t, "onboarding", res.Status)
		assert.NotEmpty(t, res.ChecklistTasks)
		assert.NotNil(t, res.CareSchedule)
		assert.Equal(t, "Salmon Supreme and Crunchies", res.CareSchedule.DietaryPlan)
		assert.Nil(t, res.VCSAccount)
		assert.Nil(t, res.Workspace)
	})

	t.Run("Execute Feline Offboarding Engine", func(t *testing.T) {
		req := OffboardingRequest{
			EmployeeID:   "emp-feline-felix",
			Reason:       "Medical Leave / Retirement",
			TargetStatus: "retired",
		}

		res, err := offboardEngine.Execute(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "emp-feline-felix", res.EmployeeID)
		assert.Equal(t, "retired", res.Status)
		assert.True(t, res.IdentityRevoked)
		assert.True(t, res.TradingHoldState)
		assert.True(t, res.SessionsRevoked)
		assert.True(t, res.StorageArchived)
		assert.NotEmpty(t, res.ArchiveVaultPath)
		assert.Contains(t, res.CareTransition, "senior veterinary medical hold")

		// Verify emergency medical hold is active for retired feline
		cs, err := wfSvc.GetCareSchedule(ctx, "emp-feline-felix")
		assert.NoError(t, err)
		assert.True(t, cs.EmergencyMedicalHold)
	})

	t.Run("Execute Human Developer Onboarding Saga with COTS IT Provisioning", func(t *testing.T) {
		req := OnboardingRequest{
			EmployeeID:   "emp-human-rick",
			EmployeeType: "human",
			FirstName:    "Rick",
			LastName:     "Newhire",
			Email:        "rick.newhire@oci.local",
			RoleTitle:    "Junior Operations Associate",
			Department:   "Workforce Operations",
			SSHKey:       "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGoZ3...",
			StorageQuota: "50 GB",
		}

		res, err := onboardSaga.Execute(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "emp-human-rick", res.EmployeeID)
		assert.Equal(t, "onboarding", res.Status)
		assert.Nil(t, res.CareSchedule)

		// Verify VCS account provisioning
		require.NotNil(t, res.VCSAccount)
		assert.Equal(t, "rick.newhire", res.VCSAccount.Username)
		assert.Equal(t, "rick.newhire@oci.local", res.VCSAccount.Email)
		assert.Equal(t, "Active", res.VCSAccount.Status)

		// Verify Nextcloud storage workspace provisioning
		require.NotNil(t, res.Workspace)
		assert.Equal(t, "rick.newhire", res.Workspace.Username)
		assert.Equal(t, "50 GB", res.Workspace.Quota)
		assert.Contains(t, res.Workspace.GroupFolders, "all-staff")
		assert.Contains(t, res.Workspace.GroupFolders, "workforce-operations")

		// Verify Snipe-IT IT hardware asset allocation
		require.NotEmpty(t, res.HardwareAssets)
		foundLaptop := false
		foundBadge := false
		for _, asset := range res.HardwareAssets {
			if asset.Category == "Laptop" {
				foundLaptop = true
			}
			if asset.Category == "Access Badge" {
				foundBadge = true
			}
		}
		assert.True(t, foundLaptop, "should allocate laptop")
		assert.True(t, foundBadge, "should allocate access badge")
	})

	t.Run("Execute Human Employee Separation & Offboarding Workflow", func(t *testing.T) {
		req := OffboardingRequest{
			EmployeeID:   "emp-human-rick",
			Reason:       "Career Transition to External Fund",
			TargetStatus: "separated",
		}

		res, err := offboardEngine.Execute(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, "emp-human-rick", res.EmployeeID)
		assert.Equal(t, "separated", res.Status)
		assert.True(t, res.IdentityRevoked)
		assert.True(t, res.SessionsRevoked)
		assert.True(t, res.StorageArchived)
		assert.Contains(t, res.ArchiveVaultPath, "emp-human-rick")

		// Verify Snipe-IT equipment recovery tickets generated
		require.NotNil(t, res.HardwareTickets)
		assert.Equal(t, "Pending Recovery", res.HardwareTickets.Status)
		assert.NotEmpty(t, res.HardwareTickets.TicketID)
		assert.GreaterOrEqual(t, res.HardwareTickets.TotalItems, 1)

		// Verify employee status updated in repository
		emp, err := wfSvc.GetEmployee(ctx, "emp-human-rick")
		assert.NoError(t, err)
		assert.Equal(t, "separated", emp.Status)
	})
}
