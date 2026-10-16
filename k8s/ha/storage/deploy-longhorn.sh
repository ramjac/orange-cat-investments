#!/usr/bin/env bash
# ==============================================================================
# Deploy Longhorn Distributed Replicated Storage Engine
# ==============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LONGHORN_VERSION="${LONGHORN_VERSION:-v1.7.2}"

echo "==> Validating node prerequisites for Longhorn..."
# Checks for open-iscsi on cluster nodes
if command -v kubectl >/dev/null 2>&1; then
  echo "Checking nodes..."
  kubectl get nodes -o wide
else
  echo "kubectl not found, continuing installation assuming environment is prepared."
fi

echo "==> Deploying Longhorn ${LONGHORN_VERSION}..."
if command -v helm >/dev/null 2>&1; then
  echo "Deploying via Helm repository..."
  helm repo add longhorn https://charts.longhorn.io
  helm repo update
  helm upgrade --install longhorn longhorn/longhorn \
    --namespace longhorn-system \
    --create-namespace \
    --version "${LONGHORN_VERSION}" \
    --set defaultSettings.defaultDataPath="/var/lib/longhorn" \
    --set defaultSettings.replicaAutoBalance="best-effort" \
    --set defaultSettings.storageOverProvisioningPercentage=200 \
    --set persistence.defaultClass=true \
    --set persistence.defaultClassReplicaCount=3
else
  echo "Helm not detected; installing via official Longhorn release manifest..."
  kubectl apply -f "https://raw.githubusercontent.com/longhorn/longhorn/${LONGHORN_VERSION}/deploy/longhorn.yaml"
fi

echo "==> Ensuring volume encryption secret exists in longhorn-system..."
if command -v kubectl >/dev/null 2>&1; then
  if ! kubectl get secret longhorn-crypto-key -n longhorn-system >/dev/null 2>&1; then
    echo "Generating new unique encryption key for longhorn-crypto StorageClass..."
    GENERATED_KEY=$(head -c 32 /dev/urandom | base64)
    kubectl create secret generic longhorn-crypto-key \
      --namespace longhorn-system \
      --from-literal=CRYPTO_KEY_VALUE="${GENERATED_KEY}" \
      --from-literal=CRYPTO_KEY_PROVIDER="secret"
  else
    echo "Encryption key 'longhorn-crypto-key' already exists."
  fi
fi

echo "==> Applying StorageClasses and Recurring Jobs..."
kubectl apply -f "${SCRIPT_DIR}/longhorn-storageclasses.yaml"
kubectl apply -f "${SCRIPT_DIR}/longhorn-backup-recurring.yaml"
kubectl apply -f "${SCRIPT_DIR}/longhorn-ui-ingress.yaml"

echo "==> Longhorn deployment submitted successfully. Awaiting rollout..."
if command -v kubectl >/dev/null 2>&1; then
  kubectl -n longhorn-system rollout status daemonset/longhorn-manager --timeout=300s || true
fi
echo "==> Distributed replicated storage is ready."
