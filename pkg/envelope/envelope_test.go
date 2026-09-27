package envelope_test

import (
	"encoding/json"
	"testing"

	"github.com/orange-cat-investments/oci/pkg/envelope"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type SamplePayload struct {
	DeviceID string  `json:"device_id"`
	Reading  float64 `json:"reading"`
}

func TestEventEnvelope(t *testing.T) {
	payload := SamplePayload{
		DeviceID: "sensor-001",
		Reading:  42.5,
	}

	env := envelope.NewEventEnvelope("evt-123", "facilities.sensor_read.v1", "corr-999", payload)

	assert.Equal(t, "evt-123", env.EventID)
	assert.Equal(t, "facilities.sensor_read.v1", env.EventType)
	assert.Equal(t, "corr-999", env.CorrelationID)
	assert.Equal(t, payload, env.Payload)
	assert.False(t, env.OccurredAt.IsZero())

	data, err := json.Marshal(env)
	require.NoError(t, err)

	var unmarshaled envelope.EventEnvelope[SamplePayload]
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, env.EventID, unmarshaled.EventID)
	assert.Equal(t, env.EventType, unmarshaled.EventType)
	assert.Equal(t, env.CorrelationID, unmarshaled.CorrelationID)
	assert.Equal(t, env.Payload, unmarshaled.Payload)
	assert.True(t, env.OccurredAt.Equal(unmarshaled.OccurredAt))
}
