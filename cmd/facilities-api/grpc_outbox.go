package main

import (
	"context"
	"time"

	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	"github.com/orange-cat-investments/oci/pkg/outbox"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *FacilitiesGRPCServer) ClaimOutboxEvents(ctx context.Context, in *facilitiesv1.ClaimOutboxEventsRequest) (*facilitiesv1.ClaimOutboxEventsResponse, error) {
	if s.repo == nil {
		return &facilitiesv1.ClaimOutboxEventsResponse{}, nil
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
		return nil, status.Errorf(codes.Internal, "claim facilities outbox events failed: %v", err)
	}
	out := make([]*facilitiesv1.OutboxEvent, 0, len(recs))
	for _, r := range recs {
		out = append(out, &facilitiesv1.OutboxEvent{
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
	return &facilitiesv1.ClaimOutboxEventsResponse{Events: out}, nil
}

func (s *FacilitiesGRPCServer) AckOutboxEvents(ctx context.Context, in *facilitiesv1.AckOutboxEventsRequest) (*facilitiesv1.AckOutboxEventsResponse, error) {
	if s.repo == nil || in == nil {
		return &facilitiesv1.AckOutboxEventsResponse{}, nil
	}
	failures := make([]outbox.Failure, 0, len(in.Failures))
	for _, f := range in.Failures {
		if f != nil {
			failures = append(failures, outbox.Failure{
				ID:     f.ID,
				Schema: outbox.SchemaFacilities,
				Error:  f.Error,
			})
		}
	}
	acked, err := s.repo.AckOutboxEvents(ctx, in.PublishedIDs, failures)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ack facilities outbox events failed: %v", err)
	}
	return &facilitiesv1.AckOutboxEventsResponse{Acknowledged: int32(acked)}, nil
}
