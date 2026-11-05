#!/usr/bin/env bash
set -euo pipefail

WAIT=false
if [[ "${1:-}" == "--wait" ]]; then
    WAIT=true
fi

FORGEJO_URL=""
for c in "http://forgejo-service.oci-internal.svc:3000" "http://forgejo.oci.local:80" "http://localhost:3000"; do
    if curl -sf --connect-timeout 2 "${c}/api/v1/version" >/dev/null 2>&1; then
        FORGEJO_URL="${c}"
        break
    fi
done

if [[ -z "${FORGEJO_URL}" ]]; then
    echo "ERROR: Forgejo instance is offline or unreachable."

    exit 1
fi

TARGET_ORG="oci"
REPO_NAME="monorepo"

RUNS_JSON=$(curl -s "${FORGEJO_URL}/api/v1/repos/${TARGET_ORG}/${REPO_NAME}/actions/runs?page=1&limit=1")

if [[ -z "$RUNS_JSON" || "$RUNS_JSON" == "[]" ]]; then
    echo "No actions runs found for ${TARGET_ORG}/${REPO_NAME}."
    exit 0
fi

read -r RUN_ID RUN_STATUS < <(python3 -c "
import sys, json
try:
    data = json.loads(sys.argv[1])
    if isinstance(data, list) and len(data) > 0:
        print(f\"{data[0].get('id')} {data[0].get('status')}\")
    elif isinstance(data, dict) and data.get('data'):
        print(f\"{data['data'][0].get('id')} {data['data'][0].get('status')}\")
    elif isinstance(data, dict):
        print(f\"{data.get('id')} {data.get('status')}\")
    else:
        print('unknown unknown')
except:
    print('unknown unknown')
" "$RUNS_JSON")

if [[ "$RUN_ID" == "unknown" ]]; then
    echo "Could not determine run ID from Forgejo API."
    exit 1
fi

echo "Latest Run ID: $RUN_ID"
echo "Status: $RUN_STATUS"

if [[ "$WAIT" == true ]]; then
    while [[ "$RUN_STATUS" != "success" && "$RUN_STATUS" != "failure" && "$RUN_STATUS" != "cancelled" && "$RUN_STATUS" != "skipped" ]]; do
        echo "Waiting for run $RUN_ID to complete... (current status: $RUN_STATUS)"
        sleep 5
        RUN_STATUS=$(curl -s "${FORGEJO_URL}/api/v1/repos/${TARGET_ORG}/${REPO_NAME}/actions/runs?page=1&limit=1" | python3 -c "
import sys, json
try:
    data = json.loads(sys.stdin.read())
    if isinstance(data, list) and len(data) > 0:
        print(data[0].get('status', 'unknown'))
    elif isinstance(data, dict) and data.get('data'):
        print(data['data'][0].get('status', 'unknown'))
    elif isinstance(data, dict):
        print(data.get('status', 'unknown'))
    else:
        print('unknown')
except:
    print('unknown')
")
    done
    echo "Final Status: $RUN_STATUS"
fi

if [[ "$RUN_STATUS" == "failure" ]]; then
    echo "Run failed. Fetching logs..."
    JOBS_JSON=$(curl -s "${FORGEJO_URL}/api/v1/repos/${TARGET_ORG}/${REPO_NAME}/actions/runs/${RUN_ID}/jobs")
    FAILED_JOB_ID=$(python3 -c "
import sys, json
try:
    data = json.loads(sys.argv[1])
    jobs = data.get('jobs', []) if isinstance(data, dict) else data
    for j in jobs:
        if j.get('status') == 'failure':
            print(j.get('id', ''))
            break
except:
    pass
" "$JOBS_JSON")

    if [[ -n "$FAILED_JOB_ID" ]]; then
        echo "Logs for failed job $FAILED_JOB_ID:"
        curl -s "${FORGEJO_URL}/api/v1/repos/${TARGET_ORG}/${REPO_NAME}/actions/jobs/${FAILED_JOB_ID}/logs" || true
    else
        echo "Could not determine which job failed or no logs available."
    fi

    exit 1
fi
