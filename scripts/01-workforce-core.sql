-- Standard Small Business Workforce Schema
CREATE SCHEMA IF NOT EXISTS workforce;

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
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS workforce.leave_requests (
    leave_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    employee_id UUID NOT NULL REFERENCES workforce.employees(employee_id) ON DELETE CASCADE,
    leave_type VARCHAR(30) NOT NULL CHECK (leave_type IN ('vacation', 'sick', 'catnip_break', 'sabbatical')),
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
