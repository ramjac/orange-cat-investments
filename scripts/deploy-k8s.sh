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
    *)
        echo "ERROR: Unknown deployment profile '${PROFILE}'. Valid profiles are 'full' and 'small-business'." >&2
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
echo "3. Deploying PostgreSQL Database..."
kubectl apply -f "${REPO_ROOT}/k8s/postgres/"

if [[ "${PROFILE}" == "small-business" || "${PROFILE}" == "smb" ]]; then
    echo ""
    echo "4. Deploying Core SMB Applications (Valkey, RabbitMQ, Forgejo)..."
    kubectl apply -f "${REPO_ROOT}/k8s/apps/app-secrets.yaml"
    kubectl apply -f "${REPO_ROOT}/k8s/apps/valkey.yaml"
    kubectl apply -f "${REPO_ROOT}/k8s/apps/rabbitmq.yaml"
    kubectl apply -f "${REPO_ROOT}/k8s/apps/forgejo.yaml"
    echo "Skipping heavyweight COTS (ERPNext, Frappe HR, Nextcloud, Zitadel, Snipe-IT) and monitoring/logging stacks."
else
    echo ""
    echo "4. Deploying Core Applications (Valkey, RabbitMQ, Forgejo, Zitadel, Nextcloud, Snipe-IT, ERPNext, Frappe HR)..."
    kubectl apply -f "${REPO_ROOT}/k8s/apps/"

    echo ""
    echo "5. Deploying Monitoring & Logging Stacks (Prometheus, Grafana, Loki, Promtail)..."
    kubectl apply -f "${REPO_ROOT}/k8s/monitoring/"
    kubectl apply -f "${REPO_ROOT}/k8s/logging/"
fi

echo ""
echo "6. Waiting for core database and messaging workloads to become ready..."
echo "Waiting for PostgreSQL..."
kubectl rollout status statefulset/postgres -n postgres --timeout=120s || true

echo "Waiting for Valkey..."
kubectl rollout status deployment/valkey -n apps --timeout=120s || true

echo "Waiting for RabbitMQ..."
kubectl rollout status deployment/rabbitmq -n apps --timeout=120s || true

echo ""
echo "=============================================================="
echo "  OCI Platform Workloads Successfully Deployed!              "
echo "=============================================================="
echo ""
echo "Cluster Status:"
kubectl get pods -A -l 'app in (postgres, valkey, rabbitmq, forgejo, zitadel, snipe-it, meilisearch, nextcloud, loki, prometheus, grafana)' || kubectl get pods -A

echo ""
echo "Quickstart: Connect from local machine using port-forwarding:"
echo "  # Terminal 1: PostgreSQL"
echo "  kubectl port-forward -n postgres svc/postgres-service 5432:5432"
echo ""
echo "  # Terminal 2: Valkey & RabbitMQ"
echo "  kubectl port-forward -n apps svc/valkey-service 6379:6379 &"
echo "  kubectl port-forward -n apps svc/rabbitmq-service 5672:5672 15672:15672 &"
echo ""
echo "  # Terminal 3: Run OCI Unified API Server"
echo "  DATABASE_URL='postgres://oci_admin:oci_secure_pass@localhost:5432/oci?sslmode=disable' PORT=8080 go run ./cmd/server"
echo ""
echo "  # Terminal 4: Run Platform Simulation Driver"
echo "  go run ./simulation/cmd/simulator -mode live -server-url http://localhost:8080 -bff-url http://localhost:8081"
echo "=============================================================="
