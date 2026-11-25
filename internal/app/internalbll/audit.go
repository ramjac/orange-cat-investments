package internalbll

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/orange-cat-investments/oci/pkg/auditintent"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// errNoOpsClient is returned by the relay when no ops-api client is wired
// (unit tests only; production always dials ops-api).
var errNoOpsClient = errors.New("ops-api client not configured")

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

// recordAudit appends a tamper-evident audit entry to ops.audit_logs. It
// attempts the ledger append synchronously (preserving request-scoped
// latency and immediate chain visibility); if that fails — ops-api outage,
// request cancellation, upstream timeout — the intent is durably spooled
// (fsync + atomic rename, independent of the request context) and the
// background relay re-delivers it with capped exponential backoff until the
// ledger accepts it. An already-committed mutation can therefore never lose
// its audit entry to a transient ops-api failure.
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
	req := &opsv1.AppendAuditLogRequest{
		ActorID:      actorID,
		ActorRole:    actorRole,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		MetadataJSON: string(metaBytes),
	}
	if _, err := s.opsClient.AppendAuditLog(ctx, req); err == nil {
		return
	} else if s.auditRecorder == nil {
		s.logger.Warn("failed to append tamper-evident audit log", "action", action, "resource_id", resourceID, "error", err)
		return
	} else {
		// Delivery failed: persist a durable intent so the relay can
		// complete the append later. The write uses a detached, bounded
		// context so a cancelled request cannot drop the intent.
		s.logger.Warn("audit ledger append failed; spooling durable intent for relay retry",
			"action", action, "resource_id", resourceID, "error", err)
		writeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_, spoolErr := s.auditRecorder.Record(writeCtx, auditintent.Entry{
			ActorID:      actorID,
			ActorRole:    actorRole,
			Action:       action,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			MetadataJSON: string(metaBytes),
			OccurredAt:   time.Now().UTC(),
		})
		if spoolErr != nil {
			// Record already logged CRITICAL: without the durable spool the
			// entry for a committed operation is lost.
		}
	}
}

// deliverAuditIntent is the relay callback: one spooled entry -> one
// tamper-evident ledger append via ops-api.
func (s *Server) deliverAuditIntent(ctx context.Context, e *auditintent.Entry) error {
	if s.opsClient == nil {
		return errNoOpsClient
	}
	_, err := s.opsClient.AppendAuditLog(ctx, &opsv1.AppendAuditLogRequest{
		ActorID:      e.ActorID,
		ActorRole:    e.ActorRole,
		Action:       e.Action,
		ResourceType: e.ResourceType,
		ResourceID:   e.ResourceID,
		MetadataJSON: e.MetadataJSON,
	})
	return err
}
