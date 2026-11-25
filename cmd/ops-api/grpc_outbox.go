package main

import (
	"context"
	"time"

	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	"github.com/orange-cat-investments/oci/pkg/outbox"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *OpsGRPCServer) ClaimOutboxEvents(ctx context.Context, in *opsv1.ClaimOutboxEventsRequest) (*opsv1.ClaimOutboxEventsResponse, error) {
	if s.repo == nil {
		return &opsv1.ClaimOutboxEventsResponse{}, nil
	}
	limit := int32(50)
	lease := outbox.DefaultLeaseDuration
	if in != nil {
		if in.Limit > 0 {
			limit = in.Limit
		}
		if in.LeaseMs > 0 {
			lease = time.Duration(in.LeaseMs) * time.Millisecond
		}
	}
	recs, err := s.repo.ClaimOutboxEvents(ctx, limit, lease)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "claim ops outbox events failed: %v", err)
	}
	out := make([]*opsv1.OutboxEvent, 0, len(recs))
	for _, r := range recs {
		out = append(out, &opsv1.OutboxEvent{
			ID:          r.ID,
			Schema:      string(r.Schema),
			EventID:     r.EventID,
			Topic:       r.Topic,
			AggregateID: r.AggregateID,
			Payload:     r.Payload,
			Attempts:    int32(r.Attempts),
			CreatedAt:   r.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	return &opsv1.ClaimOutboxEventsResponse{Events: out}, nil
}

func (s *OpsGRPCServer) AckOutboxEvents(ctx context.Context, in *opsv1.AckOutboxEventsRequest) (*opsv1.AckOutboxEventsResponse, error) {
	if s.repo == nil || in == nil {
		return &opsv1.AckOutboxEventsResponse{}, nil
	}
	failures := make([]outbox.Failure, 0, len(in.Failures))
	for _, f := range in.Failures {
		if f != nil {
			failures = append(failures, outbox.Failure{
				ID:     f.ID,
				Schema: outbox.SchemaOps,
				Error:  f.Error,
			})
		}
	}
	acked, err := s.repo.AckOutboxEvents(ctx, in.PublishedIDs, failures)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ack ops outbox events failed: %v", err)
	}
	return &opsv1.AckOutboxEventsResponse{Acknowledged: int32(acked)}, nil
}
