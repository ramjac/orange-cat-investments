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

# 6. Verify Prometheus stateful store alert rules (7.14)
echo "Verifying Prometheus stateful store alert rules in k8s/monitoring/prometheus.yaml..."
if ! python3 - <<'PY'
import sys
import yaml

with open("k8s/monitoring/prometheus.yaml", "r", encoding="utf-8") as f:
    docs = list(yaml.safe_load_all(f))

cm = next((d for d in docs if isinstance(d, dict) and d.get("kind") == "ConfigMap"), None)
if not cm:
    print("ERROR: ConfigMap not found in k8s/monitoring/prometheus.yaml")
    sys.exit(1)

alert_rules_raw = cm.get("data", {}).get("alert_rules.yml")
if not alert_rules_raw:
    print("ERROR: data.alert_rules.yml not found in prometheus ConfigMap")
    sys.exit(1)

rules_doc = yaml.safe_load(alert_rules_raw)
groups = {g["name"]: g.get("rules", []) for g in rules_doc.get("groups", [])}

expected = {
    "valkey_alerts": {"ValkeyMemorySaturation", "ValkeyKeyEvictions"},
    "postgresql_alerts": {
        "PostgresConnectionSaturation",
        "PostgresDeadTupleBloat",
        "PostgresXIDWraparoundRisk",
        "PostgresLowCacheHitRatio",
    },
    "meilisearch_alerts": {"MeilisearchDown", "MeilisearchDiskUsageHigh"},
}

errors = []
for group_name, rules in groups.items():
    for rule in rules:
        labels = rule.get("labels", {})
        if not labels.get("severity") or not labels.get("tier"):
            errors.append(f"Rule {rule.get('alert')} in {group_name} missing severity or tier label")

for group_name, expected_alerts in expected.items():
    if group_name not in groups:
        errors.append(f"Missing alert group: {group_name}")
        continue
    found = {r.get("alert") for r in groups[group_name]}
    missing = expected_alerts - found
    if missing:
        errors.append(f"Group {group_name} missing alerts: {sorted(missing)}")

if errors:
    for err in errors:
        print(f"ERROR: {err}")
    sys.exit(1)

print("All stateful store alert rules verified.")
PY
then
  failed=1
fi

# 7. Enforce Grafana Beyla removal (7.15)
echo "Checking that Grafana Beyla is absent from k8s/, cmd/, internal/, pkg/, scripts/, docs/..."
beyla_hits="$(grep -rni "beyla" k8s/ cmd/ internal/ pkg/ scripts/ docs/ 2>/dev/null | grep -v "^scripts/verify-ci-contracts.sh:" || true)"
if [ -n "$beyla_hits" ]; then
  echo "ERROR: Found prohibited 'beyla' references:"
  echo "$beyla_hits"
  failed=1
fi

# 8. Enforce every telemetry Record* helper has at least one non-test caller (7.15)
echo "Checking that every Record* helper in pkg/telemetry/metrics.go has a non-test caller..."
for fn in $(grep -oE '^func (Record[A-Za-z0-9_]+)' pkg/telemetry/metrics.go | awk '{print $2}'); do
  caller_hits="$(grep -rn "telemetry\.$fn(" cmd/ internal/ pkg/ --exclude='*_test.go' || true)"
  if [ -z "$caller_hits" ]; then
    echo "ERROR: Telemetry helper $fn has no non-test caller"
    failed=1
  fi
done

if [ "$failed" -eq 1 ]; then
    echo "CI contracts verification failed."
    exit 1
fi

echo "CI contracts verification passed."
