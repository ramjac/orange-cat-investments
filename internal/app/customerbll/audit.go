package customerbll

import (
	"context"
	"encoding/json"

	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
)

// recordAudit appends a tamper-evident audit record to ops.audit_logs via ops-api.
// Callers must pass only non-PII or hashed identifiers for GDPR-erased subjects.
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
	}); err != nil {
		s.log().Warn("failed to append customer audit log", "action", action, "resource_id", resourceID, "error", err)
	}
}
