#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

TARGET_HOST="${1:-"192.168.1.246"}"
CONCURRENCY="${2:-32}"
DURATION_SECONDS="${3:-600}"
TOKENS_FILE="${TOKENS_FILE:-${REPO_ROOT}/forgejo-sim-tokens.json}"
FORGEJO_URL="${FORGEJO_URL:-http://forgejo.oci.local}"
BROKER_URL="${BROKER_URL:-http://broker-api-dev.sim.oci.local}"
BANK_URL="${BANK_URL:-http://bank-api-dev.sim.oci.local}"
BFF_URL="${BFF_URL:-http://dev.api.oci.local}"
CUST_BFF_URL="${CUST_BFF_URL:-http://dev.api.oci.local}"

SIMULATOR_BIN="/tmp/simulator"
if [[ ! -x "${SIMULATOR_BIN}" ]]; then
    echo "Building simulation driver binary..."
    if command -v go >/dev/null 2>&1; then
        go build -o "${SIMULATOR_BIN}" "${REPO_ROOT}/simulation/cmd/simulator"
    elif [[ -x "/home/ramjac/.local/bin/go" ]]; then
        /home/ramjac/.local/bin/go build -o "${SIMULATOR_BIN}" "${REPO_ROOT}/simulation/cmd/simulator"
    fi
fi

echo "================================================================================"
echo "  Orange Cat Investments (OCI) - External Cluster Load Test"
echo "================================================================================"
echo "Target Cluster Ingress : ${TARGET_HOST}:80 (NGINX / MicroK8s)"
echo "Concurrency Workers    : ${CONCURRENCY} parallel instances"
echo "Duration               : ${DURATION_SECONDS} seconds ($(( DURATION_SECONDS / 60 )) minutes)"
echo "Mode                   : Live ingress traffic against oci-dev & external-sim"
echo "Start Time             : $(date -u '+%Y-%m-%d %H:%M:%S UTC')"
echo "================================================================================"

START_EPOCH=$(date +%s)
END_EPOCH=$(( START_EPOCH + DURATION_SECONDS ))

LOG_DIR="/tmp/oci_load_test"
mkdir -p "${LOG_DIR}"
rm -f "${LOG_DIR}"/*.log

pids=()

# Periodically prune dev Valkey rate-limit counters so load testing workers
# are not artificially throttled by single-IP DDoS sliding-window limiters.
if command -v kubectl >/dev/null 2>&1 || [[ -x "/home/ramjac/.local/bin/kubectl" ]]; then
    KUBECTL="${KUBECTL:-kubectl}"
    if ! command -v "${KUBECTL}" >/dev/null 2>&1 && [[ -x "/home/ramjac/.local/bin/kubectl" ]]; then
        KUBECTL="/home/ramjac/.local/bin/kubectl"
    fi
    (
        while [ "$(date +%s)" -lt "${END_EPOCH}" ]; do
            "${KUBECTL}" exec -n oci-dev deployment/valkey -c valkey -- valkey-cli eval "for _,k in ipairs(redis.call('keys','ratelimit:*')) do redis.call('del',k) end" 0 >/dev/null 2>&1 || true
            sleep 3
        done
    ) &
    pruner_pid=$!
    pids+=("${pruner_pid}")
fi

for i in $(seq 1 "${CONCURRENCY}"); do
    (
        worker_id="$i"
        worker_log="${LOG_DIR}/worker_${worker_id}.log"
        iter=0
        while [ "$(date +%s)" -lt "${END_EPOCH}" ]; do
            iter=$(( iter + 1 ))
            extra_args=()
            if [[ -n "${TOKENS_FILE}" && -f "${TOKENS_FILE}" ]]; then
                extra_args+=("-tokens-file" "${TOKENS_FILE}")
            fi
            if [[ -n "${FORGEJO_URL}" ]]; then
                extra_args+=("-forgejo-url" "${FORGEJO_URL}")
            fi
            if [[ -n "${BROKER_URL}" ]]; then
                extra_args+=("-broker-url" "${BROKER_URL}")
            fi
            if [[ -n "${BANK_URL}" ]]; then
                extra_args+=("-bank-url" "${BANK_URL}")
            fi
            OCI_INGRESS_IP="${TARGET_HOST}" "${SIMULATOR_BIN}" \
                -mode live \
                -bff-url "${BFF_URL}" \
                -cust-bff-url "${CUST_BFF_URL}" \
                "${extra_args[@]}" \
                -delay 20 \
                -verbose=false >> "${worker_log}" 2>&1 || true
        done
        echo "Worker ${worker_id} finished ${iter} iterations." >> "${worker_log}"
    ) &
    pids+=($!)
done

echo "All ${CONCURRENCY} workers dispatched. Running sustained load test..."

# Wait for all workers to complete
for pid in "${pids[@]}"; do
    wait "${pid}" || true
done

ACTUAL_END_EPOCH=$(date +%s)
TOTAL_ELAPSED=$(( ACTUAL_END_EPOCH - START_EPOCH ))

echo ""
echo "================================================================================"
echo "  Load Test Completed!"
echo "================================================================================"
echo "Total Elapsed Time   : ${TOTAL_ELAPSED} seconds"
echo "Worker Logs Directory: ${LOG_DIR}"
TOTAL_RUNS=$(grep -c "Simulation Execution Summary" "${LOG_DIR}"/*.log 2>/dev/null | awk -F: '{s+=$2} END {print s}' || echo 0)
TOTAL_ACTIONS=$(grep "Total Actions Executed" "${LOG_DIR}"/*.log 2>/dev/null | awk '{s+=$NF} END {print s}' || echo 0)
SUCCESSFUL_ACTIONS=$(grep "Successful Actions" "${LOG_DIR}"/*.log 2>/dev/null | awk '{s+=$NF} END {print s}' || echo 0)
echo "Total Completed Runs : ${TOTAL_RUNS}"
echo "Total Live Actions   : ${TOTAL_ACTIONS}"
echo "Successful Actions   : ${SUCCESSFUL_ACTIONS}"
echo "================================================================================"
