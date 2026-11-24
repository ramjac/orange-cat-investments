#!/usr/bin/env bash
set -euo pipefail

# Orange Cat Investments (OCI) - Development Database Retention & Seed Reset (`prune-dev-db.sh`)
# Prunes synthetic load-test growth in the 'oci-dev' PostgreSQL database while preserving
# all baseline seed rows from scripts/seed-dev.sql (plus the most recent retention window)
# and running VACUUM ANALYZE to reclaim dead tuples.
#
# CRITICAL SAFETY INVARIANT:
# This script is strictly forbidden from running against the production namespace 'oci-core'
# or production database 'oci'.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SEED_SQL="${REPO_ROOT}/scripts/seed-dev.sql"

TARGET_NAMESPACE="oci-dev"
TARGET_DB="${DEV_DATABASE_URL:-}"
KEEP_ROWS="${RETENTION_KEEP_ROWS:-500}"
DRY_RUN=false

while [[ $# -gt 0 ]]; do
    case "$1" in
        --namespace|-n)
            TARGET_NAMESPACE="$2"
            shift 2
            ;;
        --db-url)
            TARGET_DB="$2"
            shift 2
            ;;
        --keep-rows)
            KEEP_ROWS="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [--namespace oci-dev] [--db-url postgresql://...] [--keep-rows 500] [--dry-run]"
            echo "Note: Target namespace MUST be 'oci-dev'. Running against 'oci-core' is blocked."
            exit 0
            ;;
        *)
            echo "Unknown argument: $1" >&2
            exit 1
            ;;
    esac
done

if ! [[ "${KEEP_ROWS}" =~ ^[0-9]+$ ]]; then
    echo "ERROR: --keep-rows must be a non-negative integer, got '${KEEP_ROWS}'" >&2
    exit 1
fi

# 1. Strict Production Guard
if [[ "${TARGET_NAMESPACE}" == "oci-core" || "${TARGET_NAMESPACE}" == "prod"* || "${TARGET_NAMESPACE}" == *"production"* ]]; then
    echo "CRITICAL SECURITY ERROR: Retention script cannot target production namespace '${TARGET_NAMESPACE}'!" >&2
    exit 1
fi

