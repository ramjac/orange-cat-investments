#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Orange Cat Investments (OCI) - Rigorous Chaos & Observability Audit Harness
# ==============================================================================
# Runs externally from the developer workstation against the thinker cluster.
# Brings down each target workload one-by-one (or injects -dev mock faults),
# runs the full 13-persona simulator + targeted probes, dwells for alert
# evaluation, collects caller pod logs + Loki ingestion + Prometheus alerts +
# Gatus statuses + RabbitMQ queue depths, restores the target, and checks
# recovery.
#
# Safety invariants:
#   - NEVER scales, mutates, or targets oci-core.
#   - NEVER scales or mutates forgejo-runner.
#   - NEVER touches unsuffixed external-sim mocks (only *-dev).
#   - Scaling oci-internal requires explicit --allow-internal-scale opt-in.
#   - trap EXIT/INT/ERR unconditionally restores any active scale-down or fault.
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

if [[ -z "${KUBECONFIG:-}" && -f "${REPO_ROOT}/thinker-kube.config" ]]; then
    export KUBECONFIG="${REPO_ROOT}/thinker-kube.config"
fi
if ! command -v kubectl >/dev/null 2>&1 && [[ -x "/home/ramjac/.local/bin/kubectl" ]]; then
    export PATH="/home/ramjac/.local/bin:${PATH}"
fi

INGRESS_IP="${OCI_INGRESS_IP:-192.168.1.246}"
GROUP="all"
ONLY_ID=""
ALLOW_INTERNAL_SCALE=false
DRY_RUN=false
DEV_DWELL=75
INTERNAL_DWELL=100
MOCK_DWELL=75
OUT_DIR="/tmp/oci-chaos-audit-$(date -u +%Y%m%dT%H%M%SZ)"
TOKENS_FILE="${TOKENS_FILE:-${REPO_ROOT}/forgejo-sim-tokens.json}"
SIM_BIN="/tmp/oci-chaos-simulator"

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Options:
  --group GROUP            Scenario group: all | dev | internal | mocks (default: all)
  --only ID                Run only a single scenario ID (e.g. A7, B1, C5)
  --allow-internal-scale   Required to scale oci-internal COTS workloads to 0
  --alert-dwell SEC        Override alert dwell seconds for dev & mocks (default: 75)
  --internal-dwell SEC     Override alert dwell seconds for oci-internal (default: 100)
  --ingress-ip IP          Cluster ingress IP (default: 192.168.1.246)
  --out-dir DIR            Directory for raw evidence JSON/logs/summary
  --dry-run                Validate target guards and print scenarios without mutating cluster
  -h, --help               Show this help
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --group) GROUP="$2"; shift 2 ;;
        --only) ONLY_ID="$2"; shift 2 ;;
        --allow-internal-scale) ALLOW_INTERNAL_SCALE=true; shift ;;
        --alert-dwell) DEV_DWELL="$2"; MOCK_DWELL="$2"; shift 2 ;;
        --internal-dwell) INTERNAL_DWELL="$2"; shift 2 ;;
        --ingress-ip) INGRESS_IP="$2"; shift 2 ;;
        --out-dir) OUT_DIR="$2"; shift 2 ;;
        --dry-run) DRY_RUN=true; shift ;;
        -h|--help) usage; exit 0 ;;
        *) echo "ERROR: Unknown argument: $1" >&2; usage; exit 1 ;;
    esac
done

case "${GROUP}" in
    all|dev|internal|mocks) ;;
    *) echo "ERROR: Invalid --group '${GROUP}' (expected all|dev|internal|mocks)" >&2; exit 1 ;;
esac

# Active restoration state (tracked for trap cleanup)
ACTIVE_SCALE_NS=""
ACTIVE_SCALE_REF=""
ACTIVE_SCALE_ORIG=1
ACTIVE_MOCK_FAULTS=false
PF_PIDS=()

PROM_PORT=$(( 31000 + RANDOM % 1000 ))
LOKI_PORT=$(( 32100 + RANDOM % 1000 ))
RMQ_PORT=$(( 33200 + RANDOM % 1000 ))

# Guard: verify a mutation target is allowed
guard_target() {
    local ns="$1"
    local ref="$2"
    if [[ "${ns}" == "oci-core" ]]; then
        echo "FATAL GUARD: Refusing to mutate oci-core (${ref})" >&2
        exit 99
    fi
    if [[ "${ns}" == "oci-internal" ]]; then
        if [[ "${ref}" == *"forgejo-runner"* ]]; then
            echo "FATAL GUARD: Refusing to mutate forgejo-runner in oci-internal" >&2
            exit 99
        fi
        if [[ "${ALLOW_INTERNAL_SCALE}" != true ]]; then
            echo "FATAL GUARD: Scaling oci-internal (${ref}) requires --allow-internal-scale" >&2
            exit 99
        fi
    elif [[ "${ns}" == "external-sim" ]]; then
        if [[ "${ref}" != *"-dev" ]]; then
            echo "FATAL GUARD: Refusing to mutate non-dev external-sim resource (${ref})" >&2
            exit 99
        fi
    elif [[ "${ns}" != "oci-dev" ]]; then
        echo "FATAL GUARD: Refusing to mutate unexpected namespace '${ns}' (${ref})" >&2
        exit 99
    fi
}

