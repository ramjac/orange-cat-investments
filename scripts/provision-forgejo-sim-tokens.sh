#!/usr/bin/env bash
set -euo pipefail

# Provision Forgejo identity accounts and simulation access tokens for the
# developer personas that exercise direct Forgejo Git workflows in the
# platform simulator (feature branches, commits, pull requests, reviews,
# merges). The simulator consumes the tokens through -tokens-file, so the
# generated JSON is a local-only artifact (gitignored).
#
# Usage:
#   ./scripts/provision-forgejo-sim-tokens.sh [output-json-path]
#
# Requires: kubectl with a KUBECONFIG pointing at a cluster where Forgejo is
# deployed in the oci-internal namespace.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

if [[ -z "${KUBECONFIG:-}" && -f "${REPO_ROOT}/thinker-kube.config" ]]; then
    export KUBECONFIG="${REPO_ROOT}/thinker-kube.config"
fi
if ! command -v kubectl >/dev/null 2>&1 && [[ -x "/home/ramjac/.local/bin/kubectl" ]]; then
    export PATH="/home/ramjac/.local/bin:${PATH}"
fi

OUT_FILE="${1:-${REPO_ROOT}/forgejo-sim-tokens.json}"
NS="oci-internal"
DEPLOY="deployment/forgejo"
TOKEN_PREFIX="oci-sim"
TOKEN_NAME="${TOKEN_PREFIX}-$(date +%Y%m%d%H%M%S)"
PERSONA_PASSWORD="password123"
PERSONAS=("emp-human-elise" "emp-human-marcus" "emp-human-bob")

command -v kubectl >/dev/null 2>&1 || {
    echo "ERROR: kubectl is required to provision Forgejo simulation tokens." >&2
    exit 1
}
kubectl get namespace "${NS}" >/dev/null 2>&1 || {
    echo "ERROR: namespace '${NS}' not found. Is KUBECONFIG pointing at the test cluster?" >&2
    exit 1
}

echo "=============================================================="
echo "  Forgejo Simulation Token Provisioning"
echo "=============================================================="

declare -A GENERATED=()
declare -A SHA1S=()

for persona in "${PERSONAS[@]}"; do
    if ! kubectl exec -n "${NS}" "${DEPLOY}" -- forgejo admin user list 2>/dev/null | awk '{print $2}' | grep -qx "${persona}"; then
        email_part="${persona#emp-human-}"
        email="${email_part}@oci.local"
        echo "Creating Forgejo user '${persona}' (${email})..."
        kubectl exec -n "${NS}" "${DEPLOY}" -- forgejo admin user create \
            --username "${persona}" \
            --password "${PERSONA_PASSWORD}" \
            --email "${email}" \
            --must-change-password=false >/dev/null
    else
        echo "Forgejo user '${persona}' exists; syncing password..."
    fi
    kubectl exec -n "${NS}" "${DEPLOY}" -- forgejo admin user change-password \
        -u "${persona}" -p "${PERSONA_PASSWORD}" --must-change-password=false >/dev/null

    raw_token="$(kubectl exec -n "${NS}" "${DEPLOY}" -- forgejo admin user generate-access-token \
        -u "${persona}" -t "${TOKEN_NAME}" --scopes all --raw | tr -d '\r\n')"
    if [[ -z "${raw_token}" ]]; then
        echo "ERROR: failed to generate access token for '${persona}'." >&2
        exit 1
    fi
    GENERATED["${persona}"]="${raw_token}"
    SHA1S["${persona}"]="$(printf '%s' "${raw_token}" | sha1sum | awk '{print $1}')"
    echo "Generated access token for '${persona}'."
done

# Remove superseded 'oci-sim' tokens so repeated provisioning does not
# accumulate stale credentials. Done over the API through a short-lived
# port-forward, authenticated with the freshly generated token; the fresh
# token is identified by its SHA1.
PF_PORT=$(( 30000 + RANDOM % 15000 ))
kubectl port-forward -n "${NS}" svc/forgejo-service "${PF_PORT}:3000" >/dev/null 2>&1 &
PF_PID=$!
trap 'kill "${PF_PID}" 2>/dev/null || true' EXIT
for _ in $(seq 1 10); do
    if curl -sf --connect-timeout 2 "http://localhost:${PF_PORT}/api/v1/version" >/dev/null 2>&1; then
        break
    fi
    sleep 1
