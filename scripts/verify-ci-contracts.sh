#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

failed=0

# 1. Verify every localhost:32000 image referenced under k8s/ has a matching docker build target in ci.yaml
IMAGES=$(grep -r "image: localhost:32000/" k8s/ | grep -o "localhost:32000/[^:]*" | sort | uniq)

echo "Verifying docker build targets for k8s deployment images..."
for img in $IMAGES; do
    if ! grep -q "docker build -t ${img}:" .forgejo/workflows/ci.yaml; then
        echo "ERROR: Missing docker build target in .forgejo/workflows/ci.yaml for $img"
        failed=1
    fi
done

# 2. Check every Go Dockerfile to ensure GO_VERSION matches go.mod major.minor
GOMOD_VERSION=$(grep -m1 "^go " go.mod | awk '{print $2}' | cut -d. -f1,2)

echo "Expected GO_VERSION: ${GOMOD_VERSION}"
for dockerfile in cmd/*/Dockerfile web/marketing/Dockerfile; do
  if [ -f "$dockerfile" ]; then
    DOCKERFILE_VERSION=$(grep -m1 "^ARG GO_VERSION" "$dockerfile" | cut -d= -f2 | tr -d '\r\n')
    if [ "$DOCKERFILE_VERSION" != "$GOMOD_VERSION" ]; then
      echo "Error: $dockerfile has ARG GO_VERSION=$DOCKERFILE_VERSION, expected $GOMOD_VERSION"
      failed=1
    fi
  fi
done

# 3. Extract go run ./simulation/cmd/simulator commands from ci.yaml and run them with -validate-flags
echo "Testing simulation commands in ci.yaml with -validate-flags..."
if grep -q "go run ./simulation/cmd/simulator" .forgejo/workflows/ci.yaml; then
  grep -o "go run ./simulation/cmd/simulator[^\|]*" .forgejo/workflows/ci.yaml | while read -r cmd; do
    cmd="${cmd//\\/}"

    EVAL_CMD="${cmd} -validate-flags"
    echo "Running: STAGING_URL=http://test $EVAL_CMD"

    if ! STAGING_URL=http://test eval "$EVAL_CMD"; then
      echo "Error: Command failed to validate flags: $EVAL_CMD"
      exit 1
    fi
  done
fi


# 4. Verify mock environment wiring (7.6)
echo "Verifying mock environment wiring..."
if grep -qrE "mock-(bank|broker|hardware)-service" k8s/oci-dev/; then
  echo "ERROR: k8s/oci-dev/ references unsuffixed mock services"
  failed=1
fi

if grep -qrE "mock-(bank|broker|hardware)-dev-service" k8s/oci-core/; then
  echo "ERROR: k8s/oci-core/ references -dev mock services"
  failed=1
fi

if grep -q "\.oci-core\." k8s/external-sim/mock-*-dev.yaml; then
  echo "ERROR: mock-*-dev.yaml references .oci-core. hosts"
  failed=1
fi

if grep -q "\.oci-dev\." k8s/external-sim/mock-bank.yaml k8s/external-sim/mock-broker.yaml k8s/external-sim/mock-hardware.yaml; then
  echo "ERROR: unsuffixed mock manifests reference .oci-dev. hosts"
  failed=1
fi

# 5. Verify scripts/verify-deployed-images.sh tag matching with a stub kubectl
echo "Testing scripts/verify-deployed-images.sh with a stub kubectl..."
STUB_DIR=$(mktemp -d)
cat > "$STUB_DIR/kubectl" <<'STUB'
#!/usr/bin/env bash
case "$*" in
  *"-o jsonpath={..image}"*) echo "${STUB_IMAGES:-}" ;;
  *) echo "" ;;
esac
STUB
chmod +x "$STUB_DIR/kubectl"

# Matching tag (plus an external image that must be ignored) must pass.
if ! STUB_IMAGES="registry.k8s.io/pause:3.9 localhost:32000/app-one:sha-abc123" PATH="$STUB_DIR:$PATH" ./scripts/verify-deployed-images.sh test-ns sha-abc123 app-one >/dev/null 2>&1; then
  echo "ERROR: verify-deployed-images.sh rejected a matching :sha-abc123 tag"
  failed=1
fi

# A :latest image must fail.
if STUB_IMAGES="localhost:32000/app-one:latest" PATH="$STUB_DIR:$PATH" ./scripts/verify-deployed-images.sh test-ns sha-abc123 app-one >/dev/null 2>&1; then
  echo "ERROR: verify-deployed-images.sh accepted a :latest image"
  failed=1
fi

# A different SHA must fail.
if STUB_IMAGES="localhost:32000/app-one:sha-000000" PATH="$STUB_DIR:$PATH" ./scripts/verify-deployed-images.sh test-ns sha-abc123 app-one >/dev/null 2>&1; then
  echo "ERROR: verify-deployed-images.sh accepted a different SHA tag"
  failed=1
fi

rm -rf "$STUB_DIR"

if [ "$failed" -eq 1 ]; then
    echo "CI contracts verification failed."
    exit 1
fi

echo "CI contracts verification passed."
