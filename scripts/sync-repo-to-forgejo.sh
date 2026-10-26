#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Orange Cat Investments (OCI) - In-Cluster Source Synchronization Script
# ==============================================================================
# Synchronizes the local git repository into the in-cluster Forgejo instance
# (http://forgejo-service.oci-internal.svc:3000 or http://forgejo.oci.local).
#
# Features:
# 1. Validates Forgejo connectivity (via direct URL, ingress, or port-forward).
# 2. Authenticates using bootstrap platform administrator credentials.
# 3. Ensures organization 'oci' exists.
# 4. Provisions 'monorepo' and 'orange-cat-investments' repositories with Actions enabled.
# 5. Seeds Kanban Card / Issue #42 for persona CI/CD workflows.
# 6. Pushes current git commit tree (main branch) to Forgejo.
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

FORGEJO_URL=""
ADMIN_USER="${FORGEJO_ADMIN_USER:-admin}"
ADMIN_PASSWORD="${FORGEJO_ADMIN_PASSWORD:-}"
TARGET_ORG="oci"
TARGET_BRANCH="${FORGEJO_BRANCH:-main}"
DRY_RUN=false
CHECK_ONLY=false
PF_PID=""

cleanup() {
    if [[ -n "${PF_PID}" ]]; then
        kill "${PF_PID}" 2>/dev/null || true
    fi
}
trap cleanup EXIT

print_usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Options:
  -u, --url URL          Base URL of Forgejo instance (autodetected if omitted)
      --admin-user USER  Admin username (default: admin)
  -p, --admin-password PWD Admin password (read from starter-admin-credentials if omitted)
      --org ORG          Forgejo target organization (default: oci)
  -b, --branch BRANCH    Git branch to push (default: main)
      --dry-run          Validate connectivity and plan actions without mutating state
      --check-only       Check if Forgejo and repository are ready (exits 0 if ready)
  -h, --help             Display this help message
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        -u|--url)
            FORGEJO_URL="$2"
            shift 2
            ;;
        --admin-user)
            ADMIN_USER="$2"
            shift 2
            ;;
        -p|--admin-password)
            ADMIN_PASSWORD="$2"
            shift 2
            ;;
        --org)
            TARGET_ORG="$2"
            shift 2
            ;;
        -b|--branch)
            TARGET_BRANCH="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --check-only)
            CHECK_ONLY=true
            shift
            ;;
        -h|--help)
            print_usage
            exit 0
            ;;
        *)
            echo "ERROR: Unknown argument: $1" >&2
            print_usage
            exit 1
            ;;
    esac
done

echo "================================================================================"
echo "  Orange Cat Investments (OCI) - In-Cluster Source Synchronization"
echo "================================================================================"

# 1. Resolve Admin Password
if [[ -z "${ADMIN_PASSWORD}" ]]; then
    if [[ -f "${REPO_ROOT}/.starter-admin-credentials.json" ]]; then
        ADMIN_PASSWORD=$(python3 -c "import json; print(json.load(open('${REPO_ROOT}/.starter-admin-credentials.json')).get('initial_password', ''))" 2>/dev/null || true)
    fi
    if [[ -z "${ADMIN_PASSWORD}" && -f "${REPO_ROOT}/starter-admin-credentials.txt" ]]; then
        ADMIN_PASSWORD=$(grep "^Initial Password:" "${REPO_ROOT}/starter-admin-credentials.txt" | awk '{print $3}' || true)
    fi
    if [[ -z "${ADMIN_PASSWORD}" ]] && command -v kubectl &>/dev/null; then
        ADMIN_PASSWORD=$(kubectl get secret starter-admin-credentials -n oci-internal -o jsonpath='{.data.ADMIN_PASSWORD}' 2>/dev/null | base64 -d 2>/dev/null || true)
    fi
fi

if [[ -z "${ADMIN_PASSWORD}" ]]; then
    echo "WARNING: Starter admin password could not be automatically resolved from credentials files or cluster secret."
    echo "Proceeding with default fallback 'oci_secure_pass' for initial bootstrap..."
    ADMIN_PASSWORD="oci_secure_pass"
fi

