package customerbll

import (
	"context"
	"testing"

	opsrepo "github.com/orange-cat-investments/oci/internal/repository/ops"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type opaTestWFClient struct {
	workforcev1.WorkforceServiceClient
	medicalHold bool
}

func (w *opaTestWFClient) GetCareSchedule(_ context.Context, in *workforcev1.GetCareScheduleRequest) (*workforcev1.GetCareScheduleResponse, error) {
	return &workforcev1.GetCareScheduleResponse{
		CareSchedule: &workforcev1.CareSchedule{
			FelineID:             in.FelineID,
			EmergencyMedicalHold: w.medicalHold,
		},
	}, nil
}

type opaTestCIClient struct {
	coreinvestv1.CoreInvestServiceClient
	accountStatus string
}

func (c *opaTestCIClient) GetPortfolio(_ context.Context, in *coreinvestv1.GetPortfolioRequest) (*coreinvestv1.GetPortfolioResponse, error) {
	return &coreinvestv1.GetPortfolioResponse{
		Portfolio: &coreinvestv1.Portfolio{
			PortfolioID: in.PortfolioID,
			Status:      c.accountStatus,
		},
	}, nil
}

type opaTestOpsClient struct {
	opsv1.OpsServiceClient
	repo opsrepo.Repository
}

func (o *opaTestOpsClient) AppendAuditLog(ctx context.Context, in *opsv1.AppendAuditLogRequest) (*opsv1.AppendAuditLogResponse, error) {
	rec, err := o.repo.AppendAuditLog(ctx, &opsrepo.AuditLog{
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

func TestCustomerBLL_OPATradingPolicyAndAuditDenials(t *testing.T) {
	opsRepo := opsrepo.NewMockRepository()
	s := NewServer(Config{DefaultFelineID: "emp-feline-garfield"})
	s.SetOpsClient(&opaTestOpsClient{repo: opsRepo})

	t.Run("closed account denied by OPA trading policy with PermissionDenied and audited", func(t *testing.T) {
		s.SetWorkforceClient(&opaTestWFClient{medicalHold: false})
		s.SetCoreInvestClient(&opaTestCIClient{accountStatus: "closed"})

		_, err := s.ValidateAndExecuteOrder(context.Background(), &customerbllv1.ValidateAndExecuteOrderRequest{
			CustomerID:  "cust-closed-01",
			PortfolioID: "port-closed-01",
			Symbol:      "NVDA",
			Side:        "buy",
			Quantity:    5,
			OrderType:   "limit",
			LimitPrice:  100.0,
		})
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Contains(t, err.Error(), `account status is "closed"`)
	})

	t.Run("order exceeding single-order notional limit denied with PermissionDenied", func(t *testing.T) {
		s.SetWorkforceClient(&opaTestWFClient{medicalHold: false})
		s.SetCoreInvestClient(&opaTestCIClient{accountStatus: "active"})

		_, err := s.ValidateAndExecuteOrder(context.Background(), &customerbllv1.ValidateAndExecuteOrderRequest{
			CustomerID:  "cust-whale-01",
			PortfolioID: "port-whale-01",
			Symbol:      "NVDA",
			Side:        "buy",
			Quantity:    5000,
			OrderType:   "limit",
			LimitPrice:  100.0, // $500,000 > $250,000 cap
		})
		require.Error(t, err)
		assert.Equal(t, codes.PermissionDenied, status.Code(err))
		assert.Contains(t, err.Error(), "exceeds single-order policy limit")
	})

	t.Run("active medical hold denied with FailedPrecondition and audited", func(t *testing.T) {
		s.SetWorkforceClient(&opaTestWFClient{medicalHold: true})
		s.SetCoreInvestClient(&opaTestCIClient{accountStatus: "active"})

		_, err := s.ValidateAndExecuteOrder(context.Background(), &customerbllv1.ValidateAndExecuteOrderRequest{
			CustomerID:  "cust-active-chloe",
			PortfolioID: "port-chloe",
			Symbol:      "NVDA",
			Side:        "buy",
			Quantity:    5,
			OrderType:   "limit",
			LimitPrice:  100.0,
		})
		require.Error(t, err)
		assert.Equal(t, codes.FailedPrecondition, status.Code(err))
		assert.Contains(t, err.Error(), "emergency medical hold active")
	})

	denials, err := opsRepo.ListAuditLogs(context.Background(), "", "authz.policy.denied", 20, 0)
	require.NoError(t, err)
	assert.Len(t, denials, 3)

	verify, err := opsRepo.VerifyAuditChain(context.Background())
	require.NoError(t, err)
	assert.True(t, verify.Valid)
	assert.Equal(t, int64(3), verify.VerifiedRecords)
}
