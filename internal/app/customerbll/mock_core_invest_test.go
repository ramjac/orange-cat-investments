package customerbll

import (
	"context"

	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
)

type mockCoreInvestClient struct {
	coreinvestv1.CoreInvestServiceClient

	registerCustomerFn          func(ctx context.Context, in *coreinvestv1.RegisterCustomerRequest) (*coreinvestv1.RegisterCustomerResponse, error)
	closeAndLiquidateAccountFn  func(ctx context.Context, in *coreinvestv1.CloseAndLiquidateAccountRequest) (*coreinvestv1.CloseAndLiquidateAccountResponse, error)
	executeGDPRErasureFn        func(ctx context.Context, in *coreinvestv1.ExecuteGDPRErasureRequest) (*coreinvestv1.ExecuteGDPRErasureResponse, error)
	getGDPRErasureCertificateFn func(ctx context.Context, in *coreinvestv1.GetGDPRErasureCertificateRequest) (*coreinvestv1.GetGDPRErasureCertificateResponse, error)
	listStatementsFn            func(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error)
	generateMonthlyStatementFn  func(ctx context.Context, in *coreinvestv1.GenerateMonthlyStatementRequest) (*coreinvestv1.GenerateMonthlyStatementResponse, error)
	generateForm1099BFn         func(ctx context.Context, in *coreinvestv1.GenerateForm1099BRequest) (*coreinvestv1.GenerateForm1099BResponse, error)
	getStatementDocumentFn      func(ctx context.Context, in *coreinvestv1.GetStatementDocumentRequest) (*coreinvestv1.GetStatementDocumentResponse, error)
	listPortfoliosFn            func(ctx context.Context, in *coreinvestv1.ListPortfoliosRequest) (*coreinvestv1.ListPortfoliosResponse, error)
	generateStatementFn         func(ctx context.Context, in *coreinvestv1.GenerateStatementRequest) (*coreinvestv1.GenerateStatementResponse, error)
}

func (m *mockCoreInvestClient) RegisterCustomer(ctx context.Context, in *coreinvestv1.RegisterCustomerRequest) (*coreinvestv1.RegisterCustomerResponse, error) {
	if m.registerCustomerFn != nil {
		return m.registerCustomerFn(ctx, in)
	}
	return nil, nil
}

func (m *mockCoreInvestClient) CloseAndLiquidateAccount(ctx context.Context, in *coreinvestv1.CloseAndLiquidateAccountRequest) (*coreinvestv1.CloseAndLiquidateAccountResponse, error) {
	if m.closeAndLiquidateAccountFn != nil {
		return m.closeAndLiquidateAccountFn(ctx, in)
	}
	return nil, nil
}

func (m *mockCoreInvestClient) ExecuteGDPRErasure(ctx context.Context, in *coreinvestv1.ExecuteGDPRErasureRequest) (*coreinvestv1.ExecuteGDPRErasureResponse, error) {
	if m.executeGDPRErasureFn != nil {
		return m.executeGDPRErasureFn(ctx, in)
	}
	return nil, nil
}

func (m *mockCoreInvestClient) GetGDPRErasureCertificate(ctx context.Context, in *coreinvestv1.GetGDPRErasureCertificateRequest) (*coreinvestv1.GetGDPRErasureCertificateResponse, error) {
	if m.getGDPRErasureCertificateFn != nil {
		return m.getGDPRErasureCertificateFn(ctx, in)
	}
	return nil, nil
}

func (m *mockCoreInvestClient) ListStatements(ctx context.Context, in *coreinvestv1.ListStatementsRequest) (*coreinvestv1.ListStatementsResponse, error) {
	if m.listStatementsFn != nil {
		return m.listStatementsFn(ctx, in)
	}
	return nil, nil
}

func (m *mockCoreInvestClient) GenerateMonthlyStatement(ctx context.Context, in *coreinvestv1.GenerateMonthlyStatementRequest) (*coreinvestv1.GenerateMonthlyStatementResponse, error) {
	if m.generateMonthlyStatementFn != nil {
		return m.generateMonthlyStatementFn(ctx, in)
	}
	return nil, nil
}

func (m *mockCoreInvestClient) GenerateForm1099B(ctx context.Context, in *coreinvestv1.GenerateForm1099BRequest) (*coreinvestv1.GenerateForm1099BResponse, error) {
	if m.generateForm1099BFn != nil {
		return m.generateForm1099BFn(ctx, in)
	}
	return nil, nil
}

func (m *mockCoreInvestClient) GetStatementDocument(ctx context.Context, in *coreinvestv1.GetStatementDocumentRequest) (*coreinvestv1.GetStatementDocumentResponse, error) {
	if m.getStatementDocumentFn != nil {
		return m.getStatementDocumentFn(ctx, in)
	}
	return nil, nil
}

func (m *mockCoreInvestClient) ListPortfolios(ctx context.Context, in *coreinvestv1.ListPortfoliosRequest) (*coreinvestv1.ListPortfoliosResponse, error) {
	if m.listPortfoliosFn != nil {
		return m.listPortfoliosFn(ctx, in)
	}
	return nil, nil
}

func (m *mockCoreInvestClient) GenerateStatement(ctx context.Context, in *coreinvestv1.GenerateStatementRequest) (*coreinvestv1.GenerateStatementResponse, error) {
	if m.generateStatementFn != nil {
		return m.generateStatementFn(ctx, in)
	}
	return nil, nil
}
