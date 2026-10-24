# COTS Application Lifecycle Management & Upstream Security Patching Runbook

**Audience:** Platform Security & K8s Infrastructure Lead (`emp-human-frank`), Site Reliability Engineers  
**Scope:** Standard Operating Procedures (SOPs) for upstream security patching, minor updates, schema migrations, and major version upgrades across Commercial-Off-The-Shelf (COTS) business platforms in `oci-internal` (Forgejo, Nextcloud, Snipe-IT, ERPNext/Frappe HR) and core identity systems in `oci-core` (ZITADEL).

---

## 1. Overview & Maintenance Principles

Orange Cat Investments enforces strict domain isolation: proprietary Go microservices run in `oci-core`, while open-source and commercial business systems run in `oci-internal`. Relational persistence is isolated to `postgres-internal` in `oci-internal` and `postgres-core` in `oci-core`.

### Guiding Principles:
1. **Never Skip Major Versions:** Applications like Nextcloud and Frappe strictly require sequential hops (e.g., 28 → 29 → 30). Skipping versions corrupts database schemas.
2. **Pre-Upgrade Snapshot Invariant:** No deployment manifest or image tag may be updated in production without a verified, successful database snapshot.
3. **Staging Gate in `oci-dev`:** Upstream container patches must first be applied and validated in `oci-dev` before modifying production manifests.
4. **Zero Unencrypted Secrets in Manifests:** Database credentials, `APP_KEY`, and `ZITADEL_MASTERKEY` reside strictly in Kubernetes Secrets (`app-secrets` and `oci-core-secrets`). Never hardcode or overwrite encryption keys.

---

## 2. Standardized Maintenance & Pre-Flight Procedures

### 2.1 Schedule Maintenance Window & Announce
1. Post a maintenance advisory to team channels in Nextcloud Talk / Chat at least 24 hours prior to planned downtime:
   ```text
   [OCI Infrastructure Notice] Scheduled Maintenance Window
   Scope: COTS security patch upgrade (Nextcloud, Forgejo, Snipe-IT, ERPNext)
   Target Window: Sunday 02:00 - 04:00 UTC
   Impact: Temporary read-only or 503 maintenance banner on internal endpoints.
   ```

2. *(Optional)* Apply Ingress Maintenance Banner:
   To route external and staff traffic to a graceful maintenance notice, apply the maintenance configuration snippet:
   ```bash
   kubectl annotate ingress oci-internal-ingress -n oci-internal \
     nginx.ingress.kubernetes.io/server-snippet="return 503 'OCI Systems Under Scheduled Maintenance. Back shortly.';" \
     --overwrite
   ```

### 2.2 Trigger Automated Pre-Upgrade Database Snapshots
Before applying any image or schema changes, manually trigger the database backup CronJobs and verify successful completion:

1. **Trigger `postgres-internal` Snapshot:**
   ```bash
   SNAPSHOT_ID="pre-patch-$(date +%s)"
   kubectl create job --from=cronjob/postgres-internal-backup-cronjob "${SNAPSHOT_ID}-internal" -n oci-internal
   kubectl wait --for=condition=complete job/"${SNAPSHOT_ID}-internal" -n oci-internal --timeout=180s
   ```

2. **Trigger `postgres-core` Snapshot:**
   ```bash
   kubectl create job --from=cronjob/postgres-core-backup-cronjob "${SNAPSHOT_ID}-core" -n oci-core
   kubectl wait --for=condition=complete job/"${SNAPSHOT_ID}-core" -n oci-core --timeout=180s
   ```

3. **Verify Backup Files on Persistent Storage:**
   ```bash
   kubectl exec -n oci-internal statefulset/postgres-internal -- ls -lh /backups/
   kubectl exec -n oci-core statefulset/postgres-core -- ls -lh /backups/
   ```

Alternatively, use the unified maintenance helper script:
```bash
./scripts/cots-maintenance.sh snapshot
```

---

## 3. Application-Specific Upgrade Runbooks

