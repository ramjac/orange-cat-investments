package worker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/orange-cat-investments/oci/internal/app/customerbll"
	"github.com/orange-cat-investments/oci/internal/app/internalbll"
	cirepo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	facrepo "github.com/orange-cat-investments/oci/internal/repository/facilities"
	opsrepo "github.com/orange-cat-investments/oci/internal/repository/ops"
	wfrepo "github.com/orange-cat-investments/oci/internal/repository/workforce"
	"github.com/orange-cat-investments/oci/pkg/envelope"
	coreinvestv1 "github.com/orange-cat-investments/oci/pkg/gen/go/core_invest/v1"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	facilitiesv1 "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"
	internalbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/internal_bll/v1"
	opsv1 "github.com/orange-cat-investments/oci/pkg/gen/go/ops/v1"
	workforcev1 "github.com/orange-cat-investments/oci/pkg/gen/go/workforce/v1"
	"github.com/orange-cat-investments/oci/pkg/outbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type relayFlakyPublisher struct {
	mu      sync.Mutex
	outage  atomic.Bool
	sent    map[string][]*message.Message
	sentCh  chan string
}

func newRelayFlakyPublisher() *relayFlakyPublisher {
	return &relayFlakyPublisher{
		sent:   make(map[string][]*message.Message),
		sentCh: make(chan string, 32),
	}
}

func (p *relayFlakyPublisher) Publish(topic string, msgs ...*message.Message) error {
	if p.outage.Load() {
		return errors.New("rabbitmq unavailable")
	}
	p.mu.Lock()
	p.sent[topic] = append(p.sent[topic], msgs...)
	p.mu.Unlock()
	for range msgs {
		select {
		case p.sentCh <- topic:
		default:
		}
	}
	return nil
}

func (p *relayFlakyPublisher) Close() error { return nil }

func (p *relayFlakyPublisher) count(topic string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.sent[topic])
}

type inProcCoreInvestClient struct {
	coreinvestv1.CoreInvestServiceClient
	repo cirepo.Repository
}

func (c *inProcCoreInvestClient) ClaimOutboxEvents(ctx context.Context, in *coreinvestv1.ClaimOutboxEventsRequest) (*coreinvestv1.ClaimOutboxEventsResponse, error) {
	recs, err := c.repo.ClaimOutboxEvents(ctx, in.Limit, time.Duration(in.LeaseMs)*time.Millisecond)
	if err != nil {
		return nil, err
	}
	out := make([]*coreinvestv1.OutboxEvent, 0, len(recs))
	for _, r := range recs {
		out = append(out, &coreinvestv1.OutboxEvent{
			ID:          r.ID,
			Schema:      string(r.Schema),
			EventID:     r.EventID,
			Topic:       r.Topic,
			AggregateID: r.AggregateID,
			Payload:     r.Payload,
			Attempts:    int32(r.Attempts),
			CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		})
	}
	return &coreinvestv1.ClaimOutboxEventsResponse{Events: out}, nil
}

func (c *inProcCoreInvestClient) AckOutboxEvents(ctx context.Context, in *coreinvestv1.AckOutboxEventsRequest) (*coreinvestv1.AckOutboxEventsResponse, error) {
	fails := make([]outbox.Failure, 0, len(in.Failures))
	for _, f := range in.Failures {
		fails = append(fails, outbox.Failure{ID: f.ID, Schema: outbox.Schema(f.Schema), Error: f.Error})
	}
	n, err := c.repo.AckOutboxEvents(ctx, in.PublishedIDs, fails)
	return &coreinvestv1.AckOutboxEventsResponse{Acknowledged: int32(n)}, err
}

type inProcWorkforceClient struct {
	workforcev1.WorkforceServiceClient
	repo wfrepo.Repository
}

func (c *inProcWorkforceClient) ClaimOutboxEvents(ctx context.Context, in *workforcev1.ClaimOutboxEventsRequest) (*workforcev1.ClaimOutboxEventsResponse, error) {
	recs, err := c.repo.ClaimOutboxEvents(ctx, in.Limit, time.Duration(in.LeaseMs)*time.Millisecond)
	if err != nil {
		return nil, err
	}
	out := make([]*workforcev1.OutboxEvent, 0, len(recs))
	for _, r := range recs {
		out = append(out, &workforcev1.OutboxEvent{
			ID:          r.ID,
			Schema:      string(r.Schema),
			EventID:     r.EventID,
			Topic:       r.Topic,
			AggregateID: r.AggregateID,
			Payload:     r.Payload,
			Attempts:    int32(r.Attempts),
			CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		})
	}
	return &workforcev1.ClaimOutboxEventsResponse{Events: out}, nil
}

