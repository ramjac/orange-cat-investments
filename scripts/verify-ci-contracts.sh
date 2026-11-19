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

# 9. Verify ENABLE_FAULT_INJECTION is restricted to -dev mock manifests (7.17)
echo "Verifying ENABLE_FAULT_INJECTION is restricted to -dev mock manifests..."
if grep -q "ENABLE_FAULT_INJECTION" k8s/external-sim/mock-bank.yaml k8s/external-sim/mock-broker.yaml k8s/external-sim/mock-hardware.yaml; then
  echo "ERROR: ENABLE_FAULT_INJECTION must never appear in unsuffixed mock manifests"
  failed=1
fi
if grep -rq "ENABLE_FAULT_INJECTION" k8s/oci-core/; then
  echo "ERROR: ENABLE_FAULT_INJECTION must never appear in k8s/oci-core/"
  failed=1
fi
if ! grep -q "ENABLE_FAULT_INJECTION" k8s/external-sim/mock-bank-dev.yaml || ! grep -q "ENABLE_FAULT_INJECTION" k8s/external-sim/mock-broker-dev.yaml; then
  echo "ERROR: mock-bank-dev.yaml and mock-broker-dev.yaml must configure ENABLE_FAULT_INJECTION"
  failed=1
fi

# 10. Verify Prometheus resource sizing and TSDB retention flags (7.21)
echo "Verifying Prometheus resource sizing and TSDB retention in k8s/monitoring/prometheus.yaml..."
if ! python3 - <<'PY'
import sys
import yaml

with open("k8s/monitoring/prometheus.yaml", "r", encoding="utf-8") as f:
    docs = list(yaml.safe_load_all(f))

dep = next(
    (d for d in docs if isinstance(d, dict) and d.get("kind") == "Deployment" and d.get("metadata", {}).get("name") == "prometheus"),
    None,
)
if not dep:
    print("ERROR: prometheus Deployment not found in k8s/monitoring/prometheus.yaml")
    sys.exit(1)

containers = dep.get("spec", {}).get("template", {}).get("spec", {}).get("containers", [])
prom = next((c for c in containers if c.get("name") == "prometheus"), None)
if not prom:
    print("ERROR: prometheus container not found")
    sys.exit(1)

args = prom.get("args", [])
if not any(a.startswith("--storage.tsdb.retention.time=") for a in args):
    print("ERROR: missing --storage.tsdb.retention.time flag on prometheus")
    sys.exit(1)
if not any(a.startswith("--storage.tsdb.retention.size=") for a in args):
    print("ERROR: missing --storage.tsdb.retention.size flag on prometheus")
    sys.exit(1)

res = prom.get("resources", {})
req_mem = res.get("requests", {}).get("memory")
lim_mem = res.get("limits", {}).get("memory")
if req_mem != "512Mi" or lim_mem != "2Gi":
    print(f"ERROR: expected prometheus memory requests=512Mi limits=2Gi, got requests={req_mem} limits={lim_mem}")
    sys.exit(1)

print("Prometheus resource sizing and TSDB retention verified.")
PY
then
  failed=1
fi

# 11. Verify BFF rate-limiting policy isolation (7.22)
echo "Verifying BFF rate-limit configuration contracts..."
if grep -rqE "RATE_LIMIT_ENABLED|RATE_LIMIT_MULTIPLIER" k8s/oci-core/; then
  echo "ERROR: k8s/oci-core/ must never override or disable strict BFF rate limiting"
  failed=1
fi
if ! grep -qE "RATE_LIMIT_ENABLED|RATE_LIMIT_MULTIPLIER" k8s/oci-dev/oci-customer-bff.yaml; then
  echo "ERROR: k8s/oci-dev/oci-customer-bff.yaml must configure staging rate-limit policy"
  failed=1
fi
if grep -q "ratelimit:\*" scripts/run-load-test.sh; then
  echo "ERROR: scripts/run-load-test.sh must not contain kubectl exec ratelimit:* pruning workaround"
  failed=1
fi

# 12. Verify Nextcloud COTS secretKeyRef credentials on internal-bll (7.23)
echo "Verifying Nextcloud credentials wiring in internal-bll manifests..."
for bll_manifest in k8s/oci-core/bll-services.yaml k8s/oci-dev/bll-services.yaml; do
  if ! grep -q "NEXTCLOUD_ADMIN_USER" "$bll_manifest" || ! grep -q "NEXTCLOUD_ADMIN_PASS" "$bll_manifest"; then
    echo "ERROR: $bll_manifest missing NEXTCLOUD_ADMIN_USER or NEXTCLOUD_ADMIN_PASS on internal-bll"
    failed=1
  fi
done

# 13. Verify oci-dev database retention script & CronJob (7.24)
echo "Verifying oci-dev database retention script and CronJob..."
if ! ./scripts/prune-dev-db.sh --dry-run >/dev/null; then
  echo "ERROR: scripts/prune-dev-db.sh --dry-run failed"
  failed=1
fi
if ./scripts/prune-dev-db.sh --namespace oci-core --dry-run >/dev/null 2>&1; then
  echo "ERROR: scripts/prune-dev-db.sh failed to block --namespace oci-core"
  failed=1
fi
if [ ! -f k8s/oci-dev/dev-db-retention-cronjob.yaml ] || ! grep -q "namespace: oci-dev" k8s/oci-dev/dev-db-retention-cronjob.yaml; then
  echo "ERROR: k8s/oci-dev/dev-db-retention-cronjob.yaml missing or not targeting namespace oci-dev"
  failed=1
fi

if [ "$failed" -eq 1 ]; then
    echo "CI contracts verification failed."
    exit 1
fi

echo "CI contracts verification passed."
