package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInitProvider(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdown, err := InitProvider(ctx, "test-service")
	assert.NoError(t, err)
	assert.NotNil(t, shutdown)

	err = shutdown(ctx)
	assert.NoError(t, err)
}
