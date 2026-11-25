package main

import (
	"context"
	"time"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	"github.com/orange-cat-investments/oci/pkg/outbox"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *CoreInvestGRPCServer) EnqueueOutboxEvent(ctx context.Context, in *coreinvestv1.EnqueueOutboxEventRequest) (*coreinvestv1.EnqueueOutboxEventResponse, error) {
	if s.repo == nil {
		return nil, status.Error(codes.Unavailable, "core-invest repository not configured")
	}
	if in == nil || in.Topic == "" {
		return nil, status.Error(codes.InvalidArgument, "topic is required")
	}
	evID, err := s.repo.EnqueueOutboxEvent(ctx, in.EventID, in.Topic, in.AggregateID, in.Payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "enqueue outbox event failed: %v", err)
	}
	return &coreinvestv1.EnqueueOutboxEventResponse{
		EventID: evID,
		Status:  "pending",
	}, nil
}

func (s *CoreInvestGRPCServer) ClaimOutboxEvents(ctx context.Context, in *coreinvestv1.ClaimOutboxEventsRequest) (*coreinvestv1.ClaimOutboxEventsResponse, error) {
	if s.repo == nil {
		return &coreinvestv1.ClaimOutboxEventsResponse{}, nil
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
		return nil, status.Errorf(codes.Internal, "claim outbox events failed: %v", err)
	}
	out := make([]*coreinvestv1.OutboxEvent, 0, len(recs))
	for _, r := range recs {
		out = append(out, &coreinvestv1.OutboxEvent{
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
	return &coreinvestv1.ClaimOutboxEventsResponse{Events: out}, nil
}

func (s *CoreInvestGRPCServer) AckOutboxEvents(ctx context.Context, in *coreinvestv1.AckOutboxEventsRequest) (*coreinvestv1.AckOutboxEventsResponse, error) {
	if s.repo == nil || in == nil {
		return &coreinvestv1.AckOutboxEventsResponse{}, nil
	}
	failures := make([]outbox.Failure, 0, len(in.Failures))
	for _, f := range in.Failures {
		if f != nil {
			failures = append(failures, outbox.Failure{
				ID:     f.ID,
				Schema: outbox.SchemaCoreInvest,
				Error:  f.Error,
			})
		}
	}
	acked, err := s.repo.AckOutboxEvents(ctx, in.PublishedIDs, failures)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ack outbox events failed: %v", err)
	}
	return &coreinvestv1.AckOutboxEventsResponse{Acknowledged: int32(acked)}, nil
}
