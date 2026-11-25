package ops

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type auditMockBLL struct {
	internalbllv1.InternalBLLServiceClient
	verifyResp *opsv1.VerifyAuditChainResponse
	listResp   *opsv1.ListAuditLogsResponse
}

func (m *auditMockBLL) VerifyAuditChain(_ context.Context, _ *opsv1.VerifyAuditChainRequest) (*opsv1.VerifyAuditChainResponse, error) {
	return m.verifyResp, nil
}

func (m *auditMockBLL) ListAuditLogs(_ context.Context, _ *opsv1.ListAuditLogsRequest) (*opsv1.ListAuditLogsResponse, error) {
	return m.listResp, nil
}

func TestOpsHandler_AuditLogVerificationAndListing(t *testing.T) {
	t.Run("valid chain returns 200 OK with verified count", func(t *testing.T) {
		h := NewHandler(&auditMockBLL{
			verifyResp: &opsv1.VerifyAuditChainResponse{
				Valid:           true,
				VerifiedRecords: 12,
				LatestHash:      "abcdef123456",
			},
		})
		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/audit-logs/verify", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var body opsv1.VerifyAuditChainResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.True(t, body.Valid)
		assert.Equal(t, int64(12), body.VerifiedRecords)
		assert.Equal(t, "abcdef123456", body.LatestHash)
	})

	t.Run("tampered chain returns 409 Conflict with broken_at_id", func(t *testing.T) {
		h := NewHandler(&auditMockBLL{
			verifyResp: &opsv1.VerifyAuditChainResponse{
				Valid:           false,
				VerifiedRecords: 3,
				BrokenAtID:      4,
				ExpectedHash:    "aaaa",
				ActualHash:      "bbbb",
				Reason:          "record_hash mismatch (tampering detected) at audit record id=4",
			},
		})
		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/ops/audit-logs/verify", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusConflict, rec.Code)
		var body opsv1.VerifyAuditChainResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.False(t, body.Valid)
		assert.Equal(t, int64(4), body.BrokenAtID)
	})

	t.Run("list audit logs returns JSON array", func(t *testing.T) {
		h := NewHandler(&auditMockBLL{
			listResp: &opsv1.ListAuditLogsResponse{
				Records: []*opsv1.AuditLogRecord{
					{ID: 1, ActorID: "emp-human-elena", Action: "workforce.medical_hold.updated"},
				},
			},
		})
		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/audit-logs?actor_id=emp-human-elena&limit=10", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		var body map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, float64(1), body["total"])
	})
}
