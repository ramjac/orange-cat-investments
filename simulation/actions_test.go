package simulation

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomAction(t *testing.T) {
	t.Run("nil persona", func(t *testing.T) {
		act := RandomAction(nil)
		assert.Nil(t, act)
	})

	t.Run("empty actions", func(t *testing.T) {
		p := &Persona{
			ID:      "test-p",
			Actions: []Action{},
		}
		act := RandomAction(p)
		assert.Nil(t, act)
	})

	t.Run("zero activity level", func(t *testing.T) {
		// Activity level strictly zero or negative should select nil if set to 0.0 with check
		p := &Persona{
			ID:            "test-p",
			ActivityLevel: 0.0000001, // extremely low probability so rand.Float64() < activityLevel is false
			Actions: []Action{
				{Name: "action1"},
			},
		}
		// With 0.0000001, rand.Float64() will practically always be >= activityLevel
		act := RandomAction(p)
		assert.Nil(t, act)
	})

	t.Run("full activity level", func(t *testing.T) {
		p := &Persona{
			ID:            "test-p",
			ActivityLevel: 1.0,
			Actions: []Action{
				{Name: "action1"},
				{Name: "action2"},
			},
		}

		for i := 0; i < 20; i++ {
			act := RandomAction(p)
			require.NotNil(t, act)
			assert.Contains(t, []string{"action1", "action2"}, act.Name)
		}
	})

	t.Run("default activity level when unspecified", func(t *testing.T) {
		p := &Persona{
			ID: "test-p",
			Actions: []Action{
				{Name: "default_action"},
			},
		}
		act := RandomAction(p)
		require.NotNil(t, act)
		assert.Equal(t, "default_action", act.Name)
	})

	t.Run("partial activity level distribution", func(t *testing.T) {
		p := &Persona{
			ID:            "test-p",
			ActivityLevel: 0.5,
			Actions: []Action{
				{Name: "action1"},
			},
		}

		var selectedCount, nilCount int
		for i := 0; i < 100; i++ {
			act := RandomAction(p)
			if act != nil {
				selectedCount++
			} else {
				nilCount++
			}
		}

		assert.Greater(t, selectedCount, 0, "should select action at least once with 0.5 activity level over 100 trials")
		assert.Greater(t, nilCount, 0, "should return nil at least once with 0.5 activity level over 100 trials")
	})
}

func TestGarfieldSleepInHallway_TrippedAndNotTripped(t *testing.T) {
	client, err := NewClient(&Config{Mode: "mock"})
	require.NoError(t, err)
	defer client.Close()

	registry := NewRegistry()
	garfield, ok := registry.Get("emp-feline-garfield")
	require.True(t, ok)

	var sleepAction *Action
	for _, act := range garfield.Actions {
		if act.Name == "garfield_sleep_in_hallway" {
			sleepAction = &act
			break
		}
	}
	require.NotNil(t, sleepAction, "garfield_sleep_in_hallway action should exist")

	ctx := context.Background()

	var sawTripped, sawNotTripped bool
	for i := 0; i < 50; i++ {
		res, err := sleepAction.Execute(ctx, client)
		require.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.Success)

		if res.Details == "Garfield slept peacefully in the middle of the hallway. Nobody tripped." {
			sawNotTripped = true
			assert.Equal(t, http.StatusOK, res.StatusCode)
		} else {
			sawTripped = true
			assert.Contains(t, res.Details, "TRIPPED!")
			assert.Contains(t, res.Details, "Garfield sent to vet")
			assert.Contains(t, res.Details, "Workplace injury ticket created")
		}

		if sawTripped && sawNotTripped {
			break
		}
	}

	assert.True(t, sawNotTripped, "should have observed peaceful sleep outcome")
	assert.True(t, sawTripped, "should have observed tripped outcome")
}

func TestAllPersonaActionsExecution(t *testing.T) {
	client, err := NewClient(&Config{Mode: "mock"})
	require.NoError(t, err)
	defer client.Close()

	registry := NewRegistry()
	personas := registry.List()
	require.NotEmpty(t, personas)

	ctx := context.Background()

	for _, p := range personas {
		t.Run(p.Name, func(t *testing.T) {
			require.NotEmpty(t, p.Actions, "persona %s should have actions", p.Name)
			for _, act := range p.Actions {
				t.Run(act.Name, func(t *testing.T) {
					assert.NotEmpty(t, act.Name)
					assert.NotEmpty(t, act.Description)
					assert.NotNil(t, act.Execute)

					res, err := act.Execute(ctx, client)
					require.NoError(t, err, "action %s execution failed with unexpected error", act.Name)
					require.NotNil(t, res, "action %s returned nil result", act.Name)

					assert.Equal(t, p.ID, res.PersonaID)
					assert.Equal(t, p.Name, res.PersonaName)
					assert.Equal(t, act.Name, res.ActionName)
					assert.True(t, res.Success, "action %s should succeed in mock mode. Details: %s", act.Name, res.Details)
					assert.False(t, res.Timestamp.IsZero())
				})
			}
		})
	}
}

