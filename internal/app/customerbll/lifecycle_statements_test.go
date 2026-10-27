package customerbll

import (
	"context"
	"errors"
	"testing"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCoreInvestClient struct {
	coreinvestv1.CoreInvestServiceClient
	listStatementsFunc func(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error)
}

func (m *mockCoreInvestClient) ListStatements(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
	if m.listStatementsFunc != nil {
		return m.listStatementsFunc(ctx, in)
	}
	return nil, nil
}

func TestListStatements_Success(t *testing.T) {
	expectedStatements := []*coreinvestv1.CustomerStatement{
		{
			StatementID:   "stmt-101",
			CustomerID:    "cust-123",
			PortfolioID:   "port-123",
			StatementType: "monthly",
			Period:        "2026-01",
			Status:        "generated",
		},
		{
			StatementID:   "stmt-102",
			CustomerID:    "cust-123",
			PortfolioID:   "port-123",
			StatementType: "monthly",
			Period:        "2026-02",
			Status:        "generated",
		},
	}

	mockCI := &mockCoreInvestClient{
		listStatementsFunc: func(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
			assert.Equal(t, "cust-123", in.CustomerID)
			return &coreinvestv1.ListStatementsResponse{
				Statements: expectedStatements,
			}, nil
		},
	}

	server := NewServer(Config{})
	server.SetCoreInvestClient(mockCI)

	res, err := server.ListStatements(t.Context(), &customerbllv1.ListStatementsRequest{
		CustomerID: "cust-123",
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Len(t, res.Statements, 2)
	assert.Equal(t, "stmt-101", res.Statements[0].StatementID)
	assert.Equal(t, "stmt-102", res.Statements[1].StatementID)
}

func TestListStatements_CoreInvestErrorFallback(t *testing.T) {
	mockCI := &mockCoreInvestClient{
		listStatementsFunc: func(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
			return nil, errors.New("database connection failed")
		},
	}

	server := NewServer(Config{})
	server.SetCoreInvestClient(mockCI)

	res, err := server.ListStatements(t.Context(), &customerbllv1.ListStatementsRequest{
		CustomerID: "cust-123",
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Empty(t, res.Statements)
}

func TestListStatements_NilResponseFromCoreInvest(t *testing.T) {
	mockCI := &mockCoreInvestClient{
		listStatementsFunc: func(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
			return nil, nil
		},
	}

	server := NewServer(Config{})
	server.SetCoreInvestClient(mockCI)

	res, err := server.ListStatements(t.Context(), &customerbllv1.ListStatementsRequest{
		CustomerID: "cust-123",
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Empty(t, res.Statements)
}

func TestListStatements_NilCoreInvestClient(t *testing.T) {
	server := NewServer(Config{})

	res, err := server.ListStatements(t.Context(), &customerbllv1.ListStatementsRequest{
		CustomerID: "cust-123",
	})
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Empty(t, res.Statements)
}

func TestListStatements_NilRequest(t *testing.T) {
	server := NewServer(Config{})

	res, err := server.ListStatements(t.Context(), nil)
	require.NoError(t, err)
	require.NotNil(t, res)
	assert.Empty(t, res.Statements)
}
