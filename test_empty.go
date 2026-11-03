package main
import (
	"context"
	"fmt"
	"github.com/orange-cat-investments/oci/internal/app/customerbll"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
)
type mockCI struct {
	coreinvestv1.CoreInvestServiceClient
}
func (m *mockCI) ListStatements(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
	return &coreinvestv1.ListStatementsResponse{
		Statements: nil,
	}, nil
}
func main() {
	srv := customerbll.NewServer(customerbll.Config{})
	srv.SetCoreInvestClient(&mockCI{})
	res, _ := srv.ListStatements(context.Background(), &customerbllv1.ListStatementsRequest{})
	fmt.Printf("Statements: %#v\n", res.Statements)
}