reset_dev_mock_faults() {
    local bank_cid bank_sec broker_key broker_sec
    bank_cid="$(kubectl get secret mock-bank-dev-credentials -n external-sim -o jsonpath='{.data.PLAID_CLIENT_ID}' 2>/dev/null | base64 -d 2>/dev/null || echo 'client_oci_sim_01')"
    bank_sec="$(kubectl get secret mock-bank-dev-credentials -n external-sim -o jsonpath='{.data.PLAID_SECRET}' 2>/dev/null | base64 -d 2>/dev/null || echo 'sec_oci_sim_test_8829')"
    broker_key="$(kubectl get secret mock-broker-dev-credentials -n external-sim -o jsonpath='{.data.APCA_API_KEY_ID}' 2>/dev/null | base64 -d 2>/dev/null || echo 'CK_OCI_TEST_KEY_01')"
    broker_sec="$(kubectl get secret mock-broker-dev-credentials -n external-sim -o jsonpath='{.data.APCA_API_SECRET_KEY}' 2>/dev/null | base64 -d 2>/dev/null || echo 'SK_OCI_TEST_SECRET_01')"

    curl -sf --max-time 10 -X DELETE \
        --resolve "bank-api-dev.sim.oci.local:80:${INGRESS_IP}" \
        -H "PLAID-CLIENT-ID: ${bank_cid}" \
        -H "PLAID-SECRET: ${bank_sec}" \
        "http://bank-api-dev.sim.oci.local/sim/faults" >/dev/null 2>&1 || true
    curl -sf --max-time 10 -X DELETE \
        --resolve "broker-api-dev.sim.oci.local:80:${INGRESS_IP}" \
        -H "APCA-API-KEY-ID: ${broker_key}" \
        -H "APCA-API-SECRET-KEY: ${broker_sec}" \
        "http://broker-api-dev.sim.oci.local/sim/faults" >/dev/null 2>&1 || true
    ACTIVE_MOCK_FAULTS=false
}

cleanup() {
    local exit_code=$?
    set +e
    if [[ -n "${ACTIVE_SCALE_NS}" && -n "${ACTIVE_SCALE_REF}" ]]; then
        echo "[cleanup] Restoring ${ACTIVE_SCALE_REF} in ${ACTIVE_SCALE_NS} to replicas=${ACTIVE_SCALE_ORIG}..."
        kubectl scale "${ACTIVE_SCALE_REF}" -n "${ACTIVE_SCALE_NS}" --replicas="${ACTIVE_SCALE_ORIG}" >/dev/null 2>&1 || true
        ACTIVE_SCALE_NS=""
        ACTIVE_SCALE_REF=""
    fi
    if [[ "${ACTIVE_MOCK_FAULTS}" == true ]]; then
        echo "[cleanup] Resetting -dev mock faults..."
        reset_dev_mock_faults
    fi
    for pid in "${PF_PIDS[@]}"; do
        kill "${pid}" 2>/dev/null || true
    done
    exit "${exit_code}"
}
trap cleanup EXIT INT TERM

