package outbox

import (
	"context"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

// RecordRef uniquely identifies an outbox row across isolated domain schemas.
type RecordRef struct {
	ID     int64  `json:"id"`
	Schema Schema `json:"schema"`
}

// Source abstracts a schema-scoped or BLL-bridged outbox event batch provider.
type Source interface {
	Name() string
	ClaimOutbox(ctx context.Context, limit int32) ([]*Record, error)
	AckOutbox(ctx context.Context, published []RecordRef, failures []Failure) error
}

// ForwarderConfig tunes the outbox relay polling loop.
type ForwarderConfig struct {
	PollInterval time.Duration
	BatchSize    int32
	Logger       *slog.Logger
}

// Forwarder streams committed SQL outbox events to a Watermill message.Publisher
// with at-least-once delivery guarantees across broker outages.
type Forwarder struct {
	publisher    message.Publisher
	sources      []Source
	pollInterval time.Duration
	batchSize    int32
	logger       *slog.Logger
}

// NewForwarder creates a new Forwarder over one or more outbox sources.
func NewForwarder(pub message.Publisher, cfg ForwarderConfig, sources ...Source) *Forwarder {
	interval := cfg.PollInterval
	if interval <= 0 {
		interval = 500 * time.Millisecond
	}
	batch := cfg.BatchSize
	if batch <= 0 {
		batch = 50
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Forwarder{
		publisher:    pub,
		sources:      sources,
		pollInterval: interval,
		batchSize:    batch,
		logger:       logger,
	}
}

// RunOnce executes a single claim -> publish -> ack pass across all registered sources.
// Returns the number of successfully published events and the last error encountered (if any).
func (f *Forwarder) RunOnce(ctx context.Context) (int, error) {
	if f.publisher == nil {
		return 0, nil
	}
	totalPublished := 0
	var lastErr error

	for _, src := range f.sources {
		if src == nil {
			continue
		}
		records, err := src.ClaimOutbox(ctx, f.batchSize)
		if err != nil {
			f.logger.Warn("outbox claim failed", "source", src.Name(), "error", err)
			lastErr = err
			continue
		}
		if len(records) == 0 {
			continue
		}

		var published []RecordRef
		var failures []Failure

		for _, rec := range records {
			if rec == nil || rec.Topic == "" {
				continue
			}
			msgID := rec.EventID
			if msgID == "" {
				msgID = NewEventID()
			}
			msg := message.NewMessage(msgID, rec.Payload)
			msg.SetContext(ctx)
			msg.Metadata.Set("outbox_schema", string(rec.Schema))
			if rec.AggregateID != "" {
				msg.Metadata.Set("aggregate_id", rec.AggregateID)
			}

			if err := f.publisher.Publish(rec.Topic, msg); err != nil {
				f.logger.Warn("outbox publish failed; retaining event for retry",
					"source", src.Name(),
					"schema", string(rec.Schema),
					"topic", rec.Topic,
					"event_id", msgID,
					"error", err,
				)
				failures = append(failures, Failure{
					ID:     rec.ID,
					Schema: rec.Schema,
					Error:  err.Error(),
				})
				lastErr = err
				continue
			}
			published = append(published, RecordRef{ID: rec.ID, Schema: rec.Schema})
			totalPublished++
		}

		if len(published) > 0 || len(failures) > 0 {
			if err := src.AckOutbox(ctx, published, failures); err != nil {
				f.logger.Warn("outbox ack failed", "source", src.Name(), "error", err)
				lastErr = err
			}
		}
	}

	return totalPublished, lastErr
}

// Start runs the outbox polling loop until ctx is canceled.
func (f *Forwarder) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(f.pollInterval)
		defer ticker.Stop()

		f.logger.Info("transactional SQL outbox forwarder started", "poll_interval", f.pollInterval.String(), "sources", len(f.sources))
		for {
			select {
			case <-ctx.Done():
				f.logger.Info("transactional SQL outbox forwarder stopped")
				return
			case <-ticker.C:
				_, _ = f.RunOnce(ctx)
			}
		}
	}()
}
