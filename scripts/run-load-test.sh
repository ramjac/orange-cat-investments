#!/usr/bin/env bash
set -euo pipefail

TARGET_HOST="${1:-"192.168.1.246"}"
CONCURRENCY="${2:-32}"
DURATION_SECONDS="${3:-600}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

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

for i in $(seq 1 "${CONCURRENCY}"); do
    (
        worker_id="$i"
        worker_log="${LOG_DIR}/worker_${worker_id}.log"
        iter=0
        while [ $(date +%s) -lt "${END_EPOCH}" ]; do
            iter=$(( iter + 1 ))
            OCI_INGRESS_IP="${TARGET_HOST}" "${SIMULATOR_BIN}" \
                -mode live \
                -url "http://dev.api.oci.local" \
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
