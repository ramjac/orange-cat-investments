#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# COTS Application Maintenance & Upstream Patching Automation Helper
# Target: Orange Cat Investments (OCI) - oci-internal & oci-core
# Role: Platform Security & K8s Infrastructure Lead (emp-human-frank)
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

usage() {
    cat << EOF
Usage: $(basename "$0") <command> [arguments]

Commands:
  snapshot               Trigger automated pre-upgrade snapshots for postgres-internal and postgres-core
  enable-maintenance     Apply 503 maintenance banner to internal Ingress
  disable-maintenance    Remove 503 maintenance banner from internal Ingress
  verify                 Run health checks against all COTS applications (Forgejo, Nextcloud, Snipe-IT, ERPNext, Zitadel)
  rollback <deployment> [namespace]
                         Roll back a deployment to its previous revision (default namespace: oci-internal)

Examples:
  $(basename "$0") snapshot
  $(basename "$0") enable-maintenance
  $(basename "$0") verify
  $(basename "$0") rollback forgejo
  $(basename "$0") rollback zitadel oci-core
EOF
    exit 1
}

CMD="${1:-}"
if [[ -z "${CMD}" ]]; then
    usage
fi

case "${CMD}" in
    snapshot)
        echo "=========================================================="
        echo "  Triggering Pre-Upgrade Database Snapshots               "
        echo "=========================================================="
        SNAPSHOT_ID="pre-patch-$(date +%s)"

        echo "1. Triggering postgres-internal snapshot in oci-internal..."
        if kubectl get cronjob postgres-internal-backup-cronjob -n oci-internal &>/dev/null; then
            kubectl create job --from=cronjob/postgres-internal-backup-cronjob "${SNAPSHOT_ID}-internal" -n oci-internal
            echo "Waiting for postgres-internal backup to complete..."
            kubectl wait --for=condition=complete job/"${SNAPSHOT_ID}-internal" -n oci-internal --timeout=180s
            echo "✅ postgres-internal snapshot completed successfully."
        else
            echo "⚠️  postgres-internal-backup-cronjob not found in oci-internal."
        fi

        echo ""
        echo "2. Triggering postgres-core snapshot in oci-core..."
        if kubectl get cronjob postgres-core-backup-cronjob -n oci-core &>/dev/null; then
            kubectl create job --from=cronjob/postgres-core-backup-cronjob "${SNAPSHOT_ID}-core" -n oci-core
            echo "Waiting for postgres-core backup to complete..."
            kubectl wait --for=condition=complete job/"${SNAPSHOT_ID}-core" -n oci-core --timeout=180s
            echo "✅ postgres-core snapshot completed successfully."
        else
            echo "⚠️  postgres-core-backup-cronjob not found in oci-core."
        fi

        echo ""
        echo "=========================================================="
        echo "  Pre-Upgrade Snapshots Finished                          "
        echo "=========================================================="
        ;;

    enable-maintenance)
        echo "Deploying 503 maintenance responder and switching oci-internal COTS HTTPRoutes..."
        kubectl apply -f - <<'EOF'
apiVersion: v1
kind: ConfigMap
metadata:
  name: cots-maintenance-nginx-conf
  namespace: oci-internal
  labels:
    app: cots-maintenance-responder
data:
  default.conf: |
    server {
      listen 8080;
      location / {
        default_type text/plain;
        return 503 "OCI Systems Under Scheduled Maintenance. Back shortly.\n";
      }
    }
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cots-maintenance-responder
  namespace: oci-internal
  labels:
    app: cots-maintenance-responder
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cots-maintenance-responder
  template:
    metadata:
      labels:
        app: cots-maintenance-responder
    spec:
      containers:
        - name: responder
          image: nginx:1.27-alpine
          ports:
            - containerPort: 8080
              name: http
          volumeMounts:
            - name: conf
              mountPath: /etc/nginx/conf.d
              readOnly: true
            - name: cache
              mountPath: /var/cache/nginx
            - name: run
              mountPath: /var/run
          readinessProbe:
            httpGet:
              path: /
              port: 8080
            initialDelaySeconds: 2
            periodSeconds: 3
          resources:
            requests:
              cpu: 10m
              memory: 16Mi
            limits:
              cpu: 100m
              memory: 64Mi
      volumes:
        - name: conf
          configMap:
            name: cots-maintenance-nginx-conf
        - name: cache
          emptyDir: {}
        - name: run
          emptyDir: {}
---
apiVersion: v1
kind: Service
metadata:
  name: cots-maintenance-responder
  namespace: oci-internal
  labels:
    app: cots-maintenance-responder
spec:
  selector:
    app: cots-maintenance-responder
  ports:
    - name: http
      port: 8080
      targetPort: 8080