func (c *inProcWorkforceClient) AckOutboxEvents(ctx context.Context, in *workforcev1.AckOutboxEventsRequest) (*workforcev1.AckOutboxEventsResponse, error) {
	fails := make([]outbox.Failure, 0, len(in.Failures))
	for _, f := range in.Failures {
		fails = append(fails, outbox.Failure{ID: f.ID, Schema: outbox.Schema(f.Schema), Error: f.Error})
	}
	n, err := c.repo.AckOutboxEvents(ctx, in.PublishedIDs, fails)
	return &workforcev1.AckOutboxEventsResponse{Acknowledged: int32(n)}, err
}

type inProcFacilitiesClient struct {
	facilitiesv1.FacilitiesServiceClient
	repo facrepo.Repository
}

func (c *inProcFacilitiesClient) ClaimOutboxEvents(ctx context.Context, in *facilitiesv1.ClaimOutboxEventsRequest) (*facilitiesv1.ClaimOutboxEventsResponse, error) {
	recs, err := c.repo.ClaimOutboxEvents(ctx, in.Limit, time.Duration(in.LeaseMs)*time.Millisecond)
	if err != nil {
		return nil, err
	}
	out := make([]*facilitiesv1.OutboxEvent, 0, len(recs))
	for _, r := range recs {
		out = append(out, &facilitiesv1.OutboxEvent{
			ID:          r.ID,
			Schema:      string(r.Schema),
			EventID:     r.EventID,
			Topic:       r.Topic,
			AggregateID: r.AggregateID,
			Payload:     r.Payload,
			Attempts:    int32(r.Attempts),
			CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		})
	}
	return &facilitiesv1.ClaimOutboxEventsResponse{Events: out}, nil
}

func (c *inProcFacilitiesClient) AckOutboxEvents(ctx context.Context, in *facilitiesv1.AckOutboxEventsRequest) (*facilitiesv1.AckOutboxEventsResponse, error) {
	fails := make([]outbox.Failure, 0, len(in.Failures))
	for _, f := range in.Failures {
		fails = append(fails, outbox.Failure{ID: f.ID, Schema: outbox.Schema(f.Schema), Error: f.Error})
	}
	n, err := c.repo.AckOutboxEvents(ctx, in.PublishedIDs, fails)
	return &facilitiesv1.AckOutboxEventsResponse{Acknowledged: int32(n)}, err
}

type inProcOpsClient struct {
	opsv1.OpsServiceClient
	repo opsrepo.Repository
}

func (c *inProcOpsClient) ClaimOutboxEvents(ctx context.Context, in *opsv1.ClaimOutboxEventsRequest) (*opsv1.ClaimOutboxEventsResponse, error) {
	recs, err := c.repo.ClaimOutboxEvents(ctx, in.Limit, time.Duration(in.LeaseMs)*time.Millisecond)
	if err != nil {
		return nil, err
	}
	out := make([]*opsv1.OutboxEvent, 0, len(recs))
	for _, r := range recs {
		out = append(out, &opsv1.OutboxEvent{
			ID:          r.ID,
			Schema:      string(r.Schema),
			EventID:     r.EventID,
			Topic:       r.Topic,
			AggregateID: r.AggregateID,
			Payload:     r.Payload,
			Attempts:    int32(r.Attempts),
			CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		})
	}
	return &opsv1.ClaimOutboxEventsResponse{Events: out}, nil
}

func (c *inProcOpsClient) AckOutboxEvents(ctx context.Context, in *opsv1.AckOutboxEventsRequest) (*opsv1.AckOutboxEventsResponse, error) {
	fails := make([]outbox.Failure, 0, len(in.Failures))
	for _, f := range in.Failures {
		fails = append(fails, outbox.Failure{ID: f.ID, Schema: outbox.Schema(f.Schema), Error: f.Error})
	}
	n, err := c.repo.AckOutboxEvents(ctx, in.PublishedIDs, fails)
	return &opsv1.AckOutboxEventsResponse{Acknowledged: int32(n)}, err
}

type inProcCustomerBLLClient struct {
	customerbllv1.CustomerBLLServiceClient
	srv *customerbll.Server
}

func (c *inProcCustomerBLLClient) ClaimOutboxEvents(ctx context.Context, in *coreinvestv1.ClaimOutboxEventsRequest) (*coreinvestv1.ClaimOutboxEventsResponse, error) {
	return c.srv.ClaimOutboxEvents(ctx, in)
}

func (c *inProcCustomerBLLClient) AckOutboxEvents(ctx context.Context, in *coreinvestv1.AckOutboxEventsRequest) (*coreinvestv1.AckOutboxEventsResponse, error) {
	return c.srv.AckOutboxEvents(ctx, in)
}

type inProcInternalBLLClient struct {
	internalbllv1.InternalBLLServiceClient
	srv *internalbll.Server
}

