package sim

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextcloudSimClient(t *testing.T) {
	client := NewClient(Config{BaseURL: "http://unreachable-host.invalid"})
	u, err := client.CreateUser(context.Background(), CreateUserOption{UserID: "test_sim"})
	require.NoError(t, err)
	assert.Equal(t, "test_sim", u.ID)
}
