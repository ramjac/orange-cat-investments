-- Orange Cat Investments (OCI) Platform Database Initialization Script
-- PostgreSQL 16 Architecture & Schema Layout

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Function to generate RFC 4122 compliant UUIDv7 in PostgreSQL 16
CREATE OR REPLACE FUNCTION gen_random_uuid_v7()
RETURNS uuid AS $$
DECLARE
    v_time timestamp with time zone := clock_timestamp();
    v_secs bigint := extract(epoch from v_time);
    v_msec bigint := (extract(milliseconds from v_time)::bigint) % 1000;
    v_timestamp bigint := (v_secs * 1000) + v_msec;
    v_timestamp_hex text := lpad(to_hex(v_timestamp), 12, '0');
    v_bytes bytea := decode(v_timestamp_hex || encode(gen_random_bytes(10), 'hex'), 'hex');
BEGIN
    -- Set version to 7 (bits 48-51 to 0111)
    v_bytes := set_byte(v_bytes, 6, (get_byte(v_bytes, 6) & 15) | 112);
    -- Set variant to RFC 4122 (bits 64-65 to 10)
    v_bytes := set_byte(v_bytes, 8, (get_byte(v_bytes, 8) & 63) | 128);
    RETURN encode(v_bytes, 'hex')::uuid;
END;
$$ LANGUAGE plpgsql VOLATILE;

-- Create Isolated Schemas
CREATE SCHEMA IF NOT EXISTS workforce;
CREATE SCHEMA IF NOT EXISTS facilities;
CREATE SCHEMA IF NOT EXISTS core_invest;
CREATE SCHEMA IF NOT EXISTS ops;

--------------------------------------------------------------------------------
-- WORKFORCE SCHEMA
--------------------------------------------------------------------------------

-- Employees Table (Human staff & Feline employees)
CREATE TABLE IF NOT EXISTS workforce.employees (
    employee_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    employee_type VARCHAR(20) NOT NULL CHECK (employee_type IN ('human', 'feline')),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100),
    email VARCHAR(255) UNIQUE,
    role_title VARCHAR(100) NOT NULL,
    department VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'onboarding' CHECK (status IN ('onboarding', 'active', 'on_leave', 'separated', 'retired')),
    hired_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_human_email CHECK (employee_type <> 'human' OR email IS NOT NULL)
);

