package worker_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/orange-cat-investments/oci/internal/app/worker"
	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestRouter_PoisonPillAndDLQ(t *testing.T) {
	const (
		inTopic  = "events.banking.ach_transfer_requested.v1"
		outTopic = "events.banking.ach_settled.v1"
		dlqTopic = inTopic + ".dlq"

		secretMalformed  = "SECRET_MALFORMED_PAYLOAD_TOKEN_9911"
		secretMissingID  = "SECRET_MISSING_EVENTID_TOKEN_9922"
		secretMissingTs  = "SECRET_MISSING_OCCURREDAT_TOKEN_9933"
		secretHandlerErr = "SECRET_HANDLER_ERROR_TOKEN_9944"
	)

	logBuf := &lockedBuffer{}
	logger := slog.New(slog.NewJSONHandler(logBuf, nil))
	wmLogger := watermill.NopLogger{}
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 50}, wmLogger)

	var handlerAttempts atomic.Int32
	var shouldFail atomic.Bool

	mockBLL := &mockCustomerBLL{
		settleFunc: func(ctx context.Context, in *customerbllv1.SettleACHTransferRequest) (*customerbllv1.SettleACHTransferResponse, error) {
			handlerAttempts.Add(1)
			if shouldFail.Load() {
				return nil, errors.New("upstream clearing house down")
			}
			return &customerbllv1.SettleACHTransferResponse{
				TransferID:  in.TransferID,
				CustomerID:  in.CustomerID,
				PortfolioID: in.PortfolioID,
				AmountUSD:   in.AmountUSD,
				Status:      "settled",
				SettledAt:   time.Now().UTC().Format(time.RFC3339),
			}, nil
		},
	}

	fastRetry := middleware.Retry{
		MaxRetries:      2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     50 * time.Millisecond,
		Multiplier:      2.0,
		Logger:          wmLogger,
	}

	router, err := worker.CreateRouterWithRetry(pubSub, wmLogger, logger, mockBLL, nil, "", "", nil, fastRetry)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dlqCh, err := pubSub.Subscribe(ctx, dlqTopic)
	require.NoError(t, err)
	outCh, err := pubSub.Subscribe(ctx, outTopic)
	require.NoError(t, err)

	go func() {
		_ = router.Run(ctx)
	}()
	select {
	case <-router.Running():
	case <-time.After(2 * time.Second):
		t.Fatal("router did not start in time")
	}
	defer func() { _ = router.Close() }()

	// 1. Malformed JSON -> straight to .dlq without hitting handler or retry.
	malformedMsg := message.NewMessage("msg-malformed", []byte("not-json-"+secretMalformed))
	require.NoError(t, pubSub.Publish(inTopic, malformedMsg))

	select {
	case m := <-dlqCh:
		m.Ack()
		assert.Equal(t, "msg-malformed", m.UUID)
		assert.Equal(t, worker.ErrMalformedEnvelope.Error(), m.Metadata.Get(middleware.ReasonForPoisonedKey))
		assert.Equal(t, inTopic, m.Metadata.Get(middleware.PoisonedTopicKey))
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for malformed message in DLQ")
	}
	assert.Equal(t, int32(0), handlerAttempts.Load(), "malformed message must bypass handler and retry")

	// 2. Missing EventID -> straight to .dlq without hitting handler or retry.
	missingIDPayload, _ := json.Marshal(map[string]any{
		"event_type":     inTopic,
		"occurred_at":    time.Now().UTC(),
		"correlation_id": "corr-missing-id",
		"payload": map[string]any{
			"transfer_id":  "ach-miss-id",
			"bank_account": secretMissingID,
		},
	})
	require.NoError(t, pubSub.Publish(inTopic, message.NewMessage("msg-missing-id", missingIDPayload)))

	select {
	case m := <-dlqCh:
		m.Ack()
		assert.Equal(t, "msg-missing-id", m.UUID)
		assert.Equal(t, worker.ErrMissingEventID.Error(), m.Metadata.Get(middleware.ReasonForPoisonedKey))
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for missing-event_id message in DLQ")
	}
	assert.Equal(t, int32(0), handlerAttempts.Load(), "missing event_id must bypass handler and retry")

	// 3. Missing OccurredAt -> straight to .dlq without hitting handler or retry.
	missingTsPayload, _ := json.Marshal(map[string]any{
		"event_id":       "evt-missing-ts",
		"event_type":     inTopic,
		"correlation_id": "corr-missing-ts",
		"payload": map[string]any{
			"transfer_id":  "ach-miss-ts",
			"bank_account": secretMissingTs,
		},
	})
	require.NoError(t, pubSub.Publish(inTopic, message.NewMessage("msg-missing-ts", missingTsPayload)))

	select {
	case m := <-dlqCh:
		m.Ack()
		assert.Equal(t, "msg-missing-ts", m.UUID)
		assert.Equal(t, worker.ErrMissingOccurredAt.Error(), m.Metadata.Get(middleware.ReasonForPoisonedKey))
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for missing-occurred_at message in DLQ")
	}
	assert.Equal(t, int32(0), handlerAttempts.Load(), "missing occurred_at must bypass handler and retry")

	// 4. Valid envelope with persistent handler failure -> retried 3 total attempts then routed to .dlq.
	shouldFail.Store(true)
	handlerAttempts.Store(0)

	failingEnv := worker.EventEnvelope[worker.ACHTransferRequestedPayload]{
		EventID:       "evt-handler-fail",
		EventType:     inTopic,
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "corr-handler-fail",
		Payload: worker.ACHTransferRequestedPayload{
			TransferID:   "ach-fail-1",
			CustomerID:   "cust-fail",
			PortfolioID:  "port-fail",
			BankAccount:  secretHandlerErr,
			AmountUSD:    100.0,
			TransferType: "deposit",
		},
	}
	failingBytes, err := json.Marshal(failingEnv)
	require.NoError(t, err)
	require.NoError(t, pubSub.Publish(inTopic, message.NewMessage("msg-handler-fail", failingBytes)))

	select {
	case m := <-dlqCh:
		m.Ack()
		assert.Equal(t, "msg-handler-fail", m.UUID)
		assert.Contains(t, m.Metadata.Get(middleware.ReasonForPoisonedKey), "upstream clearing house down")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for handler-error message in DLQ")
	}
	assert.Equal(t, int32(3), handlerAttempts.Load(), "expected 3 total attempts (1 initial + 2 retries) before DLQ")

	// 5. Following valid message is still processed normally.
	shouldFail.Store(false)
	handlerAttempts.Store(0)

	validEnv := worker.EventEnvelope[worker.ACHTransferRequestedPayload]{
		EventID:       "evt-valid-after-poison",
		EventType:     inTopic,
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "corr-valid-after-poison",
		Payload: worker.ACHTransferRequestedPayload{
			TransferID:   "ach-valid-1",
			CustomerID:   "cust-ok",
			PortfolioID:  "port-ok",
			BankAccount:  "acct-ok-1234",
			AmountUSD:    420.0,
			TransferType: "deposit",
		},
	}
	validBytes, err := json.Marshal(validEnv)
	require.NoError(t, err)
	require.NoError(t, pubSub.Publish(inTopic, message.NewMessage("msg-valid-1", validBytes)))

	select {
	case outMsg := <-outCh:
		outMsg.Ack()
		var settled worker.EventEnvelope[worker.ACHSettledPayload]
		require.NoError(t, json.Unmarshal(outMsg.Payload, &settled))
		assert.Equal(t, "ach-valid-1", settled.Payload.TransferID)
		assert.Equal(t, "settled", settled.Payload.Status)
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for valid message after poison messages")
	}
	assert.Equal(t, int32(1), handlerAttempts.Load())

	// 6. CWE-532 assertion: captured logs contain expected metadata and ZERO payload bytes.
	logs := logBuf.String()
	assert.Contains(t, logs, inTopic)
	assert.Contains(t, logs, "corr-missing-id")
	assert.Contains(t, logs, "evt-missing-ts")
	assert.Contains(t, logs, "corr-missing-ts")
	assert.Contains(t, logs, "evt-handler-fail")
	assert.Contains(t, logs, "corr-handler-fail")

	assert.NotContains(t, logs, secretMalformed, "malformed payload bytes must never appear in logs (CWE-532)")
	assert.NotContains(t, logs, secretMissingID, "missing-event_id payload bytes must never appear in logs (CWE-532)")
	assert.NotContains(t, logs, secretMissingTs, "missing-occurred_at payload bytes must never appear in logs (CWE-532)")
	assert.NotContains(t, logs, secretHandlerErr, "handler-error payload bytes must never appear in logs (CWE-532)")
}

