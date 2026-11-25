package internalbll

import (
	"context"

	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/outbox"
)

func (s *Server) ClaimOutboxEvents(ctx context.Context, in *internalbllv1.ClaimBLLOutboxEventsRequest) (*internalbllv1.ClaimBLLOutboxEventsResponse, error) {
	limit := int32(50)
	var leaseMs int64
	if in != nil {
		if in.Limit > 0 {
			limit = in.Limit
		}
		leaseMs = in.LeaseMs
	}

	var all []*internalbllv1.BLLOutboxEvent

	if s.wfClient != nil {
		if res, err := s.wfClient.ClaimOutboxEvents(ctx, &workforcev1.ClaimOutboxEventsRequest{Limit: limit, LeaseMs: leaseMs}); err == nil && res != nil {
			for _, e := range res.Events {
				if e == nil {
					continue
				}
				all = append(all, &internalbllv1.BLLOutboxEvent{
					ID:          e.ID,
					Schema:      string(outbox.SchemaWorkforce),
					EventID:     e.EventID,
					Topic:       e.Topic,
					AggregateID: e.AggregateID,
					Payload:     e.Payload,
					Attempts:    e.Attempts,
					CreatedAt:   e.CreatedAt,
				})
			}
		}
	}

	if s.facClient != nil {
		if res, err := s.facClient.ClaimOutboxEvents(ctx, &facilitiesv1.ClaimOutboxEventsRequest{Limit: limit, LeaseMs: leaseMs}); err == nil && res != nil {
			for _, e := range res.Events {
				if e == nil {
					continue
				}
				all = append(all, &internalbllv1.BLLOutboxEvent{
					ID:          e.ID,
					Schema:      string(outbox.SchemaFacilities),
					EventID:     e.EventID,
					Topic:       e.Topic,
					AggregateID: e.AggregateID,
					Payload:     e.Payload,
					Attempts:    e.Attempts,
					CreatedAt:   e.CreatedAt,
				})
			}
		}
	}

	if s.opsClient != nil {
		if res, err := s.opsClient.ClaimOutboxEvents(ctx, &opsv1.ClaimOutboxEventsRequest{Limit: limit, LeaseMs: leaseMs}); err == nil && res != nil {
			for _, e := range res.Events {
				if e == nil {
					continue
				}
				all = append(all, &internalbllv1.BLLOutboxEvent{
					ID:          e.ID,
					Schema:      string(outbox.SchemaOps),
					EventID:     e.EventID,
					Topic:       e.Topic,
					AggregateID: e.AggregateID,
					Payload:     e.Payload,
					Attempts:    e.Attempts,
					CreatedAt:   e.CreatedAt,
				})
			}
		}
	}

	return &internalbllv1.ClaimBLLOutboxEventsResponse{Events: all}, nil
}

func (s *Server) AckOutboxEvents(ctx context.Context, in *internalbllv1.AckBLLOutboxEventsRequest) (*internalbllv1.AckBLLOutboxEventsResponse, error) {
	if in == nil {
		return &internalbllv1.AckBLLOutboxEventsResponse{}, nil
	}

	var wfPub, facPub, opsPub []int64
	var wfFail []*workforcev1.OutboxFailure
	var facFail []*facilitiesv1.OutboxFailure
	var opsFail []*opsv1.OutboxFailure

	for _, p := range in.Published {
		if p == nil {
			continue
		}
		switch outbox.Schema(p.Schema) {
		case outbox.SchemaWorkforce:
			wfPub = append(wfPub, p.ID)
		case outbox.SchemaFacilities:
			facPub = append(facPub, p.ID)
		case outbox.SchemaOps:
			opsPub = append(opsPub, p.ID)
		}
	}
	for _, f := range in.Failures {
		if f == nil {
			continue
		}
		switch outbox.Schema(f.Schema) {
		case outbox.SchemaWorkforce:
			wfFail = append(wfFail, &workforcev1.OutboxFailure{ID: f.ID, Schema: f.Schema, Error: f.Error})
		case outbox.SchemaFacilities:
			facFail = append(facFail, &facilitiesv1.OutboxFailure{ID: f.ID, Schema: f.Schema, Error: f.Error})
		case outbox.SchemaOps:
			opsFail = append(opsFail, &opsv1.OutboxFailure{ID: f.ID, Schema: f.Schema, Error: f.Error})
		}
	}

	var total int32
	if s.wfClient != nil && (len(wfPub) > 0 || len(wfFail) > 0) {
		if res, err := s.wfClient.AckOutboxEvents(ctx, &workforcev1.AckOutboxEventsRequest{PublishedIDs: wfPub, Failures: wfFail}); err == nil && res != nil {
			total += res.Acknowledged
		}
	}
	if s.facClient != nil && (len(facPub) > 0 || len(facFail) > 0) {
		if res, err := s.facClient.AckOutboxEvents(ctx, &facilitiesv1.AckOutboxEventsRequest{PublishedIDs: facPub, Failures: facFail}); err == nil && res != nil {
			total += res.Acknowledged
		}
	}
	if s.opsClient != nil && (len(opsPub) > 0 || len(opsFail) > 0) {
		if res, err := s.opsClient.AckOutboxEvents(ctx, &opsv1.AckOutboxEventsRequest{PublishedIDs: opsPub, Failures: opsFail}); err == nil && res != nil {
			total += res.Acknowledged
		}
	}

	return &internalbllv1.AckBLLOutboxEventsResponse{Acknowledged: total}, nil
}
