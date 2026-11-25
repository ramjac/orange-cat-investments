package customerbll

import (
	"context"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
)

func (s *Server) ClaimOutboxEvents(ctx context.Context, in *coreinvestv1.ClaimOutboxEventsRequest) (*coreinvestv1.ClaimOutboxEventsResponse, error) {
	if s.ciClient == nil {
		return &coreinvestv1.ClaimOutboxEventsResponse{}, nil
	}
	return s.ciClient.ClaimOutboxEvents(ctx, in)
}

func (s *Server) AckOutboxEvents(ctx context.Context, in *coreinvestv1.AckOutboxEventsRequest) (*coreinvestv1.AckOutboxEventsResponse, error) {
	if s.ciClient == nil {
		return &coreinvestv1.AckOutboxEventsResponse{}, nil
	}
	return s.ciClient.AckOutboxEvents(ctx, in)
}
