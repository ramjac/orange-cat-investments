package worker

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/ThreeDotsLabs/watermill/message"
)

func TestTraceContextMiddleware(t *testing.T) {
	// Simple test to ensure it compiles and runs
	msg := message.NewMessage("123", []byte(`{"correlation_id":""}`))

	handler := TraceContextMiddleware(func(m *message.Message) ([]*message.Message, error) {
		return nil, nil
	})

	_, err := handler(msg)
	assert.NoError(t, err)
}