func TestRouter_TransientErrorRecoversOnRetry(t *testing.T) {
	const (
		inTopic  = "events.banking.ach_transfer_requested.v1"
		outTopic = "events.banking.ach_settled.v1"
		dlqTopic = inTopic + ".dlq"
	)

	logBuf := &lockedBuffer{}
	logger := slog.New(slog.NewJSONHandler(logBuf, nil))
	wmLogger := watermill.NopLogger{}
	pubSub := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 20}, wmLogger)

	var attempts atomic.Int32
	mockBLL := &mockCustomerBLL{
		settleFunc: func(ctx context.Context, in *customerbllv1.SettleACHTransferRequest) (*customerbllv1.SettleACHTransferResponse, error) {
			if attempts.Add(1) < 3 {
				return nil, errors.New("transient blip")
			}
			return &customerbllv1.SettleACHTransferResponse{
				TransferID: in.TransferID,
				Status:     "settled",
				SettledAt:  time.Now().UTC().Format(time.RFC3339),
			}, nil
		},
	}

	fastRetry := middleware.Retry{
		MaxRetries:      2,
		InitialInterval: 10 * time.Millisecond,
		MaxInterval:     50 * time.Millisecond,
		Multiplier:      2.0,
		Logger:          wmLogger,
	}

	router, err := worker.CreateRouterWithRetry(pubSub, wmLogger, logger, mockBLL, nil, "", "", nil, fastRetry)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	outCh, err := pubSub.Subscribe(ctx, outTopic)
	require.NoError(t, err)
	dlqCh, err := pubSub.Subscribe(ctx, dlqTopic)
	require.NoError(t, err)

	go func() {
		_ = router.Run(ctx)
	}()
	<-router.Running()
	defer func() { _ = router.Close() }()

	env := worker.EventEnvelope[worker.ACHTransferRequestedPayload]{
		EventID:       "evt-retry-ok",
		EventType:     inTopic,
		OccurredAt:    time.Now().UTC(),
		CorrelationID: "corr-retry-ok",
		Payload: worker.ACHTransferRequestedPayload{
			TransferID:   "ach-retry-ok",
			AmountUSD:    75.0,
			TransferType: "deposit",
		},
	}
	data, err := json.Marshal(env)
	require.NoError(t, err)
	require.NoError(t, pubSub.Publish(inTopic, message.NewMessage("msg-retry-ok", data)))

	select {
	case outMsg := <-outCh:
		outMsg.Ack()
		assert.Equal(t, int32(3), attempts.Load(), "should succeed on 3rd attempt")
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for recovered message on output topic")
	}

	select {
	case dlqMsg := <-dlqCh:
		dlqMsg.Ack()
		t.Fatal("recovered message must not be routed to DLQ")
	case <-time.After(100 * time.Millisecond):
	}
}

