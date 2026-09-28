-- OCI Feline Observation Telemetry & Algorithmic Investment Domain Overlay
CREATE SCHEMA IF NOT EXISTS core_invest;

CREATE TABLE IF NOT EXISTS core_invest.observation_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    camera_asset_id UUID NOT NULL REFERENCES facilities.hardware_assets(asset_id),
    feline_id UUID REFERENCES workforce.employees(employee_id),
    activity_type VARCHAR(50) NOT NULL,
    confidence_score DECIMAL(5, 4) NOT NULL,
    duration_seconds INT NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    observed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS core_invest.investment_portfolios (
    portfolio_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    customer_id UUID NOT NULL,
    account_name VARCHAR(100) NOT NULL,
    total_balance_usd DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
    cash_balance_usd DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
