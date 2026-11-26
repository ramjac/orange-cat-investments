-- Phase 10 / 7.41: Tamper-Evident Cryptographic Hash-Chain Audit Log
-- Append-only ledger in ops.audit_logs with SHA-256 hash chaining and
-- PostgreSQL BEFORE UPDATE/DELETE/TRUNCATE immutability triggers.

-- +goose Up
CREATE SEQUENCE IF NOT EXISTS ops.audit_logs_id_seq;

CREATE TABLE IF NOT EXISTS ops.audit_logs (
    id BIGINT PRIMARY KEY,
    actor_id VARCHAR(128) NOT NULL,
    actor_role VARCHAR(64) NOT NULL DEFAULT '',
    action VARCHAR(128) NOT NULL,
    resource_type VARCHAR(64) NOT NULL DEFAULT '',
    resource_id VARCHAR(128) NOT NULL DEFAULT '',
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    previous_hash VARCHAR(64) NOT NULL,
    record_hash VARCHAR(64) NOT NULL UNIQUE
);

CREATE INDEX IF NOT EXISTS idx_ops_audit_logs_actor_created
    ON ops.audit_logs (actor_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_audit_logs_action_created
    ON ops.audit_logs (action, created_at DESC);


-- Phase 10 (7.41) chain-lock fix: drop any pre-existing BIGSERIAL column
-- default so the id is drawn inside ops.compute_audit_log_hash() AFTER the
-- advisory chain lock is held. A plain default would draw the sequence
-- before the lock, letting concurrent inserts acquire the lock in reverse
-- id order and build a chain that ascending-id verification rejects.
ALTER TABLE ops.audit_logs ALTER COLUMN id DROP DEFAULT;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION ops.prevent_audit_log_mutation()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'ops.audit_logs is append-only: % operations are strictly prohibited', TG_OP
        USING ERRCODE = '42501';
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS trg_ops_audit_logs_no_update_delete ON ops.audit_logs;
CREATE TRIGGER trg_ops_audit_logs_no_update_delete
    BEFORE UPDATE OR DELETE ON ops.audit_logs
    FOR EACH ROW EXECUTE FUNCTION ops.prevent_audit_log_mutation();

DROP TRIGGER IF EXISTS trg_ops_audit_logs_no_truncate ON ops.audit_logs;
CREATE TRIGGER trg_ops_audit_logs_no_truncate
    BEFORE TRUNCATE ON ops.audit_logs
    FOR EACH STATEMENT EXECUTE FUNCTION ops.prevent_audit_log_mutation();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION ops.compute_audit_log_hash()
RETURNS TRIGGER AS $$
DECLARE
    prev_hash TEXT;
    canonical_ts TEXT;
    payload TEXT;
BEGIN
    PERFORM pg_advisory_xact_lock(741001);

    -- Serialize ID allocation with the hash calculation: the id is drawn
    -- from the chain sequence only while the lock is held, so ascending id
    -- order always matches hash-chain order under concurrent appends.
    IF NEW.id IS NULL THEN
        NEW.id := nextval('ops.audit_logs_id_seq');
    END IF;

    IF NEW.created_at IS NULL THEN
        NEW.created_at := CURRENT_TIMESTAMP;
    END IF;
    IF NEW.metadata_json IS NULL THEN
        NEW.metadata_json := '{}'::jsonb;
    END IF;

    SELECT record_hash INTO prev_hash
    FROM ops.audit_logs
    ORDER BY id DESC
    LIMIT 1;

    IF prev_hash IS NULL THEN
        prev_hash := 'GENESIS';
    END IF;

    NEW.previous_hash := prev_hash;
    canonical_ts := to_char(NEW.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"');
    payload := NEW.previous_hash || '|' ||
               NEW.actor_id || '|' ||
               COALESCE(NEW.actor_role, '') || '|' ||
               NEW.action || '|' ||
               COALESCE(NEW.resource_type, '') || '|' ||
               COALESCE(NEW.resource_id, '') || '|' ||
               NEW.metadata_json::text || '|' ||
               canonical_ts;
    NEW.record_hash := encode(sha256(convert_to(payload, 'UTF8')), 'hex');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

DROP TRIGGER IF EXISTS trg_ops_audit_logs_compute_hash ON ops.audit_logs;
CREATE TRIGGER trg_ops_audit_logs_compute_hash
    BEFORE INSERT ON ops.audit_logs
    FOR EACH ROW EXECUTE FUNCTION ops.compute_audit_log_hash();
