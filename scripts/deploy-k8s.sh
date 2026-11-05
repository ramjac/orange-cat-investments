#!/usr/bin/env bash
set -euo pipefail

FAILED_ROLLOUTS=()
wait_rollout() {
    local cmd=("$@")
    echo "Running: ${cmd[*]}"
    if ! "${cmd[@]}"; then
        echo "WARNING: Rollout failed for: ${cmd[*]}"
        FAILED_ROLLOUTS+=("${cmd[*]}")
    fi
}


# Direct Kubernetes Deployment Script for Orange Cat Investments (OCI)
# Targets any existing Kubernetes cluster (MicroK8s, Minikube, kind, or cloud K8s)
# using standard kubectl without requiring host SSH or Ansible.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "=============================================================="
echo "  Orange Cat Investments (OCI) - Kubernetes Platform Deploy  "
echo "=============================================================="
echo "Working directory: ${REPO_ROOT}"

PROFILE="all"
if [[ "${1:-}" == "--profile" ]]; then
    PROFILE="${2:-all}"
fi

case "${PROFILE}" in
    all)
        echo "Deployment profile: ${PROFILE} (Default: Full platform suite, internal COTS, monitoring/logging, and staging)"
        ;;
    business-only)
        echo "Deployment profile: ${PROFILE} (Internal business COTS applications only - oci-internal without demo data or OCI custom apps)"
        ;;
    dev|staging)
        echo "Deployment profile: ${PROFILE} (Development & simulation staging suite - oci-dev & external-sim)"
        ;;
    *)
        echo "ERROR: Unknown deployment profile '${PROFILE}'. Valid profiles are 'all', 'business-only', and 'dev'." >&2
        exit 1
        ;;
esac

# 1. Verify kubectl availability
if ! command -v kubectl &>/dev/null; then
    echo "ERROR: 'kubectl' command not found. Please ensure kubectl is installed and in your PATH."
    exit 1
fi

echo ""
echo "Checking container registry pre-flight..."
REGISTRY_HOST="${REGISTRY_HOST:-localhost}"
MANIFEST_DIRS=()
if [[ "$PROFILE" == "dev" || "$PROFILE" == "staging" ]]; then
    MANIFEST_DIRS=("k8s/oci-dev" "k8s/external-sim")
elif [[ "$PROFILE" == "business-only" ]]; then
    MANIFEST_DIRS=("k8s/oci-internal")
else
    MANIFEST_DIRS=("k8s/oci-core" "k8s/oci-internal" "k8s/external-sim" "k8s/monitoring" "k8s/logging" "k8s/oci-dev")
fi

MISSING_IMAGES=()
for dir in "${MANIFEST_DIRS[@]}"; do
    if [ ! -d "${REPO_ROOT}/${dir}" ]; then continue; fi
    IMAGES=$(grep -rhoE "localhost:32000/[^\"[:space:]]+" "${REPO_ROOT}/${dir}" | sort -u || true)
    for image in $IMAGES; do
        REPO=$(echo "$image" | sed "s/localhost:32000\/\(.*\):.*/\1/")
        TAG=$(echo "$image" | sed "s/.*:\(.*\)/\1/")
        HTTP_CODE=$(curl -s -o /tmp/registry_resp -w "%{http_code}" "http://${REGISTRY_HOST}:32000/v2/${REPO}/tags/list")
        if [ "$HTTP_CODE" -ne 200 ]; then
            MISSING_IMAGES+=("$image")
            continue
        fi
        if ! grep -q "\"${TAG}\"" /tmp/registry_resp; then
            MISSING_IMAGES+=("$image")
        fi
    done
done