func (c *inProcInternalBLLClient) ClaimOutboxEvents(ctx context.Context, in *internalbllv1.ClaimBLLOutboxEventsRequest) (*internalbllv1.ClaimBLLOutboxEventsResponse, error) {
	return c.srv.ClaimOutboxEvents(ctx, in)
}

func (c *inProcInternalBLLClient) AckOutboxEvents(ctx context.Context, in *internalbllv1.AckBLLOutboxEventsRequest) (*internalbllv1.AckBLLOutboxEventsResponse, error) {
	return c.srv.AckOutboxEvents(ctx, in)
}

func TestOutboxRelay_BridgesAllFourSchemasAcrossRabbitMQOutage(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	ciMockRepo := cirepo.NewMockRepository()
	wfMockRepo := wfrepo.NewMockRepository()
	facMockRepo := facrepo.NewMockRepository()
	opsMockRepo := opsrepo.NewMockRepository()

	custBLL := customerbll.NewServer(customerbll.Config{})
	custBLL.SetCoreInvestClient(&inProcCoreInvestClient{repo: ciMockRepo})

	intBLL := internalbll.NewServer(
		&inProcWorkforceClient{repo: wfMockRepo},
		&inProcFacilitiesClient{repo: facMockRepo},
		&inProcOpsClient{repo: opsMockRepo},
		nil,
		logger,
		internalbll.Config{},
	)

	// Commit domain mutations while RabbitMQ is down.
	pub := newRelayFlakyPublisher()
	pub.outage.Store(true)

	_, err := ciMockRepo.CreateBrokerageOrder(ctx, &cirepo.BrokerageOrder{
		PortfolioID: "port-1",
		BrokerName:  "alpaca",
		Symbol:      "NVDA",
		Side:        "buy",
		Quantity:    10,
		Price:       125.0,
		Status:      "filled",
	})
	require.NoError(t, err)

	_, err = wfMockRepo.CreateWorkplaceIncident(ctx, &wfrepo.WorkplaceIncident{
		Title:       "Laser pointer malfunction",
		Category:    "habitat_disruption",
		Severity:    "high",
		Description: "Auto-laser stuck on corridor wall",
	})
	require.NoError(t, err)

	_, err = facMockRepo.CreateOTAJob(ctx, &facrepo.DeviceOTAJob{
		AssetID:   "asset-cam-1",
		ReleaseID: "rel-v2",
	})
	require.NoError(t, err)

	_, err = opsMockRepo.CreateITTicket(ctx, &opsrepo.ITTicket{
		Title:          "Critical rack alert",
		AuthorUsername: "frank",
		IsUrgent:       true,
	})
	require.NoError(t, err)

	fwd := outbox.NewForwarder(pub, outbox.ForwarderConfig{
		PollInterval: 50 * time.Millisecond,
		BatchSize:    25,
		Logger:       logger,
	},
		&customerBLLOutboxSource{client: &inProcCustomerBLLClient{srv: custBLL}},
		&internalBLLOutboxSource{client: &inProcInternalBLLClient{srv: intBLL}},
	)

	// Pass 1 during RabbitMQ outage: 0 delivered, events stay durable in outbox.
	delivered, runErr := fwd.RunOnce(ctx)
	require.Error(t, runErr)
	assert.Equal(t, 0, delivered)
	assert.Equal(t, 0, pub.count(envelope.TopicOrderPlaced))

	// RabbitMQ recovers: Pass 2 drains all 4 schemas without cross-schema ID collision.
	pub.outage.Store(false)
	delivered, runErr = fwd.RunOnce(ctx)
	require.NoError(t, runErr)
	assert.Equal(t, 4, delivered)

	assert.Equal(t, 1, pub.count(envelope.TopicOrderPlaced))
	assert.Equal(t, 1, pub.count(envelope.TopicIncidentCreated))
	assert.Equal(t, 1, pub.count(envelope.TopicOTATriggered))
	assert.Equal(t, 1, pub.count(envelope.TopicIncidentAlerted))

	// Verify envelope integrity on published order message
	pub.mu.Lock()
	orderMsg := pub.sent[envelope.TopicOrderPlaced][0]
	pub.mu.Unlock()
	var env envelope.EventEnvelope[map[string]any]
	require.NoError(t, json.Unmarshal(orderMsg.Payload, &env))
	assert.Equal(t, envelope.TopicOrderPlaced, env.EventType)
	assert.Equal(t, "NVDA", env.Payload["symbol"])

	// Pass 3: nothing left pending (no duplicate delivery)
	delivered, runErr = fwd.RunOnce(ctx)
	require.NoError(t, runErr)
	assert.Equal(t, 0, delivered)
}
