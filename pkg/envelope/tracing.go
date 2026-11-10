package envelope

import (
	"context"

	"github.com/ThreeDotsLabs/watermill/message"
	"go.opentelemetry.io/otel"
)

type metadataCarrier message.Metadata

func (m metadataCarrier) Get(key string) string {
	return m[key]
}

func (m metadataCarrier) Set(key string, value string) {
	m[key] = value
}

func (m metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// InjectTraceContext injects tracing context into a message's metadata.
func InjectTraceContext(ctx context.Context, msg *message.Message) {
	if msg.Metadata == nil {
		msg.Metadata = make(message.Metadata)
	}
	propagator := otel.GetTextMapPropagator()
	propagator.Inject(ctx, metadataCarrier(msg.Metadata))
}

// ExtractTraceContext extracts tracing context from a message's metadata.
func ExtractTraceContext(ctx context.Context, msg *message.Message) context.Context {
	propagator := otel.GetTextMapPropagator()
	return propagator.Extract(ctx, metadataCarrier(msg.Metadata))
}