func TestActionsErrorHandling_CanceledContext(t *testing.T) {
	client, err := NewClient(&Config{Mode: "mock"})
	require.NoError(t, err)
	defer client.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context immediately

	registry := NewRegistry()
	garfield, ok := registry.Get("emp-feline-garfield")
	require.True(t, ok)

	action := garfield.Actions[0]
	res, err := action.Execute(ctx, client)
	assert.Error(t, err, "action execution with canceled context should return an error")
	assert.Nil(t, res)
}

func TestActionsErrorHandling_InvalidServerURL(t *testing.T) {
	client := &Client{
		serverURL:  "http://invalid.localhost:99999",
		bffURL:     "http://invalid.localhost:99999",
		custBFFURL: "http://invalid.localhost:99999",
		brokerURL:  "http://invalid.localhost:99999",
		bankURL:    "http://invalid.localhost:99999",
		forgejoURL: "http://invalid.localhost:99999",
		httpClient: &http.Client{Timeout: 100 * time.Millisecond},
	}

	ctx := context.Background()
	registry := NewRegistry()
	alice, ok := registry.Get("emp-human-alice")
	require.True(t, ok)

	action := alice.Actions[0]
	res, err := action.Execute(ctx, client)
	assert.Error(t, err, "action execution with invalid server should return an error")
	assert.Nil(t, res)
}

func TestGarfieldSleepInHallway_VetErrorHandling(t *testing.T) {
	// Create client with valid server for initial GET but broken server for vet hold call
	client, err := NewClient(&Config{Mode: "mock"})
	require.NoError(t, err)
	defer client.Close()

	// Mutate BFF URL to an invalid address to force vet call error when tripped
	client.bffURL = "http://127.0.0.1:1"

	registry := NewRegistry()
	garfield, ok := registry.Get("emp-feline-garfield")
	require.True(t, ok)

	var sleepAction *Action
	for _, act := range garfield.Actions {
		if act.Name == "garfield_sleep_in_hallway" {
			sleepAction = &act
			break
		}
	}
	require.NotNil(t, sleepAction)

	ctx := context.Background()

	var sawVetError bool
	for i := 0; i < 50; i++ {
		res, err := sleepAction.Execute(ctx, client)
		if err != nil {
			assert.Nil(t, res)
			assert.Contains(t, err.Error(), "failed to toggle vet medical hold")
			sawVetError = true
			break
		}
	}

	assert.True(t, sawVetError, "should have triggered vet medical hold error when tripped")
}

func TestBarnebySlapWaterGlass_ErrorHandling(t *testing.T) {
	client, err := NewClient(&Config{Mode: "mock"})
	require.NoError(t, err)
	defer client.Close()

	// Mutate serverURL to invalid address so ticket creation fails
	client.serverURL = "http://127.0.0.1:1"

	registry := NewRegistry()
	barneby, ok := registry.Get("emp-feline-barneby")
	require.True(t, ok)

	var slapAction *Action
	for _, act := range barneby.Actions {
		if act.Name == "barneby_slap_water_glass" {
			slapAction = &act
			break
		}
	}
	require.NotNil(t, slapAction)

	ctx := context.Background()
	res, err := slapAction.Execute(ctx, client)
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "failed to create maintenance ticket")
}

func TestAlicePublishOnboardingDoc_ErrorHandling(t *testing.T) {
	client, err := NewClient(&Config{Mode: "mock"})
	require.NoError(t, err)
	defer client.Close()

	client.serverURL = "http://127.0.0.1:1"

	registry := NewRegistry()
	alice, ok := registry.Get("emp-human-alice")
	require.True(t, ok)

	var docAction *Action
	for _, act := range alice.Actions {
		if act.Name == "alice_publish_onboarding_doc" {
			docAction = &act
			break
		}
	}
	require.NotNil(t, docAction)

	ctx := context.Background()
	res, err := docAction.Execute(ctx, client)
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "failed to create Nextcloud document")
}

func TestFrankPublishSecurityDoc_ErrorHandling(t *testing.T) {
	client, err := NewClient(&Config{Mode: "mock"})
	require.NoError(t, err)
	defer client.Close()

	client.serverURL = "http://127.0.0.1:1"

	registry := NewRegistry()
	frank, ok := registry.Get("emp-human-frank")
	require.True(t, ok)

	var secAction *Action
	for _, act := range frank.Actions {
		if act.Name == "frank_publish_security_doc" {
			secAction = &act
			break
		}
	}
	require.NotNil(t, secAction)

	ctx := context.Background()
	res, err := secAction.Execute(ctx, client)
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "failed to write security doc in Nextcloud")
}
