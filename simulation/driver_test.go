package simulation

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, "http://localhost:8080", cfg.ServerURL)
	assert.Equal(t, "http://localhost:8081", cfg.EmployeeBFFURL)
	assert.Equal(t, "mock", cfg.Mode)
	assert.Equal(t, "all", cfg.TargetPersona)
	assert.Equal(t, 1, cfg.Iterations)
}

func TestRegistryPersonas(t *testing.T) {
	registry := NewRegistry()
	personas := registry.List()
	assert.Len(t, personas, 12)

	expectedPersonas := []string{
		"emp-feline-garfield",
		"emp-feline-barneby",
		"emp-human-alice",
		"emp-human-bob",
		"emp-human-carol",
		"emp-human-david",
		"emp-human-elena",
		"emp-human-frank",
		"cust-longterm-arthur",
		"cust-active-chloe",
		"emp-human-rick",
		"emp-human-elise",
	}

	for _, id := range expectedPersonas {
		p, ok := registry.Get(id)
		require.True(t, ok, "persona %s should exist in registry", id)
		assert.NotEmpty(t, p.Name)
		assert.NotEmpty(t, p.RoleTitle)
		assert.NotEmpty(t, p.Department)
		assert.NotEmpty(t, p.Actions)
	}
}

func TestDriverMockExecutionAllPersonas(t *testing.T) {
	cfg := &Config{
		Mode:          "mock",
		TargetPersona: "all",
		Iterations:    1,
		ActionDelay:   1 * time.Millisecond,
		Verbose:       false,
	}

	driver, err := NewDriver(cfg)
	require.NoError(t, err)
	defer driver.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	summary, err := driver.Run(ctx)
	require.NoError(t, err)

	assert.GreaterOrEqual(t, summary.TotalActions, 30)
	assert.Equal(t, summary.TotalActions, summary.Successes)
	assert.Equal(t, 0, summary.Failures)
}

func TestDriverSinglePersonaGarfield(t *testing.T) {
	cfg := &Config{
		Mode:          "mock",
		TargetPersona: "emp-feline-garfield",
		Iterations:    1,
		ActionDelay:   1 * time.Millisecond,
		Verbose:       false,
	}

	driver, err := NewDriver(cfg)
	require.NoError(t, err)
	defer driver.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	summary, err := driver.Run(ctx)
	require.NoError(t, err)

	assert.Equal(t, 4, summary.TotalActions)
	assert.Equal(t, 4, summary.Successes)
	assert.Equal(t, 0, summary.Failures)
}

func TestDriverSinglePersonaBarneby(t *testing.T) {
	cfg := &Config{
		Mode:          "mock",
		TargetPersona: "emp-feline-barneby",
		Iterations:    1,
		ActionDelay:   1 * time.Millisecond,
		Verbose:       false,
	}

	driver, err := NewDriver(cfg)
	require.NoError(t, err)
	defer driver.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	summary, err := driver.Run(ctx)
	require.NoError(t, err)

	assert.Equal(t, 3, summary.TotalActions)
	assert.Equal(t, 3, summary.Successes)
	assert.Equal(t, 0, summary.Failures)
}

func TestDriverSinglePersonaAlice(t *testing.T) {
	cfg := &Config{
		Mode:          "mock",
		TargetPersona: "emp-human-alice",
		Iterations:    1,
		ActionDelay:   1 * time.Millisecond,
		Verbose:       false,
	}

	driver, err := NewDriver(cfg)
	require.NoError(t, err)
	defer driver.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	summary, err := driver.Run(ctx)
	require.NoError(t, err)

	assert.Equal(t, 5, summary.TotalActions)
	assert.Equal(t, 5, summary.Successes)
	assert.Equal(t, 0, summary.Failures)
}

func TestDriverSinglePersonaBob(t *testing.T) {
	cfg := &Config{
		Mode:          "mock",
		TargetPersona: "emp-human-bob",
		Iterations:    1,
		ActionDelay:   1 * time.Millisecond,
		Verbose:       false,
	}

	driver, err := NewDriver(cfg)
	require.NoError(t, err)
	defer driver.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	summary, err := driver.Run(ctx)
	require.NoError(t, err)

	assert.Equal(t, 3, summary.TotalActions)
	assert.Equal(t, 3, summary.Successes)
	assert.Equal(t, 0, summary.Failures)
}

func TestDriverSinglePersonaRick(t *testing.T) {
	cfg := &Config{
		Mode:          "mock",
		TargetPersona: "emp-human-rick",
		Iterations:    1,
		ActionDelay:   1 * time.Millisecond,
		Verbose:       false,
	}

	driver, err := NewDriver(cfg)
	require.NoError(t, err)
	defer driver.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	summary, err := driver.Run(ctx)
	require.NoError(t, err)

	assert.Equal(t, 5, summary.TotalActions)
	assert.Equal(t, 5, summary.Successes)
	assert.Equal(t, 0, summary.Failures)
}

func TestDriverSinglePersonaElise(t *testing.T) {
	cfg := &Config{
		Mode:          "mock",
		TargetPersona: "emp-human-elise",
		Iterations:    1,
		ActionDelay:   1 * time.Millisecond,
		Verbose:       false,
	}

	driver, err := NewDriver(cfg)
	require.NoError(t, err)
	defer driver.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	summary, err := driver.Run(ctx)
	require.NoError(t, err)

	assert.Equal(t, 5, summary.TotalActions)
	assert.Equal(t, 5, summary.Successes)
	assert.Equal(t, 0, summary.Failures)
}

func TestDriverSinglePersonaFrank(t *testing.T) {
	cfg := &Config{
		Mode:          "mock",
		TargetPersona: "emp-human-frank",
		Iterations:    1,
		ActionDelay:   1 * time.Millisecond,
		Verbose:       false,
	}

	driver, err := NewDriver(cfg)
	require.NoError(t, err)
	defer driver.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	summary, err := driver.Run(ctx)
	require.NoError(t, err)

	assert.Equal(t, 4, summary.TotalActions)
	assert.Equal(t, 4, summary.Successes)
	assert.Equal(t, 0, summary.Failures)
}

func TestDriverSinglePersonaArthur(t *testing.T) {
	cfg := &Config{
		Mode:          "mock",
		TargetPersona: "cust-longterm-arthur",
		Iterations:    1,
		ActionDelay:   1 * time.Millisecond,
		Verbose:       false,
	}

	driver, err := NewDriver(cfg)
	require.NoError(t, err)
	defer driver.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	summary, err := driver.Run(ctx)
	require.NoError(t, err)

	assert.Equal(t, 3, summary.TotalActions)
	assert.Equal(t, 3, summary.Successes)
	assert.Equal(t, 0, summary.Failures)
}

func TestDriverSinglePersonaChloe(t *testing.T) {
	cfg := &Config{
		Mode:          "mock",
		TargetPersona: "cust-active-chloe",
		Iterations:    1,
		ActionDelay:   1 * time.Millisecond,
		Verbose:       false,
	}

	driver, err := NewDriver(cfg)
	require.NoError(t, err)
	defer driver.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	summary, err := driver.Run(ctx)
	require.NoError(t, err)

	assert.Equal(t, 4, summary.TotalActions)
	assert.Equal(t, 4, summary.Successes)
	assert.Equal(t, 0, summary.Failures)
}
