-- Standard Small Business IT Operations & Issue Tracking Schema
CREATE SCHEMA IF NOT EXISTS ops;

CREATE TABLE IF NOT EXISTS ops.it_tickets (
    ticket_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    forgejo_issue_id BIGINT UNIQUE,
    forgejo_repo VARCHAR(200) NOT NULL,
    title VARCHAR(255) NOT NULL,
    body TEXT,
    state VARCHAR(20) NOT NULL DEFAULT 'open',
    author_username VARCHAR(100) NOT NULL,
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);
