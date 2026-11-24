package outbox

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type memorySource struct {
	name  string
	store *MemoryStore
	lease time.Duration
}

func (s *memorySource) Name() string { return s.name }

func (s *memorySource) ClaimOutbox(_ context.Context, limit int32) ([]*Record, error) {
	return s.store.ClaimPending(limit, s.lease), nil
}

func (s *memorySource) AckOutbox(_ context.Context, published []RecordRef, failures []Failure) error {
	ids := make([]int64, 0, len(published))
	for _, ref := range published {
		ids = append(ids, ref.ID)
	}
	s.store.MarkPublished(ids)
	for _, f := range failures {
		s.store.MarkFailed(f.ID, f.Error, 0)
	}
	return nil
}

type flakyPublisher struct {
	mu       sync.Mutex
	failNext atomic.Bool
	sent     map[string][]*message.Message
}

func newFlakyPublisher() *flakyPublisher {
	return &flakyPublisher{sent: make(map[string][]*message.Message)}
}

func (p *flakyPublisher) Publish(topic string, messages ...*message.Message) error {
	if p.failNext.Load() {
		return errors.New("simulated rabbitmq connection refused")
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sent[topic] = append(p.sent[topic], messages...)
	return nil
}

func (p *flakyPublisher) Close() error { return nil }

func TestForwarder_SurvivesRabbitMQOutageAndPublishesOnRecovery(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore(SchemaCoreInvest)

	eventID, envBytes, err := BuildEnvelopeBytes(ctx, "", "events.investment.order_placed.v1", "ord-123", map[string]any{
		"order_id": "ord-123",
		"symbol":   "NVDA",
		"quantity": 10.0,
	})
	require.NoError(t, err)
	store.Enqueue(eventID, "events.investment.order_placed.v1", "ord-123", envBytes)

	pub := newFlakyPublisher()
	pub.failNext.Store(true)

	src := &memorySource{name: "core_invest", store: store, lease: time.Second}
	fwd := NewForwarder(pub, ForwarderConfig{PollInterval: 10 * time.Millisecond, BatchSize: 10}, src)

	// 1. Outage pass: publish fails, event remains pending in outbox with error recorded.
	published, err := fwd.RunOnce(ctx)
	require.Error(t, err)
	assert.Equal(t, 0, published)

	snap := store.Snapshot()
	require.Len(t, snap, 1)
	assert.Equal(t, "pending", snap[0].Status)
	assert.Nil(t, snap[0].PublishedAt)
	assert.Equal(t, 1, snap[0].Attempts)
	assert.Contains(t, snap[0].LastError, "simulated rabbitmq connection refused")

	// 2. RabbitMQ recovers: next pass delivers the event and marks it published.
	pub.failNext.Store(false)
	published, err = fwd.RunOnce(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, published)

	snap = store.Snapshot()
	require.Len(t, snap, 1)
	assert.Equal(t, "published", snap[0].Status)
	assert.NotNil(t, snap[0].PublishedAt)

	pub.mu.Lock()
	msgs := pub.sent["events.investment.order_placed.v1"]
	pub.mu.Unlock()
	require.Len(t, msgs, 1)
	assert.Equal(t, eventID, msgs[0].UUID)
	assert.JSONEq(t, string(envBytes), string(msgs[0].Payload))

	// 3. Subsequent pass is a no-op (no duplicate delivery).
	published, err = fwd.RunOnce(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, published)
}

func TestMemoryStore_SkipLockedConcurrentClaims(t *testing.T) {
	store := NewMemoryStore(SchemaWorkforce)
	for i := 0; i < 10; i++ {
		store.Enqueue("", "events.workforce.incident_created.v1", "inc", []byte(`{"ok":true}`))
	}

	var wg sync.WaitGroup
	claimedByWorker := make([][]*Record, 4)
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			claimedByWorker[idx] = store.ClaimPending(5, 5*time.Second)
		}(w)
	}
	wg.Wait()

	seen := make(map[int64]bool)
	for _, batch := range claimedByWorker {
		for _, rec := range batch {
			assert.False(t, seen[rec.ID], "record %d claimed concurrently by multiple workers", rec.ID)
			seen[rec.ID] = true
		}
	}
	assert.Len(t, seen, 10)
}

func TestValidateSchema(t *testing.T) {
	assert.NoError(t, ValidateSchema(SchemaWorkforce))
	assert.NoError(t, ValidateSchema(SchemaFacilities))
	assert.NoError(t, ValidateSchema(SchemaCoreInvest))
	assert.NoError(t, ValidateSchema(SchemaOps))
	assert.Error(t, ValidateSchema("public; DROP TABLE users"))
}