### 3.1 Forgejo (`k8s/oci-internal/forgejo.yaml`)

* **Role:** Internal Git source control, CI/CD Actions, and OAuth2 identity provider for employee portal.
* **Database:** `forgejo` database on `postgres-internal-service:5432`.
* **State Directory:** `/data` (rootless home: `/var/lib/gitea`).

#### Upgrade Procedure:
1. **Update Manifest:**
   Edit [`k8s/oci-internal/forgejo.yaml`](../../k8s/oci-internal/forgejo.yaml) and update `spec.template.spec.containers[0].image` to the target target version (e.g., `codeberg.org/forgejo/forgejo:7.0-rootless` → `8.0-rootless` or patch release).
2. **Apply Update:**
   ```bash
   kubectl apply -f k8s/oci-internal/forgejo.yaml
   kubectl rollout status deployment/forgejo -n oci-internal --timeout=180s
   ```
3. **Verify Migrations & OAuth2 Providers:**
   Check container logs to verify schema migrations executed without error:
   ```bash
   kubectl logs -n oci-internal deployment/forgejo -c forgejo --tail=100
   ```
4. **Update Forgejo Actions Runner (`k8s/oci-internal/forgejo-runner.yaml`):**
   If a major version requires runner updates, bump `gitea/act_runner:latest` and verify runner registration:
   ```bash
   kubectl rollout restart deployment/forgejo-runner -n oci-internal
   kubectl logs -n oci-internal deployment/forgejo-runner --tail=50
   ```
5. **Verify OAuth2 Flow:**
   Ensure the Employee BFF can successfully query Forgejo user endpoints (`GET http://forgejo-service.oci-internal.svc:3000/api/v1/user`).

---

### 3.2 Nextcloud (`k8s/oci-internal/nextcloud.yaml`)

* **Role:** Workplace productivity, document collaboration, chat, and executive file storage.
* **Database:** `nextcloud` database on `postgres-internal-service:5432`.
* **Data Volume:** `nextcloud-data-pvc` mounted at `/var/www/html`.

> [!CRITICAL]
> **Sequential Upgrades Only:** Nextcloud does NOT support skipping major versions (e.g. updating 28 → 30 directly is strictly forbidden). You must upgrade 28 → 29, run migrations, and then upgrade 29 → 30.

#### Upgrade Procedure:
1. **Auditing App Compatibility:**
   Check installed third-party apps for compatibility with target version:
   ```bash
   kubectl exec -n oci-internal deployment/nextcloud -c nextcloud -- \
     su -s /bin/sh www-data -c "php occ app:list"
   ```
2. **Update Manifest Image:**
   Edit [`k8s/oci-internal/nextcloud.yaml`](../../k8s/oci-internal/nextcloud.yaml) and bump `image: nextcloud:28-apache` to the next sequential major release (e.g., `nextcloud:29-apache`).
3. **Trigger Rolling Update:**
   ```bash
   kubectl apply -f k8s/oci-internal/nextcloud.yaml
   kubectl rollout status deployment/nextcloud -n oci-internal --timeout=300s
   ```
4. **Execute Post-Upgrade Maintenance & Optimization:**
   After the container starts, execute schema updates and index creation:
   ```bash
   kubectl exec -n oci-internal deployment/nextcloud -c nextcloud -- \
     su -s /bin/sh www-data -c "php occ upgrade"

   kubectl exec -n oci-internal deployment/nextcloud -c nextcloud -- \
     su -s /bin/sh www-data -c "php occ db:add-missing-indices"

   kubectl exec -n oci-internal deployment/nextcloud -c nextcloud -- \
     su -s /bin/sh www-data -c "php occ db:add-missing-columns"
   ```
5. **Disable Maintenance Mode:**
   If Nextcloud remains in maintenance mode:
   ```bash
   kubectl exec -n oci-internal deployment/nextcloud -c nextcloud -- \
     su -s /bin/sh www-data -c "php occ maintenance:mode --off"
   ```

---