if [ ${#MISSING_IMAGES[@]} -gt 0 ]; then
    echo "ERROR: The following images are missing from the local registry (${REGISTRY_HOST}:32000):"
    for img in "${MISSING_IMAGES[@]}"; do
        echo "  - $img"
    done
    echo ""
    echo "Hint: Run scripts/sync-repo-to-forgejo.sh to build and push these images via CI, or build them manually."
    exit 1
fi
echo "Pre-flight checks passed."
echo ""
echo "1. Checking Kubernetes cluster connectivity..."
kubectl cluster-info
echo "Connected context: $(kubectl config current-context)"

echo ""
echo "2. Applying Namespaces..."
kubectl apply -f "${REPO_ROOT}/k8s/namespaces/"

echo ""
echo "Checking starter platform administrator bootstrap credentials..."
"${REPO_ROOT}/scripts/generate-starter-admin.sh"

echo ""
echo "Checking cert-manager & PKI prerequisites..."
if kubectl get crd certificates.cert-manager.io &>/dev/null; then
    if [[ -f "${REPO_ROOT}/k8s/ha/lan/tls/oci-lan-root-ca.crt" && -f "${REPO_ROOT}/k8s/ha/lan/tls/oci-lan-root-ca.key" ]]; then
        kubectl create namespace cert-manager --dry-run=client -o yaml | kubectl apply -f -
        kubectl create secret tls oci-lan-root-ca-key-pair \
            --namespace cert-manager \
            --cert="${REPO_ROOT}/k8s/ha/lan/tls/oci-lan-root-ca.crt" \
            --key="${REPO_ROOT}/k8s/ha/lan/tls/oci-lan-root-ca.key" \
            --dry-run=client -o yaml | kubectl apply -f -
        kubectl apply -f "${REPO_ROOT}/k8s/ha/lan/tls/intranet-cluster-issuer.yaml"
        echo "PKI ClusterIssuer 'oci-lan-issuer' verified."

        # Ensure fallback oci-ca-bundle ConfigMap exists in core application namespaces
        for ns in oci-core oci-internal oci-dev default; do
            kubectl create configmap oci-ca-bundle \
                --namespace "${ns}" \
                --from-file=ca.crt="${REPO_ROOT}/k8s/ha/lan/tls/oci-lan-root-ca.crt" \
                --dry-run=client -o yaml | kubectl apply -f - &>/dev/null || true
        done
    fi
else
    echo "WARNING: cert-manager CRDs not detected. Please install cert-manager (e.g., 'kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.17.1/cert-manager.yaml' or 'microk8s enable cert-manager')."
fi

echo ""
echo "Checking local container registry (:32000)..."
if ! kubectl get svc -n kube-system container-registry &>/dev/null && ! kubectl get svc -n container-registry registry &>/dev/null; then
    echo "Deploying in-cluster container registry on NodePort 32000..."
    kubectl apply -f "${REPO_ROOT}/k8s/registry/registry.yaml"
fi

echo ""
if [[ "${PROFILE}" == "dev" || "${PROFILE}" == "staging" ]]; then
    echo ""
    echo "3. Deploying Development & Simulation Staging Suite (oci-dev & external-sim)..."
    kubectl apply -f "${REPO_ROOT}/k8s/oci-dev/"
    kubectl apply -f "${REPO_ROOT}/k8s/external-sim/"
    wait_rollout kubectl rollout status deployment/mock-hardware -n external-sim --timeout=120s
    echo "Staging environment deployed. Production (oci-core) and internal COTS left untouched."
elif [[ "${PROFILE}" == "business-only" ]]; then
    echo ""
    echo "3. Deploying Internal Business Applications & COTS (oci-internal)..."
    kubectl apply -f "${REPO_ROOT}/k8s/oci-internal/"
    echo "Deployed internal COTS suite (Nextcloud, Snipe-IT, ERPNext, Frappe HR, Forgejo). OCI demo apps and simulation seeds skipped."
else
    echo ""
    echo "3. Deploying Core Applications & Infrastructure (oci-core)..."
    kubectl apply -f "${REPO_ROOT}/k8s/oci-core/"

    echo ""
    echo "4. Deploying Internal Business Applications (oci-internal)..."
    kubectl apply -f "${REPO_ROOT}/k8s/oci-internal/"

    echo ""
    echo "5. Deploying External Simulation Mocks (external-sim)..."
    kubectl apply -f "${REPO_ROOT}/k8s/external-sim/"
    wait_rollout kubectl rollout status deployment/mock-hardware -n external-sim --timeout=120s

    echo ""
    echo "6. Deploying Monitoring & Logging Stacks (Prometheus, Grafana, Loki, Promtail)..."
    kubectl apply -f "${REPO_ROOT}/k8s/monitoring/"
    kubectl apply -f "${REPO_ROOT}/k8s/logging/"

    echo ""
    echo "7. Deploying Development & Simulation Staging (oci-dev)..."
    kubectl apply -f "${REPO_ROOT}/k8s/oci-dev/"
fi

echo ""
echo "8. Waiting for database and messaging workloads to become ready..."
if [[ "${PROFILE}" == "business-only" ]]; then
    echo "Waiting for Internal PostgreSQL (postgres-internal)..."
    wait_rollout kubectl rollout status statefulset/postgres-internal -n oci-internal --timeout=120s
elif [[ "${PROFILE}" == "dev" || "${PROFILE}" == "staging" ]]; then
    echo "Waiting for Dev PostgreSQL..."
    wait_rollout kubectl rollout status statefulset/postgres-dev -n oci-dev --timeout=120s
    echo "Waiting for Dev Valkey..."
    wait_rollout kubectl rollout status deployment/valkey -n oci-dev --timeout=120s
    echo "Waiting for Dev RabbitMQ..."
    wait_rollout kubectl rollout status deployment/rabbitmq -n oci-dev --timeout=120s
else
    echo "Waiting for OCI Core PostgreSQL..."
    wait_rollout kubectl rollout status statefulset/postgres-core -n oci-core --timeout=120s
    echo "Waiting for Internal PostgreSQL..."
    wait_rollout kubectl rollout status statefulset/postgres-internal -n oci-internal --timeout=120s
    echo "Waiting for Valkey..."
    wait_rollout kubectl rollout status deployment/valkey -n oci-core --timeout=120s
    echo "Waiting for RabbitMQ..."
    wait_rollout kubectl rollout status deployment/rabbitmq -n oci-core --timeout=120s
fi

# Provision starter platform administrator in Forgejo IdP if deployed
if kubectl get namespace oci-internal &>/dev/null && kubectl get secret starter-admin-credentials -n oci-internal &>/dev/null; then
    echo ""
    echo "9. Provisioning starter platform administrator account in Forgejo IdP..."
    if wait_rollout kubectl rollout status deployment/forgejo -n oci-internal --timeout=90s 2>/dev/null; then
        ADMIN_PWD=$(kubectl get secret starter-admin-credentials -n oci-internal -o jsonpath='{.data.ADMIN_PASSWORD}' | base64 -d 2>/dev/null || true)
        if [[ -n "${ADMIN_PWD}" ]]; then
            kubectl exec -n oci-internal deployment/forgejo -- \
                forgejo admin user create --admin --username "admin" --password "${ADMIN_PWD}" --email "admin@oci.local" 2>/dev/null || true
            echo "Starter admin account 'admin' provisioned in Forgejo."
        fi

        # Auto-provision Forgejo Actions runner registration token if needed
        if ! kubectl get secret forgejo-runner-secret -n oci-internal &>/dev/null; then
            RUNNER_TOKEN=$(kubectl exec -n oci-internal deployment/forgejo -- forgejo actions generate-runner-token 2>/dev/null | tr -d '\r\n' || true)
            if [[ -n "${RUNNER_TOKEN}" ]]; then
                kubectl create secret generic forgejo-runner-secret \
                    --namespace oci-internal \
                    --from-literal=RUNNER_TOKEN="${RUNNER_TOKEN}" \
                    --dry-run=client -o yaml | kubectl apply -f - >/dev/null 2>&1 || true
                echo "Forgejo runner registration token secret provisioned."
                kubectl rollout restart deployment/forgejo-runner -n oci-internal 2>/dev/null || true
            fi
        fi

        echo ""
        echo "10. Synchronizing repository and seeding CI/CD pipelines in Forgejo..."
        "${REPO_ROOT}/scripts/sync-repo-to-forgejo.sh" 2>/dev/null || true
    fi
fi

echo ""
echo "=============================================================="
echo "  Platform Workloads Successfully Deployed!                   "
echo "=============================================================="
echo ""
if [[ "${PROFILE}" == "business-only" ]]; then
    echo "Cluster Status (oci-internal):"
    kubectl get pods -n oci-internal
    echo ""
    echo "Quickstart: Access Internal Business Applications:"
    echo "  # Forgejo Git & Identity (port 3000):"
    echo "  kubectl port-forward -n oci-internal svc/forgejo-service 3000:3000 &"
    echo "  # Nextcloud Workspace (port 80):"
    echo "  kubectl port-forward -n oci-internal svc/nextcloud-service 8080:80 &"
    echo "  # Snipe-IT Asset Management (port 80):"
    echo "  kubectl port-forward -n oci-internal svc/snipe-it-service 8088:80 &"
    echo "  # ERPNext / Frappe (port 8000):"
    echo "  kubectl port-forward -n oci-internal svc/erpnext-service 8000:8000 &"
    echo "  # Initial bootstrap admin credentials: starter-admin-credentials.txt"
elif [[ "${PROFILE}" == "dev" || "${PROFILE}" == "staging" ]]; then
    echo "Cluster Status (oci-dev & external-sim):"
    kubectl get pods -n oci-dev
    kubectl get pods -n external-sim
    echo ""
    echo "Seed staging database fixtures:"
    echo "  ./scripts/seed-dev.sh"
else
    echo "Cluster Status:"
    kubectl get pods -A -l 'app in (postgres-core, postgres-internal, valkey, rabbitmq, forgejo, zitadel, snipe-it, meilisearch, nextcloud, loki, prometheus, grafana, workforce-api, facilities-api, core-invest-api, ops-api, customer-bll, internal-bll, oci-worker)' || kubectl get pods -A
    echo ""
    echo "Quickstart: Connect from local machine using port-forwarding:"
    echo "  # Employee BFF & Web Portal:"
    echo "  kubectl port-forward -n oci-core svc/oci-employee-bff-service 8081:8081 &"
    echo "  # Customer BFF & Web Portal:"
    echo "  kubectl port-forward -n oci-core svc/oci-customer-bff-service 8083:8083 &"
    echo "  # Forgejo Git & Identity:"
    echo "  kubectl port-forward -n oci-internal svc/forgejo-service 3000:3000 &"
fi
echo "=============================================================="
if [ ${#FAILED_ROLLOUTS[@]} -gt 0 ]; then
    echo ""
    echo "=============================================================="
    echo "  Deployment completed with ERRORS!                           "
    echo "=============================================================="
    echo "The following rollouts failed:"
    for f in "${FAILED_ROLLOUTS[@]}"; do
        echo "  - $f"
    done
    exit 1
fi
