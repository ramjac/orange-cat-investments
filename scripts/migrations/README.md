# Database Migrations Protocol

This directory contains ordered, idempotent SQL migrations for Orange Cat Investments (OCI).

## Protocol Rules (from README.md §7.0 Rule 4)

1. **Three-Copy Synchronization:**
   The database bootstrap SQL exists in three copies that must always stay in sync:
   - `scripts/init.sql`
   - `k8s/oci-core/postgres-core-configmap.yaml`
   - `k8s/oci-dev/postgres-dev-configmap.yaml`
   (Seed rows: `scripts/seed-dev.sql` and the `oci-dev` configmap).

2. **Idempotent DDL:**
   Bootstrap scripts only run against an empty data volume, so every schema change must be written idempotently (`ALTER TABLE ... ADD COLUMN IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`).

3. **Numbered Migration Files:**
   Every schema change must also be added to a numbered migration file:
   `scripts/migrations/NNN-<name>.sql`
   which can be applied to existing, running databases with `psql -f`.
