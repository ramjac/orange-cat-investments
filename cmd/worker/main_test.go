package main

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
)

func TestHandleCatSpotted_DoesNotLogRawPayload(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, nil))

	secretPayloadStr := `{"secret_key":"TOP_SECRET_PAYLOAD_DATA","payload":{"camera_id":"CAM-1","feline_id":"garfield","activity_type":"zooming","confidence_score":0.99}}`
	msg := message.NewMessage("test-msg-uuid-123", []byte(secretPayloadStr))

	outMsgs, err := handleCatSpotted(logger, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(outMsgs) != 1 {
		t.Fatalf("expected 1 output message, got %d", len(outMsgs))
	}

	logs := buf.String()
	if strings.Contains(logs, "TOP_SECRET_PAYLOAD_DATA") {
		t.Errorf("found raw sensitive payload in log output: %s", logs)
	}
	if !strings.Contains(logs, "test-msg-uuid-123") {
		t.Errorf("expected message UUID in log output, got: %s", logs)
	}
}

func TestHandleBacktestRun_DoesNotLogRawPayload(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := slog.New(slog.NewJSONHandler(buf, nil))

	secretPayloadStr := `{"secret_strategy":"CLASSIFIED_QUANT_MODEL"}`
	msg := message.NewMessage("test-msg-uuid-456", []byte(secretPayloadStr))

	outMsgs, err := handleBacktestRun(logger, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(outMsgs) != 1 {
		t.Fatalf("expected 1 output message, got %d", len(outMsgs))
	}

	logs := buf.String()
	if strings.Contains(logs, "CLASSIFIED_QUANT_MODEL") {
		t.Errorf("found raw sensitive payload in log output: %s", logs)
	}
	if !strings.Contains(logs, "test-msg-uuid-456") {
		t.Errorf("expected message UUID in log output, got: %s", logs)
	}
}
