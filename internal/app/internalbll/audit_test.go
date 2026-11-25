package internalbll

import (
	"context"
	"testing"
	"time"

	opsrepo "github.com/orange-cat-investments/oci/internal/repository/ops"
	"github.com/orange-cat-investments/oci/pkg/auditintent"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type auditOpsStub struct {
	opsv1.OpsServiceClient
	repo opsrepo.Repository
}

func (a *auditOpsStub) AppendAuditLog(ctx context.Context, in *opsv1.AppendAuditLogRequest) (*opsv1.AppendAuditLogResponse, error) {
	rec, err := a.repo.AppendAuditLog(ctx, &opsrepo.AuditLog{
		ActorID:      in.ActorID,
		ActorRole:    in.ActorRole,
		Action:       in.Action,
		ResourceType: in.ResourceType,
		ResourceID:   in.ResourceID,
		MetadataJSON: in.MetadataJSON,
	})
	if err != nil {
		return nil, err
	}
	return &opsv1.AppendAuditLogResponse{
		Record: &opsv1.AuditLogRecord{
			ID:           rec.ID,
			ActorID:      rec.ActorID,
			ActorRole:    rec.ActorRole,
			Action:       rec.Action,
			ResourceType: rec.ResourceType,
			ResourceID:   rec.ResourceID,
			MetadataJSON: rec.MetadataJSON,
			PreviousHash: rec.PreviousHash,
			RecordHash:   rec.RecordHash,
		},
	}, nil
}

func (a *auditOpsStub) VerifyAuditChain(ctx context.Context, _ *opsv1.VerifyAuditChainRequest) (*opsv1.VerifyAuditChainResponse, error) {
	res, err := a.repo.VerifyAuditChain(ctx)
	if err != nil {
		return nil, err
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

type auditWFStub struct {
	workforcev1.WorkforceServiceClient
}

func (w *auditWFStub) SetMedicalHold(_ context.Context, in *workforcev1.SetMedicalHoldRequest) (*workforcev1.SetMedicalHoldResponse, error) {
	return &workforcev1.SetMedicalHoldResponse{
		CareSchedule: &workforcev1.CareSchedule{
			FelineID:             in.FelineID,
			EmergencyMedicalHold: in.EmergencyMedicalHold,
		},
	}, nil
}

func (w *auditWFStub) GetLeaveRequest(_ context.Context, in *workforcev1.GetLeaveRequestRequest) (*workforcev1.GetLeaveRequestResponse, error) {
	return &workforcev1.GetLeaveRequestResponse{
		LeaveRequest: &workforcev1.LeaveRequest{
			LeaveID:    in.LeaveID,
			EmployeeID: "emp-human-rick",
		},
	}, nil
}

func (w *auditWFStub) UpdateLeaveStatus(_ context.Context, in *workforcev1.UpdateLeaveStatusRequest) (*workforcev1.UpdateLeaveStatusResponse, error) {
	return &workforcev1.UpdateLeaveStatusResponse{
		LeaveRequest: &workforcev1.LeaveRequest{
			LeaveID: in.LeaveID,
			Status:  in.Status,
		},
	}, nil
}

func (w *auditWFStub) GetEmployee(_ context.Context, in *workforcev1.GetEmployeeRequest) (*workforcev1.GetEmployeeResponse, error) {
	return &workforcev1.GetEmployeeResponse{
		Employee: &workforcev1.Employee{
			EmployeeID:   in.EmployeeID,
			EmployeeType: "human",
			FirstName:    "Test",
			Status:       "active",
		},
	}, nil
}

func (w *auditWFStub) UpdateEmployeeStatus(_ context.Context, in *workforcev1.UpdateEmployeeStatusRequest) (*workforcev1.UpdateEmployeeStatusResponse, error) {
	return &workforcev1.UpdateEmployeeStatusResponse{
		Employee: &workforcev1.Employee{
			EmployeeID: in.EmployeeID,
			Status:     in.Status,
		},
	}, nil
}

type auditCIStub struct {
	coreinvestv1.CoreInvestServiceClient
}

func (c *auditCIStub) CreateOrder(_ context.Context, in *coreinvestv1.CreateOrderRequest) (*coreinvestv1.CreateOrderResponse, error) {
	return &coreinvestv1.CreateOrderResponse{
		Order: &coreinvestv1.BrokerageOrder{
			OrderID:     "ord-audit-1",
			PortfolioID: in.PortfolioID,
			Symbol:      in.Symbol,
			Side:        in.Side,
			Quantity:    in.Quantity,
			Status:      "filled",
		},
	}, nil
}

func TestInternalBLL_EmitsTamperEvidentAuditChain(t *testing.T) {
	repo := opsrepo.NewMockRepository()
	srv := NewServer(&auditWFStub{}, nil, &auditOpsStub{repo: repo}, nil, nil, Config{
		CoreInvestClient: &auditCIStub{},
	})

	vetCtx := grpcutil.WithIncomingActor(context.Background(), grpcutil.ActorContext{
		ActorID: "emp-human-elena",
		Role:    "veterinary",
	})
	_, err := srv.SetMedicalHold(vetCtx, &workforcev1.SetMedicalHoldRequest{
		FelineID:             "emp-feline-garfield",
		EmergencyMedicalHold: true,
	})
	require.NoError(t, err)

	hrCtx := grpcutil.WithIncomingActor(context.Background(), grpcutil.ActorContext{
		ActorID: "emp-human-alice",
		Role:    "hr",
	})
	_, err = srv.UpdateLeaveStatus(hrCtx, &workforcev1.UpdateLeaveStatusRequest{
		LeaveID: "leave-001",
		Status:  "approved",
	})
	require.NoError(t, err)

	_, err = srv.OffboardEmployee(hrCtx, &internalbllv1.OffboardEmployeeSagaRequest{
		EmployeeID: "emp-temp-01",
		Reason:     "contract complete",
	})
	require.NoError(t, err)

	quantCtx := grpcutil.WithIncomingActor(context.Background(), grpcutil.ActorContext{
		ActorID: "emp-human-carol",
		Role:    "investment",
	})
	_, err = srv.ExecuteBrokerageOrder(quantCtx, &coreinvestv1.CreateOrderRequest{
		PortfolioID: "port-alpha",
		Symbol:      "NVDA",
		Side:        "buy",
		Quantity:    10,
		OrderType:   "market",
	})
	require.NoError(t, err)

	verifyResp, err := srv.VerifyAuditChain(context.Background(), &opsv1.VerifyAuditChainRequest{})
	require.NoError(t, err)
	assert.True(t, verifyResp.Valid)
	assert.Equal(t, int64(4), verifyResp.VerifiedRecords)
	assert.NotEmpty(t, verifyResp.LatestHash)
}

// flakyOpsStub fails AppendAuditLog until released, then succeeds and
// records every accepted entry.
type flakyOpsStub struct {
	opsv1.OpsServiceClient
	repo    opsrepo.Repository
	failing *bool
}

func (f *flakyOpsStub) AppendAuditLog(ctx context.Context, in *opsv1.AppendAuditLogRequest) (*opsv1.AppendAuditLogResponse, error) {
	if f.failing != nil && *f.failing {
		return nil, status.Errorf(codes.Unavailable, "simulated ops-api outage")
	}
	rec, err := f.repo.AppendAuditLog(ctx, &opsrepo.AuditLog{
		ActorID:      in.ActorID,
		ActorRole:    in.ActorRole,
		Action:       in.Action,
		ResourceType: in.ResourceType,
		ResourceID:   in.ResourceID,
		MetadataJSON: in.MetadataJSON,
	})
	if err != nil {
		return nil, err
	}
	return &opsv1.AppendAuditLogResponse{
		Record: &opsv1.AuditLogRecord{ID: rec.ID, Action: rec.Action},
	}, nil
}

func TestInternalBLL_AuditIntentSurvivesOpsAPIOutage(t *testing.T) {
	repo := opsrepo.NewMockRepository()
	failing := true
	srv := NewServer(&auditWFStub{}, nil, &flakyOpsStub{repo: repo, failing: &failing}, nil, nil, Config{
		AuditSpoolDir: t.TempDir(),
	})
	defer srv.Close()
	require.NotNil(t, srv.auditRecorder, "durable audit spool must be active when ops client is configured")

	// Sensitive mutation while the audit ledger is unreachable: the
	// operation itself must still succeed (it is already committed upstream).
	vetCtx := grpcutil.WithIncomingActor(context.Background(), grpcutil.ActorContext{
		ActorID: "emp-human-elena", Role: "veterinary",
	})
	_, err := srv.SetMedicalHold(vetCtx, &workforcev1.SetMedicalHoldRequest{
		FelineID: "emp-feline-garfield", EmergencyMedicalHold: true,
	})
	require.NoError(t, err)

	// The audit intent must be durably spooled (not lost with the RPC error).
	require.Eventually(t, func() bool {
		n, err := srv.auditRecorder.Spool().(*auditintent.FileSpool).PendingCount()
		return err == nil && n == 1
	}, 2*time.Second, 10*time.Millisecond)

	// Ledger recovers: the relay must deliver the spooled intent.
	failing = false
	require.Eventually(t, func() bool {
		res, err := repo.VerifyAuditChain(context.Background())
		return err == nil && res.VerifiedRecords == 1
	}, 5*time.Second, 50*time.Millisecond)

	n, err := srv.auditRecorder.Spool().(*auditintent.FileSpool).PendingCount()
	require.NoError(t, err)
	require.Zero(t, n, "spooled intent must be acknowledged after relay delivery")
}
