package internalbll

import (
	"context"
	"errors"
	"testing"

	opsrepo "github.com/orange-cat-investments/oci/internal/repository/ops"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// correlationWFStub serves a configurable workforce roster and review/leave
// ownership so actor-correlation behavior can be exercised end to end.
type correlationWFStub struct {
	workforcev1.WorkforceServiceClient
	roster        []*workforcev1.Employee
	reviewOwner   string
	leaveOwner    string
	reviewErr     error
	updateLeaveOK bool
}

func (w *correlationWFStub) ListEmployees(_ context.Context, _ *workforcev1.ListEmployeesRequest) (*workforcev1.ListEmployeesResponse, error) {
	return &workforcev1.ListEmployeesResponse{Employees: w.roster}, nil
}

func (w *correlationWFStub) GetReviewCycle(_ context.Context, in *workforcev1.GetReviewCycleRequest) (*workforcev1.GetReviewCycleResponse, error) {
	if w.reviewErr != nil {
		return nil, w.reviewErr
	}
	return &workforcev1.GetReviewCycleResponse{
		ReviewCycle: &workforcev1.ReviewCycle{ReviewID: in.ReviewID, EmployeeID: w.reviewOwner},
	}, nil
}

func (w *correlationWFStub) GetLeaveRequest(_ context.Context, in *workforcev1.GetLeaveRequestRequest) (*workforcev1.GetLeaveRequestResponse, error) {
	return &workforcev1.GetLeaveRequestResponse{
		LeaveRequest: &workforcev1.LeaveRequest{LeaveID: in.LeaveID, EmployeeID: w.leaveOwner},
	}, nil
}

func (w *correlationWFStub) SubmitManagerAssessment(_ context.Context, in *workforcev1.SubmitManagerAssessmentRequest) (*workforcev1.SubmitManagerAssessmentResponse, error) {
	return &workforcev1.SubmitManagerAssessmentResponse{
		ReviewCycle: &workforcev1.ReviewCycle{ReviewID: in.ReviewID},
	}, nil
}

func (w *correlationWFStub) UpdateLeaveStatus(_ context.Context, in *workforcev1.UpdateLeaveStatusRequest) (*workforcev1.UpdateLeaveStatusResponse, error) {
	if !w.updateLeaveOK {
		return nil, errors.New("mutation must not reach the domain API when ownership is unverifiable")
	}
	return &workforcev1.UpdateLeaveStatusResponse{
		LeaveRequest: &workforcev1.LeaveRequest{LeaveID: in.LeaveID, Status: in.Status},
	}, nil
}

func (w *correlationWFStub) UpdateReviewCycle(_ context.Context, in *workforcev1.UpdateReviewCycleRequest) (*workforcev1.UpdateReviewCycleResponse, error) {
	return &workforcev1.UpdateReviewCycleResponse{
		ReviewCycle: &workforcev1.ReviewCycle{ReviewID: in.ReviewID},
	}, nil
}

func (w *correlationWFStub) CreateReviewCycle(_ context.Context, in *workforcev1.CreateReviewCycleRequest) (*workforcev1.CreateReviewCycleResponse, error) {
	return &workforcev1.CreateReviewCycleResponse{
		ReviewCycle: &workforcev1.ReviewCycle{EmployeeID: in.EmployeeID},
	}, nil
}

func TestInternalBLL_ForgejoActorCorrelationBlocksSelfAssessment(t *testing.T) {
	opsRepo := opsrepo.NewMockRepository()
	wfStub := &correlationWFStub{
		roster: []*workforcev1.Employee{
			{EmployeeID: "wf-uuid-jdoe", Email: "jdoe@oci.example", Status: "active"},
			{EmployeeID: "wf-uuid-alice", Email: "alice@oci.example", Status: "active"},
		},
		reviewOwner: "wf-uuid-jdoe",
	}
	srv := NewServer(wfStub, nil, &auditOpsStub{repo: opsRepo}, nil, nil, Config{AuditSpoolDir: t.TempDir()})
	defer srv.Close()

	// Forgejo login yields actor id emp-<username>; the review owner is the
	// canonical workforce UUID. The engine must correlate the identity via
	// email and deny the self-assessment instead of letting the ID mismatch
	// silently disable the self-approval restriction.
	ctx := grpcutil.WithIncomingActor(context.Background(), grpcutil.ActorContext{
		ActorID: "emp-jdoe",
		Role:    "engineering",
		Email:   "jdoe@oci.example",
	})
	_, err := srv.SubmitManagerAssessment(ctx, &workforcev1.SubmitManagerAssessmentRequest{ReviewID: "rev-1"})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Contains(t, err.Error(), "own review cycle")
}

func TestInternalBLL_ForgejoActorCorrelationAllowsOtherEmployees(t *testing.T) {
	opsRepo := opsrepo.NewMockRepository()
	wfStub := &correlationWFStub{
		roster: []*workforcev1.Employee{
			{EmployeeID: "wf-uuid-alice", Email: "alice@oci.example", Status: "active"},
			{EmployeeID: "wf-uuid-jdoe", Email: "jdoe@oci.example", Status: "active"},
		},
		reviewOwner: "wf-uuid-jdoe",
	}
	srv := NewServer(wfStub, nil, &auditOpsStub{repo: opsRepo}, nil, nil, Config{AuditSpoolDir: t.TempDir()})
	defer srv.Close()

	// Alice (hr, a manager role) assessing J. Doe's review: correlation
	// resolves to a different canonical ID, so the self-restriction does not
	// apply and the assessment must go through.
	ctx := grpcutil.WithIncomingActor(context.Background(), grpcutil.ActorContext{
		ActorID: "emp-alice",
		Role:    "hr",
		Email:   "alice@oci.example",
	})
	_, err := srv.SubmitManagerAssessment(ctx, &workforcev1.SubmitManagerAssessmentRequest{ReviewID: "rev-1"})
	require.NoError(t, err)
}

func TestInternalBLL_ForgejoActorCorrelationFailsClosed(t *testing.T) {
	opsRepo := opsrepo.NewMockRepository()
	wfStub := &correlationWFStub{
		roster: []*workforcev1.Employee{
			{EmployeeID: "wf-uuid-alice", Email: "alice@oci.example", Status: "active"},
		},
		reviewOwner: "wf-uuid-unknown",
	}
	srv := NewServer(wfStub, nil, &auditOpsStub{repo: opsRepo}, nil, nil, Config{AuditSpoolDir: t.TempDir()})
	defer srv.Close()

	// The actor's email matches no workforce employee: owner-based
	// authorization must fail closed rather than compare mismatched IDs.
	ctx := grpcutil.WithIncomingActor(context.Background(), grpcutil.ActorContext{
		ActorID: "emp-ghost",
		Role:    "hr",
		Email:   "ghost@oci.example",
	})
	_, err := srv.SubmitManagerAssessment(ctx, &workforcev1.SubmitManagerAssessmentRequest{ReviewID: "rev-1"})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
	assert.Contains(t, err.Error(), "canonical workforce identity")
}

func TestInternalBLL_LeaveStatusFailsClosedWithoutOwner(t *testing.T) {
	opsRepo := opsrepo.NewMockRepository()

	// Lookup failure: the mutation must not reach the domain API.
	wfErr := &correlationWFStub{roster: nil}
	wfErr.reviewErr = nil
	srvErr := NewServer(wfErr, nil, &auditOpsStub{repo: opsRepo}, nil, nil, Config{AuditSpoolDir: t.TempDir()})
	defer srvErr.Close()

	// Lookup failure case
	failStub := &failingLeaveStub{}
	srvFail := NewServer(failStub, nil, &auditOpsStub{repo: opsRepo}, nil, nil, Config{AuditSpoolDir: t.TempDir()})
	defer srvFail.Close()
	ctx := grpcutil.WithIncomingActor(context.Background(), grpcutil.ActorContext{
		ActorID: "emp-alice", Role: "hr", Email: "alice@oci.example",
	})
	_, err := srvFail.UpdateLeaveStatus(ctx, &workforcev1.UpdateLeaveStatusRequest{LeaveID: "leave-404", Status: "approved"})
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))

	// Missing owner case: empty EmployeeID on a successfully fetched request.
	emptyStub := &correlationWFStub{roster: nil, leaveOwner: ""}
	srvEmpty := NewServer(emptyStub, nil, &auditOpsStub{repo: opsRepo}, nil, nil, Config{AuditSpoolDir: t.TempDir()})
	defer srvEmpty.Close()
	_, err = srvEmpty.UpdateLeaveStatus(ctx, &workforcev1.UpdateLeaveStatusRequest{LeaveID: "leave-x", Status: "approved"})
	require.Error(t, err)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

