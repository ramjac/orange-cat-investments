# Stateful Store Alerts Runbook

This runbook documents the Prometheus alerting rules for OCI's stateful data stores (**Valkey**, **PostgreSQL**, and **Meilisearch**), defined in [`k8s/monitoring/prometheus.yaml`](../../k8s/monitoring/prometheus.yaml).

## Alert Summary

| Alert Name | Group | Severity | Tier | Threshold / Condition | `for` |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `ValkeyMemorySaturation` | `valkey_alerts` | `warning` | `cache` | `redis_memory_used_bytes / redis_memory_max_bytes > 0.85` (when `maxmemory > 0`) | `5m` |
| `ValkeyKeyEvictions` | `valkey_alerts` | `warning` | `cache` | `increase(redis_evicted_keys_total[5m]) > 0` | `2m` |
| `PostgresConnectionSaturation` | `postgresql_alerts` | `warning` | `database` | `sum by (instance) (pg_stat_activity_count) / pg_settings_max_connections > 0.80` | `5m` |
| `PostgresDeadTupleBloat` | `postgresql_alerts` | `warning` | `database` | `n_dead_tup > 10000` and dead ratio `> 0.20` | `15m` |
| `PostgresXIDWraparoundRisk` | `postgresql_alerts` | `critical` | `database` | `pg_database_age > 1500000000` (1.5B XIDs) | `5m` |
| `PostgresLowCacheHitRatio` | `postgresql_alerts` | `warning` | `database` | 5m buffer cache hit ratio `< 0.99` (when active) | `10m` |
| `MeilisearchDown` | `meilisearch_alerts` | `critical` | `search` | `up{job="meilisearch"} == 0` | `1m` |
| `MeilisearchDiskUsageHigh` | `meilisearch_alerts` | `warning` | `search` | `meilisearch-data-pvc` used / capacity `> 0.80` | `5m` |

---

## 1. Valkey Alerts (`tier: cache`)

### `ValkeyMemorySaturation`
* **Severity:** `warning`
* **Threshold:** Memory usage exceeds **85%** of `maxmemory` for **5 minutes** (only evaluated when `redis_memory_max_bytes > 0`).
* **Impact:** High memory pressure increases the risk of key evictions or OOM kills, impacting BFF session cookies (`__Host-customer-session`, `__Host-employee-session`), rate limiters, and distributed caches.
* **Triage:**
  ```bash
  kubectl -n oci-core exec deploy/valkey -c valkey -- valkey-cli INFO memory
  kubectl -n oci-core exec deploy/valkey -c valkey -- valkey-cli --bigkeys
  ```
* **Remediation:**
  1. Identify keys without TTLs or unexpectedly large cached values using `--bigkeys` or `MEMORY USAGE <key>`.
  2. Verify session TTLs in `pkg/session` and cache TTLs in `pkg/cache` are being honored.
  3. If legitimate working-set growth is confirmed, increase Valkey container memory limits and `--maxmemory` in `k8s/oci-core/valkey.yaml` (or `k8s/oci-dev/valkey.yaml`).

### `ValkeyKeyEvictions`
* **Severity:** `warning`
* **Threshold:** `increase(redis_evicted_keys_total[5m]) > 0` for **2 minutes**.
* **Impact:** Active key eviction indicates Valkey has hit `maxmemory`. Active user sessions or CSRF tokens may be dropped, causing unexpected logouts or `403` CSRF errors.
* **Triage:**
  ```bash
  kubectl -n oci-core exec deploy/valkey -c valkey -- valkey-cli INFO stats | grep -E "evicted_keys|expired_keys"
  kubectl -n oci-core exec deploy/valkey -c valkey -- valkey-cli CONFIG GET maxmemory-policy
  ```
* **Remediation:**
  1. Treat as urgent if evictions occur in `oci-core`. Inspect which key prefixes are consuming memory.
  2. Increase `maxmemory` and pod memory limits, or move non-critical cache keys to shorter TTLs.

---

## 2. PostgreSQL Alerts (`tier: database`)

### `PostgresConnectionSaturation`
* **Severity:** `warning`
* **Threshold:** Active/idle connections exceed **80%** of `pg_settings_max_connections` for **5 minutes**.
* **Impact:** Once `max_connections` is reached, BLLs, Domain APIs (`workforce-api`, `facilities-api`, `core-invest-api`, `ops-api`), and COTS apps (Forgejo, Nextcloud, ZITADEL) will fail to open new connections.
* **Triage:**
  ```bash
  kubectl -n oci-core exec deploy/postgres-core -c postgres -- psql -U "$POSTGRES_USER" -d oci -c \
    "SELECT datname, usename, state, count(*) FROM pg_stat_activity GROUP BY 1, 2, 3 ORDER BY 4 DESC;"
  ```