if [[ -n "${TARGET_DB}" ]]; then
    DB_VALIDATION=$(python3 -c "
import urllib.parse, sys
try:
    parsed = urllib.parse.urlparse(sys.argv[1])
    path = parsed.path.strip('/')
    host = parsed.hostname or ''
    if path == 'oci' or 'oci-core' in host or 'prod' in host:
        print('INVALID_PROD')
    elif path != 'oci_dev' and not path.endswith('_dev') and not path.startswith('dev'):
        print('INVALID_NOT_DEV')
    else:
        print('VALID')
except Exception:
    print('ERROR')
" "${TARGET_DB}")

    if [[ "${DB_VALIDATION}" != "VALID" ]]; then
        echo "CRITICAL SECURITY ERROR: Database URL rejected by dev-only guard (${DB_VALIDATION}): ${TARGET_DB}" >&2
        exit 1
    fi
fi

generate_retention_sql() {
    local keep="${1}"
    cat <<SQL
-- OCI oci-dev Database Retention & Seed Hygiene (keep latest ${keep} non-seed rows per table)
BEGIN;

-- 1. Ops schema (child comments first, then tickets)
DELETE FROM ops.ticket_comments
WHERE comment_id NOT IN (
    '018e0000-0000-7000-8000-000000000050'::uuid,
    '018e0000-0000-7000-8000-000000000051'::uuid
)
AND comment_id NOT IN (
    SELECT comment_id FROM ops.ticket_comments
    ORDER BY created_at DESC, comment_id DESC
    LIMIT ${keep}
);

DELETE FROM ops.it_tickets
WHERE ticket_id NOT IN (
    '018e0000-0000-7000-8000-000000000003'::uuid,
    '018e0000-0000-7000-8000-000000000004'::uuid,
    '018e0000-0000-7000-8000-000000000005'::uuid
)
AND ticket_id NOT IN (
    SELECT ticket_id FROM ops.it_tickets
    ORDER BY created_at DESC, ticket_id DESC
    LIMIT ${keep}
);

-- 2. Core Invest schema (dependent child tables before observation_events & portfolios)
DELETE FROM core_invest.brokerage_orders
WHERE order_id NOT IN (
    SELECT order_id FROM core_invest.brokerage_orders
    ORDER BY created_at DESC, order_id DESC
    LIMIT ${keep}
);

DELETE FROM core_invest.portfolio_snapshots
WHERE NOT (
    snapshot_type = 'daily_eod'
    AND portfolio_id IN (
        '018f3a9a-2222-7000-8000-000000000002'::uuid,
        '018f3a9a-2222-7000-8000-000000000003'::uuid
    )
    AND snapshot_date >= (CURRENT_DATE - INTERVAL '90 days')
)
AND snapshot_id NOT IN (
    SELECT snapshot_id FROM core_invest.portfolio_snapshots
    ORDER BY snapshot_at DESC, snapshot_id DESC
    LIMIT ${keep}
);

DELETE FROM core_invest.observation_events
WHERE event_id NOT IN (
    SELECT event_id FROM core_invest.observation_events
    ORDER BY observed_at DESC, event_id DESC
    LIMIT ${keep}
)
AND event_id NOT IN (SELECT event_id FROM core_invest.brokerage_orders WHERE event_id IS NOT NULL)
AND event_id NOT IN (SELECT triggered_by_event_id FROM core_invest.portfolio_snapshots WHERE triggered_by_event_id IS NOT NULL);

DELETE FROM core_invest.backtest_runs
WHERE backtest_id NOT IN (
    SELECT backtest_id FROM core_invest.backtest_runs
    ORDER BY created_at DESC, backtest_id DESC
    LIMIT ${keep}
);

DELETE FROM core_invest.recurring_deposits
WHERE deposit_id NOT IN (
    SELECT deposit_id FROM core_invest.recurring_deposits
    ORDER BY created_at DESC, deposit_id DESC
    LIMIT ${keep}
);

DELETE FROM core_invest.investment_statements
WHERE statement_id NOT IN (
    SELECT statement_id FROM core_invest.investment_statements
    ORDER BY created_at DESC, statement_id DESC
    LIMIT ${keep}
);

DELETE FROM core_invest.customer_api_keys
WHERE key_id NOT IN (
    SELECT key_id FROM core_invest.customer_api_keys
    ORDER BY created_at DESC, key_id DESC
    LIMIT ${keep}
);

DELETE FROM core_invest.webhook_subscriptions
WHERE subscription_id NOT IN (
    SELECT subscription_id FROM core_invest.webhook_subscriptions
    ORDER BY created_at DESC, subscription_id DESC
    LIMIT ${keep}
);

DELETE FROM core_invest.model_drift_logs
WHERE drift_id NOT IN (
    SELECT drift_id FROM core_invest.model_drift_logs
    ORDER BY evaluated_at DESC, drift_id DESC
    LIMIT ${keep}
);

DELETE FROM core_invest.gdpr_erasure_certificates
WHERE certificate_id NOT IN (
    SELECT certificate_id FROM core_invest.gdpr_erasure_certificates
    ORDER BY created_at DESC, certificate_id DESC
    LIMIT ${keep}
);

DELETE FROM core_invest.camera_streams
WHERE stream_id NOT IN (
    '018f3a9a-c001-7000-8000-000000000001'::uuid,
    '018f3a9a-c002-7000-8000-000000000002'::uuid
)
AND stream_id NOT IN (
    SELECT stream_id FROM core_invest.camera_streams
    ORDER BY created_at DESC, stream_id DESC
    LIMIT ${keep}
);

DELETE FROM core_invest.investment_portfolios
WHERE portfolio_id NOT IN (
    '018f3a9a-2222-7000-8000-000000000002'::uuid,
    '018f3a9a-2222-7000-8000-000000000003'::uuid
)
AND portfolio_id NOT IN (
    SELECT portfolio_id FROM core_invest.investment_portfolios
    ORDER BY created_at DESC, portfolio_id DESC
    LIMIT ${keep}
);

-- 3. Facilities schema (child logs/tickets/jobs/telemetry before hardware_assets)
DELETE FROM facilities.maintenance_logs
WHERE log_id NOT IN (
    SELECT log_id FROM facilities.maintenance_logs
    ORDER BY created_at DESC, log_id DESC
    LIMIT ${keep}
);

DELETE FROM facilities.maintenance_tickets
WHERE ticket_id NOT IN (
    SELECT ticket_id FROM facilities.maintenance_tickets
    ORDER BY created_at DESC, ticket_id DESC
    LIMIT ${keep}
);

DELETE FROM facilities.device_ota_jobs
WHERE job_id NOT IN (
    SELECT job_id FROM facilities.device_ota_jobs
    ORDER BY created_at DESC, job_id DESC
    LIMIT ${keep}
);

DELETE FROM facilities.environmental_telemetry
WHERE telemetry_id NOT IN (
    SELECT telemetry_id FROM facilities.environmental_telemetry
    ORDER BY recorded_at DESC, telemetry_id DESC
    LIMIT ${keep}
);

DELETE FROM facilities.hardware_assets
WHERE asset_id NOT IN (
    '018e0000-0000-7000-8000-000000000001'::uuid,
    '018f3a9a-3333-7000-8000-000000000003'::uuid,
    '018f3a9a-5555-7000-8000-000000000005'::uuid,
    '018f3a9a-6666-7000-8000-000000000006'::uuid,
    '018f3a9a-7777-7000-8000-000000000007'::uuid,
    '018f3a9a-7777-7000-8000-000000000008'::uuid,
    '018f3a9a-8888-7000-8000-000000000001'::uuid
)
AND asset_id NOT IN (
    SELECT asset_id FROM facilities.hardware_assets
    ORDER BY created_at DESC, asset_id DESC
    LIMIT ${keep}
)
AND asset_id NOT IN (SELECT camera_asset_id FROM core_invest.observation_events WHERE camera_asset_id IS NOT NULL)
AND asset_id NOT IN (SELECT camera_asset_id FROM core_invest.camera_streams WHERE camera_asset_id IS NOT NULL);

-- 4. Workforce schema (child tables before employees)
DELETE FROM workforce.onboarding_checklists
WHERE checklist_id NOT IN (
    '018f3a9a-c001-7000-8000-000000000001'::uuid,
    '018f3a9a-c002-7000-8000-000000000002'::uuid
)
AND checklist_id NOT IN (
    SELECT checklist_id FROM workforce.onboarding_checklists
    ORDER BY created_at DESC, checklist_id DESC
    LIMIT ${keep}
);

DELETE FROM workforce.leave_requests
WHERE leave_id NOT IN (
    SELECT leave_id FROM workforce.leave_requests
    ORDER BY created_at DESC, leave_id DESC
    LIMIT ${keep}
);

DELETE FROM workforce.vhr_records
WHERE record_id NOT IN (
    SELECT record_id FROM workforce.vhr_records
    ORDER BY created_at DESC, record_id DESC
    LIMIT ${keep}
);

DELETE FROM workforce.incidents
WHERE incident_id NOT IN (
    SELECT incident_id FROM workforce.incidents
    ORDER BY reported_at DESC, incident_id DESC
    LIMIT ${keep}
);

DELETE FROM workforce.review_cycles
WHERE review_id NOT IN (
    '018e0000-0000-7000-8000-000000000020'::uuid,
    '018e0000-0000-7000-8000-000000000021'::uuid,
    '018e0000-0000-7000-8000-000000000022'::uuid
)
AND review_id NOT IN (
    SELECT review_id FROM workforce.review_cycles
    ORDER BY created_at DESC, review_id DESC
    LIMIT ${keep}
);

DELETE FROM workforce.executive_reports
WHERE report_id NOT IN ('rep-2026-08', 'rep-2026-09')
AND report_id NOT IN (
    SELECT report_id FROM workforce.executive_reports
    ORDER BY created_at DESC, report_id DESC
    LIMIT ${keep}
);

DELETE FROM workforce.employees
WHERE employee_id NOT IN (
    '018f0000-0000-7000-8000-000000000001'::uuid,
    '018f3a9a-e001-7000-8000-000000000001'::uuid,
    '018f3a9a-e002-7000-8000-000000000002'::uuid,
    '018f3a9a-e003-7000-8000-000000000003'::uuid,
    '018f3a9a-e004-7000-8000-000000000004'::uuid,
    '018f3a9a-e005-7000-8000-000000000005'::uuid,
    '018f3a9a-e006-7000-8000-000000000006'::uuid,
    '018f3a9a-e007-7000-8000-000000000007'::uuid,
    '018f3a9a-e008-7000-8000-000000000008'::uuid
)
AND employee_id NOT IN (
    SELECT employee_id FROM workforce.employees
    ORDER BY created_at DESC, employee_id DESC
    LIMIT ${keep}
)
AND employee_id NOT IN (SELECT assigned_to FROM workforce.onboarding_checklists WHERE assigned_to IS NOT NULL)
AND employee_id NOT IN (SELECT caretaker_id FROM workforce.care_schedules WHERE caretaker_id IS NOT NULL)
AND employee_id NOT IN (SELECT reviewer_id FROM workforce.review_cycles WHERE reviewer_id IS NOT NULL)
AND employee_id NOT IN (SELECT peer_reviewer_id FROM workforce.review_cycles WHERE peer_reviewer_id IS NOT NULL)
AND employee_id NOT IN (SELECT clinical_auditor_id FROM workforce.review_cycles WHERE clinical_auditor_id IS NOT NULL)
AND employee_id NOT IN (SELECT veterinarian_id FROM workforce.vhr_records WHERE veterinarian_id IS NOT NULL)
AND employee_id NOT IN (SELECT involved_feline_id FROM workforce.incidents WHERE involved_feline_id IS NOT NULL)
AND employee_id NOT IN (SELECT involved_human_id FROM workforce.incidents WHERE involved_human_id IS NOT NULL)
AND employee_id NOT IN (SELECT assigned_employee_id FROM facilities.hardware_assets WHERE assigned_employee_id IS NOT NULL)
AND employee_id NOT IN (SELECT assigned_technician_id FROM facilities.maintenance_tickets WHERE assigned_technician_id IS NOT NULL)
AND employee_id NOT IN (SELECT reported_by FROM facilities.maintenance_tickets WHERE reported_by IS NOT NULL)
AND employee_id NOT IN (SELECT technician_id FROM facilities.maintenance_logs WHERE technician_id IS NOT NULL)
AND employee_id NOT IN (SELECT feline_id FROM core_invest.observation_events WHERE feline_id IS NOT NULL);

-- Reset feline medical holds and baseline seed states
UPDATE workforce.care_schedules
SET emergency_medical_hold = FALSE, updated_at = CURRENT_TIMESTAMP
WHERE feline_id IN (
    '018f3a9a-e001-7000-8000-000000000001'::uuid,
    '018f3a9a-e002-7000-8000-000000000002'::uuid
);

COMMIT;

-- Reclaim dead tuples and refresh planner statistics
VACUUM ANALYZE ops.ticket_comments;
VACUUM ANALYZE ops.it_tickets;
VACUUM ANALYZE core_invest.brokerage_orders;
VACUUM ANALYZE core_invest.portfolio_snapshots;
VACUUM ANALYZE core_invest.observation_events;
VACUUM ANALYZE core_invest.backtest_runs;
VACUUM ANALYZE core_invest.recurring_deposits;
VACUUM ANALYZE core_invest.investment_statements;
VACUUM ANALYZE core_invest.customer_api_keys;
VACUUM ANALYZE core_invest.webhook_subscriptions;
VACUUM ANALYZE core_invest.model_drift_logs;
VACUUM ANALYZE core_invest.gdpr_erasure_certificates;
VACUUM ANALYZE core_invest.camera_streams;
VACUUM ANALYZE core_invest.investment_portfolios;
VACUUM ANALYZE facilities.maintenance_logs;
VACUUM ANALYZE facilities.maintenance_tickets;
VACUUM ANALYZE facilities.device_ota_jobs;
VACUUM ANALYZE facilities.environmental_telemetry;
VACUUM ANALYZE facilities.hardware_assets;
VACUUM ANALYZE workforce.onboarding_checklists;
VACUUM ANALYZE workforce.leave_requests;
VACUUM ANALYZE workforce.vhr_records;
VACUUM ANALYZE workforce.incidents;
VACUUM ANALYZE workforce.review_cycles;
VACUUM ANALYZE workforce.executive_reports;
VACUUM ANALYZE workforce.employees;
SQL
}

RETENTION_SQL="$(generate_retention_sql "${KEEP_ROWS}")"

if [[ "${DRY_RUN}" == true ]]; then
    echo "=== [DRY-RUN] OCI oci-dev Database Retention SQL (keep=${KEEP_ROWS}) ==="
    printf '%s\n' "${RETENTION_SQL}"

    if [[ -f "${SEED_SQL}" ]]; then
        python3 - "${SEED_SQL}" "${RETENTION_SQL}" <<'PY'
import re, sys

seed_path, retention_sql = sys.argv[1], sys.argv[2]
with open(seed_path, "r", encoding="utf-8") as f:
    seed_sql = f.read()

# Verify every seed employee, hardware asset, portfolio, stream, checklist, ticket, comment, and review UUID
# is explicitly protected in the retention SQL (tables that are never pruned like location_zones,
# firmware_releases, care_schedules, portfolio_holdings, tax_lot_sales, allocation_strategies are skipped).
protected_prefixes = (
    "018f3a9a-e00",   # seed employees (Garfield..Bob)
    "018e0000-0000-7000-8000-000000000001", # QR-CAM-ORANGE-01
    "018f3a9a-3333-", # CAM-RIG-2025-01
    "018f3a9a-5555-", # SW-CORE-01
    "018f3a9a-6666-", # GW-HABITAT-01
    "018f3a9a-7777-", # Pebble watches
    "018f3a9a-8888-", # MBP-RICK-M3-01
    "018f3a9a-c00",   # camera streams & checklists
    "018f3a9a-2222-", # Arthur & Chloe portfolios
    "018e0000-0000-7000-8000-000000000003", # seed tickets
    "018e0000-0000-7000-8000-000000000004",
    "018e0000-0000-7000-8000-000000000005",
    "018e0000-0000-7000-8000-000000000050", # seed ticket comments
    "018e0000-0000-7000-8000-000000000051",
    "018e0000-0000-7000-8000-000000000020", # seed review cycles
    "018e0000-0000-7000-8000-000000000021",
    "018e0000-0000-7000-8000-000000000022",
)
seed_uuids = sorted(set(re.findall(r"[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}", seed_sql)))
missing = [u for u in seed_uuids if u.startswith(protected_prefixes) and u not in retention_sql]
if missing:
    print(f"ERROR: Retention SQL is missing seed UUIDs from scripts/seed-dev.sql: {missing}", file=sys.stderr)
    sys.exit(1)
for rep_id in ("rep-2026-08", "rep-2026-09"):
    if rep_id not in retention_sql:
        print(f"ERROR: Retention SQL is missing seed report ID: {rep_id}", file=sys.stderr)
        sys.exit(1)
print("Dry-run validation passed: all pruned-table seed persona/entity IDs are preserved.")
PY
    fi
    exit 0
fi

echo "================================================================================"
echo "  Orange Cat Investments (OCI) - oci-dev Database Retention & Seed Reset"
echo "================================================================================"
echo "Target Namespace : ${TARGET_NAMESPACE}"
echo "Keep Recent Rows : ${KEEP_ROWS} per high-churn table"

if [[ -n "${TARGET_DB}" ]]; then
    echo "Executing retention SQL directly via psql..."
    printf '%s\n' "${RETENTION_SQL}" | psql "${TARGET_DB}" -v ON_ERROR_STOP=1
    if [[ -f "${SEED_SQL}" ]]; then
        echo "Re-applying idempotent baseline seed fixtures..."
        psql "${TARGET_DB}" -v ON_ERROR_STOP=1 -f "${SEED_SQL}"
    fi
elif command -v kubectl &>/dev/null; then
    if [[ -z "${KUBECONFIG:-}" && -f "${REPO_ROOT}/thinker-kube.config" ]]; then
        export KUBECONFIG="${REPO_ROOT}/thinker-kube.config"
    fi
    echo "Executing retention SQL via kubectl against statefulset/postgres-dev in '${TARGET_NAMESPACE}'..."
    printf '%s\n' "${RETENTION_SQL}" | kubectl exec -i -n "${TARGET_NAMESPACE}" statefulset/postgres-dev -c postgres -- \
        psql -U oci_admin -d oci_dev -v ON_ERROR_STOP=1
    if [[ -f "${SEED_SQL}" ]]; then
        echo "Re-applying idempotent baseline seed fixtures..."
        kubectl exec -i -n "${TARGET_NAMESPACE}" statefulset/postgres-dev -c postgres -- \
            psql -U oci_admin -d oci_dev -v ON_ERROR_STOP=1 < "${SEED_SQL}"
    fi
else
    echo "ERROR: Neither --db-url/DEV_DATABASE_URL nor kubectl found." >&2
    exit 1
fi

echo "oci-dev database retention and VACUUM ANALYZE completed successfully."