# Scenario table:
# ID|GROUP|NS|REF|APP_LABEL|MODE|FAULT_TARGET|FAULT_PAYLOAD|CALLERS_CSV|EXTRA_PROBE|DWELL
# MODE: scale | mock_fault | fault_suite
SCENARIOS=(
    # Group A: oci-dev (18 workloads)
    "A1|dev|oci-dev|deployment/oci-customer-portal-dev|oci-customer-portal-dev|scale|||ingress:nginx-ingress-microk8s-controller|http://dev.customer.oci.local/|${DEV_DWELL}"
    "A2|dev|oci-dev|deployment/oci-employee-portal-dev|oci-employee-portal-dev|scale|||ingress:nginx-ingress-microk8s-controller|http://dev.employee.oci.local/|${DEV_DWELL}"
    "A3|dev|oci-dev|deployment/oci-marketing|oci-marketing|scale|||ingress:nginx-ingress-microk8s-controller|http://dev.oci.local/|${DEV_DWELL}"
    "A4|dev|oci-dev|deployment/oci-customer-bff-dev|oci-customer-bff-dev|scale|||ingress:nginx-ingress-microk8s-controller||${DEV_DWELL}"
    "A5|dev|oci-dev|deployment/oci-employee-bff-dev|oci-employee-bff-dev|scale|||ingress:nginx-ingress-microk8s-controller||${DEV_DWELL}"
    "A6|dev|oci-dev|deployment/oci-pebble-proxy-dev|oci-pebble-proxy-dev|scale|||oci-dev:oci-worker-dev,ingress:nginx-ingress-microk8s-controller|http://dev.pebble-proxy.oci.local/healthz|${DEV_DWELL}"
    "A7|dev|oci-dev|deployment/customer-bll|customer-bll|scale|||oci-dev:oci-customer-bff-dev,oci-dev:oci-worker-dev||${DEV_DWELL}"
    "A8|dev|oci-dev|deployment/internal-bll|internal-bll|scale|||oci-dev:oci-employee-bff-dev,oci-dev:oci-worker-dev||${DEV_DWELL}"
    "A9|dev|oci-dev|deployment/workforce-api|workforce-api|scale|||oci-dev:internal-bll,oci-dev:customer-bll,oci-dev:oci-employee-bff-dev,oci-dev:oci-customer-bff-dev||${DEV_DWELL}"
    "A10|dev|oci-dev|deployment/facilities-api|facilities-api|scale|||oci-dev:internal-bll,oci-dev:oci-worker-dev,oci-dev:oci-employee-bff-dev||${DEV_DWELL}"
    "A11|dev|oci-dev|deployment/core-invest-api|core-invest-api|scale|||oci-dev:customer-bll,oci-dev:internal-bll,oci-dev:oci-customer-bff-dev,oci-dev:oci-worker-dev||${DEV_DWELL}"
    "A12|dev|oci-dev|deployment/ops-api|ops-api|scale|||oci-dev:internal-bll,oci-dev:customer-bll,oci-dev:oci-employee-bff-dev,oci-dev:oci-customer-bff-dev||${DEV_DWELL}"
    "A13|dev|oci-dev|deployment/oci-worker-dev|oci-worker-dev|scale|||oci-dev:customer-bll,oci-dev:internal-bll,oci-dev:rabbitmq|rmq_queues|${DEV_DWELL}"
    "A14|dev|oci-dev|statefulset/postgres-dev|postgres-dev|scale|||oci-dev:workforce-api,oci-dev:facilities-api,oci-dev:core-invest-api,oci-dev:ops-api,oci-dev:customer-bll,oci-dev:internal-bll||${DEV_DWELL}"
    "A15|dev|oci-dev|deployment/valkey|valkey|scale|||oci-dev:oci-customer-bff-dev,oci-dev:oci-employee-bff-dev,oci-dev:customer-bll,oci-dev:internal-bll||${DEV_DWELL}"
    "A16|dev|oci-dev|deployment/rabbitmq|rabbitmq|scale|||oci-dev:oci-worker-dev,oci-dev:customer-bll,oci-dev:internal-bll||${DEV_DWELL}"
    "A17|dev|oci-dev|deployment/meilisearch|meilisearch|scale|||oci-dev:internal-bll,oci-dev:oci-employee-bff-dev||${DEV_DWELL}"
    "A18|dev|oci-dev|deployment/zitadel|zitadel|scale|||oci-dev:oci-customer-bff-dev|http://dev.auth.oci.local/debug/healthz|${DEV_DWELL}"

    # Group B: oci-internal COTS & DB (6 workloads — excludes forgejo-runner)
    "B1|internal|oci-internal|deployment/forgejo|forgejo|scale|||oci-dev:oci-employee-bff-dev,oci-dev:internal-bll||${INTERNAL_DWELL}"
    "B2|internal|oci-internal|deployment/nextcloud|nextcloud|scale|||oci-dev:internal-bll,oci-dev:oci-employee-bff-dev||${INTERNAL_DWELL}"
    "B3|internal|oci-internal|deployment/snipe-it|snipe-it|scale|||oci-dev:internal-bll,oci-dev:oci-employee-bff-dev||${INTERNAL_DWELL}"
    "B4|internal|oci-internal|deployment/erpnext-frappe|erpnext-frappe|scale|||oci-dev:internal-bll,oci-dev:oci-employee-bff-dev||${INTERNAL_DWELL}"
    "B5|internal|oci-internal|deployment/frappe-hr|frappe-hr|scale|||oci-dev:internal-bll,oci-dev:oci-employee-bff-dev||${INTERNAL_DWELL}"
    "B6|internal|oci-internal|statefulset/postgres-internal|postgres-internal|scale|||oci-internal:forgejo,oci-internal:nextcloud,oci-internal:snipe-it,oci-internal:erpnext-frappe,oci-dev:oci-employee-bff-dev,oci-dev:internal-bll||${INTERNAL_DWELL}"

    # Group C: external-sim -dev mock faults (9 scenarios + built-in -fault-scenarios gate)
    "C1|mocks|external-sim|deployment/mock-bank-dev|mock-bank-dev|mock_fault|bank|{\"outage\":true}|oci-dev:customer-bll,oci-dev:oci-customer-bff-dev,oci-dev:oci-worker-dev,external-sim:mock-bank-dev|rmq_queues|${MOCK_DWELL}"
    "C2|mocks|external-sim|deployment/mock-bank-dev|mock-bank-dev|mock_fault|bank|{\"latency_ms\":6500}|oci-dev:customer-bll,oci-dev:oci-customer-bff-dev,oci-dev:oci-worker-dev,external-sim:mock-bank-dev|rmq_queues|${MOCK_DWELL}"
    "C3|mocks|external-sim|deployment/mock-bank-dev|mock-bank-dev|mock_fault|bank|{\"reject_transfers\":true}|oci-dev:customer-bll,oci-dev:oci-customer-bff-dev,oci-dev:oci-worker-dev,external-sim:mock-bank-dev|rmq_queues|${MOCK_DWELL}"
    "C4|mocks|external-sim|deployment/mock-bank-dev|mock-bank-dev|scale|||oci-dev:customer-bll,oci-dev:oci-customer-bff-dev,oci-dev:oci-worker-dev|rmq_queues|${MOCK_DWELL}"
    "C5|mocks|external-sim|deployment/mock-broker-dev|mock-broker-dev|mock_fault|broker|{\"outage\":true}|oci-dev:customer-bll,oci-dev:oci-customer-bff-dev,external-sim:mock-broker-dev||${MOCK_DWELL}"
    "C6|mocks|external-sim|deployment/mock-broker-dev|mock-broker-dev|mock_fault|broker|{\"latency_ms\":6500}|oci-dev:customer-bll,oci-dev:oci-customer-bff-dev,external-sim:mock-broker-dev||${MOCK_DWELL}"
    "C7|mocks|external-sim|deployment/mock-broker-dev|mock-broker-dev|mock_fault|broker|{\"reject_orders\":true}|oci-dev:customer-bll,oci-dev:oci-customer-bff-dev,external-sim:mock-broker-dev||${MOCK_DWELL}"
    "C8|mocks|external-sim|deployment/mock-broker-dev|mock-broker-dev|mock_fault|broker|{\"halted_symbols\":[\"NVDA\",\"AAPL\",\"MSFT\",\"TSLA\",\"AMZN\"]}|oci-dev:customer-bll,oci-dev:oci-customer-bff-dev,external-sim:mock-broker-dev||${MOCK_DWELL}"
    "C9|mocks|external-sim|deployment/mock-broker-dev|mock-broker-dev|scale|||oci-dev:customer-bll,oci-dev:oci-customer-bff-dev||${MOCK_DWELL}"
    "C10|mocks|external-sim|deployment/mock-broker-dev|mock-broker-dev|fault_suite|||oci-dev:customer-bll,oci-dev:oci-customer-bff-dev||15"
)