// failingLeaveStub always fails the leave lookup.
type failingLeaveStub struct {
	workforcev1.WorkforceServiceClient
}

func (f *failingLeaveStub) GetLeaveRequest(_ context.Context, _ *workforcev1.GetLeaveRequestRequest) (*workforcev1.GetLeaveRequestResponse, error) {
	return nil, errors.New("workforce-api unavailable")
}

func TestInternalBLL_ReviewRPCsRejectSubmissionFields(t *testing.T) {
	opsRepo := opsrepo.NewMockRepository()
	wfStub := &correlationWFStub{
		roster: []*workforcev1.Employee{
			{EmployeeID: "wf-uuid-alice", Email: "alice@oci.example", Status: "active"},
		},
	}
	srv := NewServer(wfStub, nil, &auditOpsStub{repo: opsRepo}, nil, nil, Config{AuditSpoolDir: t.TempDir()})
	defer srv.Close()

	hrCtx := grpcutil.WithIncomingActor(context.Background(), grpcutil.ActorContext{
		ActorID: "emp-alice", Role: "hr", Email: "alice@oci.example",
	})

	// Manager assessment payload smuggled through the general update RPC.
	_, err := srv.UpdateReviewCycle(hrCtx, &workforcev1.UpdateReviewCycleRequest{
		ReviewID:          "rev-1",
		ManagerAssessment: "great work",
		ManagerScore:      5,
		HasManagerScore:   true,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Contains(t, err.Error(), "dedicated submission RPCs")

	// Veterinary audit payload smuggled through the general update RPC.
	_, err = srv.UpdateReviewCycle(hrCtx, &workforcev1.UpdateReviewCycleRequest{
		ReviewID:           "rev-1",
		WellnessAuditScore: 9,
		HasWellnessScore:   true,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// Submission payload smuggled through the create RPC.
	_, err = srv.CreateReviewCycle(hrCtx, &workforcev1.CreateReviewCycleRequest{
		EmployeeID: "wf-uuid-jdoe",
		KPIRatings: `{"quality":5}`,
	})
	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))

	// Plain management updates remain allowed.
	_, err = srv.UpdateReviewCycle(hrCtx, &workforcev1.UpdateReviewCycleRequest{
		ReviewID: "rev-1",
		Status:   "in_progress",
	})
	require.NoError(t, err)
}
