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

CREATE TABLE IF NOT EXISTS core_invest.portfolio_holdings (
    holding_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    portfolio_id UUID NOT NULL REFERENCES core_invest.investment_portfolios(portfolio_id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL,
    quantity DECIMAL(14, 4) NOT NULL DEFAULT 0.0000,
    cost_basis_usd DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
    avg_price_usd DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
    current_price_usd DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
    market_value_usd DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
    unrealized_gain_usd DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
    unrealized_gain_pct DECIMAL(7, 4) NOT NULL DEFAULT 0.0000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT uq_portfolio_symbol UNIQUE (portfolio_id, symbol)
);

CREATE TABLE IF NOT EXISTS core_invest.tax_lot_sales (
    lot_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    portfolio_id UUID NOT NULL REFERENCES core_invest.investment_portfolios(portfolio_id) ON DELETE CASCADE,
    symbol VARCHAR(20) NOT NULL,
    description VARCHAR(100) NOT NULL,
    quantity DECIMAL(14, 4) NOT NULL DEFAULT 0.0000,
    date_acquired DATE NOT NULL,
    date_sold DATE NOT NULL,
    gross_proceeds_usd DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
    cost_basis_usd DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
    wash_sale_loss_disallowed DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
    realized_gain_loss_usd DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
    is_short_term BOOLEAN NOT NULL DEFAULT TRUE,
    tax_year INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS core_invest.investment_statements (
    statement_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    portfolio_id UUID NOT NULL REFERENCES core_invest.investment_portfolios(portfolio_id) ON DELETE CASCADE,
    customer_id VARCHAR(100) NOT NULL,
    statement_type VARCHAR(50) NOT NULL CHECK (statement_type IN ('monthly_summary', 'form_1099b')),
    period VARCHAR(50) NOT NULL,
    tax_year INT NOT NULL,
    title VARCHAR(200) NOT NULL,
    file_name VARCHAR(200) NOT NULL,
    total_proceeds_usd DECIMAL(14,2) NOT NULL DEFAULT 0.00,
    net_gain_usd DECIMAL(14,2) NOT NULL DEFAULT 0.00,
    status VARCHAR(20) NOT NULL DEFAULT 'available' CHECK (status IN ('available', 'generating', 'archived')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS core_invest.gdpr_erasure_certificates (
    certificate_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    customer_id_hash VARCHAR(128) NOT NULL,
    customer_email_hash VARCHAR(128) NOT NULL,
    legal_basis VARCHAR(100) NOT NULL DEFAULT 'GDPR Article 17',
    supervisory_authority VARCHAR(100) NOT NULL DEFAULT 'Orange Cat Investments Data Protection Office',
    tables_scrubbed TEXT[] NOT NULL DEFAULT '{}',
    sessions_purged_count INT NOT NULL DEFAULT 0,
    records_scrubbed_count INT NOT NULL DEFAULT 0,
    digital_signature VARCHAR(128) NOT NULL,
    compliance_statement TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Feline workforce extensions
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'workforce' AND table_name = 'employees') THEN
        ALTER TABLE workforce.employees DROP CONSTRAINT IF EXISTS employees_employee_type_check;
        ALTER TABLE workforce.employees ADD CONSTRAINT employees_employee_type_check 
            CHECK (employee_type IN ('full_time', 'part_time', 'contractor', 'intern', 'human', 'feline'));
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'workforce' AND table_name = 'leave_requests') THEN
        ALTER TABLE workforce.leave_requests DROP CONSTRAINT IF EXISTS leave_requests_leave_type_check;
        ALTER TABLE workforce.leave_requests ADD CONSTRAINT leave_requests_leave_type_check 
            CHECK (leave_type IN ('vacation', 'sick', 'personal', 'bereavement', 'parental', 'sabbatical', 'catnip_break'));
    END IF;
END $$;

