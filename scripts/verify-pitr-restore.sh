#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# OCI PgBackRest Point-In-Time Recovery (PITR) Verification Drill (Phase 11 / 7.44)
# ==============================================================================
# Creates a test table in oci-dev, captures a pre-drop timestamp, switches WAL,
# drops the table, and verifies PgBackRest PITR restore to the target timestamp.
# Strictly refuses to run against production namespace oci-core.
# ==============================================================================

NAMESPACE="oci-dev"
DRY_RUN=false

while [[ $# -gt 0 ]]; do
  case "$1" in
    --namespace)
      NAMESPACE="${2:-oci-dev}"
      shift 2
      ;;
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    *)
      echo "Unknown option: $1" >&2
      exit 1
      ;;
  esac
done

if [[ "${NAMESPACE}" == "oci-core" ]]; then
  echo "ERROR: verify-pitr-restore.sh refuses to run destructive PITR drills against production namespace oci-core" >&2
  exit 1
fi

if [[ "${DRY_RUN}" == "true" ]]; then
  echo "DRY-RUN: PgBackRest PITR verification drill validated for namespace=${NAMESPACE} (--type=time --target=YYYY-MM-DD HH:MM:SS)"
  exit 0
fi

POD_NAME="postgres-dev-0"
PG_DB="oci_dev"

echo "1. Ensuring PgBackRest stanza and initial backup exist in ${NAMESPACE}/${POD_NAME}..."
for _ in $(seq 1 30); do
  if kubectl exec -n "${NAMESPACE}" "${POD_NAME}" -c pgbackrest -- pgbackrest --stanza=oci info 2>/dev/null | grep -q "full backup:"; then
    break
  fi
  kubectl exec -n "${NAMESPACE}" "${POD_NAME}" -c pgbackrest -- pgbackrest --stanza=oci stanza-create 2>/dev/null || true
  kubectl exec -n "${NAMESPACE}" "${POD_NAME}" -c pgbackrest -- pgbackrest --stanza=oci backup --type=full 2>/dev/null || true
  sleep 2
done
kubectl exec -n "${NAMESPACE}" "${POD_NAME}" -c pgbackrest -- pgbackrest --stanza=oci info | grep -E "status:|full backup:"

echo "2. Creating synthetic test table ops.pitr_drill_verification and capturing target timestamp..."
kubectl exec -n "${NAMESPACE}" "${POD_NAME}" -c postgres -- sh -ec "
  psql -U \"\${POSTGRES_USER}\" -d \"${PG_DB}\" -v ON_ERROR_STOP=1 -c \"
    CREATE TABLE IF NOT EXISTS ops.pitr_drill_verification (
      id INT PRIMARY KEY,
      sentinel TEXT NOT NULL,
      created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
    );
    TRUNCATE TABLE ops.pitr_drill_verification;
    INSERT INTO ops.pitr_drill_verification (id, sentinel) VALUES (1, 'pitr-ok-before-drop');
  \"
"

TARGET_TS=$(kubectl exec -n "${NAMESPACE}" "${POD_NAME}" -c postgres -- sh -ec "psql -U \"\${POSTGRES_USER}\" -d \"${PG_DB}\" -tAc \"SELECT to_char((clock_timestamp() + interval '500 milliseconds') AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS.US+00');\"" | tr -d '\r\n')
echo "Captured pre-drop target timestamp (UTC): ${TARGET_TS}"
sleep 2

