package ops

import (
	"encoding/json"
	"net/http"
	"strconv"

	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
)

// VerifyAuditChain walks the cryptographic SHA-256 hash chain in ops.audit_logs via internal-bll -> ops-api.
func (h *Handler) VerifyAuditChain(w http.ResponseWriter, r *http.Request) {
	if h.bllClient == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
		return
	}

	resp, err := h.bllClient.VerifyAuditChain(r.Context(), &opsv1.VerifyAuditChainRequest{})
	if err != nil {
		writeOpsGRPCError(w, "VerifyAuditChain", err, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if !resp.Valid {
		w.WriteHeader(http.StatusConflict)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// ListAuditLogs retrieves paginated tamper-evident audit entries filtered optionally by actor_id or action.
func (h *Handler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	if h.bllClient == nil {
		http.Error(w, "ops service unavailable", http.StatusServiceUnavailable)
		return
	}

	limit := int32(50)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = int32(v)
		}
	}
	offset := int32(0)
	if raw := r.URL.Query().Get("offset"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v >= 0 {
			offset = int32(v)
		}
	}

	resp, err := h.bllClient.ListAuditLogs(r.Context(), &opsv1.ListAuditLogsRequest{
		ActorID: r.URL.Query().Get("actor_id"),
		Action:  r.URL.Query().Get("action"),
		Limit:   limit,
		Offset:  offset,
	})
	if err != nil {
		writeOpsGRPCError(w, "ListAuditLogs", err, http.StatusInternalServerError)
		return
	}

	records := resp.Records
	if records == nil {
		records = []*opsv1.AuditLogRecord{}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"records": records,
		"total":   len(records),
	})
}
