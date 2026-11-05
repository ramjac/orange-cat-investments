package envelope

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
)

// PublisherWithTracing wraps a message.Publisher to automatically inject trace context
// into published messages.
type PublisherWithTracing struct {
	publisher message.Publisher
}

// NewPublisherWithTracing creates a wrapper that injects OpenTelemetry trace context.
func NewPublisherWithTracing(pub message.Publisher) message.Publisher {
	return &PublisherWithTracing{publisher: pub}
}

// Publish publishes the messages after injecting trace context from the given context
// if the message has one, or does its best. Since Watermill Publisher interface doesn't
// take a context, we only can inject if the context is available through msg.Context().
func (p *PublisherWithTracing) Publish(topic string, messages ...*message.Message) error {
	for _, msg := range messages {
		InjectTraceContext(msg.Context(), msg)
	}
	return p.publisher.Publish(topic, messages...)
}

// Close closes the underlying publisher.
func (p *PublisherWithTracing) Close() error {
	return p.publisher.Close()
}
