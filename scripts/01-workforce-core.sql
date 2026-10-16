-- Standard Small Business Workforce Schema
CREATE SCHEMA IF NOT EXISTS workforce;

CREATE TABLE IF NOT EXISTS workforce.employees (
    employee_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    employee_type VARCHAR(20) NOT NULL CHECK (employee_type IN ('full_time', 'part_time', 'contractor', 'intern', 'human')),
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100),
    email VARCHAR(255) UNIQUE,
    role_title VARCHAR(100) NOT NULL,
    department VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'onboarding' CHECK (status IN ('onboarding', 'active', 'on_leave', 'separated', 'retired')),
    hired_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS workforce.leave_requests (
    leave_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    employee_id UUID NOT NULL REFERENCES workforce.employees(employee_id) ON DELETE CASCADE,
    leave_type VARCHAR(30) NOT NULL CHECK (leave_type IN ('vacation', 'sick', 'personal', 'bereavement', 'parental', 'sabbatical')),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS workforce.executive_reports (
    report_id VARCHAR(64) PRIMARY KEY,
    period VARCHAR(16) NOT NULL,
    title VARCHAR(255) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'archived',
    total_revenue NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    operating_expenses NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    payroll_expenses NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    net_profit NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    operational_burn NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    cash_reserves NUMERIC(15, 2) NOT NULL DEFAULT 0.00,
    runway_months NUMERIC(6, 2) NOT NULL DEFAULT 0.00,
    total_headcount INT NOT NULL DEFAULT 0,
    human_headcount INT NOT NULL DEFAULT 0,
    feline_headcount INT NOT NULL DEFAULT 0,
    attendance_rate NUMERIC(5, 2) NOT NULL DEFAULT 0.00,
    overtime_hours NUMERIC(8, 2) NOT NULL DEFAULT 0.00,
    timesheets_approved INT NOT NULL DEFAULT 0,
    webdav_path_pdf VARCHAR(512),
    webdav_path_csv VARCHAR(512),
    csv_content TEXT,
    pdf_content BYTEA,
    generated_by VARCHAR(64) NOT NULL DEFAULT 'system-reporting-worker',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_workforce_executive_reports_period ON workforce.executive_reports(period);
