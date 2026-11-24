package internalbll

import (
	"context"
	"testing"

	opsrepo "github.com/orange-cat-investments/oci/internal/repository/ops"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type authzWFStub struct {
	auditWFStub
	leaveOwner  string
	reviewOwner string
}

func (w *authzWFStub) GetLeaveRequest(_ context.Context, in *workforcev1.GetLeaveRequestRequest) (*workforcev1.GetLeaveRequestResponse, error) {
	return &workforcev1.GetLeaveRequestResponse{
		LeaveRequest: &workforcev1.LeaveRequest{
			LeaveID:    in.LeaveID,
			EmployeeID: w.leaveOwner,
		},
	}, nil
}

func (w *authzWFStub) GetReviewCycle(_ context.Context, in *workforcev1.GetReviewCycleRequest) (*workforcev1.GetReviewCycleResponse, error) {
	return &workforcev1.GetReviewCycleResponse{
		ReviewCycle: &workforcev1.ReviewCycle{
			ReviewID:   in.ReviewID,
			EmployeeID: w.reviewOwner,
		},
	}, nil
}

func (w *authzWFStub) SubmitVeterinaryAudit(_ context.Context, in *workforcev1.SubmitVeterinaryAuditRequest) (*workforcev1.SubmitVeterinaryAuditResponse, error) {
	return &workforcev1.SubmitVeterinaryAuditResponse{
		ReviewCycle: &workforcev1.ReviewCycle{ReviewID: in.ReviewID},
	}, nil
}

func (w *authzWFStub) SubmitManagerAssessment(_ context.Context, in *workforcev1.SubmitManagerAssessmentRequest) (*workforcev1.SubmitManagerAssessmentResponse, error) {
	return &workforcev1.SubmitManagerAssessmentResponse{
		ReviewCycle: &workforcev1.ReviewCycle{ReviewID: in.ReviewID},
	}, nil
}

func TestInternalBLL_OPAPolicyEnforcementAndAuditDenials(t *testing.T) {
	opsRepo := opsrepo.NewMockRepository()
	wfStub := &authzWFStub{leaveOwner: "emp-human-alice", reviewOwner: "emp-human-alice"}
	srv := NewServer(wfStub, nil, &auditOpsStub{repo: opsRepo}, nil, nil, Config{})

	// 1. Staff persona (Rick) attempting SetMedicalHold -> PermissionDenied + audit log
	staffCtx := grpcutil.WithIncomingActor(context.Background(), grpcutil.ActorContext{
		ActorID: "emp-human-rick",
		Role:    "staff",
	})
	_, err := srv.SetMedicalHold(staffCtx, &workforcev1.SetMedicalHoldRequest{
		FelineID:             "emp-feline-garfield",
		EmergencyMedicalHold: true,
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Contains(t, err.Error(), "toggle_medical_hold")

	// 2. Veterinary persona (Dr. Elena) allowed SetMedicalHold
	vetCtx := grpcutil.WithIncomingActor(context.Background(), grpcutil.ActorContext{
		ActorID: "emp-human-elena",
		Role:    "veterinary",
	})
	res, err := srv.SetMedicalHold(vetCtx, &workforcev1.SetMedicalHoldRequest{
		FelineID:             "emp-feline-garfield",
		EmergencyMedicalHold: true,
	})
	require.NoError(t, err)
	assert.True(t, res.CareSchedule.EmergencyMedicalHold)

	// 3. Engineering persona attempting OffboardEmployee -> PermissionDenied
	engCtx := grpcutil.WithIncomingActor(context.Background(), grpcutil.ActorContext{
		ActorID: "emp-human-elise",
		Role:    "engineering",
	})
	_, err = srv.OffboardEmployee(engCtx, &internalbllv1.OffboardEmployeeSagaRequest{
		EmployeeID: "emp-human-rick",
		Reason:     "unauthorized test",
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))

	// 4. HR persona attempting to self-approve own leave -> PermissionDenied
	hrSelfCtx := grpcutil.WithIncomingActor(context.Background(), grpcutil.ActorContext{
		ActorID: "emp-human-alice",
		Role:    "hr",
	})
	_, err = srv.UpdateLeaveStatus(hrSelfCtx, &workforcev1.UpdateLeaveStatusRequest{
		LeaveID: "leave-alice-1",
		Status:  "approved",
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Contains(t, err.Error(), "cannot self-approve")

	// 5. Verify all 3 denials were recorded in ops.audit_logs and chain is intact
	denials, err := opsRepo.ListAuditLogs(context.Background(), "", "authz.policy.denied", 20, 0)
	require.NoError(t, err)
	assert.Len(t, denials, 3)

	verify, err := opsRepo.VerifyAuditChain(context.Background())
	require.NoError(t, err)
	assert.True(t, verify.Valid)
	assert.Equal(t, int64(4), verify.VerifiedRecords) // 3 denials + 1 allowed medical hold
}
