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
        echo "Enabling maintenance banner on oci-internal ingress..."
        if kubectl get ingress oci-internal-ingress -n oci-internal &>/dev/null; then
            kubectl annotate ingress oci-internal-ingress -n oci-internal \
                nginx.ingress.kubernetes.io/server-snippet="return 503 'OCI Systems Under Scheduled Maintenance. Back shortly.';" \
                --overwrite
            echo "✅ Maintenance banner applied to oci-internal-ingress."
        else
            echo "⚠️  oci-internal-ingress not found in oci-internal."
        fi
        ;;

    disable-maintenance)
        echo "Removing maintenance banner from oci-internal ingress..."
        if kubectl get ingress oci-internal-ingress -n oci-internal &>/dev/null; then
            kubectl annotate ingress oci-internal-ingress -n oci-internal \
                nginx.ingress.kubernetes.io/server-snippet- || true
            echo "✅ Maintenance banner removed from oci-internal-ingress."
        else
            echo "⚠️  oci-internal-ingress not found in oci-internal."
        fi
        ;;

    verify)
        echo "=========================================================="
        echo "  Verifying COTS Workload Health                          "
        echo "=========================================================="

        echo "Checking oci-internal pods..."
        kubectl get pods -n oci-internal -l 'app in (forgejo, nextcloud, snipe-it, erpnext-frappe, frappe-hr, postgres-internal)' || true

        echo ""
        echo "Checking oci-core identity pods..."
        kubectl get pods -n oci-core -l 'app in (zitadel, postgres-core)' || true

        echo ""
        echo "Running simulated persona verification for Frank Operations..."
        if command -v go >/dev/null 2>&1 || [[ -x "/home/ramjac/.local/bin/go" ]]; then
            export PATH="/home/ramjac/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-amd64/bin:$PATH"
            go run "${REPO_ROOT}/simulation/cmd/simulator" -mode mock -persona emp-human-frank
        fi
        echo "✅ Health verification completed."
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
