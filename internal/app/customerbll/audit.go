package customerbll

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/orange-cat-investments/oci/pkg/auditintent"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
)

// errNoOpsClient is returned by the relay when no ops-api client is wired
// (unit tests only; production always dials ops-api).
var errNoOpsClient = errors.New("ops-api client not configured")

// recordAudit appends a tamper-evident audit record to ops.audit_logs via ops-api.
// Callers must pass only non-PII or hashed identifiers for GDPR-erased subjects.
//
// The ledger append is attempted synchronously; when it fails (ops-api
// outage, request cancellation, upstream timeout) the intent is durably
// spooled (fsync + atomic rename, independent of the request context) and
// the background relay re-delivers it with capped exponential backoff until
// the ledger accepts it, so committed customer operations and policy-denial
// records can never be lost to a transient ops-api failure.
func (s *Server) recordAudit(ctx context.Context, actorID, actorRole, action, resourceType, resourceID string, meta map[string]any) {
	if s.opsClient == nil {
		return
	}
	if actorID == "" {
		actorID = "system-customer-bll"
	}
	if actorRole == "" {
		actorRole = "customer"
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
	}); err == nil {
		return
	} else if s.auditRecorder == nil {
		s.log().Warn("failed to append customer audit log", "action", action, "resource_id", resourceID, "error", err)
		return
	} else {
		// Delivery failed: persist a durable intent so the relay can
		// complete the append later. The write uses a detached, bounded
		// context so a cancelled request cannot drop the intent.
		s.log().Warn("audit ledger append failed; spooling durable intent for relay retry",
			"action", action, "resource_id", resourceID, "error", err)
		writeCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		// Record already logged CRITICAL if the durable spool fails.
		_, _ = s.auditRecorder.Record(writeCtx, auditintent.Entry{
			ActorID:      actorID,
			ActorRole:    actorRole,
			Action:       action,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			MetadataJSON: string(metaBytes),
			OccurredAt:   time.Now().UTC(),
		})
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