should_run() {
    local sid="$1"
    local sgrp="$2"
    if [[ -n "${ONLY_ID}" ]]; then
        [[ "${sid}" == "${ONLY_ID}" ]]
        return
    fi
    if [[ "${GROUP}" == "all" || "${GROUP}" == "${sgrp}" ]]; then
        return 0
    fi
    return 1
}

if [[ "${DRY_RUN}" == true ]]; then
    echo "DRY RUN: Validating scenario guards and printing execution plan..."
    for entry in "${SCENARIOS[@]}"; do
        IFS='|' read -r sid sgrp sns sref slabel smode sfault_target sfault_payload scallers sprobe sdwell <<< "${entry}"
        if should_run "${sid}" "${sgrp}"; then
            guard_target "${sns}" "${sref}"
            echo "  [${sid}] group=${sgrp} ns=${sns} ref=${sref} mode=${smode} fault=${sfault_target}:${sfault_payload} dwell=${sdwell}s"
        fi
    done
    echo "Dry run complete."
    exit 0
fi

mkdir -p "${OUT_DIR}"

echo "================================================================================"
echo "  Orange Cat Investments (OCI) - Rigorous Chaos & Observability Audit"
echo "================================================================================"
echo "Group          : ${GROUP} (only: ${ONLY_ID:-none})"
echo "Ingress IP     : ${INGRESS_IP}"
echo "Output Dir     : ${OUT_DIR}"
echo "Dwell (dev/int): ${DEV_DWELL}s / ${INTERNAL_DWELL}s"
echo "================================================================================"

# Build simulator binary once
echo "[preflight] Building simulator binary (${SIM_BIN})..."
(cd "${REPO_ROOT}" && go build -o "${SIM_BIN}" ./simulation/cmd/simulator)

# Ensure Forgejo sim tokens exist
if [[ ! -f "${TOKENS_FILE}" ]]; then
    echo "[preflight] Provisioning Forgejo simulation tokens..."
    "${REPO_ROOT}/scripts/provision-forgejo-sim-tokens.sh" "${TOKENS_FILE}"
fi

# Start port-forwards to observability backends (Prometheus, Loki, oci-dev RabbitMQ mgmt)
start_pf() {
    local ns="$1"
    local svc="$2"
    local mapping="$3"
    local check_url="$4"
    kubectl port-forward -n "${ns}" "svc/${svc}" "${mapping}" >/dev/null 2>&1 &
    local pid=$!
    PF_PIDS+=("${pid}")
    for _ in $(seq 1 15); do
        if curl -sf --max-time 2 "${check_url}" >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
    done
    echo "WARNING: Port-forward ${ns}/${svc} (${mapping}) did not respond on ${check_url}" >&2
    return 0
}

echo "[preflight] Opening port-forwards for Prometheus, Loki, and oci-dev RabbitMQ..."
start_pf monitoring prometheus-service "${PROM_PORT}:9090" "http://127.0.0.1:${PROM_PORT}/-/ready"
start_pf logging loki-service "${LOKI_PORT}:3100" "http://127.0.0.1:${LOKI_PORT}/ready"
start_pf oci-dev rabbitmq-service "${RMQ_PORT}:15672" "http://127.0.0.1:${RMQ_PORT}/api/overview"

clear_dev_ratelimits() {
    kubectl exec -n oci-dev deployment/valkey -c valkey -- \
        valkey-cli eval "for _,k in ipairs(redis.call('keys','ratelimit:*')) do redis.call('del',k) end" 0 >/dev/null 2>&1 || true
}

fetch_prom_alerts() {
    curl -sf --max-time 8 "http://127.0.0.1:${PROM_PORT}/api/v1/alerts" 2>/dev/null || echo '{"status":"error","data":{"alerts":[]}}'
}

fetch_prom_targets() {
    curl -sf --max-time 8 "http://127.0.0.1:${PROM_PORT}/api/v1/targets?state=active" 2>/dev/null || echo '{"status":"error","data":{"activeTargets":[]}}'
}

fetch_gatus_statuses() {
    curl -sf --max-time 8 --resolve "status.oci.local:80:${INGRESS_IP}" "http://status.oci.local/api/v1/endpoints/statuses" 2>/dev/null || echo '[]'
}

fetch_loki_count() {
    local start_ns="$1"
    local end_ns="$2"
    local query='{namespace=~"oci-dev|oci-internal|external-sim"}'
    curl -sf --max-time 8 -G "http://127.0.0.1:${LOKI_PORT}/loki/api/v1/query_range" \
        --data-urlencode "query=${query}" \
        --data-urlencode "start=${start_ns}" \
        --data-urlencode "end=${end_ns}" \
        --data-urlencode "limit=20" 2>/dev/null || echo '{"status":"error","data":{"result":[]}}'
}

