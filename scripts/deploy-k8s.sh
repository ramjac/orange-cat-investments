#!/usr/bin/env bash
set -euo pipefail

# Direct Kubernetes Deployment Script for Orange Cat Investments (OCI)
# Targets any existing Kubernetes cluster (MicroK8s, K3s, Minikube, kind, or cloud K8s)
# using standard kubectl without requiring host SSH or Ansible.

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

echo "=============================================================="
echo "  Orange Cat Investments (OCI) - Kubernetes Platform Deploy  "
echo "=============================================================="
echo "Working directory: ${REPO_ROOT}"

PROFILE="full"
if [[ "${1:-}" == "--profile" ]]; then
    PROFILE="${2:-full}"
fi

case "${PROFILE}" in
    full)
        echo "Deployment profile: ${PROFILE} (Complete enterprise suite)"
        ;;
    small-business|smb)
        echo "Deployment profile: ${PROFILE} (Lightweight core SMB suite)"
        ;;
    dev|staging)
        echo "Deployment profile: ${PROFILE} (Development & simulation staging suite - oci-dev)"
        ;;
    all)
        echo "Deployment profile: ${PROFILE} (Full production suite + oci-dev staging)"
        ;;
    *)
        echo "ERROR: Unknown deployment profile '${PROFILE}'. Valid profiles are 'full', 'small-business', 'dev', and 'all'." >&2
        exit 1
        ;;
esac

# 1. Verify kubectl availability
if ! command -v kubectl &>/dev/null; then
    echo "ERROR: 'kubectl' command not found. Please ensure kubectl is installed and in your PATH."
    exit 1
fi

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
    echo "Staging environment deployed. Clean production (oci-core) left untouched."
elif [[ "${PROFILE}" == "small-business" || "${PROFILE}" == "smb" ]]; then
    echo ""
    echo "3. Deploying Core Infrastructure & Custom Apps (oci-core) and Internal Auth (Forgejo)..."
    kubectl apply -f "${REPO_ROOT}/k8s/oci-core/"
    kubectl apply -f "${REPO_ROOT}/k8s/oci-internal/internal-secrets.yaml"
    kubectl apply -f "${REPO_ROOT}/k8s/oci-internal/postgres-internal.yaml"
    kubectl apply -f "${REPO_ROOT}/k8s/oci-internal/forgejo.yaml"
    kubectl apply -f "${REPO_ROOT}/k8s/external-sim/"
    echo "Skipping heavyweight COTS (ERPNext, Frappe HR, Nextcloud, Snipe-IT) and monitoring/logging stacks."
else
    echo ""
    echo "3. Deploying Clean Core Applications & Infrastructure (oci-core)..."
    kubectl apply -f "${REPO_ROOT}/k8s/oci-core/"

    echo ""
    echo "4. Deploying Internal Business Applications (oci-internal)..."
    kubectl apply -f "${REPO_ROOT}/k8s/oci-internal/"

    echo ""
    echo "5. Deploying External Simulation Mocks (external-sim)..."
    kubectl apply -f "${REPO_ROOT}/k8s/external-sim/"

    echo ""
    echo "6. Deploying Monitoring & Logging Stacks (Prometheus, Grafana, Loki, Promtail)..."
    kubectl apply -f "${REPO_ROOT}/k8s/monitoring/"
    kubectl apply -f "${REPO_ROOT}/k8s/logging/"

    if [[ "${PROFILE}" == "all" ]]; then
        echo ""
        echo "7. Deploying Development & Simulation Staging (oci-dev)..."
        kubectl apply -f "${REPO_ROOT}/k8s/oci-dev/"
    fi
fi

echo ""
echo "7. Waiting for core database and messaging workloads to become ready..."
echo "Waiting for OCI Core PostgreSQL..."
kubectl rollout status statefulset/postgres-core -n oci-core --timeout=120s || true

echo "Waiting for Valkey..."
kubectl rollout status deployment/valkey -n oci-core --timeout=120s || true

echo "Waiting for RabbitMQ..."
kubectl rollout status deployment/rabbitmq -n oci-core --timeout=120s || true

# Provision starter platform administrator in Forgejo IdP if deployed
if kubectl get namespace oci-internal &>/dev/null && kubectl get secret starter-admin-credentials -n oci-internal &>/dev/null; then
    echo ""
    echo "8. Provisioning starter platform administrator account in Forgejo IdP..."
    if kubectl rollout status deployment/forgejo -n oci-internal --timeout=90s 2>/dev/null; then
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
            fi
        fi
    fi
fi

echo ""
echo "=============================================================="
echo "  OCI Platform Workloads Successfully Deployed!              "
echo "=============================================================="
echo ""
echo "Cluster Status:"
kubectl get pods -A -l 'app in (postgres-core, postgres-internal, valkey, rabbitmq, forgejo, zitadel, snipe-it, meilisearch, nextcloud, loki, prometheus, grafana, oci-server, oci-worker)' || kubectl get pods -A

echo ""
echo "Quickstart: Connect from local machine using port-forwarding:"
echo "  # Terminal 1: PostgreSQL Core"
echo "  kubectl port-forward -n oci-core svc/postgres-core-service 5432:5432"
echo ""
echo "  # Terminal 2: Valkey & RabbitMQ"
echo "  kubectl port-forward -n oci-core svc/valkey-service 6379:6379 &"
echo "  kubectl port-forward -n oci-core svc/rabbitmq-service 5672:5672 15672:15672 &"
echo ""
echo "  # Terminal 3: Run OCI Unified API Server"
echo "  DATABASE_URL='postgres://oci_admin:oci_secure_pass@localhost:5432/oci?sslmode=disable' PORT=8080 go run ./cmd/server"
echo ""
echo "  # Terminal 4: Run Platform Simulation Driver"
echo "  go run ./simulation/cmd/simulator -mode live -server-url http://localhost:8080 -bff-url http://localhost:8081"
echo "=============================================================="
