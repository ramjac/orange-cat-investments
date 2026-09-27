package core_invest_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	handler "github.com/orange-cat-investments/oci/internal/handler/core_invest"
	repo "github.com/orange-cat-investments/oci/internal/repository/core_invest"
	service "github.com/orange-cat-investments/oci/internal/service/core_invest"
	"github.com/stretchr/testify/assert"
)

func TestCoreInvestHandler(t *testing.T) {
	repository := repo.NewMockRepository()
	svc := service.NewService(repository)
	h := handler.NewHandler(svc)

	t.Run("ListCameraStreams", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/core-invest/streams", nil)
		rec := httptest.NewRecorder()

		h.ListCameraStreams(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "stream-uuid-001")
	})

	t.Run("RegisterCameraStream", func(t *testing.T) {
		body := handler.CreateStreamRequest{
			CameraAssetID: "cam-01",
			StreamURL:     "rtsp://cam/live",
			Protocol:      "rtsp",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/core-invest/streams", bytes.NewBuffer(jsonBody))
		rec := httptest.NewRecorder()

		h.RegisterCameraStream(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Contains(t, rec.Body.String(), "stream-uuid-created")
	})

	t.Run("GetCameraStreamByID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/core-invest/streams/id?id=stream-uuid-001", nil)
		rec := httptest.NewRecorder()

		h.GetCameraStreamByID(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "stream-uuid-001")
	})

	t.Run("ExecuteBrokerageOrder", func(t *testing.T) {
		body := handler.CreateOrderRequest{
			PortfolioID: "port-01",
			BrokerName:  "InteractiveBrokers",
			Symbol:      "ORNG",
			Side:        "buy",
			Quantity:    100,
			Price:       50,
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/core-invest/brokerage/orders", bytes.NewBuffer(jsonBody))
		rec := httptest.NewRecorder()

		h.ExecuteBrokerageOrder(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Contains(t, rec.Body.String(), "executed")
	})

	t.Run("GetBrokerageOrderByID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/core-invest/brokerage/orders/id?id=order-uuid-001", nil)
		rec := httptest.NewRecorder()

		h.GetBrokerageOrderByID(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "order-uuid-001")
	})

	t.Run("StartBacktestRun", func(t *testing.T) {
		body := handler.CreateBacktestRequest{
			StrategyID: "strat-01",
			StartDate:  time.Now().Add(-24 * time.Hour),
			EndDate:    time.Now(),
			Parameters: "{}",
		}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/core-invest/backtesting/runs", bytes.NewBuffer(jsonBody))
		rec := httptest.NewRecorder()

		h.StartBacktestRun(rec, req)

		assert.Equal(t, http.StatusCreated, rec.Code)
		assert.Contains(t, rec.Body.String(), "backtest-uuid-created")
	})

	t.Run("GetBacktestRunByID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/core-invest/backtesting/runs/id?id=backtest-uuid-001", nil)
		rec := httptest.NewRecorder()

		h.GetBacktestRunByID(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "backtest-uuid-001")
	})

	t.Run("RegisterRoutes", func(t *testing.T) {
		mux := http.NewServeMux()
		h.RegisterRoutes(mux)

		req := httptest.NewRequest(http.MethodGet, "/core-invest/streams/stream-uuid-001", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "stream-uuid-001")
	})
}
