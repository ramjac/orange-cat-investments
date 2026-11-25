package main

import (
	"context"
	"time"

	"github.com/orange-cat-investments/oci/internal/repository/ops"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func toProtoAuditLog(rec *ops.AuditLog) *opsv1.AuditLogRecord {
	if rec == nil {
		return nil
	}
	created := rec.CanonicalTS
	if created == "" && !rec.CreatedAt.IsZero() {
		created = rec.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return &opsv1.AuditLogRecord{
		ID:           rec.ID,
		ActorID:      rec.ActorID,
		ActorRole:    rec.ActorRole,
		Action:       rec.Action,
		ResourceType: rec.ResourceType,
		ResourceID:   rec.ResourceID,
		MetadataJSON: rec.MetadataJSON,
		CreatedAt:    created,
		PreviousHash: rec.PreviousHash,
		RecordHash:   rec.RecordHash,
	}
}

func (s *OpsGRPCServer) AppendAuditLog(ctx context.Context, in *opsv1.AppendAuditLogRequest) (*opsv1.AppendAuditLogResponse, error) {
	if in == nil || in.ActorID == "" || in.Action == "" {
		return nil, status.Error(codes.InvalidArgument, "actor_id and action are required")
	}
	rec, err := s.svc.AppendAuditLog(ctx, &ops.AuditLog{
		ActorID:      in.ActorID,
		ActorRole:    in.ActorRole,
		Action:       in.Action,
		ResourceType: in.ResourceType,
		ResourceID:   in.ResourceID,
		MetadataJSON: in.MetadataJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append audit log failed: %v", err)
	}
	return &opsv1.AppendAuditLogResponse{Record: toProtoAuditLog(rec)}, nil
}

func (s *OpsGRPCServer) VerifyAuditChain(ctx context.Context, _ *opsv1.VerifyAuditChainRequest) (*opsv1.VerifyAuditChainResponse, error) {
	res, err := s.svc.VerifyAuditChain(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "verify audit chain failed: %v", err)
	}
	return &opsv1.VerifyAuditChainResponse{
		Valid:           res.Valid,
		VerifiedRecords: res.VerifiedRecords,
		BrokenAtID:      res.BrokenAtID,
		ExpectedHash:    res.ExpectedHash,
		ActualHash:      res.ActualHash,
		LatestHash:      res.LatestHash,
		Reason:          res.Reason,
	}, nil
}

func (s *OpsGRPCServer) ListAuditLogs(ctx context.Context, in *opsv1.ListAuditLogsRequest) (*opsv1.ListAuditLogsResponse, error) {
	var actorID, action string
	var limit, offset int32 = 50, 0
	if in != nil {
		actorID = in.ActorID
		action = in.Action
		if in.Limit > 0 {
			limit = in.Limit
		}
		offset = in.Offset
	}
	recs, err := s.svc.ListAuditLogs(ctx, actorID, action, limit, offset)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list audit logs failed: %v", err)
	}
	out := make([]*opsv1.AuditLogRecord, 0, len(recs))
	for _, rec := range recs {
		out = append(out, toProtoAuditLog(rec))
	}
	return &opsv1.ListAuditLogsResponse{Records: out}, nil
}
