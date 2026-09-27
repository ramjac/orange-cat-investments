#!/usr/bin/env bash
set -euo pipefail

# Kubernetes Cluster Cleanup Script for Orange Cat Investments (OCI)
# Removes all OCI domain applications, databases, logging, monitoring, and namespaces
# while safely preserving cluster system infrastructure, ingress controllers, and access.

echo "=============================================================="
echo "  Orange Cat Investments (OCI) - Kubernetes Cluster Cleanup  "
echo "=============================================================="

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
echo "2. Identifying OCI-specific namespaces to remove..."
OCI_NAMESPACES=("apps" "logging" "monitoring" "oci-core" "postgres")

for ns in "${OCI_NAMESPACES[@]}"; do
    if kubectl get namespace "${ns}" &>/dev/null; then
        echo "Found OCI namespace: ${ns}. Deleting..."
        kubectl delete namespace "${ns}" --wait=true --timeout=120s || {
            echo "Warning: Timeout waiting for namespace ${ns} deletion. Checking resources..."
        }
    else
        echo "Namespace ${ns} does not exist (already clean)."
    fi
done

echo ""
echo "3. Verifying all OCI persistent volume claims have been released..."
if kubectl get pvc -A | grep -E "postgres|apps|logging|monitoring|oci-core" &>/dev/null; then
    echo "Warning: Lingering PVCs detected:"
    kubectl get pvc -A | grep -E "postgres|apps|logging|monitoring|oci-core" || true
else
    echo "All OCI PVCs and bound storage successfully deleted."
fi

echo ""
echo "4. Preserved Cluster Infrastructure Status:"
kubectl get namespaces
echo ""
echo "Preserved Storage / PVs:"
kubectl get pv

echo ""
echo "=============================================================="
echo "  OCI Cluster Cleanup Complete!                               "
echo "  Cluster is reset and ready for fresh deployment tomorrow via: "
echo "  ./scripts/deploy-k8s.sh                                     "
echo "=============================================================="
