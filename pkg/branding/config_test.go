package branding_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orange-cat-investments/oci/pkg/branding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := branding.DefaultConfig()
	assert.Equal(t, "Orange Cat Investments LLC", cfg.Organization.LegalName)
	assert.Equal(t, "Orange Cat Investments", cfg.Organization.TradingName)
	assert.Equal(t, "#f97316", cfg.Branding.PrimaryColor)
	assert.True(t, cfg.Modules.EnableCoreInvest)
}

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "branding.yaml")

	content := `
organization:
  legal_name: "Acme Corp LLC"
  trading_name: "Acme Corp"
  tagline: "Custom Business Solutions"
  base_domain: "acme.local"
branding:
  primary_color: "#2563eb"
  secondary_color: "#1d4ed8"
  accent_color: "#60a5fa"
  background_color: "#0f172a"
  card_color: "#1e293b"
  text_color: "#f8fafc"
  logo_icon: "⚡"
  font_family: "Inter, sans-serif"
modules:
  enable_core_invest: false
  enable_feline_workforce: false
  enable_pebble_gateway: true
`
	err := os.WriteFile(configPath, []byte(content), 0644)
	require.NoError(t, err)

	cfg, err := branding.LoadConfig(configPath)
	require.NoError(t, err)
	assert.Equal(t, "Acme Corp LLC", cfg.Organization.LegalName)
	assert.Equal(t, "Acme Corp", cfg.Organization.TradingName)
	assert.Equal(t, "#2563eb", cfg.Branding.PrimaryColor)
	assert.False(t, cfg.Modules.EnableCoreInvest)
}

func TestLoadConfig_MissingFileFallback(t *testing.T) {
	cfg, err := branding.LoadConfig("/path/does/not/exist.yaml")
	require.NoError(t, err)
	assert.Equal(t, "Orange Cat Investments LLC", cfg.Organization.LegalName)
}

func TestLoadConfig_ReadErrorPropagated(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := branding.LoadConfig(tmpDir)
	assert.Error(t, err)
}

