#!/usr/bin/env bash
# scripts/run-local-dev.sh
#
# Launches the complete 5-Tier OCI Backend Architecture for local development:
#   Tier 4 - Domain APIs:
#     - workforce-api   (:50051)
#     - facilities-api  (:50052)
#     - core-invest-api (:50053)
#     - ops-api         (:50054)
#   Tier 3 - Business Logic Layer (BLL):
#     - customer-bll    (:8085)
#     - internal-bll    (:8086)
#   Tier 2 - Backend-for-Frontend (BFF):
#     - employee-bff    (:8081)
#     - customer-bff    (:8083)
#     - pebble-proxy    (:8082)
#   Tier 5 - Async Workers:
#     - worker          (:8080 healthz / event subscriber)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default connection environment variables
export DATABASE_URL="${DATABASE_URL:-"postgres://oci:catnip@localhost:5432/oci?sslmode=disable"}"
export VALKEY_ADDR="${VALKEY_ADDR:-"localhost:6379"}"
export VALKEY_PASSWORD="${VALKEY_PASSWORD:-""}"
export RABBITMQ_URL="${RABBITMQ_URL:-"amqp://guest:guest@localhost:5672/"}"
export AMQP_URL="${RABBITMQ_URL}"

# Inter-service routing URLs (BFF -> BLL -> Domain APIs)
export WORKFORCE_API_URL="${WORKFORCE_API_URL:-"localhost:50051"}"
export WORKFORCE_GRPC_ADDR="${WORKFORCE_API_URL}"
export FACILITIES_API_URL="${FACILITIES_API_URL:-"localhost:50052"}"
export FACILITIES_GRPC_ADDR="${FACILITIES_API_URL}"
export CORE_INVEST_API_URL="${CORE_INVEST_API_URL:-"localhost:50053"}"
export CORE_INVEST_GRPC_ADDR="${CORE_INVEST_API_URL}"
export OPS_API_URL="${OPS_API_URL:-"localhost:50054"}"
export OPS_GRPC_ADDR="${OPS_API_URL}"

export CUSTOMER_BLL_URL="${CUSTOMER_BLL_URL:-"localhost:8085"}"
export INTERNAL_BLL_URL="${INTERNAL_BLL_URL:-"localhost:8086"}"
export EMPLOYEE_BFF_URL="${EMPLOYEE_BFF_URL:-"http://localhost:8081"}"
export CUSTOMER_BFF_URL="${CUSTOMER_BFF_URL:-"http://localhost:8083"}"
export PEBBLE_PROXY_URL="${PEBBLE_PROXY_URL:-"http://localhost:8082"}"

# Feature flags & defaults
export ENABLE_FELINE_WORKFORCE="${ENABLE_FELINE_WORKFORCE:-"true"}"
export ENABLE_CORE_INVEST="${ENABLE_CORE_INVEST:-"true"}"
export ENABLE_PEBBLE_GATEWAY="${ENABLE_PEBBLE_GATEWAY:-"true"}"
export HEALTH_PORT="${HEALTH_PORT:-"8080"}"

LOG_DIR="${REPO_ROOT}/logs/local-dev"
mkdir -p "${LOG_DIR}"

declare -a PIDS=()
declare -a SERVICE_NAMES=()

cleanup() {
    echo ""
    echo "================================================================================"
    echo "  Shutting down OCI 5-Tier Local Development Stack..."
    echo "================================================================================"
    for i in "${!PIDS[@]}"; do
        pid="${PIDS[$i]}"
        name="${SERVICE_NAMES[$i]}"
        if kill -0 "${pid}" 2>/dev/null; then
            echo "Terminating ${name} (PID ${pid})..."
            kill "${pid}" 2>/dev/null || true
        fi
    done

    # Give processes a moment to gracefully terminate
    sleep 1

    # Force kill any lingering processes
    for pid in "${PIDS[@]}"; do
        if kill -0 "${pid}" 2>/dev/null; then
            kill -9 "${pid}" 2>/dev/null || true
        fi
    done
    echo "All local development backend services stopped."
}

trap cleanup EXIT INT TERM

start_service() {
    local name="$1"
    local cmd_dir="$2"
    local port="$3"
    local log_file="${LOG_DIR}/${name}.log"

    echo "  -> Starting ${name} (port ${port})..."
    PORT="${port}" go run "${REPO_ROOT}/${cmd_dir}" > "${log_file}" 2>&1 &
    local pid=$!
    PIDS+=("${pid}")
    SERVICE_NAMES+=("${name}")
}

echo "================================================================================"
echo "  Orange Cat Investments (OCI) - Production 5-Tier Local Topology Launcher"
echo "================================================================================"
echo "Database URL : ${DATABASE_URL}"
echo "Valkey Addr  : ${VALKEY_ADDR}"
echo "RabbitMQ URL : ${RABBITMQ_URL}"
echo "Logs Directory: ${LOG_DIR}"
echo "================================================================================"

echo ""
echo "[1/4] Launching Tier 4: Pure gRPC Domain APIs..."
start_service "workforce-api"   "cmd/workforce-api"   "50051"
start_service "facilities-api"  "cmd/facilities-api"  "50052"
start_service "core-invest-api" "cmd/core-invest-api" "50053"
start_service "ops-api"         "cmd/ops-api"         "50054"

# Wait briefly for Domain APIs to bind
sleep 2

echo ""
echo "[2/4] Launching Tier 3: Business Logic Layer (BLL)..."
start_service "customer-bll"    "cmd/customer-bll"    "8085"
start_service "internal-bll"    "cmd/internal-bll"    "8086"

# Wait briefly for BLL APIs to bind
sleep 2

echo ""
echo "[3/4] Launching Tier 2: Backend-for-Frontend (BFF) Proxies..."
start_service "employee-bff"    "cmd/employee-bff"    "8081"
start_service "pebble-proxy"    "cmd/pebble-proxy"    "8082"
start_service "customer-bff"    "cmd/customer-bff"    "8083"

echo ""
echo "[4/4] Launching Tier 5: Background Event Worker..."
start_service "worker"          "cmd/worker"          "8080"

echo ""
echo "================================================================================"
echo "  All 5-Tier Services Running Successfully!"
echo "================================================================================"
echo "  - Employee BFF        : http://localhost:8081"
echo "  - Pebble Gateway      : http://localhost:8082"
echo "  - Customer BFF        : http://localhost:8083"
echo "  - Customer BLL        : localhost:8085 (gRPC)"
echo "  - Internal BLL        : localhost:8086 (gRPC)"
echo "  - Workforce Domain API: localhost:50051 (gRPC)"
echo "  - Facilities Domain API: localhost:50052 (gRPC)"
echo "  - Core Invest Domain API: localhost:50053 (gRPC)"
echo "  - Ops Domain API      : localhost:50054 (gRPC)"
echo "  - Background Worker   : http://localhost:8080/healthz"
echo "================================================================================"
echo "Press Ctrl+C to stop all services."
echo ""

# Wait on all background jobs
wait
