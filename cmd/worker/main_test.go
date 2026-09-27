package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
)

func TestHandleOTATriggered(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	event := EventEnvelope[OTATriggeredPayload]{
		EventID:       watermill.NewUUID(),
		EventType:     "facilities.ota_triggered.v1",
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "test-corr-id",
		Payload: OTATriggeredPayload{
			JobID:      "job-123",
			AssetID:    "asset-456",
			ReleaseID:  "release-789",
			DeviceType: "edge_camera",
			Version:    "2.1.0",
			FileURL:    "https://firmware.oci.local/v2.1.0.bin",
			Checksum:   "sha256checksum",
		},
	}

	data, err := json.Marshal(event)
	assert.NoError(t, err)

	msg := message.NewMessage(event.EventID, data)
	outputMsgs, err := HandleOTATriggered(logger, msg)

	assert.NoError(t, err)
	assert.Len(t, outputMsgs, 1)

	var completionEvent EventEnvelope[OTACompletedPayload]
	err = json.Unmarshal(outputMsgs[0].Payload, &completionEvent)
	assert.NoError(t, err)
	assert.Equal(t, "job-123", completionEvent.Payload.JobID)
	assert.Equal(t, "asset-456", completionEvent.Payload.AssetID)
	assert.Equal(t, "completed", completionEvent.Payload.Status)
}