fetch_loki_all_labels() {
    curl -sf --max-time 8 "http://127.0.0.1:${LOKI_PORT}/loki/api/v1/labels" 2>/dev/null || echo '{"status":"error","data":[]}'
}

RMQ_USER="$(kubectl get secret oci-dev-secrets -n oci-dev -o jsonpath='{.data.RABBITMQ_DEFAULT_USER}' 2>/dev/null | base64 -d 2>/dev/null || echo 'oci')"
RMQ_PASS="$(kubectl get secret oci-dev-secrets -n oci-dev -o jsonpath='{.data.RABBITMQ_DEFAULT_PASS}' 2>/dev/null | base64 -d 2>/dev/null || echo 'oci_rabbit_pass')"

fetch_rmq_queues() {
    # Re-open RMQ port-forward if RabbitMQ restarted
    if ! curl -sf --max-time 3 -u "${RMQ_USER}:${RMQ_PASS}" "http://127.0.0.1:${RMQ_PORT}/api/queues" >/dev/null 2>&1; then
        kubectl port-forward -n oci-dev svc/rabbitmq-service "${RMQ_PORT}:15672" >/dev/null 2>&1 &
        PF_PIDS+=("$!")
        sleep 2
    fi
    curl -sf --max-time 5 -u "${RMQ_USER}:${RMQ_PASS}" "http://127.0.0.1:${RMQ_PORT}/api/queues" 2>/dev/null || echo '[]'
}

run_simulator_once() {
    local log_out="$1"
    local extra_flags=()
    if [[ -f "${TOKENS_FILE}" ]]; then
        extra_flags+=("-tokens-file" "${TOKENS_FILE}")
    fi
    OCI_INGRESS_IP="${INGRESS_IP}" "${SIM_BIN}" \
        -mode step \
        -iterations 1 \
        -bff-url "http://dev.api.oci.local" \
        -cust-bff-url "http://dev.api.oci.local" \
        -forgejo-url "http://forgejo.oci.local" \
        -broker-url "http://broker-api-dev.sim.oci.local" \
        -bank-url "http://bank-api-dev.sim.oci.local" \
        "${extra_flags[@]}" \
        -delay 10 \
        -verbose=true > "${log_out}" 2>&1 || true
}

# Capture baseline state
echo "[preflight] Resetting -dev mock faults and Valkey rate limits..."
reset_dev_mock_faults
clear_dev_ratelimits

echo "[preflight] Capturing baseline Prometheus alerts, Gatus statuses, Loki labels, and clean simulator run..."
fetch_prom_alerts > "${OUT_DIR}/baseline_prom_alerts.json"
fetch_prom_targets > "${OUT_DIR}/baseline_prom_targets.json"
fetch_gatus_statuses > "${OUT_DIR}/baseline_gatus.json"
fetch_loki_all_labels > "${OUT_DIR}/baseline_loki_labels.json"
run_simulator_once "${OUT_DIR}/baseline_sim.log"
grep -E "Total Actions Executed|Successful Actions|Failed Actions" "${OUT_DIR}/baseline_sim.log" || true

