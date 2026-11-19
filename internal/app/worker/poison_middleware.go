package worker

import (
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
)

var (
	// ErrMalformedEnvelope indicates that an incoming message payload is not valid JSON.
	ErrMalformedEnvelope = errors.New("malformed event envelope JSON")
	// ErrMissingEventID indicates that an incoming envelope lacks a non-empty event_id.
	ErrMissingEventID = errors.New("envelope missing required field: event_id")
	// ErrMissingOccurredAt indicates that an incoming envelope lacks a valid occurred_at timestamp.
	ErrMissingOccurredAt = errors.New("envelope missing required field: occurred_at")
)

type envelopeHeader struct {
	EventID       string    `json:"event_id"`
	OccurredAt    time.Time `json:"occurred_at"`
	CorrelationID string    `json:"correlation_id"`
}

func extractEnvelopeMeta(payload []byte) (eventID, correlationID string, occurredAt time.Time, ok bool) {
	var hdr envelopeHeader
	if err := json.Unmarshal(payload, &hdr); err != nil {
		return "", "", time.Time{}, false
	}
	return strings.TrimSpace(hdr.EventID), strings.TrimSpace(hdr.CorrelationID), hdr.OccurredAt, true
}

// EnvelopeValidationMiddleware validates that every incoming message parses as a JSON
// EventEnvelope with non-empty EventID and OccurredAt fields. It is registered outside
// Retry and inside PoisonQueue so invalid messages bypass retries and route straight to
// the topic's dead-letter queue.
func EnvelopeValidationMiddleware() message.HandlerMiddleware {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			eventID, _, occurredAt, ok := extractEnvelopeMeta(msg.Payload)
			if !ok {
				return nil, ErrMalformedEnvelope
			}
			if eventID == "" {
				return nil, ErrMissingEventID
			}
			if occurredAt.IsZero() {
				return nil, ErrMissingOccurredAt
			}
			return h(msg)
		}
	}
}

// NewPoisonLoggingMiddleware logs a CWE-532-safe error entry whenever a message fails
// validation or exhausts retries and is about to be published to the dead-letter queue.
// Only event_id (when parseable), topic, correlation_id, and the error are logged —
// never raw message payload bytes.
func NewPoisonLoggingMiddleware(logger *slog.Logger, defaultTopic string) message.HandlerMiddleware {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			produced, err := h(msg)
			if err != nil && logger != nil {
				topic := message.SubscribeTopicFromCtx(msg.Context())
				if topic == "" {
					topic = defaultTopic
				}
				eventID, correlationID, _, _ := extractEnvelopeMeta(msg.Payload)
				attrs := []any{
					"topic", topic,
					"error", err.Error(),
				}
				if eventID != "" {
					attrs = append(attrs, "event_id", eventID)
				}
				if correlationID != "" {
					attrs = append(attrs, "correlation_id", correlationID)
				}
				logger.Error("routing message to dead-letter queue", attrs...)
			}
			return produced, err
		}
	}
}

// DefaultRetryConfig returns the standard exponential-backoff retry configuration
// (3 total attempts: 1 initial attempt + 2 retries).
func DefaultRetryConfig(wmLogger watermill.LoggerAdapter) middleware.Retry {
	return middleware.Retry{
		MaxRetries:      2,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     2 * time.Second,
		Multiplier:      2.0,
		Logger:          wmLogger,
	}
}

// AttachHandlerResilience declares <subTopic>.dlq when supported by ps and attaches
// PoisonQueue, CWE-532 poison logging, pre-retry envelope validation, and exponential
// backoff retry middlewares to h.
func AttachHandlerResilience(
	h *message.Handler,
	subTopic string,
	ps PubSub,
	retryCfg middleware.Retry,
	logger *slog.Logger,
) error {
	dlqTopic := subTopic + ".dlq"

	if init, ok := ps.(message.SubscribeInitializer); ok {
		if err := init.SubscribeInitialize(dlqTopic); err != nil && logger != nil {
			logger.Warn("failed to pre-declare dead-letter queue", "topic", dlqTopic, "error", err)
		}
	}

	poisonMW, err := middleware.PoisonQueue(ps, dlqTopic)
	if err != nil {
		return err
	}

	// Watermill executes handler middlewares in registration order (first = outermost):
	// PoisonQueue -> PoisonLogging -> EnvelopeValidation -> Retry -> Handler.
	// Validation errors bypass Retry and unwind directly into PoisonLogging + PoisonQueue.
	h.AddMiddleware(
		poisonMW,
		NewPoisonLoggingMiddleware(logger, subTopic),
		EnvelopeValidationMiddleware(),
		retryCfg.Middleware,
	)
	return nil
}