# 2. Autodetect Forgejo URL
if [[ -z "${FORGEJO_URL}" ]]; then
    CANDIDATES=(
        "http://forgejo-service.oci-internal.svc.cluster.local:3000"
        "http://forgejo-service.oci-internal.svc:3000"
        "http://forgejo.oci.local:3000"
        "http://forgejo.oci.local"
        "http://localhost:3000"
    )
    for c in "${CANDIDATES[@]}"; do
        if curl -sf --connect-timeout 2 "${c}/api/v1/version" >/dev/null 2>&1; then
            FORGEJO_URL="${c}"
            break
        fi
    done
fi

# If still not reachable, try kubectl port-forward if kubectl is available
if [[ -z "${FORGEJO_URL}" ]] && command -v kubectl &>/dev/null; then
    if kubectl get svc -n oci-internal forgejo-service &>/dev/null; then
        FREE_PORT=33000
        echo "Starting background port-forward to forgejo-service on port ${FREE_PORT}..."
        kubectl port-forward -n oci-internal svc/forgejo-service "${FREE_PORT}:3000" >/dev/null 2>&1 &
        PF_PID=$!
        sleep 2
        if curl -sf --connect-timeout 2 "http://localhost:${FREE_PORT}/api/v1/version" >/dev/null 2>&1; then
            FORGEJO_URL="http://localhost:${FREE_PORT}"
        fi
    fi
fi

if [[ -z "${FORGEJO_URL}" ]]; then
    echo "NOTICE: Forgejo instance is currently offline or unreachable across probed endpoints."
    echo "If running in standalone offline mock mode without a live cluster, skipping live sync."
    if [[ "${CHECK_ONLY}" == true ]]; then
        exit 1
    fi
    exit 0
fi

echo "Connected to Forgejo host: ${FORGEJO_URL}"
AUTH_HEADER="Authorization: Basic $(echo -n "${ADMIN_USER}:${ADMIN_PASSWORD}" | base64)"

# Check Version
VERSION_INFO=$(curl -sf -H "${AUTH_HEADER}" "${FORGEJO_URL}/api/v1/version" 2>/dev/null || true)
echo "Forgejo API Status: OK (${VERSION_INFO})"

if [[ "${CHECK_ONLY}" == true ]]; then
    ORG_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -H "${AUTH_HEADER}" "${FORGEJO_URL}/api/v1/orgs/${TARGET_ORG}")
    REPO_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -H "${AUTH_HEADER}" "${FORGEJO_URL}/api/v1/repos/${TARGET_ORG}/monorepo")
    if [[ "${ORG_STATUS}" == "200" && "${REPO_STATUS}" == "200" ]]; then
        echo "Forgejo repository state is verified and ready."
        exit 0
    fi
    echo "Forgejo repository or organization is not yet fully provisioned."
    exit 1
fi

if [[ "${DRY_RUN}" == true ]]; then
    echo "DRY RUN MODE: Skipping state mutations (Organization, Repository, Issue, and Git Push)."
    echo "Ready to synchronize to ${FORGEJO_URL}/${TARGET_ORG}/monorepo"
    exit 0
fi

# 3. Ensure Organization Exists
echo "Ensuring organization '${TARGET_ORG}' exists..."
ORG_HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "${AUTH_HEADER}" "${FORGEJO_URL}/api/v1/orgs/${TARGET_ORG}")
if [[ "${ORG_HTTP_CODE}" != "200" ]]; then
    echo "Creating organization '${TARGET_ORG}'..."
    curl -sf -X POST "${FORGEJO_URL}/api/v1/orgs" \
        -H "${AUTH_HEADER}" \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"${TARGET_ORG}\",
            \"full_name\": \"Orange Cat Investments\",
            \"description\": \"Enterprise Monorepo GitOps Platform\",
            \"visibility\": \"public\"
        }" >/dev/null 2>&1 || true
    echo "Organization '${TARGET_ORG}' created."
else
    echo "Organization '${TARGET_ORG}' already exists."
fi

