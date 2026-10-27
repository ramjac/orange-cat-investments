package internalbllv1_test

import (
	"context"
	"net"
	"testing"

	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/grpcutil"
)

type testInternalBLLServer struct {
	internalbllv1.UnimplementedInternalBLLServiceServer
}

func (s *testInternalBLLServer) GetEmployee(_ context.Context, req *workforcev1.GetEmployeeRequest) (*workforcev1.GetEmployeeResponse, error) {
	return &workforcev1.GetEmployeeResponse{
		Employee: &workforcev1.Employee{
			EmployeeID: req.EmployeeID,
			FirstName:  "Elise",
		},
	}, nil
}

func TestInternalBLLServiceRoundTrip(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpcutil.NewServer(nil)
	internalbllv1.RegisterInternalBLLServiceServer(srv, &testInternalBLLServer{})
	go func() { _ = srv.Serve(lis) }()
	defer srv.Stop()

	conn, err := grpcutil.Dial(context.Background(), lis.Addr().String(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	client := internalbllv1.NewInternalBLLServiceClient(conn)
	resp, err := client.GetEmployee(context.Background(), &workforcev1.GetEmployeeRequest{EmployeeID: "emp-001"})
	if err != nil {
		t.Fatalf("GetEmployee: %v", err)
	}
	if resp.Employee == nil || resp.Employee.EmployeeID != "emp-001" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}
