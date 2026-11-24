-- 003-transactional-outbox.sql
-- Idempotent migration adding schema-scoped transactional SQL outbox tables
-- across workforce, facilities, core_invest, and ops schemas (Phase 10 / 7.40).

CREATE TABLE IF NOT EXISTS workforce.outbox_events (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL UNIQUE,
    topic VARCHAR(200) NOT NULL,
    aggregate_id VARCHAR(128) NOT NULL DEFAULT '',
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'published', 'failed')),
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_workforce_outbox_pending
    ON workforce.outbox_events (created_at ASC)
    WHERE published_at IS NULL;

CREATE TABLE IF NOT EXISTS facilities.outbox_events (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL UNIQUE,
    topic VARCHAR(200) NOT NULL,
    aggregate_id VARCHAR(128) NOT NULL DEFAULT '',
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'published', 'failed')),
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_facilities_outbox_pending
    ON facilities.outbox_events (created_at ASC)
    WHERE published_at IS NULL;

CREATE TABLE IF NOT EXISTS core_invest.outbox_events (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL UNIQUE,
    topic VARCHAR(200) NOT NULL,
    aggregate_id VARCHAR(128) NOT NULL DEFAULT '',
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'published', 'failed')),
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_core_invest_outbox_pending
    ON core_invest.outbox_events (created_at ASC)
    WHERE published_at IS NULL;

CREATE TABLE IF NOT EXISTS ops.outbox_events (
    id BIGSERIAL PRIMARY KEY,
    event_id UUID NOT NULL UNIQUE,
    topic VARCHAR(200) NOT NULL,
    aggregate_id VARCHAR(128) NOT NULL DEFAULT '',
    payload JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'published', 'failed')),
    attempts INT NOT NULL DEFAULT 0,
    last_error TEXT,
    locked_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS idx_ops_outbox_pending
    ON ops.outbox_events (created_at ASC)
    WHERE published_at IS NULL;
