package worker

import (
	"context"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	"github.com/orange-cat-investments/oci/pkg/outbox"
)

type customerBLLOutboxSource struct {
	client customerbllv1.CustomerBLLServiceClient
}

func (s *customerBLLOutboxSource) Name() string { return "customer-bll:core_invest" }

func (s *customerBLLOutboxSource) ClaimOutbox(ctx context.Context, limit int32) ([]*outbox.Record, error) {
	if s.client == nil {
		return nil, nil
	}
	res, err := s.client.ClaimOutboxEvents(ctx, &coreinvestv1.ClaimOutboxEventsRequest{
		Limit:   limit,
		LeaseMs: outbox.DefaultLeaseDuration.Milliseconds(),
	})
	if err != nil || res == nil {
		return nil, err
	}
	out := make([]*outbox.Record, 0, len(res.Events))
	for _, e := range res.Events {
		if e == nil {
			continue
		}
		ts, _ := time.Parse(time.RFC3339, e.CreatedAt)
		out = append(out, &outbox.Record{
			ID:          e.ID,
			Schema:      outbox.SchemaCoreInvest,
			EventID:     e.EventID,
			Topic:       e.Topic,
			AggregateID: e.AggregateID,
			Payload:     e.Payload,
			Attempts:    int(e.Attempts),
			CreatedAt:   ts,
		})
	}
	return out, nil
}

func (s *customerBLLOutboxSource) AckOutbox(ctx context.Context, published []outbox.RecordRef, failures []outbox.Failure) error {
	if s.client == nil {
		return nil
	}
	pubIDs := make([]int64, 0, len(published))
	for _, ref := range published {
		pubIDs = append(pubIDs, ref.ID)
	}
	protoFailures := make([]*coreinvestv1.OutboxFailure, 0, len(failures))
	for _, f := range failures {
		protoFailures = append(protoFailures, &coreinvestv1.OutboxFailure{
			ID:     f.ID,
			Schema: string(f.Schema),
			Error:  f.Error,
		})
	}
	_, err := s.client.AckOutboxEvents(ctx, &coreinvestv1.AckOutboxEventsRequest{
		PublishedIDs: pubIDs,
		Failures:     protoFailures,
	})
	return err
}

type internalBLLOutboxSource struct {
	client internalbllv1.InternalBLLServiceClient
}

func (s *internalBLLOutboxSource) Name() string {
	return "internal-bll:workforce+facilities+ops"
}

func (s *internalBLLOutboxSource) ClaimOutbox(ctx context.Context, limit int32) ([]*outbox.Record, error) {
	if s.client == nil {
		return nil, nil
	}
	res, err := s.client.ClaimOutboxEvents(ctx, &internalbllv1.ClaimBLLOutboxEventsRequest{
		Limit:   limit,
		LeaseMs: outbox.DefaultLeaseDuration.Milliseconds(),
	})
	if err != nil || res == nil {
		return nil, err
	}
	out := make([]*outbox.Record, 0, len(res.Events))
	for _, e := range res.Events {
		if e == nil {
			continue
		}
		ts, _ := time.Parse(time.RFC3339, e.CreatedAt)
		out = append(out, &outbox.Record{
			ID:          e.ID,
			Schema:      outbox.Schema(e.Schema),
			EventID:     e.EventID,
			Topic:       e.Topic,
			AggregateID: e.AggregateID,
			Payload:     e.Payload,
			Attempts:    int(e.Attempts),
			CreatedAt:   ts,
		})
	}
	return out, nil
}

func (s *internalBLLOutboxSource) AckOutbox(ctx context.Context, published []outbox.RecordRef, failures []outbox.Failure) error {
	if s.client == nil {
		return nil
	}
	pubItems := make([]*internalbllv1.BLLOutboxAckItem, 0, len(published))
	for _, ref := range published {
		pubItems = append(pubItems, &internalbllv1.BLLOutboxAckItem{
			ID:     ref.ID,
			Schema: string(ref.Schema),
		})
	}
	failItems := make([]*internalbllv1.BLLOutboxFailure, 0, len(failures))
	for _, f := range failures {
		failItems = append(failItems, &internalbllv1.BLLOutboxFailure{
			ID:     f.ID,
			Schema: string(f.Schema),
			Error:  f.Error,
		})
	}
	_, err := s.client.AckOutboxEvents(ctx, &internalbllv1.AckBLLOutboxEventsRequest{
		Published: pubItems,
		Failures:  failItems,
	})
	return err
}

// StartOutboxForwarder starts the background SQL outbox relay polling both BLL tiers
// and publishing committed domain events to RabbitMQ / Watermill.
func StartOutboxForwarder(
	ctx context.Context,
	pub message.Publisher,
	customerBLL customerbllv1.CustomerBLLServiceClient,
	internalBLL internalbllv1.InternalBLLServiceClient,
	logger *slog.Logger,
) *outbox.Forwarder {
	var sources []outbox.Source
	if customerBLL != nil {
		sources = append(sources, &customerBLLOutboxSource{client: customerBLL})
	}
	if internalBLL != nil {
		sources = append(sources, &internalBLLOutboxSource{client: internalBLL})
	}
	fwd := outbox.NewForwarder(pub, outbox.ForwarderConfig{
		PollInterval: 500 * time.Millisecond,
		BatchSize:    50,
		Logger:       logger,
	}, sources...)
	fwd.Start(ctx)
	return fwd
}