# Execute each selected scenario
for entry in "${SCENARIOS[@]}"; do
    IFS='|' read -r sid sgrp sns sref slabel smode sfault_target sfault_payload scallers sprobe sdwell <<< "${entry}"
    if ! should_run "${sid}" "${sgrp}"; then
        continue
    fi

    guard_target "${sns}" "${sref}"
    sdir="${OUT_DIR}/${sid}"
    mkdir -p "${sdir}"

    echo ""
    echo "--------------------------------------------------------------------------------"
    echo "  Scenario ${sid}: ${sref} (${sns}) [mode=${smode}, dwell=${sdwell}s]"
    echo "--------------------------------------------------------------------------------"

    # Clear single-IP rate limits before scenario (unless testing valkey itself)
    if [[ "${slabel}" != "valkey" ]]; then
        clear_dev_ratelimits
    fi

    T_START_ISO="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    T_START_EPOCH="$(date -u +%s)"
    T_START_NS="${T_START_EPOCH}000000000"

    # 1. Inject fault / scale down
    if [[ "${smode}" == "scale" ]]; then
        ACTIVE_SCALE_NS="${sns}"
        ACTIVE_SCALE_REF="${sref}"
        ACTIVE_SCALE_ORIG=1
        echo "[${sid}] Scaling ${sref} in ${sns} to 0..."
        kubectl scale "${sref}" -n "${sns}" --replicas=0 >/dev/null
        # Wait until all pods for this app terminate
        for _ in $(seq 1 60); do
            pod_count="$(kubectl get pods -n "${sns}" -l "app=${slabel}" --no-headers 2>/dev/null | wc -l | tr -d ' ')"
            if [[ "${pod_count}" == "0" ]]; then
                break
            fi
            sleep 1
        done
        echo "[${sid}] Target pods remaining: $(kubectl get pods -n "${sns}" -l "app=${slabel}" --no-headers 2>/dev/null | wc -l | tr -d ' ')"
    elif [[ "${smode}" == "mock_fault" ]]; then
        ACTIVE_MOCK_FAULTS=true
        local_auth_headers=()
        fault_host="bank-api-dev.sim.oci.local"
        if [[ "${sfault_target}" == "broker" ]]; then
            fault_host="broker-api-dev.sim.oci.local"
            broker_key="$(kubectl get secret mock-broker-dev-credentials -n external-sim -o jsonpath='{.data.APCA_API_KEY_ID}' 2>/dev/null | base64 -d 2>/dev/null || echo 'CK_OCI_TEST_KEY_01')"
            broker_sec="$(kubectl get secret mock-broker-dev-credentials -n external-sim -o jsonpath='{.data.APCA_API_SECRET_KEY}' 2>/dev/null | base64 -d 2>/dev/null || echo 'SK_OCI_TEST_SECRET_01')"
            local_auth_headers=(-H "APCA-API-KEY-ID: ${broker_key}" -H "APCA-API-SECRET-KEY: ${broker_sec}")
        else
            bank_cid="$(kubectl get secret mock-bank-dev-credentials -n external-sim -o jsonpath='{.data.PLAID_CLIENT_ID}' 2>/dev/null | base64 -d 2>/dev/null || echo 'client_oci_sim_01')"
            bank_sec="$(kubectl get secret mock-bank-dev-credentials -n external-sim -o jsonpath='{.data.PLAID_SECRET}' 2>/dev/null | base64 -d 2>/dev/null || echo 'sec_oci_sim_test_8829')"
            local_auth_headers=(-H "PLAID-CLIENT-ID: ${bank_cid}" -H "PLAID-SECRET: ${bank_sec}")
        fi
        echo "[${sid}] Injecting fault on http://${fault_host}/sim/faults: ${sfault_payload}"
        http_code="$(curl -s -o "${sdir}/fault_inject_resp.txt" -w "%{http_code}" \
            -X PUT --resolve "${fault_host}:80:${INGRESS_IP}" \
            -H "Content-Type: application/json" \
            "${local_auth_headers[@]}" \
            -d "${sfault_payload}" \
            "http://${fault_host}/sim/faults")"
        echo "[${sid}] Fault inject HTTP status: ${http_code}"
    fi

    # 2. Run simulator (or -fault-scenarios gate for C10)
    if [[ "${smode}" == "fault_suite" ]]; then
        echo "[${sid}] Running simulator -fault-scenarios gate..."
        extra_flags=()
        [[ -f "${TOKENS_FILE}" ]] && extra_flags+=("-tokens-file" "${TOKENS_FILE}")
        OCI_INGRESS_IP="${INGRESS_IP}" "${SIM_BIN}" \
            -mode live \
            -persona cust-longterm-arthur \
            -fault-scenarios \
            -bff-url "http://dev.api.oci.local" \
            -cust-bff-url "http://dev.api.oci.local" \
            -forgejo-url "http://forgejo.oci.local" \
            -broker-url "http://broker-api-dev.sim.oci.local" \
            -bank-url "http://bank-api-dev.sim.oci.local" \
            "${extra_flags[@]}" \
            -delay 10 > "${sdir}/sim.log" 2>&1 || true
    else
        echo "[${sid}] Running 13-persona simulator against oci-dev..."
        run_simulator_once "${sdir}/sim.log"
    fi

    # Extra HTTP probe if configured (SPA / marketing / pebble-proxy / zitadel)
    probe_result=""
    if [[ "${sprobe}" == http://* ]]; then
        probe_host="$(echo "${sprobe}" | sed -E 's|https?://([^/:]+).*|\1|')"
        probe_code="$(curl -s --max-time 8 -o /dev/null -w "%{http_code}" --resolve "${probe_host}:80:${INGRESS_IP}" "${sprobe}" || echo "000")"
        probe_result="${sprobe} -> HTTP ${probe_code}"
        echo "[${sid}] Extra probe: ${probe_result}"
    fi
    echo "${probe_result}" > "${sdir}/extra_probe.txt"

    # 3. Alert evaluation dwell
    now_epoch="$(date -u +%s)"
    elapsed=$(( now_epoch - T_START_EPOCH ))
    if [[ "${elapsed}" -lt "${sdwell}" ]]; then
        rem=$(( sdwell - elapsed ))
        echo "[${sid}] Dwelling ${rem}s more for Prometheus/Gatus alert evaluation (total ${sdwell}s)..."
        sleep "${rem}"
    fi

    T_END_EPOCH="$(date -u +%s)"
    T_END_NS="${T_END_EPOCH}000000000"

    # 4. Collect observability evidence
    echo "[${sid}] Collecting Prometheus alerts/targets, Gatus statuses, Loki logs, and caller pod logs..."
    fetch_prom_alerts > "${sdir}/prom_alerts.json"
    fetch_prom_targets > "${sdir}/prom_targets.json"
    fetch_gatus_statuses > "${sdir}/gatus.json"
    fetch_loki_count "${T_START_NS}" "${T_END_NS}" > "${sdir}/loki.json"
    if [[ "${sprobe}" == "rmq_queues" ]]; then
        fetch_rmq_queues > "${sdir}/rmq_queues.json"
    else
        echo '[]' > "${sdir}/rmq_queues.json"
    fi

    # Collect caller pod logs since T_START_ISO
    : > "${sdir}/caller_logs.txt"
    IFS=',' read -ra caller_list <<< "${scallers}"
    for caller in "${caller_list[@]}"; do
        [[ -z "${caller}" ]] && continue
        c_ns="${caller%%:*}"
        c_app="${caller#*:}"
        {
            echo "=== CALLER ${c_ns}/${c_app} (since ${T_START_ISO}) ==="
            if [[ "${c_ns}" == "ingress" ]]; then
                kubectl logs -n ingress -l name=nginx-ingress-microk8s --since-time="${T_START_ISO}" --tail=200 2>/dev/null \
                    | grep -iE "50[0-9]|40[0-9]|error|upstream|refused|timeout|dev\.|forgejo" | tail -n 40 || true
            else
                kubectl logs -n "${c_ns}" -l "app=${c_app}" --all-containers=true --since-time="${T_START_ISO}" --tail=300 2>/dev/null \
                    | tail -n 80 || true
            fi
        } >> "${sdir}/caller_logs.txt"
    done

    # 5. Restore target and verify recovery
    recovery_status="N/A"
    if [[ "${smode}" == "scale" ]]; then
        echo "[${sid}] Restoring ${sref} in ${sns} to 1..."
        kubectl scale "${sref}" -n "${sns}" --replicas=1 >/dev/null
        ACTIVE_SCALE_NS=""
        ACTIVE_SCALE_REF=""
        if kubectl rollout status "${sref}" -n "${sns}" --timeout=240s >/dev/null 2>&1; then
            recovery_status="READY"
        else
            recovery_status="ROLLOUT_TIMEOUT"
            echo "WARNING: [${sid}] Rollout timed out for ${sref} in ${sns}" >&2
        fi
        # Give dependent connection pools 5s to stabilize after restore
        sleep 5
    elif [[ "${smode}" == "mock_fault" ]]; then
        echo "[${sid}] Resetting -dev mock faults..."
        reset_dev_mock_faults
        recovery_status="FAULT_RESET"
    else
        reset_dev_mock_faults
        recovery_status="SUITE_DONE"
    fi

    # Post-restore quick recovery check (verify BFF healthz + lightweight action)
    bff_emp_code="$(curl -s --max-time 5 -o /dev/null -w "%{http_code}" --resolve "dev.employee-bff.oci.local:80:${INGRESS_IP}" "http://dev.employee-bff.oci.local/healthz" || echo "000")"
    bff_cust_code="$(curl -s --max-time 5 -o /dev/null -w "%{http_code}" --resolve "dev.customer-bff.oci.local:80:${INGRESS_IP}" "http://dev.customer-bff.oci.local/healthz" || echo "000")"
    echo "${recovery_status} (emp-bff=${bff_emp_code}, cust-bff=${bff_cust_code})" > "${sdir}/recovery.txt"

    # Parse per-scenario summary JSON via python3
    python3 - "${OUT_DIR}" "${sid}" "${sgrp}" "${sns}" "${sref}" "${smode}" "${sfault_target}" "${sfault_payload}" "${T_START_ISO}" <<'PY'
import json, os, re, sys

out_dir, sid, sgrp, sns, sref, smode, ft, fp, t_start = sys.argv[1:]
sdir = os.path.join(out_dir, sid)

def load_json(path, default):
    try:
        with open(path) as f:
            return json.load(f)
    except Exception:
        return default

def read_text(path):
    try:
        with open(path) as f:
            return f.read()
    except Exception:
        return ""

sim_txt = read_text(os.path.join(sdir, "sim.log"))
total_m = re.search(r"Total Actions Executed\s*:\s*(\d+)", sim_txt)
succ_m = re.search(r"Successful Actions\s*:\s*(\d+)", sim_txt)
fail_m = re.search(r"Failed Actions\s*:\s*(\d+)", sim_txt)

failed_actions = []
vendor_leaks = []
for line in sim_txt.splitlines():
    line_s = line.strip()
    if line_s.startswith("{"):
        try:
            obj = json.loads(line_s)
            if obj.get("msg") == "persona action failed":
                details = str(obj.get("details", ""))
                failed_actions.append({
                    "persona": obj.get("persona", ""),
                    "action": obj.get("action", ""),
                    "details": details[:220],
                })
                if re.search(r"\b(alpaca|plaid)\b", details, re.I):
                    vendor_leaks.append(details[:220])
            elif obj.get("msg") == "persona authentication handshake failed":
                failed_actions.append({
                    "persona": obj.get("persona_name", ""),
                    "action": "authenticate",
                    "details": str(obj.get("error", ""))[:220],
                })
        except Exception:
            pass

base_alerts = load_json(os.path.join(out_dir, "baseline_prom_alerts.json"), {})
cur_alerts = load_json(os.path.join(sdir, "prom_alerts.json"), {})

def alert_key(a):
    lbl = a.get("labels", {})
    return (lbl.get("alertname", ""), lbl.get("namespace", ""), lbl.get("pod", ""), lbl.get("instance", ""), lbl.get("queue", ""), a.get("state", ""))

base_set = {alert_key(a) for a in base_alerts.get("data", {}).get("alerts", [])}
new_alerts = []
all_alerts = []
for a in cur_alerts.get("data", {}).get("alerts", []):
    lbl = a.get("labels", {})
    rec = {
        "alertname": lbl.get("alertname", ""),
        "state": a.get("state", ""),
        "namespace": lbl.get("namespace", ""),
        "instance": lbl.get("instance", ""),
        "pod": lbl.get("pod", ""),
        "queue": lbl.get("queue", ""),
    }
    all_alerts.append(rec)
    if alert_key(a) not in base_set:
        new_alerts.append(rec)

cur_targets = load_json(os.path.join(sdir, "prom_targets.json"), {})
down_targets = []
for t in cur_targets.get("data", {}).get("activeTargets", []):
    if t.get("health") != "up":
        down_targets.append({
            "job": t.get("labels", {}).get("job", ""),
            "instance": t.get("labels", {}).get("instance", ""),
            "lastError": str(t.get("lastError", ""))[:140],
        })

gatus = load_json(os.path.join(sdir, "gatus.json"), [])
gatus_unhealthy = []
if isinstance(gatus, list):
    for ep in gatus:
        results = ep.get("results") or []
        if results and results[-1].get("success") is False:
            gatus_unhealthy.append({
                "name": ep.get("name", ""),
                "group": ep.get("group", ""),
                "status": results[-1].get("status", 0),
                "errors": results[-1].get("errors", []),
            })

loki = load_json(os.path.join(sdir, "loki.json"), {})
loki_streams = len(loki.get("data", {}).get("result", []) or [])

rmq = load_json(os.path.join(sdir, "rmq_queues.json"), [])
rmq_summary = []
if isinstance(rmq, list):
    for q in rmq:
        msgs = q.get("messages", 0) or 0
        cons = q.get("consumers", 0) or 0
        name = q.get("name", "")
        if msgs > 0 or ".dlq" in name or cons == 0:
            rmq_summary.append({"name": name, "messages": msgs, "consumers": cons})

caller_txt = read_text(os.path.join(sdir, "caller_logs.txt"))
error_lines = [
    ln for ln in caller_txt.splitlines()
    if re.search(r"ERROR|WARN|level=error|level=warn|refused|timeout|deadline|unavailable|failed|50[0-9]|422|412", ln, re.I)
    and not ln.startswith("=== CALLER")
]
log_vendor_leaks = [
    ln[:220] for ln in error_lines
    if re.search(r"\b(alpaca|plaid)\b", ln, re.I) and "mock-" not in ln.lower()
]

summary = {
    "id": sid,
    "group": sgrp,
    "namespace": sns,
    "ref": sref,
    "mode": smode,
    "fault_target": ft,
    "fault_payload": fp,
    "started_at": t_start,
    "sim_total": int(total_m.group(1)) if total_m else 0,
    "sim_success": int(succ_m.group(1)) if succ_m else 0,
    "sim_failures": int(fail_m.group(1)) if fail_m else 0,
    "failed_actions": failed_actions,
    "vendor_leaks_in_sim": vendor_leaks,
    "extra_probe": read_text(os.path.join(sdir, "extra_probe.txt")).strip(),
    "recovery": read_text(os.path.join(sdir, "recovery.txt")).strip(),
    "new_prom_alerts": new_alerts,
    "all_prom_alerts_count": len(all_alerts),
    "down_prom_targets": down_targets,
    "gatus_unhealthy": gatus_unhealthy,
    "loki_matching_streams": loki_streams,
    "rmq_queues": rmq_summary,
    "caller_error_line_count": len(error_lines),
    "caller_error_samples": [ln[:240] for ln in error_lines[:8]],
    "vendor_leaks_in_caller_logs": log_vendor_leaks[:5],
}

with open(os.path.join(sdir, "result.json"), "w") as f:
    json.dump(summary, f, indent=2)

print(
    f"[{sid}] DONE | sim={summary['sim_success']}/{summary['sim_total']} (fail={summary['sim_failures']}) "
    f"| probe='{summary['extra_probe']}' | caller_err_lines={summary['caller_error_line_count']} "
    f"| loki_streams={loki_streams} | new_prom_alerts={len(new_alerts)} "
    f"| gatus_unhealthy={len(gatus_unhealthy)} | recovery={summary['recovery']}"
)
PY
done

# Post-run clean verification simulator pass
echo ""
echo "================================================================================"
echo "  Post-Run Hygiene & Clean Simulator Pass"
echo "================================================================================"
reset_dev_mock_faults
clear_dev_ratelimits
run_simulator_once "${OUT_DIR}/post_sim.log"
grep -E "Total Actions Executed|Successful Actions|Failed Actions" "${OUT_DIR}/post_sim.log" || true

# Aggregate Markdown table
python3 - "${OUT_DIR}" <<'PY'
import glob, json, os, sys

out_dir = sys.argv[1]
results = []
for p in sorted(glob.glob(os.path.join(out_dir, "*", "result.json")), key=lambda x: (os.path.basename(os.path.dirname(x))[0], int(os.path.basename(os.path.dirname(x))[1:]))):
    with open(p) as f:
        results.append(json.load(f))

md_path = os.path.join(out_dir, "summary.md")
with open(md_path, "w") as f:
    f.write("# Chaos Audit Raw Summary\n\n")
    f.write("| ID | Target | Mode/Fault | Sim (Pass/Total, Fail) | Extra Probe | Caller Err Lines | Loki Streams | New Prom Alerts | Gatus Unhealthy | Recovery |\n")
    f.write("| :-: | :--- | :--- | :--- | :--- | :-: | :-: | :--- | :--- | :--- |\n")
    for r in results:
        fault = r["mode"] if not r["fault_payload"] else f"{r['fault_target']}:{r['fault_payload']}"
        prom_names = ", ".join(f"{a['alertname']}({a['state']})" for a in r["new_prom_alerts"]) or "NONE"
        gatus_names = ", ".join(g["name"] for g in r["gatus_unhealthy"]) or "NONE"
        f.write(
            f"| {r['id']} | `{r['namespace']}/{r['ref']}` | `{fault}` | "
            f"{r['sim_success']}/{r['sim_total']} (fail {r['sim_failures']}) | "
            f"{r['extra_probe'] or '-'} | {r['caller_error_line_count']} | {r['loki_matching_streams']} | "
            f"{prom_names} | {gatus_names} | {r['recovery']} |\n"
        )

print(f"Summary table written to: {md_path}")
PY

echo "================================================================================"
echo "  Chaos Audit Complete! Evidence stored in: ${OUT_DIR}"
echo "================================================================================"
