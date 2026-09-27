package pki

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCARotationManager(t *testing.T) {
	ctx := context.Background()

	t.Run("Default Config Initialization", func(t *testing.T) {
		manager := NewCARotationManager(CARotationConfig{})
		assert.NotNil(t, manager)

		status, err := manager.GetCAStatus(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "oci-intermediate-ca", status.CAName)
		assert.NotEmpty(t, status.SerialNumber)
		assert.True(t, status.NeedsRotation)
	})

	t.Run("CheckAndRotateCA When Rotation Needed", func(t *testing.T) {
		manager := NewCARotationManager(CARotationConfig{
			RenewalThresholdDays: 30,
		})

		initialStatus, _ := manager.GetCAStatus(ctx)
		initialSerial := initialStatus.SerialNumber

		res, err := manager.CheckAndRotateCA(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "CA_ROTATED_SUCCESSFULLY", res.Status)
		assert.Equal(t, "oci-intermediate-ca", res.IntermediateCAName)
		assert.NotEqual(t, initialSerial, res.NewSerialNumber)
		assert.Equal(t, 12, res.ReissuedCertificatesCount)

		// Check status after rotation
		updatedStatus, errStatus := manager.GetCAStatus(ctx)
		assert.NoError(t, errStatus)
		assert.False(t, updatedStatus.NeedsRotation)
		assert.Equal(t, res.NewSerialNumber, updatedStatus.SerialNumber)
	})

	t.Run("CheckAndRotateCA When No Rotation Needed", func(t *testing.T) {
		manager := NewCARotationManager(CARotationConfig{
			RenewalThresholdDays: 30,
		})

		// Perform initial rotation to make CA fresh
		_, _ = manager.CheckAndRotateCA(ctx)

		// Second check should indicate no rotation needed
		res, err := manager.CheckAndRotateCA(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "NO_ROTATION_NEEDED", res.Status)
	})
}
