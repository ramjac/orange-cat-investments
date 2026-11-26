-- 001-portfolio-snapshot-daily-nav.sql
-- Idempotent migration extending core_invest.portfolio_snapshots for Daily NAV EOD snapshots.

-- +goose Up
ALTER TABLE core_invest.portfolio_snapshots
    ADD COLUMN IF NOT EXISTS snapshot_type VARCHAR(20) NOT NULL DEFAULT 'observation';

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'chk_portfolio_snapshots_type'
          AND conrelid = 'core_invest.portfolio_snapshots'::regclass
    ) THEN
        ALTER TABLE core_invest.portfolio_snapshots
            ADD CONSTRAINT chk_portfolio_snapshots_type
            CHECK (snapshot_type IN ('observation', 'daily_eod'));
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE core_invest.portfolio_snapshots
    ADD COLUMN IF NOT EXISTS snapshot_date DATE,
    ADD COLUMN IF NOT EXISTS cash_usd DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS equity_usd DECIMAL(14, 2) NOT NULL DEFAULT 0.00,
    ADD COLUMN IF NOT EXISTS daily_return_pct DECIMAL(8, 4) NOT NULL DEFAULT 0.0000;

CREATE UNIQUE INDEX IF NOT EXISTS uq_portfolio_daily_eod_snapshot
    ON core_invest.portfolio_snapshots (portfolio_id, snapshot_date)
    WHERE snapshot_type = 'daily_eod';
