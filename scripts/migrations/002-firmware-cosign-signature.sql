-- 002-firmware-cosign-signature.sql
-- Idempotent migration adding asymmetric Cosign signature verification metadata
-- to facilities.firmware_releases (Phase 8 / 7.34).

ALTER TABLE facilities.firmware_releases
    ADD COLUMN IF NOT EXISTS cosign_signature TEXT NOT NULL DEFAULT '';