* **Remediation:**
  1. Check for connection leaks or long-running `idle in transaction` sessions; terminate stuck backends if needed (`pg_terminate_backend(pid)`).
  2. Review `pgxpool` `MaxConns` across Domain APIs and worker pods relative to `max_connections`.
  3. If workload concurrency legitimately requires more connections, raise `max_connections` in PostgreSQL configuration and ensure memory limits are sized accordingly.

### `PostgresDeadTupleBloat`
* **Severity:** `warning`
* **Threshold:** A user table has `> 10,000` dead tuples and a dead-tuple ratio `> 20%` for **15 minutes**.
* **Impact:** Table and index bloat degrades query latency and wastes disk space on `postgres-core` / `postgres-internal` PVCs.
* **Triage:**
  ```bash
  kubectl -n oci-core exec deploy/postgres-core -c postgres -- psql -U "$POSTGRES_USER" -d oci -c \
    "SELECT schemaname, relname, n_live_tup, n_dead_tup, last_autovacuum FROM pg_stat_user_tables ORDER BY n_dead_tup DESC LIMIT 15;"
  ```
* **Remediation:**
  1. Check if long-running transactions are holding back the vacuum horizon (`SELECT pid, now() - xact_start AS duration, state, query FROM pg_stat_activity WHERE xact_start IS NOT NULL ORDER BY duration DESC;`).
  2. Run a manual `VACUUM (VERBOSE, ANALYZE) <schema>.<table>;` during low traffic, or tune per-table `autovacuum_vacuum_scale_factor` for high-churn tables.

### `PostgresXIDWraparoundRisk`
* **Severity:** `critical`
* **Threshold:** `pg_database_age > 1,500,000,000` (1.5 billion transactions, ~75% of the 2^31 XID wraparound limit) for **5 minutes**.
* **Impact:** PostgreSQL will refuse all write transactions and shut down if XID age reaches ~2 billion to prevent data corruption.
* **Triage:**
  ```bash
  kubectl -n oci-core exec deploy/postgres-core -c postgres -- psql -U "$POSTGRES_USER" -d oci -c \
    "SELECT datname, age(datfrozenxid) AS xid_age FROM pg_database ORDER BY xid_age DESC;"
  ```
* **Remediation:**
  1. Immediately check for stale replication slots, prepared transactions (`pg_prepared_xacts`), or long-running transactions blocking `autovacuum` freeze.
  2. Run `VACUUM FREEZE` on the affected database/tables to advance `datfrozenxid`.

### `PostgresLowCacheHitRatio`
* **Severity:** `warning`
* **Threshold:** 5-minute shared buffer cache hit ratio drops below **99%** (`< 0.99`) for **10 minutes** on an active database.
* **Impact:** Excessive physical disk reads increase I/O wait and degrade Domain API / COTS response times.
* **Triage:**
  ```bash
  kubectl -n oci-core exec deploy/postgres-core -c postgres -- psql -U "$POSTGRES_USER" -d oci -c \
    "SELECT query, calls, shared_blks_hit, shared_blks_read FROM pg_stat_statements ORDER BY shared_blks_read DESC LIMIT 10;"
  ```
* **Remediation:**
  1. Identify queries performing sequential scans on large tables and add missing indexes.
  2. Verify `shared_buffers` sizing relative to pod memory limits in `postgres-core.yaml` / `postgres-internal.yaml`.

---

## 3. Meilisearch Alerts (`tier: search`)

### `MeilisearchDown`
* **Severity:** `critical`
* **Threshold:** `up{job="meilisearch"} == 0` for **1 minute**.
* **Impact:** Full-text search endpoints fail or return stale/unavailable errors.
* **Triage:**
  ```bash
  kubectl -n oci-core get pods -l app=meilisearch
  kubectl -n oci-core logs -l app=meilisearch --tail=100
  kubectl -n oci-core describe pod -l app=meilisearch
  ```
* **Remediation:**
  1. Check pod events for `OOMKilled`, crash loops, or PVC mount issues.
  2. Verify the `MEILI_METRICS_KEY` secret mounted in both Meilisearch and Prometheus (`k8s/monitoring/prometheus.yaml`) matches if `/metrics` returns `401`/`403`.

### `MeilisearchDiskUsageHigh`
* **Severity:** `warning`
* **Threshold:** `meilisearch-data-pvc` disk utilization exceeds **80%** (`used / capacity > 0.80`) for **5 minutes**.
* **Impact:** Meilisearch will fail to index new documents or perform background LMDB compaction if the volume fills completely.
* **Triage:**
  ```bash
  kubectl -n oci-core get pvc meilisearch-data-pvc
  kubectl -n oci-core exec deploy/meilisearch -- df -h /meili_data
  ```
* **Remediation:**
  1. Inspect index sizes and purge obsolete indexes or completed task history if applicable.
  2. Expand `meilisearch-data-pvc` storage request (if supported by the StorageClass) or provision a larger volume.
