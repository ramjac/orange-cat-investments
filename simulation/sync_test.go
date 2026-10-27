package simulation

import (
	"context"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncRepoToForgejoScript_MockParity(t *testing.T) {
	mockForgejo := NewMockForgejoServer()
	server := httptest.NewServer(mockForgejo.Handler())
	defer server.Close()

	// Locate scripts/sync-repo-to-forgejo.sh
	scriptPath, err := filepath.Abs("../scripts/sync-repo-to-forgejo.sh")
	require.NoError(t, err)

	// 1. Dry Run Execution
	cmdDry := exec.CommandContext(context.Background(), scriptPath,
		"--url", server.URL,
		"--admin-user", "admin",
		"--admin-password", "mock-pass",
		"--dry-run",
	)
	outDry, err := cmdDry.CombinedOutput()
	require.NoError(t, err, "dry run should succeed against mock forgejo: %s", string(outDry))
	assert.Contains(t, string(outDry), "Connected to Forgejo host")
	assert.Contains(t, string(outDry), "DRY RUN MODE")

	// 2. Full Execution (Check Only after verification)
	cmdCheck := exec.CommandContext(context.Background(), scriptPath,
		"--url", server.URL,
		"--admin-user", "admin",
		"--admin-password", "mock-pass",
		"--check-only",
	)
	outCheck, err := cmdCheck.CombinedOutput()
	require.NoError(t, err, "check-only should verify seeded state: %s", string(outCheck))
	assert.Contains(t, string(outCheck), "Forgejo repository state is verified and ready")
}
