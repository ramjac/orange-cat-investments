#!/bin/bash
set -e

# 1. Check every Go Dockerfile to ensure GO_VERSION matches go.mod major.minor
GOMOD_VERSION=$(grep -m1 "^go " go.mod | awk '{print $2}' | cut -d. -f1,2)

echo "Expected GO_VERSION: ${GOMOD_VERSION}"

FAIL=0
for dockerfile in cmd/*/Dockerfile web/marketing/Dockerfile; do
  if [ -f "$dockerfile" ]; then
    DOCKERFILE_VERSION=$(grep -m1 "^ARG GO_VERSION" "$dockerfile" | cut -d= -f2 | tr -d '\r\n')
    if [ "$DOCKERFILE_VERSION" != "$GOMOD_VERSION" ]; then
      echo "Error: $dockerfile has ARG GO_VERSION=$DOCKERFILE_VERSION, expected $GOMOD_VERSION"
      FAIL=1
    fi
  fi
done

if [ $FAIL -ne 0 ]; then
  exit 1
fi

echo "All Dockerfiles match go.mod GO_VERSION"

# 2. Extract go run ./simulation/cmd/simulator commands from ci.yaml and run them with -validate-flags
echo "Testing simulation commands in ci.yaml with -validate-flags..."
grep -o "go run ./simulation/cmd/simulator[^\|]*" .forgejo/workflows/ci.yaml | while read -r cmd; do
  # Replace environment variables with empty strings or default values so it parses
  # Note: in this specific file the command uses "${STAGING_URL}"
  # We just evaluate the command with -validate-flags
  # Using eval here because $cmd can contain quotes and env vars that we want bash to evaluate if we export them,
  # but here we'll just remove the env vars to avoid undefined behavior or we can just export a dummy var.
  # Let's clean up $cmd to remove `\` continuation if any, though grep -o avoids trailing \|
  cmd="${cmd//\\/}"

  EVAL_CMD="${cmd} -validate-flags"
  echo "Running: STAGING_URL=http://test $EVAL_CMD"

  if ! STAGING_URL=http://test eval "$EVAL_CMD"; then
    echo "Error: Command failed to validate flags: $EVAL_CMD"
    exit 1
  fi
done

echo "All simulation commands validated successfully."