### 3.3 Snipe-IT (`k8s/oci-internal/snipe-it.yaml`)

* **Role:** Asset management for non-networked physical hardware, monitors, laptops, and field kits.
* **Database:** `snipeit` database on `postgres-internal-service:5432`.
* **Critical Secret:** `SNIPEIT_APP_KEY` stored in `secret/app-secrets`.

> [!WARNING]
> **Preserve `APP_KEY`:** Snipe-IT encrypts custom fields, user details, and passwords using `APP_KEY`. Changing or losing this key renders encrypted records permanently unreadable.

#### Upgrade Procedure:
1. **Update Manifest Image:**
   Edit [`k8s/oci-internal/snipe-it.yaml`](../../k8s/oci-internal/snipe-it.yaml) and update `image: snipe/snipe-it:<version>`.
2. **Apply Update:**
   ```bash
   kubectl apply -f k8s/oci-internal/snipe-it.yaml
   kubectl rollout status deployment/snipe-it -n oci-internal --timeout=180s
   ```
3. **Execute Laravel Database Migrations:**
   ```bash
   kubectl exec -n oci-internal deployment/snipe-it -- php artisan migrate --force
   ```
4. **Purge Route & Config Caches:**
   ```bash
   kubectl exec -n oci-internal deployment/snipe-it -- php artisan config:clear
   kubectl exec -n oci-internal deployment/snipe-it -- php artisan route:clear
   kubectl exec -n oci-internal deployment/snipe-it -- php artisan cache:clear
   ```
5. **Verify QR Code & Asset Lookup:**
   Confirm asset endpoints respond with HTTP 200:
   ```bash
   kubectl exec -n oci-internal deployment/snipe-it -- php artisan snipeit:system-health
   ```

---

### 3.4 ERPNext & Frappe HR (`k8s/oci-internal/erpnext-frappe.yaml`, `frappe-hr.yaml`)

* **Role:** Corporate accounting, double-entry general ledger, legal payroll, and leave management.
* **Database:** `erpnext` database on `postgres-internal-service:5432`.
* **Cache & Queues:** `valkey-service.oci-core.svc.cluster.local:6379`.

#### Upgrade Procedure:
1. **Synchronize Multi-App Version Tags:**
   Frappe HR requires compatible versions of the underlying Frappe framework and ERPNext. Verify version compatibility matrix before patching:
   - Edit [`k8s/oci-internal/erpnext-frappe.yaml`](../../k8s/oci-internal/erpnext-frappe.yaml)
   - Edit [`k8s/oci-internal/frappe-hr.yaml`](../../k8s/oci-internal/frappe-hr.yaml)
2. **Deploy Container Updates:**
   ```bash
   kubectl apply -f k8s/oci-internal/erpnext-frappe.yaml
   kubectl apply -f k8s/oci-internal/frappe-hr.yaml
   kubectl rollout status deployment/erpnext-frappe -n oci-internal --timeout=240s
   kubectl rollout status deployment/frappe-hr -n oci-internal --timeout=240s
   ```
3. **Execute Bench Migrations & DocType Sync:**
   ```bash
   kubectl exec -n oci-internal deployment/erpnext-frappe -c erpnext -- \
     bench --site all migrate

   kubectl exec -n oci-internal deployment/erpnext-frappe -c erpnext -- \
     bench --site all clear-cache
   ```
4. **Verify Ledger & Timesheet Integrity:**
   Ensure background Celery workers and scheduler processes are active:
   ```bash
   kubectl exec -n oci-internal deployment/erpnext-frappe -c erpnext -- \
     bench --site all doctor
   ```

---

### 3.5 ZITADEL Customer Identity Provider (`k8s/oci-core/zitadel.yaml`)

* **Role:** Customer IAM (OIDC/OAuth2) for retail investors accessing `web/customer-portal`.
* **Database:** `oci` database on `postgres-core-service.oci-core.svc.cluster.local:5432`.
* **Master Key:** `ZITADEL_MASTERKEY` in `secret/oci-core-secrets`.

