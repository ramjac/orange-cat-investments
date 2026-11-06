#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -lt 2 ]; then
    echo "Usage: $0 <namespace> <sha-tag> [deployment...]"
    exit 1
fi

NAMESPACE="$1"
SHA_TAG="$2"
shift 2
DEPLOYMENTS=("$@")

if [ ${#DEPLOYMENTS[@]} -eq 0 ]; then
    mapfile -t DEPLOYMENTS < <(kubectl get deployment -n "$NAMESPACE" -o custom-columns=NAME:.metadata.name --no-headers)
fi

HAS_ERROR=0
for deploy in "${DEPLOYMENTS[@]}"; do
    IMAGES=$(kubectl get deployment "$deploy" -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[*].image}' || true)
    for image in $IMAGES; do
        if [[ "$image" != *":$SHA_TAG" ]]; then
            echo "ERROR: Deployment $deploy in $NAMESPACE uses image $image which does not end in :$SHA_TAG"
            HAS_ERROR=1
        fi
    done
done

if [ $HAS_ERROR -eq 1 ]; then
    exit 1
fi
echo "All checked deployments in $NAMESPACE are running images with tag :$SHA_TAG."
exit 0
