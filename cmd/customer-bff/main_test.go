package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateRandomToken(t *testing.T) {
	token, err := generateRandomToken(16)
	require.NoError(t, err)
	assert.Len(t, token, 32) // 16 bytes encoded in hex is 32 chars

	token2, err := generateRandomToken(16)
	require.NoError(t, err)
	assert.NotEqual(t, token, token2, "subsequent tokens should be cryptographically unique")
}

func TestNewSessionStore_NoHardcodedTokensByDefault(t *testing.T) {
	store := NewSessionStore()
	_, okArthur := store.Get("sess-arthur-token")
	assert.False(t, okArthur, "sess-arthur-token should not exist by default")
	_, okChloe := store.Get("sess-chloe-token")
	assert.False(t, okChloe, "sess-chloe-token should not exist by default")

	store.SetSession(&CustomerSession{
		SessionID:  "sess-test-token",
		CustomerID: "cust-test",
		ExpiresAt:  time.Now().Add(time.Hour),
	})
	sess, okAfter := store.Get("sess-test-token")
	assert.True(t, okAfter)
	assert.Equal(t, "cust-test", sess.CustomerID)
}

func TestSessionStore_Create_And_GenerateRandomToken(t *testing.T) {
	store := NewSessionStore()
	sess, err := store.Create("cust-123", "Test User", "test@oci.local")
	require.NoError(t, err)
	require.NotNil(t, sess)
	assert.Equal(t, "cust-123", sess.CustomerID)
	assert.True(t, len(sess.SessionID) > 5)
	assert.Equal(t, "sess-", sess.SessionID[:5])

	gotSess, ok := store.Get(sess.SessionID)
	assert.True(t, ok)
	assert.Equal(t, sess, gotSess)

	tok, err := generateRandomToken(16)
	require.NoError(t, err)
	assert.Len(t, tok, 32) // 16 bytes encoded as hex = 32 chars
}

func TestNewCustomerBFF_RequiresBLLClient(t *testing.T) {
	assert.Panics(t, func() {
		NewCustomerBFF(nil, nil)
	}, "NewCustomerBFF must panic when bllClient is nil in production code")
}
