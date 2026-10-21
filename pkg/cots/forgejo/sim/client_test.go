package sim

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestForgejoSimClient(t *testing.T) {
	client := NewClient(Config{BaseURL: "http://unreachable-host.invalid"})
	u, err := client.CreateUser(context.Background(), CreateUserOption{
		Username: "sim_tester",
		Email:    "tester@oci.local",
	})
	require.NoError(t, err)
	assert.Equal(t, "sim_tester", u.UserName)
}