-- Onboarding Checklists
CREATE TABLE IF NOT EXISTS workforce.onboarding_checklists (
    checklist_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    employee_id UUID NOT NULL REFERENCES workforce.employees(employee_id) ON DELETE CASCADE,
    task_name VARCHAR(255) NOT NULL,
    category VARCHAR(50) NOT NULL CHECK (category IN ('hardware', 'access', 'dietary', 'medical', 'training')),
    is_completed BOOLEAN NOT NULL DEFAULT FALSE,
    completed_at TIMESTAMPTZ,
    assigned_to UUID REFERENCES workforce.employees(employee_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Feline Care Schedules
CREATE TABLE IF NOT EXISTS workforce.care_schedules (
    schedule_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    feline_id UUID NOT NULL REFERENCES workforce.employees(employee_id) ON DELETE CASCADE,
    dietary_plan TEXT NOT NULL,
    feeding_times TEXT[] NOT NULL,
    special_medical_needs TEXT,
    preferred_perch_zone VARCHAR(100),
    caretaker_id UUID REFERENCES workforce.employees(employee_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Review Cycles (Performance reviews for humans, Health/Care assessments for felines)
CREATE TABLE IF NOT EXISTS workforce.review_cycles (
    review_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    employee_id UUID NOT NULL REFERENCES workforce.employees(employee_id) ON DELETE CASCADE,
    review_type VARCHAR(30) NOT NULL CHECK (review_type IN ('performance', 'feline_health_assessment')),
    scheduled_for DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'in_progress', 'completed', 'overdue')),
    reviewer_id UUID REFERENCES workforce.employees(employee_id),
    score DECIMAL(3, 2),
    notes TEXT,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Leave Requests
CREATE TABLE IF NOT EXISTS workforce.leave_requests (
    leave_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    employee_id UUID NOT NULL REFERENCES workforce.employees(employee_id) ON DELETE CASCADE,
    leave_type VARCHAR(30) NOT NULL CHECK (leave_type IN ('vacation', 'sick', 'catnip_break', 'sabbatical')),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_leave_dates CHECK (end_date >= start_date)
);

--------------------------------------------------------------------------------
-- FACILITIES SCHEMA
--------------------------------------------------------------------------------

-- Location Zones
CREATE TABLE IF NOT EXISTS facilities.location_zones (
    zone_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    name VARCHAR(100) NOT NULL UNIQUE,
    building VARCHAR(100) NOT NULL,
    floor_level INT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Hardware Assets (Edge cameras, Observation perches, Smart collars, Gateways)
CREATE TABLE IF NOT EXISTS facilities.hardware_assets (
    asset_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    serial_number VARCHAR(100) NOT NULL UNIQUE,
    asset_type VARCHAR(50) NOT NULL CHECK (asset_type IN ('edge_camera', 'observation_perch', 'smart_collar', 'gateway', 'feeder')),
    model VARCHAR(100) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'maintenance_required', 'offline', 'pending_return', 'decommissioned')),
    zone_id UUID REFERENCES facilities.location_zones(zone_id),
    assigned_employee_id UUID REFERENCES workforce.employees(employee_id),
    last_ping_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Observation Perches (Habitat specific details)
CREATE TABLE IF NOT EXISTS facilities.observation_perches (
    perch_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    asset_id UUID NOT NULL UNIQUE REFERENCES facilities.hardware_assets(asset_id) ON DELETE CASCADE,
    height_meters DECIMAL(4,2) NOT NULL,
    max_weight_kg DECIMAL(4,2) NOT NULL,
    cushion_type VARCHAR(50) NOT NULL,
    is_occupied BOOLEAN NOT NULL DEFAULT FALSE,
    current_feline_id UUID REFERENCES workforce.employees(employee_id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Maintenance Tickets
CREATE TABLE IF NOT EXISTS facilities.maintenance_tickets (
    ticket_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    asset_id UUID NOT NULL REFERENCES facilities.hardware_assets(asset_id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    priority VARCHAR(20) NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    status VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'in_progress', 'resolved', 'closed')),
    assigned_technician_id UUID REFERENCES workforce.employees(employee_id),
    reported_by UUID REFERENCES workforce.employees(employee_id),
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

--------------------------------------------------------------------------------
-- CORE_INVEST SCHEMA
--------------------------------------------------------------------------------

-- Cat Observation Events
CREATE TABLE IF NOT EXISTS core_invest.observation_events (
    event_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    camera_asset_id UUID NOT NULL REFERENCES facilities.hardware_assets(asset_id),
    feline_id UUID REFERENCES workforce.employees(employee_id),
    activity_type VARCHAR(50) NOT NULL CHECK (activity_type IN ('zooming', 'napping', 'perched', 'eating', 'grooming', 'playful_pounce')),
    confidence_score DECIMAL(5, 4) NOT NULL CHECK (confidence_score BETWEEN 0.0000 AND 1.0000),
    duration_seconds INT NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    observed_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Investment Allocation Strategies
CREATE TABLE IF NOT EXISTS core_invest.allocation_strategies (
    strategy_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    name VARCHAR(100) NOT NULL UNIQUE,
    trigger_activity VARCHAR(50) NOT NULL,
    action_type VARCHAR(20) NOT NULL CHECK (action_type IN ('buy', 'sell', 'rebalance', 'hold')),
    target_asset_symbol VARCHAR(20) NOT NULL,
    multiplier DECIMAL(6, 4) NOT NULL DEFAULT 1.0000,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Investment Portfolios
CREATE TABLE IF NOT EXISTS core_invest.investment_portfolios (
    portfolio_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    customer_id UUID NOT NULL, -- ZITADEL Subject / External Customer ID
    account_name VARCHAR(100) NOT NULL,
    total_balance_usd DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
    cash_balance_usd DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Portfolio Snapshots
CREATE TABLE IF NOT EXISTS core_invest.portfolio_snapshots (
    snapshot_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    portfolio_id UUID NOT NULL REFERENCES core_invest.investment_portfolios(portfolio_id) ON DELETE CASCADE,
    valuation_usd DECIMAL(14, 2) NOT NULL,
    holdings_summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    triggered_by_event_id UUID REFERENCES core_invest.observation_events(event_id),
    snapshot_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

--------------------------------------------------------------------------------
-- OPS SCHEMA
--------------------------------------------------------------------------------

-- IT Tickets (Ingested via Forgejo Webhooks)
CREATE TABLE IF NOT EXISTS ops.it_tickets (
    ticket_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    forgejo_issue_id BIGINT UNIQUE,
    forgejo_repo VARCHAR(200) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    state VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (state IN ('open', 'closed')),
    author_username VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
