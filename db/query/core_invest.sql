-- name: GetCameraStreamByID :one
SELECT stream_id, camera_asset_id, stream_url, protocol, status, ingest_settings, created_at, updated_at
FROM core_invest.camera_streams
WHERE stream_id = $1;

-- name: ListCameraStreams :many
SELECT stream_id, camera_asset_id, stream_url, protocol, status, ingest_settings, created_at, updated_at
FROM core_invest.camera_streams
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateCameraStream :one
INSERT INTO core_invest.camera_streams (
    camera_asset_id, stream_url, protocol, status, ingest_settings
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING stream_id, camera_asset_id, stream_url, protocol, status, ingest_settings, created_at, updated_at;

-- name: GetBrokerageOrderByID :one
SELECT order_id, portfolio_id, strategy_id, event_id, broker_name, symbol, side, quantity, price, status, executed_at, created_at, updated_at
FROM core_invest.brokerage_orders
WHERE order_id = $1;

-- name: ListBrokerageOrders :many
SELECT order_id, portfolio_id, strategy_id, event_id, broker_name, symbol, side, quantity, price, status, executed_at, created_at, updated_at
FROM core_invest.brokerage_orders
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateBrokerageOrder :one
INSERT INTO core_invest.brokerage_orders (
    portfolio_id, strategy_id, event_id, broker_name, symbol, side, quantity, price, status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING order_id, portfolio_id, strategy_id, event_id, broker_name, symbol, side, quantity, price, status, executed_at, created_at, updated_at;

-- name: UpdateBrokerageOrderStatus :one
UPDATE core_invest.brokerage_orders
SET status = $2, executed_at = $3, updated_at = CURRENT_TIMESTAMP
WHERE order_id = $1
RETURNING order_id, portfolio_id, strategy_id, event_id, broker_name, symbol, side, quantity, price, status, executed_at, created_at, updated_at;

-- name: GetBacktestRunByID :one
SELECT backtest_id, strategy_id, start_date, end_date, parameters, results, status, created_at, updated_at
FROM core_invest.backtest_runs
WHERE backtest_id = $1;

-- name: ListBacktestRuns :many
SELECT backtest_id, strategy_id, start_date, end_date, parameters, results, status, created_at, updated_at
FROM core_invest.backtest_runs
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateBacktestRun :one
INSERT INTO core_invest.backtest_runs (
    strategy_id, start_date, end_date, parameters, status
) VALUES (
    $1, $2, $3, $4, 'pending'
) RETURNING backtest_id, strategy_id, start_date, end_date, parameters, results, status, created_at, updated_at;

-- name: UpdateBacktestRunResults :one
UPDATE core_invest.backtest_runs
SET status = $2, results = $3, updated_at = CURRENT_TIMESTAMP
WHERE backtest_id = $1
RETURNING backtest_id, strategy_id, start_date, end_date, parameters, results, status, created_at, updated_at;