#### Upgrade Procedure:
1. **Inspect Release Notes & Breaking Schema Changes:**
   Review upstream ZITADEL release notes for breaking database changes or mandatory migration flags.
2. **Update Manifest Image:**
   Edit [`k8s/oci-core/zitadel.yaml`](../../k8s/oci-core/zitadel.yaml) and update `image: ghcr.io/zitadel/zitadel:<version>`.
   *(Notice: ZITADEL is configured with `start-from-init`, which automatically runs schema setup on startup).*
3. **Trigger Rolling Deployment:**
   ```bash
   kubectl apply -f k8s/oci-core/zitadel.yaml
   kubectl rollout status deployment/zitadel -n oci-core --timeout=240s
   ```
4. **Verify OIDC Discovery & Health Probes:**
   ```bash
   kubectl exec -n oci-core deployment/zitadel -- curl -sf http://localhost:8080/debug/healthz
   kubectl exec -n oci-core deployment/zitadel -- curl -sf http://localhost:8080/.well-known/openid-configuration | jq .issuer
   ```

---

## 4. Post-Upgrade Verification & Smoke Testing

Once all target components are updated, execute the platform verification pipeline:

1. **Check All Cluster Pods:**
   ```bash
   kubectl get pods -n oci-internal
   kubectl get pods -n oci-core
   ```
   All pods must report `Running` status with `0` restarts.

2. **Execute Frank Infrastructure Persona Smoke Tests:**
   Run Frank's automated verification routine via the platform simulation engine:
   ```bash
   go run ./simulation/cmd/simulator -mode live -persona emp-human-frank
   ```

3. **Execute Full Multi-Persona Verification (1 Iteration):**
   ```bash
   go run ./simulation/cmd/simulator -mode live -iterations 1 -delay 50
   ```

4. **Disable Ingress Maintenance Banner:**
   ```bash
   kubectl annotate ingress oci-internal-ingress -n oci-internal \
     nginx.ingress.kubernetes.io/server-snippet-
   ```

---

## 5. Emergency Rollback Procedures

If an upgrade fails, causes database corruption, or fails health checks:

### 5.1 Deployment Reversion (Single Command Rollback)
If the database schema has not been destructively altered:
```bash
# Roll back specific COTS deployment to previous revision
kubectl rollout undo deployment/<app-name> -n oci-internal
# For example:
kubectl rollout undo deployment/nextcloud -n oci-internal
kubectl rollout undo deployment/forgejo -n oci-internal
kubectl rollout undo deployment/snipe-it -n oci-internal
kubectl rollout undo deployment/erpnext-frappe -n oci-internal

# Or for Zitadel:
kubectl rollout undo deployment/zitadel -n oci-core
```
Verify the rollback completes:
```bash
kubectl rollout status deployment/<app-name> -n oci-internal --timeout=120s
```

### 5.2 Full Database Snapshot Restore (Post-Schema Corruption)
If an upstream migration failed mid-way or corrupted tables:
1. Scale down application pods to terminate active connections:
   ```bash
   kubectl scale deployment/<app-name> --replicas=0 -n oci-internal
   ```
2. Locate the pre-upgrade backup file in `/backups`:
   ```bash
   kubectl exec -it -n oci-internal statefulset/postgres-internal -- ls -lh /backups/
   ```
3. Restore database snapshot:
   ```bash
   BACKUP_FILE="/backups/oci_internal_backup_<TIMESTAMP>.sql"
   kubectl exec -it -n oci-internal statefulset/postgres-internal -- \
     sh -c "psql -U oci_admin -f ${BACKUP_FILE}"
   ```
4. Revert deployment image manifest to the prior known-good tag.
5. Scale application pods back up:
   ```bash
   kubectl scale deployment/<app-name> --replicas=1 -n oci-internal
   kubectl rollout status deployment/<app-name> -n oci-internal --timeout=180s
   ```
6. Verify operational status and emit an incident review ticket.
