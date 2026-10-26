package customerbff

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (m *mockCustomerBLLClient) ListWebhooks(ctx context.Context, in *customerbllv1.ListWebhooksRequest) (*customerbllv1.ListWebhooksResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	hooks := m.webhooks[in.CustomerID]
	if hooks == nil {
		hooks = []*coreinvestv1.CustomerWebhook{}
	}
	return &customerbllv1.ListWebhooksResponse{Subscriptions: hooks}, nil
}

func (m *mockCustomerBLLClient) CreateWebhook(ctx context.Context, in *customerbllv1.CreateWebhookRequest) (*customerbllv1.CreateWebhookResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	whID := "wh-" + uuid.NewString()[:8]
	wh := &coreinvestv1.CustomerWebhook{
		WebhookID:  whID,
		CustomerID: in.CustomerID,
		TargetURL:  in.TargetURL,
		EventTypes: in.Events,
		Status:     "active",
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	m.webhooks[in.CustomerID] = append(m.webhooks[in.CustomerID], wh)
	return &customerbllv1.CreateWebhookResponse{
		Subscription: wh,
	}, nil
}

func (m *mockCustomerBLLClient) GetESGTransparency(ctx context.Context, in *customerbllv1.GetESGTransparencyRequest) (*customerbllv1.GetESGTransparencyResponse, error) {
	return &customerbllv1.GetESGTransparencyResponse{
		OverallWelfareScore:    98.2,
		VeterinaryAdherencePct: 99.4,
		NutritionalCompliance:  "100% Grain-Free Prescription Salmon & Turkey",
		AvgDailyNapHours:       14.6,
		PerchComfortRating:     "5.0 / 5.0 (Ergonomic Thermal Cushioning)",
		WorkplaceIncidentsCnt:  0,
		MedicalHoldsIssuedYTD:  0,
		FelineExecutives: []*customerbllv1.FelineExecutiveSummary{
			{EmployeeID: "fel-01", Name: "Barnaby", Role: "Chief Happiness Officer", HealthStatus: "excellent", PurrFrequencyHz: 26.5, PreferredSunbeam: "Library South Perch"},
		},
	}, nil
}

func (m *mockCustomerBLLClient) ListCameraStreams(ctx context.Context, in *customerbllv1.ListCameraStreamsRequest) (*customerbllv1.ListCameraStreamsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return &customerbllv1.ListCameraStreamsResponse{Streams: m.streams}, nil
}

func (m *mockCustomerBLLClient) ListStatements(ctx context.Context, in *customerbllv1.ListStatementsRequest) (*customerbllv1.ListStatementsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stmts := m.statements[in.CustomerID]
	if stmts == nil {
		stmts = []*coreinvestv1.CustomerStatement{}
	}
	return &customerbllv1.ListStatementsResponse{Statements: stmts}, nil
}

func (m *mockCustomerBLLClient) GenerateStatement(ctx context.Context, in *customerbllv1.GenerateStatementRequest) (*customerbllv1.GenerateStatementResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stmtID := "stmt-" + uuid.NewString()[:8]
	stmt := &coreinvestv1.CustomerStatement{
		StatementID:   stmtID,
		CustomerID:    in.CustomerID,
		PortfolioID:   in.PortfolioID,
		StatementType: in.Type,
		Period:        fmt.Sprintf("%d-%02d", in.Year, in.Month),
	}
	m.statements[in.CustomerID] = append(m.statements[in.CustomerID], stmt)
	return &customerbllv1.GenerateStatementResponse{
		Statement: stmt,
		PDFBytes:  []byte("%PDF-1.4 mock pdf"),
	}, nil
}

func (m *mockCustomerBLLClient) DownloadStatement(ctx context.Context, in *customerbllv1.DownloadStatementRequest) (*customerbllv1.DownloadStatementResponse, error) {
	return &customerbllv1.DownloadStatementResponse{
		Statement: &coreinvestv1.CustomerStatement{
			StatementID: in.StatementID,
		},
		PDFBytes:    []byte("%PDF-1.4 mock pdf"),
		ContentType: "application/pdf",
		FileName:    "statement_" + in.StatementID + ".pdf",
	}, nil
}

func (m *mockCustomerBLLClient) GenerateBatchStatements(ctx context.Context, in *customerbllv1.GenerateBatchStatementsRequest) (*customerbllv1.GenerateBatchStatementsResponse, error) {
	return &customerbllv1.GenerateBatchStatementsResponse{TotalPortfolios: 0}, nil
}

func (m *mockCustomerBLLClient) ListCustomerTickets(ctx context.Context, in *customerbllv1.ListCustomerTicketsRequest) (*customerbllv1.ListCustomerTicketsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var list []*opsv1.ITTicket
	for _, t := range m.tickets {
		if in.CustomerID == "" || t.CustomerID == in.CustomerID {
			list = append(list, t)
		}
	}
	return &customerbllv1.ListCustomerTicketsResponse{Tickets: list}, nil
}

func (m *mockCustomerBLLClient) CreateCustomerTicket(ctx context.Context, in *customerbllv1.CreateCustomerTicketRequest) (*customerbllv1.CreateCustomerTicketResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tickID := "tick-" + uuid.NewString()[:8]
	t := &opsv1.ITTicket{
		TicketID:   tickID,
		CustomerID: in.CustomerID,
		Title:      in.Title,
		Body:       in.Body,
		Severity:   in.Severity,
		State:      "open",
		CreatedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	m.tickets[tickID] = t
	return &customerbllv1.CreateCustomerTicketResponse{Ticket: t}, nil
}

func (m *mockCustomerBLLClient) GetCustomerTicket(ctx context.Context, in *customerbllv1.GetCustomerTicketRequest) (*customerbllv1.GetCustomerTicketResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tickets[in.TicketID]; ok {
		return &customerbllv1.GetCustomerTicketResponse{Ticket: t}, nil
	}
	return nil, status.Error(codes.NotFound, "ticket not found")
}

func (m *mockCustomerBLLClient) ListCustomerTicketComments(ctx context.Context, in *customerbllv1.ListCustomerTicketCommentsRequest) (*customerbllv1.ListCustomerTicketCommentsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	comments := m.comments[in.TicketID]
	var filtered []*opsv1.TicketComment
	for _, c := range comments {
		if !c.IsInternal {
			filtered = append(filtered, c)
		}
	}
	return &customerbllv1.ListCustomerTicketCommentsResponse{Comments: filtered}, nil
}

func (m *mockCustomerBLLClient) AddCustomerTicketComment(ctx context.Context, in *customerbllv1.AddCustomerTicketCommentRequest) (*customerbllv1.AddCustomerTicketCommentResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	isInternal := strings.Contains(in.Comment, "INTERNAL")
	c := &opsv1.TicketComment{
		CommentID:      "comm-" + uuid.NewString()[:8],
		TicketID:       in.TicketID,
		AuthorUsername: in.AuthorUsername,
		Comment:        in.Comment,
		IsInternal:     isInternal,
		CreatedAt:      time.Now().UTC().Format(time.RFC3339),
	}
	m.comments[in.TicketID] = append(m.comments[in.TicketID], c)
	return &customerbllv1.AddCustomerTicketCommentResponse{Comment: c}, nil
}