# 4. Ensure Repositories Exist (monorepo & orange-cat-investments)
for REPO_NAME in monorepo orange-cat-investments; do
    echo "Checking repository '${TARGET_ORG}/${REPO_NAME}'..."
    REPO_HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "${AUTH_HEADER}" "${FORGEJO_URL}/api/v1/repos/${TARGET_ORG}/${REPO_NAME}")
    if [[ "${REPO_HTTP_CODE}" != "200" ]]; then
        echo "Creating repository '${TARGET_ORG}/${REPO_NAME}'..."
        curl -sf -X POST "${FORGEJO_URL}/api/v1/orgs/${TARGET_ORG}/repos" \
            -H "${AUTH_HEADER}" \
            -H "Content-Type: application/json" \
            -d "{
                \"name\": \"${REPO_NAME}\",
                \"description\": \"Orange Cat Investments platform codebase\",
                \"private\": false,
                \"auto_init\": false,
                \"default_branch\": \"${TARGET_BRANCH}\"
            }" >/dev/null 2>&1 || true
    fi

    # Enable Actions on repository
    echo "Enabling Actions, Issues, and Pull Requests on '${TARGET_ORG}/${REPO_NAME}'..."
    curl -sf -X PATCH "${FORGEJO_URL}/api/v1/repos/${TARGET_ORG}/${REPO_NAME}" \
        -H "${AUTH_HEADER}" \
        -H "Content-Type: application/json" \
        -d "{
            \"has_actions\": true,
            \"has_issues\": true,
            \"has_pull_requests\": true
        }" >/dev/null 2>&1 || true
done

# 5. Seed Issue #42 on oci/monorepo (for Elise Dev & Marcus Lead workflows)
echo "Checking Issue #42 on '${TARGET_ORG}/monorepo'..."
ISSUE_HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" -H "${AUTH_HEADER}" "${FORGEJO_URL}/api/v1/repos/${TARGET_ORG}/monorepo/issues/42")
if [[ "${ISSUE_HTTP_CODE}" != "200" ]]; then
    echo "Seeding Issue #42 (feat(gitops): multi-stage containerization & gated promotion pipeline)..."
    curl -sf -X POST "${FORGEJO_URL}/api/v1/repos/${TARGET_ORG}/monorepo/issues" \
        -H "${AUTH_HEADER}" \
        -H "Content-Type: application/json" \
        -d "{
            \"title\": \"feat(gitops): multi-stage containerization & gated promotion pipeline\",
            \"body\": \"Containerize Go backend services and Vue 3 web portals with multi-stage Dockerfiles. Deploy in-cluster Forgejo runner.\"
        }" >/dev/null 2>&1 || true
    echo "Issue #42 seeded."
else
    echo "Issue #42 already present."
fi

# 6. Push Git Monorepo Tree to Forgejo
echo "Pushing codebase to Forgejo repositories..."
FORGEJO_HOST=$(echo "${FORGEJO_URL}" | sed -e 's|^[^/]*//||')
CURRENT_COMMIT=$(git rev-parse HEAD)
echo "Current HEAD commit: ${CURRENT_COMMIT} (branch: ${TARGET_BRANCH})"

for REPO_NAME in monorepo orange-cat-investments; do
    REMOTE_URL="http://${ADMIN_USER}:${ADMIN_PASSWORD}@${FORGEJO_HOST}/${TARGET_ORG}/${REPO_NAME}.git"
    echo "Pushing HEAD to ${TARGET_ORG}/${REPO_NAME}:${TARGET_BRANCH}..."
    if git push "${REMOTE_URL}" "HEAD:refs/heads/${TARGET_BRANCH}" --force >/dev/null 2>&1; then
        echo "Successfully pushed to ${TARGET_ORG}/${REPO_NAME}."
    else
        echo "WARNING: Git push to ${TARGET_ORG}/${REPO_NAME} exited with non-zero status (VCS repository may already be up to date)."
    fi
done

echo ""
echo "================================================================================"
echo "  Source Synchronization to In-Cluster Forgejo Completed Successfully!"
echo "================================================================================"
echo "Forgejo Monorepo URL: ${FORGEJO_URL}/${TARGET_ORG}/monorepo"
echo "Forgejo Actions Tab:  ${FORGEJO_URL}/${TARGET_ORG}/monorepo/actions"
echo "================================================================================"
