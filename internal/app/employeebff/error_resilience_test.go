package employeebff

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type failingInternalBLLClient struct {
	internalbllv1.InternalBLLServiceClient
	err error
}

func (f *failingInternalBLLClient) ListAssets(ctx context.Context, in *facilitiesv1.ListAssetsRequest) (*facilitiesv1.ListAssetsResponse, error) {
	return nil, f.err
}

func (f *failingInternalBLLClient) AcknowledgeAlert(ctx context.Context, in *opsv1.AcknowledgeAlertRequest) (*opsv1.AcknowledgeAlertResponse, error) {
	return nil, f.err
}

func TestEmployeeBFF_LogsAndSanitizesUnavailableErrors(t *testing.T) {
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))

	rawErr := status.Error(codes.Unavailable, `connection error: desc = "transport: Error while dialing: dial tcp 10.152.183.189:50054: connect: connection refused"`)
	bff := NewEmployeeBFF(&failingInternalBLLClient{err: rawErr}, logger)
	mux := http.NewServeMux()
	bff.RegisterFacilitiesRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/facilities/assets", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.NotContains(t, rec.Body.String(), "10.152.183.189")
	assert.NotContains(t, rec.Body.String(), "dial tcp")
	assert.Contains(t, logBuf.String(), "internal-bll gRPC call failed")
	assert.Contains(t, logBuf.String(), "ListAssets")

	// Also verify ops PebbleAck returns 503 (not 400) when ops-api is Unavailable
	ackReq := httptest.NewRequest(http.MethodPost, "/ops/pebble/ack", bytes.NewBufferString(`{"ticket_id":"t-1","acknowledged_by":"emp-human-bob"}`))
	ackRec := httptest.NewRecorder()
	bff.OpsHandler().SubmitPebbleAck(ackRec, ackReq)

	assert.Equal(t, http.StatusServiceUnavailable, ackRec.Code)
	assert.NotContains(t, ackRec.Body.String(), "10.152.183.189")
	assert.NotContains(t, ackRec.Body.String(), "dial tcp")
}