EOF
        kubectl rollout status deployment/cots-maintenance-responder -n oci-internal --timeout=60s

        if kubectl get httproute -n oci-internal -l oci.local/cots-route=true &>/dev/null; then
            while IFS= read -r route; do
                [[ -n "${route}" ]] || continue
                kubectl patch "${route}" -n oci-internal --type=json \
                    -p='[{"op":"replace","path":"/spec/rules/0/backendRefs","value":[{"name":"cots-maintenance-responder","port":8080}]}]'
            done < <(kubectl get httproute -n oci-internal -l oci.local/cots-route=true -o name)
            kubectl annotate httproute -n oci-internal -l oci.local/cots-route=true \
                oci.local/maintenance-banner="OCI Systems Under Scheduled Maintenance. Back shortly." \
                --overwrite
            echo "✅ Maintenance 503 responder active on oci-internal COTS HTTPRoutes."
        else
            echo "⚠️  oci-internal COTS HTTPRoutes not found in oci-internal."
        fi
        ;;

    disable-maintenance)
        echo "Restoring oci-internal COTS HTTPRoutes and removing maintenance responder..."
        kubectl apply -f "${REPO_ROOT}/k8s/oci-internal/http-routes.yaml"
        if kubectl get httproute -n oci-internal -l oci.local/cots-route=true &>/dev/null; then
            kubectl annotate httproute -n oci-internal -l oci.local/cots-route=true \
                oci.local/maintenance-banner- || true
            echo "✅ Original COTS HTTPRoutes restored."
        else
            echo "⚠️  oci-internal COTS HTTPRoutes not found in oci-internal."
        fi
        kubectl delete deployment/cots-maintenance-responder service/cots-maintenance-responder configmap/cots-maintenance-nginx-conf \
            -n oci-internal --ignore-not-found
        ;;

    verify)
        echo "=========================================================="
        echo "  Verifying COTS Workload Health                          "
        echo "=========================================================="

        echo "1. Verifying oci-internal workload rollouts..."
        kubectl rollout status statefulset/postgres-internal -n oci-internal --timeout=120s
        kubectl rollout status deployment/forgejo -n oci-internal --timeout=120s
        kubectl rollout status deployment/nextcloud -n oci-internal --timeout=120s
        kubectl rollout status deployment/snipe-it -n oci-internal --timeout=120s
        kubectl rollout status deployment/erpnext-frappe -n oci-internal --timeout=120s
        kubectl rollout status deployment/frappe-hr -n oci-internal --timeout=120s

        echo "2. Verifying oci-core identity workload rollouts..."
        kubectl rollout status statefulset/postgres-core -n oci-core --timeout=120s
        kubectl rollout status deployment/zitadel -n oci-core --timeout=120s

        echo "3. Asserting pod readiness across workloads..."
        kubectl wait --for=condition=ready pod -l 'app in (forgejo, nextcloud, snipe-it, erpnext-frappe, frappe-hr, postgres-internal)' -n oci-internal --timeout=120s
        kubectl wait --for=condition=ready pod -l 'app in (zitadel, postgres-core)' -n oci-core --timeout=120s

        echo "4. Probing live application health endpoints..."
        FAILED_CHECKS=0

        # Postgres Internal
        echo -n "Checking postgres-internal... "
        if kubectl exec statefulset/postgres-internal -n oci-internal -- pg_isready -U oci_admin >/dev/null 2>&1; then
            echo "OK"
        else
            echo "FAILED"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
        fi

        # Postgres Core
        echo -n "Checking postgres-core... "
        if kubectl exec statefulset/postgres-core -n oci-core -- pg_isready -U oci_admin >/dev/null 2>&1; then
            echo "OK"
        else
            echo "FAILED"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
        fi

        # Forgejo
        echo -n "Checking Forgejo HTTP endpoint... "
        if kubectl exec deployment/forgejo -n oci-internal -- sh -c 'wget -q -O - http://localhost:3000/api/v1/version 2>/dev/null || curl -sf http://localhost:3000/api/v1/version 2>/dev/null' >/dev/null 2>&1; then
            echo "OK"
        else
            echo "FAILED"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
        fi

        # Nextcloud
        echo -n "Checking Nextcloud HTTP endpoint... "
        if kubectl exec deployment/nextcloud -n oci-internal -- sh -c 'curl -sf http://localhost:80/status.php 2>/dev/null || php /var/www/html/occ status 2>/dev/null' >/dev/null 2>&1; then
            echo "OK"
        else
            echo "FAILED"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
        fi

        # Snipe-IT
        echo -n "Checking Snipe-IT HTTP endpoint... "
        if kubectl exec deployment/snipe-it -n oci-internal -- sh -c 'curl -sf http://localhost:80/health 2>/dev/null || curl -sf http://localhost:80/ 2>/dev/null' >/dev/null 2>&1; then
            echo "OK"
        else
            echo "FAILED"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
        fi

        # Zitadel
        echo -n "Checking Zitadel health endpoint... "
        if kubectl exec deployment/zitadel -n oci-core -- sh -c 'curl -sf http://localhost:8080/debug/healthz 2>/dev/null || wget -q -O - http://localhost:8080/debug/healthz 2>/dev/null' >/dev/null 2>&1; then
            echo "OK"
        else
            echo "FAILED"
            FAILED_CHECKS=$((FAILED_CHECKS + 1))
        fi

        if [[ ${FAILED_CHECKS} -gt 0 ]]; then
            echo "❌ COTS health verification failed with ${FAILED_CHECKS} failing health check(s)." >&2
            exit 1
        fi

        echo "✅ All COTS applications are healthy and ready."
        ;;

    rollback)
        DEPLOYMENT="${2:-}"
        if [[ -z "${DEPLOYMENT}" ]]; then
            echo "ERROR: Please specify deployment name to roll back (e.g., forgejo, nextcloud, snipe-it, erpnext-frappe, zitadel)." >&2
            exit 1
        fi
        NAMESPACE="${3:-oci-internal}"

        echo "Rolling back deployment/${DEPLOYMENT} in namespace ${NAMESPACE}..."
        kubectl rollout undo "deployment/${DEPLOYMENT}" -n "${NAMESPACE}"
        echo "Waiting for rollback rollout status..."
        kubectl rollout status "deployment/${DEPLOYMENT}" -n "${NAMESPACE}" --timeout=180s
        echo "✅ Rollback of deployment/${DEPLOYMENT} in ${NAMESPACE} completed successfully."
        ;;

    *)
        usage
        ;;
esac
