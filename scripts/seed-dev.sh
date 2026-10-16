#!/usr/bin/env bash
set -euo pipefail

# Orange Cat Investments (OCI) - Development Database Seeding Script (`seed-dev.sh`)
# Populates development and simulation fixtures into the 'oci-dev' namespace database ONLY.
#
# CRITICAL SAFETY INVARIANT:
# This script is strictly forbidden from running against the production namespace 'oci-core'.
# Production databases must remain clean with only the initial bootstrap administrator account.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
SEED_SQL="${REPO_ROOT}/scripts/seed-dev.sql"

TARGET_NAMESPACE="oci-dev"
TARGET_DB="${DEV_DATABASE_URL:-}"

# Parse optional arguments
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
        --help|-h)
            echo "Usage: $0 [--namespace oci-dev] [--db-url postgresql://...]"
            echo "Note: Target namespace MUST be 'oci-dev'. Running against 'oci-core' is blocked."
            exit 0
            ;;
        *)
            echo "Unknown argument: $1" >&2
            exit 1
            ;;
    esac
done

echo "================================================================================"
echo "  Orange Cat Investments (OCI) - Development Database Seeder"
echo "================================================================================"
echo "Target Namespace: ${TARGET_NAMESPACE}"

# 1. Strict Production Guard
if [[ "${TARGET_NAMESPACE}" == "oci-core" || "${TARGET_NAMESPACE}" == "prod"* || "${TARGET_NAMESPACE}" == *"production"* ]]; then
    echo "❌ CRITICAL SECURITY ERROR: Seeding script cannot target production namespace '${TARGET_NAMESPACE}'!" >&2
    echo "Production (oci-core) must remain clean of mock seed fixtures." >&2
    echo "Seed data may only be populated in development ('oci-dev')." >&2
    exit 1
fi

if [[ -n "${TARGET_DB}" ]]; then
    if [[ "${TARGET_DB}" == *"oci-core"* || "${TARGET_DB}" == *"/oci?"* || "${TARGET_DB}" == *"/oci\""* ]]; then
        echo "❌ CRITICAL SECURITY ERROR: Database URL appears to target production 'oci-core' database!" >&2
        echo "Seed data is strictly restricted to development ('oci-dev')." >&2
        exit 1
    fi
fi

if [[ ! -f "${SEED_SQL}" ]]; then
    echo "ERROR: Seed SQL file not found at ${SEED_SQL}" >&2
    exit 1
fi

# 2. Execute Seeding
if [[ -n "${TARGET_DB}" ]]; then
    echo "Applying seed data directly via database connection string..."
    psql "${TARGET_DB}" -f "${SEED_SQL}"
elif command -v kubectl &>/dev/null; then
    echo "Applying seed data via kubectl to StatefulSet 'postgres-dev' in namespace '${TARGET_NAMESPACE}'..."
    if ! kubectl get namespace "${TARGET_NAMESPACE}" &>/dev/null; then
        echo "ERROR: Kubernetes namespace '${TARGET_NAMESPACE}' not found." >&2
        echo "Ensure the cluster is deployed and namespace exists: kubectl get ns ${TARGET_NAMESPACE}" >&2
        exit 1
    fi

    echo "Waiting for postgres-dev to be ready..."
    kubectl rollout status statefulset/postgres-dev -n "${TARGET_NAMESPACE}" --timeout=60s || true

    kubectl exec -i -n "${TARGET_NAMESPACE}" statefulset/postgres-dev -- \
        psql -U oci_admin -d oci_dev < "${SEED_SQL}"
else
    echo "ERROR: Neither TARGET_DB nor 'kubectl' command found." >&2
    echo "Please set DEV_DATABASE_URL or ensure kubectl is configured for namespace '${TARGET_NAMESPACE}'." >&2
    exit 1
fi

echo ""
echo "================================================================================"
echo "  ✅ Development Database Seeding Completed Successfully for ${TARGET_NAMESPACE}!"
echo "================================================================================"
echo "Seeded entities: Habitat Zones, Hardware Devices, Firmware Releases,"
echo "                 Workforce Personas, Portfolios, Holdings, Review Cycles & Reports."
echo "================================================================================"
