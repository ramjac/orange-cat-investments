package internalbll

import (
	"context"
	"encoding/json"

	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) AppendAuditLog(ctx context.Context, in *opsv1.AppendAuditLogRequest) (*opsv1.AppendAuditLogResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	res, err := s.opsClient.AppendAuditLog(ctx, in)
	return res, s.logDomainError("ops-api", "AppendAuditLog", err)
}

func (s *Server) VerifyAuditChain(ctx context.Context, in *opsv1.VerifyAuditChainRequest) (*opsv1.VerifyAuditChainResponse, error) {
	if s.opsClient == nil {
		return nil, status.Error(codes.Unavailable, "ops client not configured")
	}
	res, err := s.opsClient.VerifyAuditChain(ctx, in)
	return res, s.logDomainError("ops-api", "VerifyAuditChain", err)
}

func (s *Server) ListAuditLogs(ctx context.Context, in *opsv1.ListAuditLogsRequest) (*opsv1.ListAuditLogsResponse, error) {
	if s.opsClient == nil {
		return &opsv1.ListAuditLogsResponse{Records: []*opsv1.AuditLogRecord{}}, nil
	}
	res, err := s.opsClient.ListAuditLogs(ctx, in)
	return res, s.logDomainError("ops-api", "ListAuditLogs", err)
}

// recordAudit appends a tamper-evident entry to ops.audit_logs via ops-api.
func (s *Server) recordAudit(ctx context.Context, action, resourceType, resourceID string, meta map[string]any) {
	if s.opsClient == nil {
		return
	}
	actorID := "system-internal-bll"
	actorRole := "system"
	if actor, ok := grpcutil.ActorFromIncomingContext(ctx); ok {
		if actor.ActorID != "" {
			actorID = actor.ActorID
		}
		if actor.Role != "" {
			actorRole = actor.Role
		}
	}
	metaBytes := []byte("{}")
	if len(meta) > 0 {
		if b, err := json.Marshal(meta); err == nil {
			metaBytes = b
		}
	}
	if _, err := s.opsClient.AppendAuditLog(ctx, &opsv1.AppendAuditLogRequest{
		ActorID:      actorID,
		ActorRole:    actorRole,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		MetadataJSON: string(metaBytes),
	}); err != nil && s.logger != nil {
		s.logger.Warn("failed to append tamper-evident audit log", "action", action, "resource_id", resourceID, "error", err)
	}
}
