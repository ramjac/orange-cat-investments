package customerbff

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	customerbllv1 "github.com/orange-cat-investments/oci/pkg/gen/go/customer_bll/v1"
	sessionvalkey "github.com/orange-cat-investments/oci/pkg/session/valkey"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingOrderBLLClient struct {
	*mockCustomerBLLClient
	orderCalls   atomic.Int32
	depositCalls atomic.Int32
	blockOrderCh chan struct{}
	orderStarted chan struct{}
}

func (c *countingOrderBLLClient) ValidateAndExecuteOrder(ctx context.Context, in *customerbllv1.ValidateAndExecuteOrderRequest) (*customerbllv1.ValidateAndExecuteOrderResponse, error) {
	c.orderCalls.Add(1)
	if c.orderStarted != nil {
		select {
		case c.orderStarted <- struct{}{}:
		default:
		}
	}
	if c.blockOrderCh != nil {
		<-c.blockOrderCh
	}
	return c.mockCustomerBLLClient.ValidateAndExecuteOrder(ctx, in)
}

func (c *countingOrderBLLClient) InitiateACHDeposit(ctx context.Context, in *customerbllv1.InitiateACHDepositRequest) (*customerbllv1.InitiateACHDepositResponse, error) {
	c.depositCalls.Add(1)
	return c.mockCustomerBLLClient.InitiateACHDeposit(ctx, in)
}

func setupIdempotencyTestBFF(bll *countingOrderBLLClient) *http.ServeMux {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	bff := NewCustomerBFF(bll, logger)
	sessStore := sessionvalkey.NewMemoryCustomerSessionStore()
	sessStore.SetSession(&CustomerSession{
		SessionID:  "sess-chloe-idem",
		CustomerID: "cust-active-chloe",
		Email:      "chloe@oci.local",
		Name:       "Chloe Spark",
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	})
	bff.SetSessionStore(sessStore)
	mux := http.NewServeMux()
	bff.RegisterRoutes(mux)
	return mux
}