type initializingPubSub struct {
	worker.PubSub
	mu          sync.Mutex
	initialized []string
}

func (p *initializingPubSub) SubscribeInitialize(topic string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.initialized = append(p.initialized, topic)
	return nil
}

func TestRouter_PreDeclaresDLQTopics(t *testing.T) {
	wmLogger := watermill.NopLogger{}
	logger := slog.New(slog.NewJSONHandler(&lockedBuffer{}, nil))
	base := gochannel.NewGoChannel(gochannel.Config{OutputChannelBuffer: 10}, wmLogger)
	ps := &initializingPubSub{PubSub: base}

	_, err := worker.CreateRouter(ps, wmLogger, logger, &mockCustomerBLL{}, &mockInternalBLL{}, "", "", nil)
	require.NoError(t, err)

	ps.mu.Lock()
	defer ps.mu.Unlock()
	expectedDLQs := []string{
		"events.observation.cat_spotted.v1.dlq",
		"events.investment.backtest_requested.v1.dlq",
		"events.banking.ach_transfer_requested.v1.dlq",
		"events.facilities.ota_triggered.v1.dlq",
		"events.reporting.generate_monthly_requested.v1.dlq",
		"events.workforce.incident_created.v1.dlq",
	}
	assert.ElementsMatch(t, expectedDLQs, ps.initialized)
}
