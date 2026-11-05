#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

failed=0

# Extract all unique localhost:32000 images from k8s/**/*.yaml
IMAGES=$(grep -r "image: localhost:32000/" k8s/ | grep -o "localhost:32000/[^:]*" | sort | uniq)

echo "Verifying docker build targets for k8s deployment images..."
for img in $IMAGES; do
    if ! grep -q "docker build -t ${img}:" .forgejo/workflows/ci.yaml; then
        echo "ERROR: Missing docker build target in .forgejo/workflows/ci.yaml for $img"
        failed=1
    fi
done

if [ "$failed" -eq 1 ]; then
    echo "CI contracts verification failed."
    exit 1
fi

echo "CI contracts verification passed."
