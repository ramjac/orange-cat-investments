package valkey

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCustomerSessionStore(t *testing.T) {
	store := NewMemoryCustomerSessionStore()
	defer store.Close()

	// 1. Create session
	sess, err := store.Create("cust-101", "Chloe Cat", "chloe@example.com")
	require.NoError(t, err)
	assert.NotEmpty(t, sess.SessionID)
	assert.Equal(t, "cust-101", sess.CustomerID)
	assert.Equal(t, "Chloe Cat", sess.Name)
	assert.Equal(t, "chloe@example.com", sess.Email)

	// 2. Get valid session
	retrieved, ok := store.Get(sess.SessionID)
	require.True(t, ok)
	assert.Equal(t, sess.SessionID, retrieved.SessionID)
	assert.Equal(t, sess.CustomerID, retrieved.CustomerID)

	// 3. Get non-existent session
	_, ok = store.Get("non-existent-token")
	assert.False(t, ok)

	// 4. Set expired session
	expiredSess := &CustomerSession{
		SessionID:  "expired-token",
		CustomerID: "cust-101",
		Email:      "chloe@example.com",
		Name:       "Chloe Cat",
		ExpiresAt:  time.Now().Add(-1 * time.Hour),
	}
	store.SetSession(expiredSess)
	_, ok = store.Get("expired-token")
	assert.False(t, ok, "expired session should return false")

	// 5. Create multiple sessions for same customer and purge
	sess2, err := store.Create("cust-101", "Chloe Cat", "chloe@example.com")
	require.NoError(t, err)
	sessOther, err := store.Create("cust-999", "Other Customer", "other@example.com")
	require.NoError(t, err)

	purged, err := store.PurgeCustomerSessions("cust-101")
	require.NoError(t, err)
	assert.Equal(t, 3, purged) // sess, expiredSess, and sess2 were all cust-101

	_, ok = store.Get(sess.SessionID)
	assert.False(t, ok)
	_, ok = store.Get(sess2.SessionID)
	assert.False(t, ok)

	// Other customer session must remain
	_, ok = store.Get(sessOther.SessionID)
	assert.True(t, ok)

	// 6. Delete specific session
	store.Delete(sessOther.SessionID)
	_, ok = store.Get(sessOther.SessionID)
	assert.False(t, ok)

	// 7. Test PruneStaleSessions
	activeForPrune, err := store.Create("cust-prune", "Prune User", "prune@example.com")
	require.NoError(t, err)
	expiredForPrune := &CustomerSession{
		SessionID:  "sess-prune-expired",
		CustomerID: "cust-prune",
		ExpiresAt:  time.Now().Add(-10 * time.Minute),
	}
	store.SetSession(expiredForPrune)

	prunedCount := store.PruneStaleSessions("cust-prune")
	assert.Equal(t, 1, prunedCount)
	_, ok = store.Get(activeForPrune.SessionID)
	assert.True(t, ok, "active session should remain after pruning stale sessions")
	_, ok = store.Get("sess-prune-expired")
	assert.False(t, ok)
}

func TestMemoryEmployeeSessionStore(t *testing.T) {
	store := NewMemoryEmployeeSessionStore()
	defer store.Close()

	// 1. Create session
	sess, err := store.Create("emp-101", "elise@example.com", "Elise", "ENGINEERING", "Core Platform")
	require.NoError(t, err)
	assert.NotEmpty(t, sess.SessionID)
	assert.Equal(t, "emp-101", sess.EmployeeID)
	assert.Equal(t, "elise@example.com", sess.Email)
	assert.Equal(t, "Elise", sess.Name)
	assert.Equal(t, "engineering", sess.Role, "role should be lower-cased and trimmed")
	assert.Equal(t, "Core Platform", sess.Department)

	// 2. Get valid session
	retrieved, ok := store.Get(sess.SessionID)
	require.True(t, ok)
	assert.Equal(t, sess.SessionID, retrieved.SessionID)

	// 3. Get non-existent session
	_, ok = store.Get("unknown-token")
	assert.False(t, ok)

	// 4. Set expired session
	expiredSess := &EmployeeSession{
		SessionID:  "emp-expired",
		EmployeeID: "emp-101",
		Email:      "elise@example.com",
		ExpiresAt:  time.Now().Add(-1 * time.Hour),
	}
	store.SetSession(expiredSess)
	_, ok = store.Get("emp-expired")
	assert.False(t, ok)

	// 5. Purge employee sessions
	sess2, err := store.Create("emp-101", "elise@example.com", "Elise", "engineering", "Core Platform")
	require.NoError(t, err)
	sessOther, err := store.Create("emp-202", "marcus@example.com", "Marcus", "engineering", "Core Platform")
	require.NoError(t, err)

	purged, err := store.PurgeEmployeeSessions("emp-101")
	require.NoError(t, err)
	assert.Equal(t, 3, purged) // sess, expiredSess, and sess2 were all emp-101

	_, ok = store.Get(sess.SessionID)
	assert.False(t, ok)
	_, ok = store.Get(sess2.SessionID)
	assert.False(t, ok)

	// Other employee session must remain
	_, ok = store.Get(sessOther.SessionID)
	assert.True(t, ok)

	// 6. Test PruneStaleSessions
	activeEmp, err := store.Create("emp-prune", "emp-prune@example.com", "Prune Emp", "engineering", "Core")
	require.NoError(t, err)
	expiredEmp := &EmployeeSession{
		SessionID:  "emp-prune-expired",
		EmployeeID: "emp-prune",
		ExpiresAt:  time.Now().Add(-10 * time.Minute),
	}
	store.SetSession(expiredEmp)

	prunedEmp := store.PruneStaleSessions("emp-prune")
	assert.Equal(t, 1, prunedEmp)
	_, ok = store.Get(activeEmp.SessionID)
	assert.True(t, ok)
	_, ok = store.Get("emp-prune-expired")
	assert.False(t, ok)
}

func TestStoreFallbackWhenNoValkey(t *testing.T) {
	// Empty address falls back to memory store
	custStore := NewCustomerSessionStore("", "", 0, nil)
	require.NotNil(t, custStore)
	sess, err := custStore.Create("cust-1", "Chloe", "chloe@test.com")
	require.NoError(t, err)
	retrieved, ok := custStore.Get(sess.SessionID)
	assert.True(t, ok)
	assert.Equal(t, "cust-1", retrieved.CustomerID)

	// Unreachable address also falls back gracefully without panic or failure
	empStore := NewEmployeeSessionStore("127.0.0.1:54321", "", 0, nil)
	require.NotNil(t, empStore)
	empSess, err := empStore.Create("emp-1", "frank@test.com", "Frank", "facilities", "Ops")
	require.NoError(t, err)
	empRetrieved, ok := empStore.Get(empSess.SessionID)
	assert.True(t, ok)
	assert.Equal(t, "emp-1", empRetrieved.EmployeeID)
}
