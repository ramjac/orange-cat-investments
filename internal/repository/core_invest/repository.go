package core_invest

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type CameraStream struct {
	StreamID       string    `json:"stream_id"`
	CameraAssetID  string    `json:"camera_asset_id"`
	StreamURL      string    `json:"stream_url"`
	Protocol       string    `json:"protocol"`
	Status         string    `json:"status"`
	IngestSettings string    `json:"ingest_settings"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type BrokerageOrder struct {
	OrderID     string     `json:"order_id"`
	PortfolioID string     `json:"portfolio_id"`
	StrategyID  *string    `json:"strategy_id,omitempty"`
	EventID     *string    `json:"event_id,omitempty"`
	BrokerName  string     `json:"broker_name"`
	Symbol      string     `json:"symbol"`
	Side        string     `json:"side"`
	Quantity    float64    `json:"quantity"`
	Price       float64    `json:"price"`
	Status      string     `json:"status"`
	ExecutedAt  *time.Time `json:"executed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type BacktestRun struct {
	BacktestID string    `json:"backtest_id"`
	StrategyID string    `json:"strategy_id"`
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	Parameters string    `json:"parameters"`
	Results    string    `json:"results"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Repository interface {
	GetCameraStreamByID(ctx context.Context, id string) (*CameraStream, error)
	ListCameraStreams(ctx context.Context, limit, offset int32) ([]*CameraStream, error)
	CreateCameraStream(ctx context.Context, stream *CameraStream) (*CameraStream, error)

	GetBrokerageOrderByID(ctx context.Context, id string) (*BrokerageOrder, error)
	ListBrokerageOrders(ctx context.Context, limit, offset int32) ([]*BrokerageOrder, error)
	CreateBrokerageOrder(ctx context.Context, order *BrokerageOrder) (*BrokerageOrder, error)
	UpdateBrokerageOrderStatus(ctx context.Context, id string, status string, executedAt *time.Time) (*BrokerageOrder, error)

	GetBacktestRunByID(ctx context.Context, id string) (*BacktestRun, error)
	ListBacktestRuns(ctx context.Context, limit, offset int32) ([]*BacktestRun, error)
	CreateBacktestRun(ctx context.Context, run *BacktestRun) (*BacktestRun, error)
	UpdateBacktestRunResults(ctx context.Context, id string, status string, results string) (*BacktestRun, error)
}

type pgxRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) Repository {
	return &pgxRepository{db: db}
}

func (r *pgxRepository) GetCameraStreamByID(ctx context.Context, id string) (*CameraStream, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	var s CameraStream
	query := `SELECT stream_id, camera_asset_id, stream_url, protocol, status, ingest_settings, created_at, updated_at
              FROM core_invest.camera_streams WHERE stream_id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(&s.StreamID, &s.CameraAssetID, &s.StreamURL, &s.Protocol, &s.Status, &s.IngestSettings, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *pgxRepository) ListCameraStreams(ctx context.Context, limit, offset int32) ([]*CameraStream, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT stream_id, camera_asset_id, stream_url, protocol, status, ingest_settings, created_at, updated_at
              FROM core_invest.camera_streams ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var streams []*CameraStream
	for rows.Next() {
		var s CameraStream
		if err := rows.Scan(&s.StreamID, &s.CameraAssetID, &s.StreamURL, &s.Protocol, &s.Status, &s.IngestSettings, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		streams = append(streams, &s)
	}
	return streams, rows.Err()
}

func (r *pgxRepository) CreateCameraStream(ctx context.Context, stream *CameraStream) (*CameraStream, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO core_invest.camera_streams (
                  camera_asset_id, stream_url, protocol, status, ingest_settings
              ) VALUES ($1, $2, $3, $4, $5)
              RETURNING stream_id, camera_asset_id, stream_url, protocol, status, ingest_settings, created_at, updated_at`
	var s CameraStream
	err := r.db.QueryRow(ctx, query, stream.CameraAssetID, stream.StreamURL, stream.Protocol, stream.Status, stream.IngestSettings).
		Scan(&s.StreamID, &s.CameraAssetID, &s.StreamURL, &s.Protocol, &s.Status, &s.IngestSettings, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *pgxRepository) GetBrokerageOrderByID(ctx context.Context, id string) (*BrokerageOrder, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	var o BrokerageOrder
	query := `SELECT order_id, portfolio_id, strategy_id, event_id, broker_name, symbol, side, quantity, price, status, executed_at, created_at, updated_at
              FROM core_invest.brokerage_orders WHERE order_id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(&o.OrderID, &o.PortfolioID, &o.StrategyID, &o.EventID, &o.BrokerName, &o.Symbol, &o.Side, &o.Quantity, &o.Price, &o.Status, &o.ExecutedAt, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *pgxRepository) ListBrokerageOrders(ctx context.Context, limit, offset int32) ([]*BrokerageOrder, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT order_id, portfolio_id, strategy_id, event_id, broker_name, symbol, side, quantity, price, status, executed_at, created_at, updated_at
              FROM core_invest.brokerage_orders ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*BrokerageOrder
	for rows.Next() {
		var o BrokerageOrder
		if err := rows.Scan(&o.OrderID, &o.PortfolioID, &o.StrategyID, &o.EventID, &o.BrokerName, &o.Symbol, &o.Side, &o.Quantity, &o.Price, &o.Status, &o.ExecutedAt, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		orders = append(orders, &o)
	}
	return orders, rows.Err()
}

func (r *pgxRepository) CreateBrokerageOrder(ctx context.Context, order *BrokerageOrder) (*BrokerageOrder, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO core_invest.brokerage_orders (
                  portfolio_id, strategy_id, event_id, broker_name, symbol, side, quantity, price, status
              ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
              RETURNING order_id, portfolio_id, strategy_id, event_id, broker_name, symbol, side, quantity, price, status, executed_at, created_at, updated_at`
	var o BrokerageOrder
	err := r.db.QueryRow(ctx, query, order.PortfolioID, order.StrategyID, order.EventID, order.BrokerName, order.Symbol, order.Side, order.Quantity, order.Price, order.Status).
		Scan(&o.OrderID, &o.PortfolioID, &o.StrategyID, &o.EventID, &o.BrokerName, &o.Symbol, &o.Side, &o.Quantity, &o.Price, &o.Status, &o.ExecutedAt, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *pgxRepository) UpdateBrokerageOrderStatus(ctx context.Context, id string, status string, executedAt *time.Time) (*BrokerageOrder, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `UPDATE core_invest.brokerage_orders
              SET status = $2, executed_at = $3, updated_at = CURRENT_TIMESTAMP
              WHERE order_id = $1
              RETURNING order_id, portfolio_id, strategy_id, event_id, broker_name, symbol, side, quantity, price, status, executed_at, created_at, updated_at`
	var o BrokerageOrder
	err := r.db.QueryRow(ctx, query, id, status, executedAt).
		Scan(&o.OrderID, &o.PortfolioID, &o.StrategyID, &o.EventID, &o.BrokerName, &o.Symbol, &o.Side, &o.Quantity, &o.Price, &o.Status, &o.ExecutedAt, &o.CreatedAt, &o.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &o, nil
}

func (r *pgxRepository) GetBacktestRunByID(ctx context.Context, id string) (*BacktestRun, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	var b BacktestRun
	query := `SELECT backtest_id, strategy_id, start_date, end_date, parameters, results, status, created_at, updated_at
              FROM core_invest.backtest_runs WHERE backtest_id = $1`
	err := r.db.QueryRow(ctx, query, id).Scan(&b.BacktestID, &b.StrategyID, &b.StartDate, &b.EndDate, &b.Parameters, &b.Results, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *pgxRepository) ListBacktestRuns(ctx context.Context, limit, offset int32) ([]*BacktestRun, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `SELECT backtest_id, strategy_id, start_date, end_date, parameters, results, status, created_at, updated_at
              FROM core_invest.backtest_runs ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var runs []*BacktestRun
	for rows.Next() {
		var b BacktestRun
		if err := rows.Scan(&b.BacktestID, &b.StrategyID, &b.StartDate, &b.EndDate, &b.Parameters, &b.Results, &b.Status, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		runs = append(runs, &b)
	}
	return runs, rows.Err()
}

func (r *pgxRepository) CreateBacktestRun(ctx context.Context, run *BacktestRun) (*BacktestRun, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `INSERT INTO core_invest.backtest_runs (
                  strategy_id, start_date, end_date, parameters, status
              ) VALUES ($1, $2, $3, $4, 'pending')
              RETURNING backtest_id, strategy_id, start_date, end_date, parameters, results, status, created_at, updated_at`
	var b BacktestRun
	err := r.db.QueryRow(ctx, query, run.StrategyID, run.StartDate, run.EndDate, run.Parameters).
		Scan(&b.BacktestID, &b.StrategyID, &b.StartDate, &b.EndDate, &b.Parameters, &b.Results, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *pgxRepository) UpdateBacktestRunResults(ctx context.Context, id string, status string, results string) (*BacktestRun, error) {
	if r.db == nil {
		return nil, fmt.Errorf("database connection is nil")
	}
	query := `UPDATE core_invest.backtest_runs
              SET status = $2, results = $3, updated_at = CURRENT_TIMESTAMP
              WHERE backtest_id = $1
              RETURNING backtest_id, strategy_id, start_date, end_date, parameters, results, status, created_at, updated_at`
	var b BacktestRun
	err := r.db.QueryRow(ctx, query, id, status, results).
		Scan(&b.BacktestID, &b.StrategyID, &b.StartDate, &b.EndDate, &b.Parameters, &b.Results, &b.Status, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &b, nil
}
