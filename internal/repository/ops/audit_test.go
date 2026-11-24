package ops

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditLog_HashChainIntegrityAndTamperDetection(t *testing.T) {
	ctx := context.Background()
	repo := NewMockRepository()

	// Empty chain is valid with 0 verified records.
	emptyRes, err := repo.VerifyAuditChain(ctx)
	require.NoError(t, err)
	assert.True(t, emptyRes.Valid)
	assert.Equal(t, int64(0), emptyRes.VerifiedRecords)

	// Append 3 sequential records.
	r1, err := repo.AppendAuditLog(ctx, &AuditLog{
		ActorID:      "emp-human-elena",
		ActorRole:    "veterinary",
		Action:       "workforce.medical_hold.updated",
		ResourceType: "care_schedule",
		ResourceID:   "emp-feline-garfield",
		MetadataJSON: `{"emergency_medical_hold":true}`,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(1), r1.ID)
	assert.Equal(t, AuditGenesisHash, r1.PreviousHash)
	assert.Len(t, r1.RecordHash, 64)

	r2, err := repo.AppendAuditLog(ctx, &AuditLog{
		ActorID:      "emp-human-alice",
		ActorRole:    "hr",
		Action:       "workforce.employee.offboarded",
		ResourceType: "employee",
		ResourceID:   "emp-test-01",
		MetadataJSON: `{"target_status":"separated"}`,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(2), r2.ID)
	assert.Equal(t, r1.RecordHash, r2.PreviousHash)

	r3, err := repo.AppendAuditLog(ctx, &AuditLog{
		ActorID:      "cust-active-chloe",
		ActorRole:    "customer",
		Action:       "customer.api_key.revoked",
		ResourceType: "api_key",
		ResourceID:   "key-999",
		MetadataJSON: `{"status":"revoked"}`,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(3), r3.ID)
	assert.Equal(t, r2.RecordHash, r3.PreviousHash)

	// Verify intact chain.
	okRes, err := repo.VerifyAuditChain(ctx)
	require.NoError(t, err)
	assert.True(t, okRes.Valid)
	assert.Equal(t, int64(3), okRes.VerifiedRecords)
	assert.Equal(t, r3.RecordHash, okRes.LatestHash)
	assert.Equal(t, int64(0), okRes.BrokenAtID)

	// ListAuditLogs filters correctly.
	filtered, err := repo.ListAuditLogs(ctx, "emp-human-elena", "", 10, 0)
	require.NoError(t, err)
	require.Len(t, filtered, 1)
	assert.Equal(t, r1.ID, filtered[0].ID)

	// Simulate single-field tampering on middle record #2.
	require.True(t, TamperAuditLogForTest(repo, 2, `{"target_status":"active"}`))

	brokenRes, err := repo.VerifyAuditChain(ctx)
	require.NoError(t, err)
	assert.False(t, brokenRes.Valid)
	assert.Equal(t, int64(1), brokenRes.VerifiedRecords)
	assert.Equal(t, int64(2), brokenRes.BrokenAtID)
	assert.NotEqual(t, brokenRes.ExpectedHash, brokenRes.ActualHash)
	assert.Contains(t, brokenRes.Reason, "tampering detected")
}
