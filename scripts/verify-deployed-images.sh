#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <namespace> <sha-tag> [deployment...]"
    exit 1
fi

NAMESPACE="$1"
SHA_TAG="$2"
shift 2
DEPS=("$@")

if [ ${#DEPS[@]} -eq 0 ]; then
  # If no deployments provided, fetch all deployments in the namespace
  DEPS_STR=$(kubectl get deployment -n "${NAMESPACE}" -o jsonpath='{.items[*].metadata.name}')
  if [ -n "$DEPS_STR" ]; then
    read -ra DEPS <<< "$DEPS_STR"
  fi
fi

FAILED=0

for dep in "${DEPS[@]}"; do
  # Get all images for the deployment
  IMAGES_STR=$(kubectl get deployment "${dep}" -n "${NAMESPACE}" -o jsonpath='{..image}')
  if [ -n "$IMAGES_STR" ]; then
    read -ra IMAGES <<< "$IMAGES_STR"
    for img in "${IMAGES[@]}"; do
      if [[ "$img" != *":${SHA_TAG}" ]]; then
        echo "ERROR: Deployment ${dep} in namespace ${NAMESPACE} is using image ${img} (expected tag :${SHA_TAG})"
        FAILED=1
      fi
    done
  fi
done

if [ "$FAILED" -ne 0 ]; then
  exit 1
fi

echo "All specified deployments in namespace ${NAMESPACE} are correctly pinned to :${SHA_TAG}"
exit 0