echo "3. Executing synthetic DROP TABLE ops.pitr_drill_verification and archiving WAL..."
END_WAL=$(kubectl exec -n "${NAMESPACE}" "${POD_NAME}" -c postgres -- sh -ec "
  psql -U \"\${POSTGRES_USER}\" -d \"${PG_DB}\" -v ON_ERROR_STOP=1 -c \"DROP TABLE ops.pitr_drill_verification;\" >/dev/null
  psql -U \"\${POSTGRES_USER}\" -d \"${PG_DB}\" -tAc \"SELECT pg_walfile_name(pg_switch_wal());\"
" | tr -d '\r\n')
sleep 3

echo "4. Running PgBackRest PITR restore drill (--type=time \"--target=${TARGET_TS}\", end_wal=${END_WAL})..."
kubectl exec -n "${NAMESPACE}" "${POD_NAME}" -c pgbackrest -- sh -ec "
  for wal in /var/lib/postgresql/wal_archive/0*; do
    [ -f \"\$wal\" ] || continue
    pgbackrest --stanza=oci archive-push \"\$wal\" && rm -f \"\$wal\"
  done
  RESTORE_DIR=/var/lib/postgresql/wal_archive/pitr-restore
  WAL_STAGE=/var/lib/postgresql/wal_archive/pitr-wal
  rm -rf \"\${RESTORE_DIR}\" \"\${WAL_STAGE}\"
  mkdir -p \"\${RESTORE_DIR}\" \"\${WAL_STAGE}\"
  chmod 700 \"\${RESTORE_DIR}\"
  pgbackrest --stanza=oci --pg1-path=\"\${RESTORE_DIR}\" --type=time \"--target=${TARGET_TS}\" --target-action=promote --delta restore
  START_WAL=\$(awk '/START WAL LOCATION:/ {gsub(/[()]/, \"\", \$6); print \$6}' \"\${RESTORE_DIR}/backup_label\")
  PREFIX=\$(echo \"\${START_WAL}\" | cut -c1-16)
  START_HEX=\$(echo \"\${START_WAL}\" | cut -c17-24)
  END_HEX=\$(echo \"${END_WAL}\" | cut -c17-24)
  CUR=\$((0x\${START_HEX}))
  END=\$((0x\${END_HEX}))
  while [ \"\${CUR}\" -le \"\${END}\" ]; do
    SEG=\$(printf \"%s%08X\" \"\${PREFIX}\" \"\${CUR}\")
    pgbackrest --stanza=oci archive-get \"\${SEG}\" \"\${WAL_STAGE}/\${SEG}\" || true
    CUR=\$((CUR + 1))
  done
  sed -i \"s|^restore_command = .*|restore_command = 'cp \${WAL_STAGE}/%f %p'|\" \"\${RESTORE_DIR}/postgresql.auto.conf\"
"

echo "5. Verifying restored PostgreSQL data directory contains pre-drop table state..."
kubectl exec -n "${NAMESPACE}" "${POD_NAME}" -c postgres -- sh -ec "
  RESTORE_DIR=/var/lib/postgresql/wal_archive/pitr-restore
  WAL_STAGE=/var/lib/postgresql/wal_archive/pitr-wal
  chown -R postgres:postgres \"\${RESTORE_DIR}\" \"\${WAL_STAGE}\"
  chmod 700 \"\${RESTORE_DIR}\"
  su - postgres -c \"pg_ctl -D \${RESTORE_DIR} -o '-p 5499 -k /tmp -c archive_mode=off' -l /tmp/pitr-pg.log start\"
  trap 'su - postgres -c \"pg_ctl -D \${RESTORE_DIR} stop -m immediate\" || true; rm -rf \"\${RESTORE_DIR}\" \"\${WAL_STAGE}\"' EXIT
  for _ in \$(seq 1 20); do
    if psql -h /tmp -p 5499 -U \"\${POSTGRES_USER}\" -d \"${PG_DB}\" -tAc \"SELECT NOT pg_is_in_recovery();\" 2>/dev/null | grep -q '^t$'; then
      break
    fi
    sleep 1
  done
  RESULT=\$(psql -h /tmp -p 5499 -U \"\${POSTGRES_USER}\" -d \"${PG_DB}\" -tAc \"SELECT sentinel FROM ops.pitr_drill_verification WHERE id = 1;\")
  if [ \"\${RESULT}\" != \"pitr-ok-before-drop\" ]; then
    echo \"ERROR: Expected sentinel 'pitr-ok-before-drop', got '\${RESULT}'\" >&2
    exit 1
  fi
  echo \"PITR restore verification PASSED: recovered sentinel='\${RESULT}' at target='${TARGET_TS}'\"
"