func TestCustomerBFF_IdempotencyKey_OrdersAndDeposits(t *testing.T) {
	bll := &countingOrderBLLClient{mockCustomerBLLClient: newMockCustomerBLLClient()}
	mux := setupIdempotencyTestBFF(bll)
	csrfVal := "idem-csrf-token"

	t.Run("missing or invalid UUID Idempotency-Key returns 400", func(t *testing.T) {
		for _, badKey := range []string{"", "not-a-uuid", "12345"} {
			req := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", bytes.NewBufferString(`{"symbol":"NVDA","side":"buy","quantity":1}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", csrfVal)
			if badKey != "" {
				req.Header.Set("Idempotency-Key", badKey)
			}
			req.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-chloe-idem"})
			req.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})

			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Contains(t, rec.Body.String(), "Idempotency-Key")
		}
		assert.Equal(t, int32(0), bll.orderCalls.Load())
	})

	t.Run("duplicate order retry replays cached response without re-invoking BLL", func(t *testing.T) {
		bll.orderCalls.Store(0)
		key := "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
		payload := `{"symbol":"NVDA","side":"buy","quantity":5}`

		// First attempt
		req1 := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", bytes.NewBufferString(payload))
		req1.Header.Set("Content-Type", "application/json")
		req1.Header.Set("X-CSRF-Token", csrfVal)
		req1.Header.Set("Idempotency-Key", key)
		req1.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-chloe-idem"})
		req1.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
		rec1 := httptest.NewRecorder()
		mux.ServeHTTP(rec1, req1)

		require.Equal(t, http.StatusOK, rec1.Code)
		assert.Empty(t, rec1.Header().Get("X-Idempotent-Replayed"))
		assert.Equal(t, int32(1), bll.orderCalls.Load())

		// Replay attempt with identical Idempotency-Key
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", bytes.NewBufferString(payload))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("X-CSRF-Token", csrfVal)
		req2.Header.Set("Idempotency-Key", key)
		req2.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-chloe-idem"})
		req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
		rec2 := httptest.NewRecorder()
		mux.ServeHTTP(rec2, req2)

		require.Equal(t, http.StatusOK, rec2.Code)
		assert.Equal(t, "true", rec2.Header().Get("X-Idempotent-Replayed"))
		assert.Equal(t, rec1.Body.String(), rec2.Body.String())
		assert.Equal(t, int32(1), bll.orderCalls.Load(), "BLL must not execute duplicate order")

		var resp1, resp2 map[string]any
		require.NoError(t, json.Unmarshal(rec1.Body.Bytes(), &resp1))
		require.NoError(t, json.Unmarshal(rec2.Body.Bytes(), &resp2))
		assert.Equal(t, resp1["order_id"], resp2["order_id"])
	})

	t.Run("duplicate deposit retry replays cached 201 response without re-invoking BLL", func(t *testing.T) {
		bll.depositCalls.Store(0)
		key := "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
		payload := `{"amount_usd":250.0,"frequency":"one_time","bank_account":"Chase Checking"}`

		req1 := httptest.NewRequest(http.MethodPost, "/api/v1/customer/deposits", bytes.NewBufferString(payload))
		req1.Header.Set("Content-Type", "application/json")
		req1.Header.Set("X-CSRF-Token", csrfVal)
		req1.Header.Set("Idempotency-Key", key)
		req1.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-chloe-idem"})
		req1.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
		rec1 := httptest.NewRecorder()
		mux.ServeHTTP(rec1, req1)

		require.Equal(t, http.StatusCreated, rec1.Code)
		assert.Equal(t, int32(1), bll.depositCalls.Load())

		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/customer/deposits", bytes.NewBufferString(payload))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("X-CSRF-Token", csrfVal)
		req2.Header.Set("Idempotency-Key", key)
		req2.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-chloe-idem"})
		req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
		rec2 := httptest.NewRecorder()
		mux.ServeHTTP(rec2, req2)

		require.Equal(t, http.StatusCreated, rec2.Code)
		assert.Equal(t, "true", rec2.Header().Get("X-Idempotent-Replayed"))
		assert.Equal(t, rec1.Body.String(), rec2.Body.String())
		assert.Equal(t, int32(1), bll.depositCalls.Load())
	})

	t.Run("concurrent in-flight request with same Idempotency-Key returns 409 Conflict", func(t *testing.T) {
		blockingBLL := &countingOrderBLLClient{
			mockCustomerBLLClient: newMockCustomerBLLClient(),
			blockOrderCh:          make(chan struct{}),
			orderStarted:          make(chan struct{}, 1),
		}
		blockingMux := setupIdempotencyTestBFF(blockingBLL)
		key := "cccccccc-cccc-4ccc-8ccc-cccccccccccc"

		var wg sync.WaitGroup
		rec1 := httptest.NewRecorder()
		wg.Add(1)
		go func() {
			defer wg.Done()
			req1 := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", bytes.NewBufferString(`{"symbol":"AAPL","side":"buy","quantity":2}`))
			req1.Header.Set("Content-Type", "application/json")
			req1.Header.Set("X-CSRF-Token", csrfVal)
			req1.Header.Set("Idempotency-Key", key)
			req1.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-chloe-idem"})
			req1.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
			blockingMux.ServeHTTP(rec1, req1)
		}()

		// Wait until first request is inside ValidateAndExecuteOrder
		<-blockingBLL.orderStarted

		// Concurrent second request with identical key must receive 409 Conflict
		req2 := httptest.NewRequest(http.MethodPost, "/api/v1/customer/trading/orders", bytes.NewBufferString(`{"symbol":"AAPL","side":"buy","quantity":2}`))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("X-CSRF-Token", csrfVal)
		req2.Header.Set("Idempotency-Key", key)
		req2.AddCookie(&http.Cookie{Name: "customer_session", Value: "sess-chloe-idem"})
		req2.AddCookie(&http.Cookie{Name: "csrf_token", Value: csrfVal})
		rec2 := httptest.NewRecorder()
		blockingMux.ServeHTTP(rec2, req2)
		assert.Equal(t, http.StatusConflict, rec2.Code)

		// Unblock first request and verify it completes with 200
		close(blockingBLL.blockOrderCh)
		wg.Wait()
		assert.Equal(t, http.StatusOK, rec1.Code)
		assert.Equal(t, int32(1), blockingBLL.orderCalls.Load())
	})
}
