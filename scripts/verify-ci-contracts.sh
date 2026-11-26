#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

failed=0

# 1. Verify every localhost:32000 image referenced under k8s/ (excluding negative test fixtures) has a matching docker build target in ci.yaml
IMAGES=$(grep -r --exclude-dir=fixtures "image: localhost:32000/" k8s/ | grep -o "localhost:32000/[^:]*" | sort | uniq)

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
        "PostgresWALArchiveFailing",
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
while IFS= read -r fn; do
  caller_hits="$(grep -rn "telemetry\.$fn(" cmd/ internal/ pkg/ --exclude='*_test.go' || true)"
  if [ -z "$caller_hits" ]; then
    echo "ERROR: Telemetry helper $fn has no non-test caller"
    failed=1
  fi
done < <(grep -oE '^func (Record[A-Za-z0-9_]+)' pkg/telemetry/metrics.go | awk '{print $2}')

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

# 14. Verify Promtail Kubernetes pod log scraping and CRI parsing (7.25)
echo "Verifying Promtail Kubernetes pod log scraping in k8s/logging/promtail.yaml..."
for pat in "kind: ServiceAccount" "kind: ClusterRole" "/var/log/pods" "kubernetes_sd_configs" "cri:" "target_label: namespace" "target_label: pod" "target_label: container" "target_label: app"; do
  if ! grep -q "$pat" k8s/logging/promtail.yaml; then
    echo "ERROR: k8s/logging/promtail.yaml missing required pod-logging setting: $pat"
    failed=1
  fi
done

# 15. Verify Prometheus availability/error/upstream alerts and oci-dev RabbitMQ/ZITADEL scraping (7.26)
echo "Verifying Phase 7.26 Prometheus alert rules and oci-dev RabbitMQ metrics..."
if ! grep -q "15692" k8s/oci-dev/rabbitmq.yaml || ! grep -q "rabbitmq_prometheus" k8s/oci-dev/rabbitmq.yaml; then
  echo "ERROR: k8s/oci-dev/rabbitmq.yaml must enable rabbitmq_prometheus plugin and expose port 15692"
  failed=1
fi
for pat in "rabbitmq-service.oci-dev.svc.cluster.local:15692" "zitadel-service.oci-dev.svc.cluster.local:8080" "DeploymentZeroAvailableReplicas" "StatefulSetReplicasUnavailable" "ScrapeTargetDown" "HighHTTP5xxErrorRate" "UpstreamBankOrBrokerErrorSpike" "UpstreamBankOrBrokerTimeoutSpike"; do
  if ! grep -q "$pat" k8s/monitoring/prometheus.yaml; then
    echo "ERROR: k8s/monitoring/prometheus.yaml missing required Phase 7.26 config/alert: $pat"
    failed=1
  fi
done

# 16. Verify Gatus baseline probes use in-cluster URLs and TCP checks for mTLS gRPC & Frappe (7.27)
echo "Verifying Gatus probes in k8s/monitoring/gatus.yaml..."
if grep -qE "url: \"https://[a-z.]*oci\.local" k8s/monitoring/gatus.yaml; then
  echo "ERROR: k8s/monitoring/gatus.yaml must not probe external https://*.oci.local hostnames"
  failed=1
fi
for pat in "tcp://customer-bll-service.oci-core.svc.cluster.local:8085" "tcp://internal-bll-service.oci-core.svc.cluster.local:8086" "tcp://workforce-api-service.oci-core.svc.cluster.local:50051" "tcp://facilities-api-service.oci-core.svc.cluster.local:50052" "tcp://core-invest-api-service.oci-core.svc.cluster.local:50053" "tcp://ops-api-service.oci-core.svc.cluster.local:50054" "tcp://erpnext-service.oci-internal.svc.cluster.local:8000" "tcp://frappe-hr-service.oci-internal.svc.cluster.local:8000"; do
  if ! grep -q "$pat" k8s/monitoring/gatus.yaml; then
    echo "ERROR: k8s/monitoring/gatus.yaml missing required TCP probe: $pat"
    failed=1
  fi
done

# 17. Enforce ephemeral SPIFFE CSI SVIDs across custom mTLS microservices (7.20b)
echo "Verifying ephemeral SPIFFE CSI SVID mounts across custom workloads..."
spiffe_tls_manifests=(
  k8s/oci-core/domain-apis.yaml
  k8s/oci-core/bll-services.yaml
  k8s/oci-core/oci-customer-bff.yaml
  k8s/oci-core/oci-employee-bff.yaml
  k8s/oci-core/oci-pebble-proxy.yaml
  k8s/oci-core/oci-worker.yaml
  k8s/oci-core/batch-jobs.yaml
  k8s/oci-dev/domain-apis.yaml
  k8s/oci-dev/bll-services.yaml
  k8s/oci-dev/oci-customer-bff.yaml
  k8s/oci-dev/oci-employee-bff.yaml
  k8s/oci-dev/oci-pebble-proxy.yaml
  k8s/oci-dev/oci-worker.yaml
  k8s/oci-dev/batch-jobs.yaml
  k8s/external-sim/mock-hardware.yaml
  k8s/external-sim/mock-hardware-dev.yaml
)
for manifest in "${spiffe_tls_manifests[@]}"; do
  if ! grep -q "driver: spiffe.csi.cert-manager.io" "$manifest"; then
    echo "ERROR: $manifest must mount ephemeral SPIFFE SVIDs via driver: spiffe.csi.cert-manager.io"
    failed=1
  fi
  if ! grep -q "spiffe.csi.cert-manager.io/fs-group" "$manifest"; then
    echo "ERROR: $manifest must set spiffe.csi.cert-manager.io/fs-group on SPIFFE CSI volumes"
    failed=1
  fi
  if grep -qE "secretName: .*-tls-certs" "$manifest"; then
    echo "ERROR: $manifest still references static TLS secret (*-tls-certs)"
    failed=1
  fi
  if grep -q "^kind: Certificate$" "$manifest"; then
    echo "ERROR: $manifest must not define standalone kind: Certificate resources"
    failed=1
  fi
done

# 18. Enforce ExternalSecret usage and block raw static kind: Secret manifests (7.20d)
echo "Verifying elimination of static kind: Secret manifests in k8s/oci-core, oci-dev, oci-internal..."
if grep -rnE "^kind: Secret$" k8s/oci-core/*.yaml k8s/oci-dev/*.yaml k8s/oci-internal/*.yaml 2>/dev/null; then
  echo "ERROR: Static kind: Secret manifests are prohibited in k8s/oci-core/, k8s/oci-dev/, and k8s/oci-internal/"
  failed=1
fi
for es_manifest in k8s/oci-core/oci-secrets.yaml k8s/oci-dev/oci-secrets.yaml k8s/oci-internal/internal-secrets.yaml; do
  if ! grep -q "^kind: ExternalSecret$" "$es_manifest"; then
    echo "ERROR: $es_manifest must define ExternalSecret resources"
    failed=1
  fi
done

# 19. Verify Architectural Invariant SAST, govulncheck, gosec & gitleaks gates (7.30)
echo "Verifying Phase 8 (7.30) SAST, linter, vulnerability, and secret gates..."
for cfg_file in .semgrep.yml .golangci.yml .gitleaks.toml; do
  if [ ! -f "$cfg_file" ]; then
    echo "ERROR: missing required Phase 8 config file: $cfg_file"
    failed=1
  fi
done
for ci_cmd in "semgrep --config .semgrep.yml --error" "govulncheck ./..." "golangci-lint run" "gitleaks detect"; do
  if ! grep -q "$ci_cmd" .forgejo/workflows/ci.yaml; then
    echo "ERROR: .forgejo/workflows/ci.yaml missing required Phase 8 scanner command: $ci_cmd"
    failed=1
  fi
done
if command -v semgrep &>/dev/null; then
  if ! semgrep --config .semgrep.yml --error; then
    echo "ERROR: semgrep --config .semgrep.yml --error reported violations"
    failed=1
  fi
fi
if command -v govulncheck &>/dev/null; then
  if ! govulncheck ./...; then
    echo "ERROR: govulncheck ./... reported vulnerabilities"
    failed=1
  fi
fi
if command -v golangci-lint &>/dev/null; then
  if ! golangci-lint run; then
    echo "ERROR: golangci-lint run failed"
    failed=1
  fi
fi
if command -v gitleaks &>/dev/null; then
  gitleaks_tmp="$(mktemp -d)"
  git ls-files -z | xargs -0 tar -cf - | tar -xf - -C "$gitleaks_tmp"
  if ! gitleaks detect --no-git --source "$gitleaks_tmp" --config .gitleaks.toml; then
    echo "ERROR: gitleaks detect reported secrets"
    failed=1
  fi
  rm -rf "$gitleaks_tmp"
fi
if ! python3 - <<'PY'
import fnmatch, os, re, sys, yaml

cfg = yaml.safe_load(open(".semgrep.yml", "r", encoding="utf-8"))
rules = cfg.get("rules", [])
expected_ids = {
    "oci-bff-no-direct-db-or-domain-import",
    "oci-workforce-api-no-cross-domain-import",
    "oci-facilities-api-no-cross-domain-import",
    "oci-core-invest-api-no-cross-domain-import",
    "oci-ops-api-no-cross-domain-import",
    "oci-no-vendor-identifiers-outside-drivers",
    "oci-no-hardcoded-persona-ids",
}
found_ids = {r.get("id") for r in rules}
if not expected_ids.issubset(found_ids):
    print(f"ERROR: .semgrep.yml missing rules: {sorted(expected_ids - found_ids)}")
    sys.exit(1)

gcfg = yaml.safe_load(open(".golangci.yml", "r", encoding="utf-8"))
if "gosec" not in gcfg.get("linters", {}).get("enable", []):
    print("ERROR: .golangci.yml must enable gosec")
    sys.exit(1)

def match_globs(path, globs):
    for g in globs:
        if g == "**/*_test.go" and path.endswith("_test.go"):
            return True
        if g == "**/mock*.go" and os.path.basename(path).startswith("mock"):
            return True
        if g.endswith("/**") and path.startswith(g[:-2]):
            return True
        if fnmatch.fnmatch(path, g):
            return True
    return False

def eval_rule(spec, text):
    if "pattern-regex" in spec:
        return [m.group(0) for m in re.finditer(spec["pattern-regex"], text, re.MULTILINE)]
    if "pattern-either" in spec:
        res = []
        for sub in spec["pattern-either"]:
            res.extend(eval_rule(sub, text))
        return res
    if "patterns" in spec:
        pos = []
        negs = []
        for sub in spec["patterns"]:
            if "pattern-not-regex" in sub:
                negs.append(sub["pattern-not-regex"])
            elif "pattern-regex" in sub:
                for m in re.finditer(sub["pattern-regex"], text, re.MULTILINE):
                    pos.append(m.group(0))
            elif "pattern-either" in sub:
                for sub2 in sub["pattern-either"]:
                    for m in re.finditer(sub2["pattern-regex"], text, re.MULTILINE):
                        pos.append(m.group(0))
        return [m for m in pos if not any(re.search(n, m) for n in negs)]
    return []

# 1) Verify zero hits on the live monorepo
go_files = []
for root in ["cmd", "internal", "pkg"]:
    for dp, _, fns in os.walk(root):
        for fn in fns:
            if fn.endswith(".go"):
                go_files.append(os.path.join(dp, fn))

for rule in rules:
    inc = rule.get("paths", {}).get("include", [])
    exc = rule.get("paths", {}).get("exclude", [])
    for p in go_files:
        if inc and not match_globs(p, inc):
            continue
        if exc and match_globs(p, exc):
            continue
        txt = open(p, "r", encoding="utf-8").read()
        hits = eval_rule(rule, txt)
        if hits:
            print(f"ERROR: .semgrep.yml rule {rule['id']} triggered on {p}: {hits}")
            sys.exit(1)

# 2) Verify negative fixtures trigger every architectural invariant rule
neg_fixtures = [
    ("oci-bff-no-direct-db-or-domain-import", "internal/app/customerbff/bad.go", 'import "github.com/jackc/pgx/v5/pgxpool"'),
    ("oci-bff-no-direct-db-or-domain-import", "cmd/employee-bff/bad.go", 'c := workforcev1.NewWorkforceServiceClient(conn)'),
    ("oci-workforce-api-no-cross-domain-import", "cmd/workforce-api/bad.go", 'import "github.com/orange-cat-investments/oci/pkg/gen/go/facilities/v1"'),
    ("oci-no-vendor-identifiers-outside-drivers", "internal/handler/bad.go", 'var alpacaClient = "bad"'),
    ("oci-no-hardcoded-persona-ids", "internal/app/customerbll/bad.go", 'const id = "emp-feline-garfield"'),
]
rule_map = {r["id"]: r for r in rules}
for rid, vpath, snippet in neg_fixtures:
    r = rule_map[rid]
    inc = r.get("paths", {}).get("include", [])
    exc = r.get("paths", {}).get("exclude", [])
    assert match_globs(vpath, inc) and not match_globs(vpath, exc), f"Fixture path {vpath} not matched by {rid}"
    if not eval_rule(r, snippet):
        print(f"ERROR: Negative fixture failed to trigger {rid} for snippet: {snippet}")
        sys.exit(1)

print("All .semgrep.yml rules and negative fixture checks verified.")
PY
then
  failed=1
fi

# 20. Verify Distroless Go base images & Restricted Pod Security Standards (7.31)
echo "Verifying Phase 8 (7.31) Distroless Dockerfiles and K8s securityContext..."
for df in cmd/*/Dockerfile web/marketing/Dockerfile; do
  if [ -f "$df" ]; then
    if ! grep -q "^FROM gcr.io/distroless/static-debian12:nonroot" "$df"; then
      echo "ERROR: $df must use FROM gcr.io/distroless/static-debian12:nonroot in runtime stage"
      failed=1
    fi
    if ! grep -q "^USER nonroot:nonroot" "$df"; then
      echo "ERROR: $df must set USER nonroot:nonroot"
      failed=1
    fi
    if [ "$(grep -c "^FROM " "$df")" -gt 1 ] && grep -A 50 "^FROM gcr.io/distroless" "$df" | grep -q "alpine"; then
      echo "ERROR: $df runtime stage must not reference alpine"
      failed=1
    fi
  fi
done
if ! grep -q "containerPort: 8080" k8s/oci-core/marketing.yaml || ! grep -q "containerPort: 8080" k8s/oci-dev/marketing.yaml; then
  echo "ERROR: oci-marketing manifests must bind unprivileged containerPort 8080 for distroless nonroot execution"
  failed=1
fi
if ! python3 - <<'PY'
import glob, sys, yaml

go_manifests = [
    "k8s/oci-core/domain-apis.yaml",
    "k8s/oci-core/bll-services.yaml",
    "k8s/oci-core/oci-customer-bff.yaml",
    "k8s/oci-core/oci-employee-bff.yaml",
    "k8s/oci-core/oci-pebble-proxy.yaml",
    "k8s/oci-core/oci-worker.yaml",
    "k8s/oci-core/batch-jobs.yaml",
    "k8s/oci-core/db-migrate-job.yaml",
    "k8s/oci-core/marketing.yaml",
    "k8s/oci-dev/domain-apis.yaml",
    "k8s/oci-dev/bll-services.yaml",
    "k8s/oci-dev/oci-customer-bff.yaml",
    "k8s/oci-dev/oci-employee-bff.yaml",
    "k8s/oci-dev/oci-pebble-proxy.yaml",
    "k8s/oci-dev/oci-worker.yaml",
    "k8s/oci-dev/batch-jobs.yaml",
    "k8s/oci-dev/db-migrate-job.yaml",
    "k8s/oci-dev/marketing.yaml",
    "k8s/external-sim/mock-bank.yaml",
    "k8s/external-sim/mock-bank-dev.yaml",
    "k8s/external-sim/mock-broker.yaml",
    "k8s/external-sim/mock-broker-dev.yaml",
    "k8s/external-sim/mock-hardware.yaml",
    "k8s/external-sim/mock-hardware-dev.yaml",
]
errors = []
for path in go_manifests:
    with open(path, "r", encoding="utf-8") as f:
        for doc in yaml.safe_load_all(f):
            if not isinstance(doc, dict):
                continue
            kind = doc.get("kind")
            if kind in ("Deployment", "Job"):
                containers = doc.get("spec", {}).get("template", {}).get("spec", {}).get("containers", [])
            elif kind == "CronJob":
                containers = doc.get("spec", {}).get("jobTemplate", {}).get("spec", {}).get("template", {}).get("spec", {}).get("containers", [])
            else:
                continue
            for c in containers:
                sc = c.get("securityContext", {})
                if sc.get("runAsNonRoot") is not True or sc.get("runAsUser") != 65532 or sc.get("readOnlyRootFilesystem") is not True or sc.get("allowPrivilegeEscalation") is not False or "ALL" not in sc.get("capabilities", {}).get("drop", []):
                    errors.append(f"{path} ({doc.get('metadata', {}).get('name')}/{c.get('name')}) missing restricted securityContext")
if errors:
    for e in errors:
        print("ERROR:", e)
    sys.exit(1)
print("All Go Dockerfiles and Kubernetes securityContexts verified.")
PY
then
  failed=1
fi

# 21. Verify Forgejo Package Registry configuration and firmware publisher (7.32)
echo "Verifying Phase 8 (7.32) Forgejo Package Registry..."
if ! grep -q "FORGEJO__packages__ENABLED" k8s/oci-internal/forgejo.yaml; then
  echo "ERROR: k8s/oci-internal/forgejo.yaml must enable FORGEJO__packages__ENABLED"
  failed=1
fi
if [ ! -x scripts/publish-firmware-package.sh ] || ! bash -n scripts/publish-firmware-package.sh; then
  echo "ERROR: scripts/publish-firmware-package.sh must exist, be executable, and pass bash syntax validation"
  failed=1
fi

# 22. Verify Harbor Enterprise Container Registry & Trivy gate (7.33)
echo "Verifying Phase 8 (7.33) Harbor Registry & Trivy scan gate..."
if [ -f k8s/registry/registry.yaml ] || [ -f k8s/registry/registry-gc-cronjob.yaml ]; then
  echo "ERROR: Deprecated barebones registry:2 manifests must be removed in favor of k8s/registry/harbor.yaml"
  failed=1
fi
if [ ! -f k8s/registry/harbor.yaml ]; then
  echo "ERROR: Missing k8s/registry/harbor.yaml"
  failed=1
fi
for pat in "harbor-core" "harbor-portal" "harbor-registry" "harbor-jobservice" "harbor-trivy" "nodePort: 32000" "registry.oci.local" "oidc_auth" "proxy-dockerhub" "proxy-ghcr" "proxy-gcr"; do
  if ! grep -q "$pat" k8s/registry/harbor.yaml; then
    echo "ERROR: k8s/registry/harbor.yaml missing required Harbor component/config: $pat"
    failed=1
  fi
done
if ! grep -q "trivy image --exit-code 1 --severity CRITICAL,HIGH" .forgejo/workflows/ci.yaml; then
  echo "ERROR: .forgejo/workflows/ci.yaml missing Trivy CRITICAL,HIGH vulnerability scan gate"
  failed=1
fi

# 23. Verify CycloneDX SBOM, Cosign signing & 3-copy SQL sync (7.34)
echo "Verifying Phase 8 (7.34) CycloneDX SBOMs, Cosign signing, and SQL 3-copy sync..."
for sql_file in scripts/init.sql k8s/oci-core/postgres-core-configmap.yaml k8s/oci-dev/postgres-dev-configmap.yaml scripts/migrations/002-firmware-cosign-signature.sql; do
  if ! grep -q "cosign_signature" "$sql_file"; then
    echo "ERROR: $sql_file missing cosign_signature column on facilities.firmware_releases"
    failed=1
  fi
done
for cm in k8s/oci-core/cosign-pub-configmap.yaml k8s/oci-dev/cosign-pub-configmap.yaml k8s/external-sim/cosign-pub-configmap.yaml; do
  if [ ! -f "$cm" ] || ! grep -q "BEGIN PUBLIC KEY" "$cm"; then
    echo "ERROR: $cm missing valid oci-cosign-pub ConfigMap"
    failed=1
  fi
done
for ci_pat in "syft" "cyclonedx" "cosign sign" "cosign attest"; do
  if ! grep -q "$ci_pat" .forgejo/workflows/ci.yaml; then
    echo "ERROR: .forgejo/workflows/ci.yaml missing SBOM/signing command: $ci_pat"
    failed=1
  fi
done

# 24. Verify Kyverno Admission Control Policies & Negative Fixtures (7.35)
echo "Verifying Phase 8 (7.35) Kyverno Admission Policies..."
for pol in "verify-oci-image-signatures" "disallow-latest-tag" "enforce-restricted-pss" "forbid-tls-subpath-mounts"; do
  if ! grep -q "name: ${pol}" k8s/security/kyverno-policies.yaml; then
    echo "ERROR: k8s/security/kyverno-policies.yaml missing ClusterPolicy: ${pol}"
    failed=1
  fi
done
if command -v kyverno &>/dev/null; then
  if ! kyverno apply k8s/security/kyverno-policies.yaml --resource k8s/oci-core/ >/dev/null 2>&1; then
    echo "ERROR: kyverno apply failed on k8s/oci-core/"
    failed=1
  fi
  if kyverno apply k8s/security/kyverno-policies.yaml --resource k8s/security/fixtures/invalid-manifests.yaml >/dev/null 2>&1; then
    echo "ERROR: kyverno apply failed to reject k8s/security/fixtures/invalid-manifests.yaml"
    failed=1
  fi
fi
if ! python3 - <<'PY'
import glob, re, sys, yaml

policies = list(yaml.safe_load_all(open("k8s/security/kyverno-policies.yaml", "r", encoding="utf-8")))
pol_names = {p.get("metadata", {}).get("name"): p for p in policies if isinstance(p, dict)}
for req in ("verify-oci-image-signatures", "disallow-latest-tag", "enforce-restricted-pss", "forbid-tls-subpath-mounts"):
    if req not in pol_names or pol_names[req].get("spec", {}).get("validationFailureAction") != "Enforce":
        print(f"ERROR: Policy {req} missing or validationFailureAction != Enforce")
        sys.exit(1)

pss_excluded = set(
    pol_names["enforce-restricted-pss"]["spec"]["rules"][0]["exclude"]["any"][0]["resources"]["selector"]["matchExpressions"][0]["values"]
)

def evaluate_pod_policies(ns, labels, spec):
    violations = []
    containers = spec.get("containers", [])
    app_label = labels.get("app", "")
    # 1. disallow-latest-tag (oci-core)
    if ns == "oci-core":
        for c in containers:
            img = c.get("image", "")
            if img.endswith(":latest") or img.endswith(":dev-latest"):
                violations.append(("disallow-latest-tag", img))
            if img.startswith("localhost:32000/") and "unsigned" in img:
                violations.append(("verify-oci-image-signatures", img))
    # 2. enforce-restricted-pss (oci-core, oci-dev excluding COTS/stateful allowlist)
    if ns in ("oci-core", "oci-dev") and app_label not in pss_excluded:
        for c in containers:
            sc = c.get("securityContext", {})
            if sc.get("runAsNonRoot") is not True or sc.get("readOnlyRootFilesystem") is not True or sc.get("allowPrivilegeEscalation") is not False or "ALL" not in sc.get("capabilities", {}).get("drop", []):
                violations.append(("enforce-restricted-pss", c.get("name")))
    # 3. forbid-tls-subpath-mounts (oci-core, oci-dev)
    if ns in ("oci-core", "oci-dev"):
        for c in containers:
            for vm in c.get("volumeMounts", []):
                if vm.get("subPath") and (re.search(r"(tls|cert|ca)", vm.get("mountPath", ""), re.I) or re.search(r"(tls|cert|ca)", vm.get("name", ""), re.I)):
                    violations.append(("forbid-tls-subpath-mounts", vm.get("mountPath")))
    return violations

# Verify 100% of k8s/oci-core/ and k8s/oci-dev/ pass
for path in sorted(glob.glob("k8s/oci-core/*.yaml") + glob.glob("k8s/oci-dev/*.yaml")):
    for doc in yaml.safe_load_all(open(path, "r", encoding="utf-8")):
        if not isinstance(doc, dict):
            continue
        kind = doc.get("kind")
        ns = doc.get("metadata", {}).get("namespace", "")
        if kind == "Deployment":
            tpl = doc["spec"]["template"]
        elif kind == "StatefulSet":
            tpl = doc["spec"]["template"]
        elif kind == "CronJob":
            tpl = doc["spec"]["jobTemplate"]["spec"]["template"]
        else:
            continue
        v = evaluate_pod_policies(ns, tpl.get("metadata", {}).get("labels", {}), tpl.get("spec", {}))
        if v:
            print(f"ERROR: Valid manifest {path} ({doc.get('metadata', {}).get('name')}) failed Kyverno policy check: {v}")
            sys.exit(1)

# Verify all 4 negative fixtures are rejected by their expected policy
fixtures = [d for d in yaml.safe_load_all(open("k8s/security/fixtures/invalid-manifests.yaml", "r", encoding="utf-8")) if isinstance(d, dict)]
seen_policies = set()
for pod in fixtures:
    ns = pod["metadata"]["namespace"]
    v = evaluate_pod_policies(ns, pod["metadata"].get("labels", {}), pod["spec"])
    if not v:
        print(f"ERROR: Negative fixture {pod['metadata']['name']} unexpectedly passed policies!")
        sys.exit(1)
    for pol_id, _ in v:
        seen_policies.add(pol_id)

expected_neg = {"disallow-latest-tag", "forbid-tls-subpath-mounts", "verify-oci-image-signatures", "enforce-restricted-pss"}
if not expected_neg.issubset(seen_policies):
    print(f"ERROR: Negative fixtures did not trigger all policies: missing {sorted(expected_neg - seen_policies)}")
    sys.exit(1)

print("All Kyverno policies and negative fixture evaluations verified.")
PY
then
  failed=1
fi

# 25. Verify Zero-Trust Default-Deny NetworkPolicy Microsegmentation Matrix (7.36)
echo "Verifying Phase 9 (7.36) Zero-Trust NetworkPolicy microsegmentation..."
if ! python3 - <<'PY'
import sys, yaml

manifests = {
    "oci-core": "k8s/oci-core/network-policies.yaml",
    "oci-dev": "k8s/oci-dev/network-policies.yaml",
    "oci-internal": "k8s/oci-internal/network-policies.yaml",
    "external-sim": "k8s/external-sim/network-policies.yaml",
}

def extract_ports(rule_list):
    ports = []
    for r in rule_list or []:
        for p in r.get("ports", []) or []:
            if "port" in p:
                ports.append(int(p["port"]))
    return ports

errors = []
for ns, path in manifests.items():
    docs = [d for d in yaml.safe_load_all(open(path, "r", encoding="utf-8")) if isinstance(d, dict) and d.get("kind") == "NetworkPolicy"]
    default_deny = next((d for d in docs if d.get("metadata", {}).get("name") == "default-deny-all"), None)
    if not default_deny:
        errors.append(f"{path} missing default-deny-all NetworkPolicy")
    else:
        ptypes = set(default_deny.get("spec", {}).get("policyTypes", []))
        sel = default_deny.get("spec", {}).get("podSelector", None)
        if ptypes != {"Ingress", "Egress"} or sel != {}:
            errors.append(f"{path} default-deny-all must select empty podSelector {{}} with policyTypes [Ingress, Egress]")

    for d in docs:
        name = d.get("metadata", {}).get("name", "")
        spec = d.get("spec", {})
        sel_labels = spec.get("podSelector", {}).get("matchLabels", {})
        egress_rules = spec.get("egress", []) or []
        egress_ports = extract_ports(egress_rules)

        # BFFs must NEVER have egress to Postgres :5432 or Domain APIs :50051-50054
        if sel_labels.get("tier") == "bff":
            if "Egress" not in spec.get("policyTypes", []):
                errors.append(f"{path}/{name} BFF policy must restrict Egress")
            forbidden = [p for p in egress_ports if p == 5432 or 50051 <= p <= 50054]
            if forbidden:
                errors.append(f"{path}/{name} BFF policy allows forbidden egress ports: {forbidden}")
            for rule in egress_rules:
                for peer in rule.get("to", []) or []:
                    pl = peer.get("podSelector", {}).get("matchLabels", {})
                    if pl.get("tier") == "domain-api" or str(pl.get("app", "")).startswith("postgres-"):
                        errors.append(f"{path}/{name} BFF policy has forbidden egress peer: {pl}")

        # Domain APIs must ONLY egress to Postgres :5432 (zero east-west domain-api egress)
        if sel_labels.get("tier") == "domain-api":
            if "Egress" not in spec.get("policyTypes", []):
                errors.append(f"{path}/{name} Domain API policy must restrict Egress")
            for p in egress_ports:
                if p != 5432:
                    errors.append(f"{path}/{name} Domain API allows non-5432 egress port: {p}")
            for rule in egress_rules:
                for peer in rule.get("to", []) or []:
                    pl = peer.get("podSelector", {}).get("matchLabels", {})
                    if pl.get("tier") == "domain-api":
                        errors.append(f"{path}/{name} Domain API allows east-west domain-api egress")

        # external-sim mocks must NEVER egress to any PostgreSQL database (:5432)
        if ns == "external-sim":
            if 5432 in egress_ports:
                errors.append(f"{path}/{name} external-sim allows forbidden :5432 egress")
            for rule in egress_rules:
                for peer in rule.get("to", []) or []:
                    pl = peer.get("podSelector", {}).get("matchLabels", {})
                    if "postgres" in str(pl.get("app", "")):
                        errors.append(f"{path}/{name} external-sim allows postgres peer egress: {pl}")

if errors:
    for e in errors:
        print("ERROR:", e)
    sys.exit(1)
print("All Phase 9 (7.36) Zero-Trust NetworkPolicy invariants verified.")
PY
then
  failed=1
fi

# 26. Verify Kubernetes Gateway API Migration & Absence of Legacy Ingress (7.37)
echo "Verifying Phase 9 (7.37) Kubernetes Gateway API migration..."
legacy_ingress="$(grep -rn "^kind: Ingress$" k8s/ || true)"
if [ -n "$legacy_ingress" ]; then
  echo "ERROR: Legacy kind: Ingress resources remain in k8s/:"
  echo "$legacy_ingress"
  failed=1
fi
if [ ! -f k8s/ha/lan/tls/gateway.yaml ]; then
  echo "ERROR: Missing k8s/ha/lan/tls/gateway.yaml"
  failed=1
fi
if ! python3 - <<'PY'
import sys, yaml

gw_docs = [d for d in yaml.safe_load_all(open("k8s/ha/lan/tls/gateway.yaml", "r", encoding="utf-8")) if isinstance(d, dict)]
gc = next((d for d in gw_docs if d.get("kind") == "GatewayClass" and d.get("metadata", {}).get("name") == "oci-gateway-class"), None)
gw = next((d for d in gw_docs if d.get("kind") == "Gateway" and d.get("metadata", {}).get("name") == "oci-edge-gateway"), None)
if not gc or not gw:
    print("ERROR: k8s/ha/lan/tls/gateway.yaml must define GatewayClass oci-gateway-class and Gateway oci-edge-gateway")
    sys.exit(1)

route_files = [
    "k8s/oci-core/http-routes.yaml",
    "k8s/oci-dev/http-routes.yaml",
    "k8s/oci-internal/http-routes.yaml",
    "k8s/external-sim/http-routes.yaml",
]
for rf in route_files:
    docs = [d for d in yaml.safe_load_all(open(rf, "r", encoding="utf-8")) if isinstance(d, dict)]
    http_routes = [d for d in docs if d.get("kind") in ("HTTPRoute", "GRPCRoute")]
    if not http_routes:
        print(f"ERROR: {rf} must define HTTPRoute/GRPCRoute resources")
        sys.exit(1)
    for rt in http_routes:
        if rt.get("apiVersion") != "gateway.networking.k8s.io/v1":
            print(f"ERROR: {rf}/{rt.get('metadata', {}).get('name')} must use apiVersion gateway.networking.k8s.io/v1")
            sys.exit(1)
        prefs = rt.get("spec", {}).get("parentRefs", [])
        if not any(p.get("name") == "oci-edge-gateway" for p in prefs):
            print(f"ERROR: {rf}/{rt.get('metadata', {}).get('name')} missing parentRef oci-edge-gateway")
            sys.exit(1)

print("All Gateway API GatewayClass, Gateway, HTTPRoute, and GRPCRoute manifests verified.")
PY
then
  failed=1
fi
if command -v kubectl &>/dev/null && kubectl cluster-info >/dev/null 2>&1; then
  for rf in k8s/ha/lan/tls/gateway.yaml k8s/oci-core/http-routes.yaml k8s/oci-dev/http-routes.yaml k8s/oci-internal/http-routes.yaml k8s/external-sim/http-routes.yaml; do
    if ! kubectl apply --dry-run=client -f "$rf" >/dev/null 2>&1; then
      echo "ERROR: kubectl apply --dry-run=client failed for $rf"
      failed=1
    fi
  done
fi

# 27. Verify Coraza OWASP CRS WAF Middleware & Telemetry Wiring (7.38)
echo "Verifying Phase 9 (7.38) Coraza WAF middleware and telemetry..."
for pat_file in \
  "pkg/waf/coraza.go:RecordWAFBlockedRequest" \
  "internal/app/customerbff/router.go:waf.Middleware" \
  "internal/app/employeebff/router.go:waf.Middleware"; do
  fpath="${pat_file%%:*}"
  fpat="${pat_file#*:}"
  if [ ! -f "$fpath" ] || ! grep -q "$fpat" "$fpath"; then
    echo "ERROR: $fpath missing expected Phase 9 (7.38) WAF symbol: $fpat"
    failed=1
  fi
done

# 28. Verify In-Cluster coturn STUN/TURN & ICE Server Discovery (7.39)
echo "Verifying Phase 9 (7.39) coturn relay and /api/v1/customer/webrtc/ice-servers..."
for coturn_manifest in k8s/oci-core/coturn.yaml k8s/oci-dev/coturn.yaml; do
  if [ ! -f "$coturn_manifest" ] || ! grep -q "use-auth-secret" "$coturn_manifest" || ! grep -q "3478" "$coturn_manifest"; then
    echo "ERROR: $coturn_manifest missing coturn deployment or --use-auth-secret config"
    failed=1
  fi
done
if ! grep -q "/api/v1/customer/webrtc/ice-servers" internal/app/customerbff/router.go || \
   ! grep -q "/api/v1/customer/webrtc/ice-servers" api/openapi.yaml || \
   ! grep -q "/api/v1/customer/webrtc/ice-servers" web/customer-portal/src/App.vue; then
  echo "ERROR: /api/v1/customer/webrtc/ice-servers must be wired across customerbff, openapi.yaml, and customer-portal"
  failed=1
fi

# 29. Verify Phase 10 (7.40) Transactional SQL Outbox & Valkey Idempotency Middleware
echo "Verifying Phase 10 (7.40) transactional SQL outbox and Valkey idempotency..."
for sql_file in scripts/init.sql k8s/oci-core/postgres-core-configmap.yaml k8s/oci-dev/postgres-dev-configmap.yaml scripts/migrations/003-transactional-outbox.sql; do
  for schema_tbl in workforce.outbox_events facilities.outbox_events core_invest.outbox_events ops.outbox_events; do
    if [ ! -f "$sql_file" ] || ! grep -q "$schema_tbl" "$sql_file"; then
      echo "ERROR: $sql_file missing Phase 10 (7.40) outbox table: $schema_tbl"
      failed=1
    fi
  done
done
for pat_file in \
  "pkg/outbox/outbox.go:FOR UPDATE SKIP LOCKED" \
  "pkg/cache/valkey/idempotency.go:IdempotencyMiddleware" \
  "internal/app/customerbff/router.go:IdempotencyMiddleware" \
  "cmd/worker/main.go:StartOutboxForwarder"; do
  fpath="${pat_file%%:*}"
  fpat="${pat_file#*:}"
  if [ ! -f "$fpath" ] || ! grep -q "$fpat" "$fpath"; then
    echo "ERROR: $fpath missing expected Phase 10 (7.40) symbol: $fpat"
    failed=1
  fi
done

# 30. Verify Phase 10 (7.41) Tamper-Evident Hash-Chain Audit Log
echo "Verifying Phase 10 (7.41) tamper-evident hash-chain audit log (ops.audit_logs)..."
for sql_file in scripts/init.sql k8s/oci-core/postgres-core-configmap.yaml k8s/oci-dev/postgres-dev-configmap.yaml scripts/migrations/004-tamper-evident-audit-log.sql; do
  if [ ! -f "$sql_file" ] || ! grep -q "ops.audit_logs" "$sql_file" || ! grep -q "prevent_audit_log_mutation" "$sql_file" || ! grep -q "compute_audit_log_hash" "$sql_file"; then
    echo "ERROR: $sql_file missing Phase 10 (7.41) ops.audit_logs table or immutability/hash triggers"
    failed=1
  fi
  # Chain-lock fix: the audit id must be drawn from an explicit sequence
  # inside the hash trigger AFTER the advisory lock (never via a column
  # default), or concurrent appends can build a chain that ascending-id
  # verification rejects.
  if [ -f "$sql_file" ] && { ! grep -q "ops.audit_logs_id_seq" "$sql_file" || ! grep -q "nextval('ops.audit_logs_id_seq')" "$sql_file"; }; then
    echo "ERROR: $sql_file missing serialized audit id allocation (ops.audit_logs_id_seq + in-trigger nextval)"
    failed=1
  fi
done
for pat_file in \
  "internal/repository/ops/audit.go:VerifyAuditChain" \
  "internal/handler/ops/audit.go:VerifyAuditChain" \
  "internal/app/employeebff/ops_routes.go:/api/v1/ops/audit-logs/verify" \
  "api/openapi.yaml:/ops/audit-logs/verify"; do
  fpath="${pat_file%%:*}"
  fpat="${pat_file#*:}"
  if [ ! -f "$fpath" ] || ! grep -q "$fpat" "$fpath"; then
    echo "ERROR: $fpath missing expected Phase 10 (7.41) symbol: $fpat"
    failed=1
  fi
done

# 31. Verify Phase 10 (7.42) Embedded OPA Policy-as-Code Engine in BLL Tier
echo "Verifying Phase 10 (7.42) embedded OPA Rego policy engine..."
for pat_file in \
  "pkg/authz/policy/engine.go://go:embed policies/\*.rego" \
  "pkg/authz/policy/policies/workforce.rego:package oci.authz.workforce" \
  "pkg/authz/policy/policies/trading.rego:package oci.authz.trading" \
  "internal/app/internalbll/authz.go:EvaluateWorkforce" \
  "internal/app/customerbll/orders.go:EvaluateTrading"; do
  fpath="${pat_file%%:*}"
  fpat="${pat_file#*:}"
  if [ ! -f "$fpath" ] || ! grep -q "$fpat" "$fpath"; then
    echo "ERROR: $fpath missing expected Phase 10 (7.42) symbol: $fpat"
    failed=1
  fi
done

# 32. Verify Phase 11 (7.43) S3-Compatible Object Storage (Garage) & Pre-Signed Blobs
echo "Verifying Phase 11 (7.43) Garage object storage & S3 blobStore integration..."
for pat_file in \
  "pkg/blobstore/s3/client.go:EnsureBuckets" \
  "pkg/blobstore/s3/client.go:PresignGetURL" \
  "pkg/blobstore/s3/client.go:AWS4-HMAC-SHA256" \
  "k8s/oci-core/garage.yaml:dxflrs/garage:v1.0.1" \
  "k8s/oci-core/garage.yaml:oci-statements" \
  "k8s/oci-core/garage.yaml:oci-firmware" \
  "k8s/oci-core/garage.yaml:oci-backups" \
  "k8s/oci-dev/garage.yaml:dxflrs/garage:v1.0.1" \
  "k8s/oci-dev/garage.yaml:oci-statements" \
  "k8s/oci-dev/garage.yaml:oci-firmware" \
  "k8s/oci-dev/garage.yaml:oci-backups" \
  "k8s/oci-core/bll-services.yaml:BLOBSTORE_ENDPOINT" \
  "k8s/oci-dev/bll-services.yaml:BLOBSTORE_ENDPOINT" \
  "scripts/init-openbao.sh:BLOBSTORE_ACCESS_KEY" \
  "internal/app/customerbll/lifecycle_statements.go:oci-statements" \
  "internal/app/customerbll/lifecycle_statements.go:PresignGetURL" \
  "internal/app/customerbff/portfolio_handlers.go:X-PreSigned-Download-URL" \
  "internal/app/internalbll/facilities.go:oci-firmware"; do
  fpath="${pat_file%%:*}"
  fpat="${pat_file#*:}"
  if [ ! -f "$fpath" ] || ! grep -q "$fpat" "$fpath"; then
    echo "ERROR: $fpath missing expected Phase 11 (7.43) symbol: $fpat"
    failed=1
  fi
done

# 33. Verify Phase 11 (7.44) Continuous PostgreSQL WAL Archiving & PITR (PgBackRest)
echo "Verifying Phase 11 (7.44) PgBackRest WAL archiving & PITR drill..."
for pg_file in k8s/oci-core/postgres-core.yaml k8s/oci-dev/postgres-dev.yaml k8s/oci-internal/postgres-internal.yaml; do
  for pat in "wal_level=replica" "archive_mode=on" "pgbackrest" "repo1-s3-bucket=oci-backups"; do
    if ! grep -q "$pat" "$pg_file"; then
      echo "ERROR: $pg_file missing Phase 11 (7.44) WAL/PITR setting: $pat"
      failed=1
    fi
  done
done
if ! grep -q "PostgresWALArchiveFailing" docs/runbooks/stateful-store-alerts.md; then
  echo "ERROR: docs/runbooks/stateful-store-alerts.md missing PostgresWALArchiveFailing runbook"
  failed=1
fi
if [ ! -x scripts/verify-pitr-restore.sh ] || ! ./scripts/verify-pitr-restore.sh --dry-run >/dev/null; then
  echo "ERROR: scripts/verify-pitr-restore.sh must be executable and pass --dry-run"
  failed=1
fi
if ./scripts/verify-pitr-restore.sh --namespace oci-core >/dev/null 2>&1; then
  echo "ERROR: scripts/verify-pitr-restore.sh must refuse --namespace oci-core"
  failed=1
fi

# 34. Verify Phase 11 (7.45) In-Cluster Mailpit SMTP Sandbox & Notifications
echo "Verifying Phase 11 (7.45) Mailpit SMTP sandbox & simulator mail checks..."
for pat_file in \
  "k8s/oci-internal/mailpit.yaml:axllent/mailpit" \
  "k8s/oci-internal/mailpit.yaml:mailpit-service" \
  "k8s/oci-internal/http-routes.yaml:mail.oci.local" \
  "k8s/oci-core/oci-worker.yaml:mailpit-service.oci-internal.svc.cluster.local:1025" \
  "k8s/oci-dev/oci-worker.yaml:mailpit-service.oci-internal.svc.cluster.local:1025" \
  "k8s/oci-internal/forgejo.yaml:mailpit-service.oci-internal.svc.cluster.local:1025" \
  "k8s/oci-internal/nextcloud.yaml:mailpit-service.oci-internal.svc.cluster.local" \
  "k8s/oci-core/zitadel.yaml:mailpit-service.oci-internal.svc.cluster.local" \
  "k8s/oci-dev/zitadel.yaml:mailpit-service.oci-internal.svc.cluster.local" \
  "internal/app/worker/router.go:TopicEmployeeOnboarded" \
  "internal/app/worker/smtp_notifier.go:HandleEmployeeOnboarded" \
  "simulation/actions.go:rick.newhire@oci.local" \
  "simulation/actions.go:chloe@oci.local"; do
  fpath="${pat_file%%:*}"
  fpat="${pat_file#*:}"
  if [ ! -f "$fpath" ] || ! grep -q "$fpat" "$fpath"; then
    echo "ERROR: $fpath missing expected Phase 11 (7.45) symbol: $fpat"
    failed=1
  fi
done

# 35. Verify Phase 11 (7.46) Database Migration Runner, 3-Copy SQL Sync & PgBouncer
echo "Verifying Phase 11 (7.46) db-migrate pipeline, 3-copy SQL sync & PgBouncer..."
for pat_file in \
  "cmd/db-migrate/main.go:goose_db_version" \
  "cmd/db-migrate/main.go:parseGooseUpStatements" \
  "cmd/db-migrate/Dockerfile:gcr.io/distroless/static-debian12:nonroot" \
  "k8s/oci-core/db-migrate-job.yaml:localhost:32000/db-migrate:" \
  "k8s/oci-dev/db-migrate-job.yaml:localhost:32000/db-migrate:" \
  "k8s/oci-core/pgbouncer.yaml:POOL_MODE" \
  "k8s/oci-core/pgbouncer.yaml:postgres-core-backend" \
  "k8s/oci-dev/pgbouncer.yaml:postgres-dev-backend" \
  "k8s/oci-internal/pgbouncer.yaml:postgres-internal-backend"; do
  fpath="${pat_file%%:*}"
  fpat="${pat_file#*:}"
  if [ ! -f "$fpath" ] || ! grep -q "$fpat" "$fpath"; then
    echo "ERROR: $fpath missing expected Phase 11 (7.46) symbol: $fpat"
    failed=1
  fi
done
for pgb in k8s/oci-core/pgbouncer.yaml k8s/oci-dev/pgbouncer.yaml k8s/oci-internal/pgbouncer.yaml; do
  if ! grep -q 'value: transaction' "$pgb" || ! grep -q 'value: "500"' "$pgb" || ! grep -q 'value: "25"' "$pgb"; then
    echo "ERROR: $pgb must enforce transaction pool mode, MAX_CLIENT_CONN=500, DEFAULT_POOL_SIZE=25"
    failed=1
  fi
done
if ! python3 - <<'PY'
import glob, re, sys

bootstrap_files = [
    "scripts/init.sql",
    "k8s/oci-core/postgres-core-configmap.yaml",
    "k8s/oci-dev/postgres-dev-configmap.yaml",
]
boot_contents = {p: open(p, "r", encoding="utf-8").read() for p in bootstrap_files}

mig_files = sorted(glob.glob("scripts/migrations/*.sql"))
if not mig_files:
    print("ERROR: no migration files found in scripts/migrations/")
    sys.exit(1)

errors = []
for mf in mig_files:
    txt = open(mf, "r", encoding="utf-8").read()
    if "-- +goose Up" not in txt:
        errors.append(f"{mf} missing '-- +goose Up' directive")
    # Skip role-only migrations covered differently; check tables, indexes, functions, and added columns
    for tbl in re.findall(r"CREATE TABLE IF NOT EXISTS\s+([a-zA-Z0-9_.]+)", txt, flags=re.I):
        for bp, bc in boot_contents.items():
            if tbl not in bc:
                errors.append(f"3-copy SQL drift: table {tbl} from {mf} missing in {bp}")
    for fn in re.findall(r"CREATE OR REPLACE FUNCTION\s+([a-zA-Z0-9_.]+)", txt, flags=re.I):
        for bp, bc in boot_contents.items():
            if fn not in bc:
                errors.append(f"3-copy SQL drift: function {fn} from {mf} missing in {bp}")
    for idx in re.findall(r"CREATE (?:UNIQUE )?INDEX IF NOT EXISTS\s+([a-zA-Z0-9_]+)", txt, flags=re.I):
        for bp, bc in boot_contents.items():
            if idx not in bc:
                errors.append(f"3-copy SQL drift: index {idx} from {mf} missing in {bp}")
    for col in re.findall(r"ADD COLUMN IF NOT EXISTS\s+([a-zA-Z0-9_]+)", txt, flags=re.I):
        for bp, bc in boot_contents.items():
            if col not in bc:
                errors.append(f"3-copy SQL drift: column {col} from {mf} missing in {bp}")

if errors:
    for e in errors:
        print("ERROR:", e)
    sys.exit(1)
print(f"Verified {len(mig_files)} migrations across all 3 SQL bootstrap copies.")
PY
then
  failed=1
fi

# 36. Verify Phase 11 (7.47) Protobuf (buf) & OpenAPI (oasdiff) Contract Gate + Negative Tests
echo "Verifying Phase 11 (7.47) buf/oasdiff config & negative breaking-change checks..."
for pat_file in \
  "buf.yaml:version: v2" \
  "buf.yaml:STANDARD" \
  "buf.yaml:WIRE_JSON" \
  "buf.gen.yaml:pkg/gen/go" \
  ".forgejo/workflows/ci.yaml:buf lint" \
  ".forgejo/workflows/ci.yaml:buf breaking" \
  ".forgejo/workflows/ci.yaml:oasdiff breaking"; do
  fpath="${pat_file%%:*}"
  fpat="${pat_file#*:}"
  if [ ! -f "$fpath" ] || ! grep -q "$fpat" "$fpath"; then
    echo "ERROR: $fpath missing expected Phase 11 (7.47) symbol: $fpat"
    failed=1
  fi
done
if ! python3 - <<'PY'
import copy, glob, re, subprocess, sys, yaml

def parse_proto_messages(text):
    msgs = {}
    current = None
    depth = 0
    for raw_line in text.splitlines():
        line = raw_line.split("//", 1)[0].strip()
        if not line:
            continue
        m = re.match(r"message\s+([A-Za-z0-9_]+)\s*\{", line)
        if m and depth == 0:
            current = m.group(1)
            msgs[current] = {}
            depth = 1
            continue
        if current:
            if "{" in line:
                depth += line.count("{")
            if "}" in line:
                depth -= line.count("}")
                if depth <= 0:
                    current = None
                    depth = 0
                continue
            f = re.match(r"(?:repeated\s+|optional\s+)?([A-Za-z0-9_.<>]+)\s+([A-Za-z0-9_]+)\s*=\s*(\d+)\s*;", line)
            if f:
                msgs[current][int(f.group(3))] = (f.group(1), f.group(2))
    return msgs

def detect_proto_breaking(old_text, new_text):
    old_msgs = parse_proto_messages(old_text)
    new_msgs = parse_proto_messages(new_text)
    errs = []
    for msg, old_fields in old_msgs.items():
        if msg not in new_msgs:
            errs.append(f"deleted message {msg}")
            continue
        for num, (ftype, fname) in old_fields.items():
            if num not in new_msgs[msg]:
                errs.append(f"removed field {msg}.{fname}={num}")
            elif new_msgs[msg][num] != (ftype, fname):
                errs.append(f"changed field {msg} #{num}: {ftype} {fname} -> {new_msgs[msg][num]}")
    return errs

def detect_openapi_breaking(old_spec, new_spec):
    errs = []
    old_paths = old_spec.get("paths", {}) or {}
    new_paths = new_spec.get("paths", {}) or {}
    for p, pitem in old_paths.items():
        if p not in new_paths:
            errs.append(f"deleted path {p}")
            continue
        for method in ("get", "post", "put", "patch", "delete"):
            if method in (pitem or {}) and method not in (new_paths[p] or {}):
                errs.append(f"deleted operation {method.upper()} {p}")
    old_schemas = (old_spec.get("components", {}) or {}).get("schemas", {}) or {}
    new_schemas = (new_spec.get("components", {}) or {}).get("schemas", {}) or {}
    for sname, sdef in old_schemas.items():
        if sname not in new_schemas:
            continue
        old_req = set(sdef.get("required", []) or [])
        old_props = set((sdef.get("properties", {}) or {}).keys())
        new_props = set((new_schemas[sname].get("properties", {}) or {}).keys())
        for req_prop in old_req:
            if req_prop in old_props and req_prop not in new_props:
                errs.append(f"removed required schema property {sname}.{req_prop}")
    return errs

# 1. Verify current branch vs main (if main exists locally)
has_main = subprocess.run(["git", "rev-parse", "--verify", "main"], capture_output=True).returncode == 0
if has_main:
    for pfile in sorted(glob.glob("proto/oci/**/*.proto", recursive=True)):
        res = subprocess.run(["git", "show", f"main:{pfile}"], capture_output=True, text=True)
        if res.returncode == 0:
            cur = open(pfile, "r", encoding="utf-8").read()
            b = detect_proto_breaking(res.stdout, cur)
            if b:
                print(f"ERROR: breaking proto change in {pfile}: {b}")
                sys.exit(1)
    oas_res = subprocess.run(["git", "show", "main:api/openapi.yaml"], capture_output=True, text=True)
    if oas_res.returncode == 0:
        old_oas = yaml.safe_load(oas_res.stdout)
        cur_oas = yaml.safe_load(open("api/openapi.yaml", "r", encoding="utf-8"))
        b = detect_openapi_breaking(old_oas, cur_oas)
        if b:
            print(f"ERROR: breaking OpenAPI change vs main: {b}")
            sys.exit(1)

# 2. Negative self-tests (mutated proto field number & removed OpenAPI required response property MUST fail)
sample_proto = "message SampleResp {\n  string id = 1;\n}\n"
mutated_proto = "message SampleResp {\n  string id = 2;\n}\n"
if not detect_proto_breaking(sample_proto, mutated_proto):
    print("ERROR: proto negative breaking-change test unexpectedly passed")
    sys.exit(1)

cur_oas = yaml.safe_load(open("api/openapi.yaml", "r", encoding="utf-8"))
mutated_oas = copy.deepcopy(cur_oas)
target_schema = None
for sname, sdef in (mutated_oas.get("components", {}) or {}).get("schemas", {}).items():
    req = sdef.get("required", []) or []
    props = sdef.get("properties", {}) or {}
    if req and req[0] in props:
        del props[req[0]]
        target_schema = sname
        break
if not target_schema or not detect_openapi_breaking(cur_oas, mutated_oas):
    print("ERROR: OpenAPI negative breaking-change test unexpectedly passed")
    sys.exit(1)

print("Verified proto/OpenAPI compatibility and negative breaking-change assertions.")
PY
then
  failed=1
fi

if [ "$failed" -eq 1 ]; then
    echo "CI contracts verification failed."
    exit 1
fi

echo "CI contracts verification passed."