done

for persona in "${PERSONAS[@]}"; do
    listed="$(curl -sf -H "Authorization: token ${GENERATED[${persona}]}" \
        "http://localhost:${PF_PORT}/api/v1/users/${persona}/tokens" 2>/dev/null || true)"
    [[ -z "${listed}" ]] && continue
    while IFS=$'\t' read -r tid tname tsha; do
        [[ -z "${tid}" ]] && continue
        if [[ "${tname}" == ${TOKEN_PREFIX}* && "${tsha}" != "${SHA1S[${persona}]}" ]]; then
            curl -sf -X DELETE -H "Authorization: token ${GENERATED[${persona}]}" \
                "http://localhost:${PF_PORT}/api/v1/users/${persona}/tokens/${tid}" >/dev/null 2>&1 || true
            echo "Removed superseded '${TOKEN_PREFIX}*' token ${tid} for '${persona}'."
        fi
    done < <(echo "${listed}" | python3 -c "import json,sys; [print(str(t['id']) + '\t' + t['name'] + '\t' + t.get('sha1','')) for t in json.load(sys.stdin)]" 2>/dev/null || true)
done

# Ensure developer/facilities personas have collaborator access on the target repositories
# and package write access in the 'oci' organization.
ADMIN_PWD=$(kubectl get secret starter-admin-credentials -n "${NS}" -o jsonpath='{.data.ADMIN_PASSWORD}' 2>/dev/null | base64 -d 2>/dev/null || true)
if [[ -n "${ADMIN_PWD}" ]]; then
    OWNERS_TEAM_ID=$(curl -sf -u "oci-admin:${ADMIN_PWD}" "http://localhost:${PF_PORT}/api/v1/orgs/oci/teams" 2>/dev/null | \
        python3 -c "import json,sys; teams=json.load(sys.stdin); print(next((str(t['id']) for t in teams if t.get('name')=='Owners'), ''))" 2>/dev/null || true)
    for persona in "${PERSONAS[@]}"; do
        for repo in monorepo orange-cat-investments; do
            curl -sf -X PUT -u "oci-admin:${ADMIN_PWD}" "http://localhost:${PF_PORT}/api/v1/repos/oci/${repo}/collaborators/${persona}" \
                -H "Content-Type: application/json" \
                -d '{"permission": "admin"}' >/dev/null 2>&1 || true
        done
        if [[ -n "${OWNERS_TEAM_ID}" ]]; then
            curl -sf -X PUT -u "oci-admin:${ADMIN_PWD}" "http://localhost:${PF_PORT}/api/v1/teams/${OWNERS_TEAM_ID}/members/${persona}" >/dev/null 2>&1 || true
        fi
    done
fi

# Merge into the existing token file (preserving any manually saved state).
python3 - "${OUT_FILE}" <<PY
import json, os, sys

out_path = sys.argv[1]
state = {"forgejo_tokens": {}, "csrf_tokens": {}}
if os.path.exists(out_path):
    try:
        with open(out_path) as f:
            state.update(json.load(f))
    except Exception:
        pass
generated = {
$(for p in "${PERSONAS[@]}"; do printf '    "%s": "%s",\n' "${p}" "${GENERATED[${p}]}"; done)
}
if not generated:
    sys.exit("ERROR: no persona tokens generated; refusing to overwrite the token file.")
state.setdefault("forgejo_tokens", {}).update(generated)
state.setdefault("csrf_tokens", {})
with open(out_path, "w") as f:
    json.dump(state, f, indent=2)
    f.write("\n")
os.chmod(out_path, 0o600)
PY

echo ""
echo "Simulation tokens written to: ${OUT_FILE}"
echo "Run the simulator with: -tokens-file ${OUT_FILE}"
echo "=============================================================="
