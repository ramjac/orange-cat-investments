-- Standard Small Business IT Operations & Issue Tracking Schema
CREATE SCHEMA IF NOT EXISTS ops;

CREATE TABLE IF NOT EXISTS ops.it_tickets (
    ticket_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    forgejo_issue_id BIGINT UNIQUE,
    forgejo_repo VARCHAR(200) NOT NULL DEFAULT 'oci-internal/helpdesk',
    title VARCHAR(255) NOT NULL,
    body TEXT,
    state VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (state IN ('open', 'closed', 'acknowledged', 'in_progress', 'resolved')),
    severity VARCHAR(20) NOT NULL DEFAULT 'medium' CHECK (severity IN ('low', 'medium', 'high', 'urgent', 'critical')),
    category VARCHAR(50) NOT NULL DEFAULT 'software' CHECK (category IN ('hardware', 'software', 'habitat', 'trading', 'general')),
    source VARCHAR(20) NOT NULL DEFAULT 'internal' CHECK (source IN ('internal', 'customer', 'system')),
    customer_id VARCHAR(100),
    assignee_username VARCHAR(100),
    author_username VARCHAR(100) NOT NULL,
    resolution_notes TEXT,
    is_urgent BOOLEAN NOT NULL DEFAULT FALSE,
    acknowledged_at TIMESTAMPTZ,
    acknowledged_by VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ops.ticket_comments (
    comment_id UUID PRIMARY KEY DEFAULT gen_random_uuid_v7(),
    ticket_id UUID NOT NULL REFERENCES ops.it_tickets(ticket_id) ON DELETE CASCADE,
    author_username VARCHAR(100) NOT NULL,
    comment TEXT NOT NULL,
    is_internal BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

