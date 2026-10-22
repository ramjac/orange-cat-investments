# Control Plane & Host Certificate Lifecycle Runbook

**Audience:** Infrastructure & Systems Engineers (`emp-human-frank`)  
**Scope:** Kubernetes control plane certificates, cert-manager internal PKI, and automated trust distribution for Orange Cat Investments (OCI).

---

## 1. Overview & Threat Model

Kubernetes control planes (MicroK8s and K3s) generate internal TLS certificates during initial cluster bootstrapping for:
- API server (`kube-apiserver`)
- Kubelet node agents and client credentials
- etcd peer and client communication
- Aggregation layer front-proxy client

By default, these Kubernetes control plane certificates are valid for **365 days (1 year)**. If they expire, node-to-master communication halts, `kubectl` commands fail with `x509: certificate has expired or is not yet valid`, and the cluster enters an unrecoverable split-brain state unless emergency recovery is performed.

Internal application mutual TLS (mTLS) is decoupled from the control plane and managed via `cert-manager` and the in-cluster `CARotationManager` background worker.

---

## 2. Annual Control Plane Certificate Renewal SOP

### 2.1 MicroK8s Clusters (e.g. `thinker`)

Infrastructure engineers must execute the annual certificate inspection and refresh cycle at least 30 days prior to expiration:

1. **Inspect Certificate Expiration:**
   Log into the host node via SSH and check the expiration of all control plane certificates:
   ```bash
   ssh ramjac@thinker "microk8s refresh-certs --check"
   ```
   Inspect output for days remaining on `server.crt`, `ca.crt`, and `front-proxy-client.crt`.

2. **Execute In-Place Certificate Refresh:**
   MicroK8s provides a certificate renewal command with `-e/--cert` to target specific expired or expiring certificates before generating fresh 365-day certificates (backed up to `/var/snap/microk8s/current/certs/backup/`):
   ```bash
   ssh ramjac@thinker "microk8s refresh-certs -e server.crt && microk8s refresh-certs -e front-proxy-client.crt"
   ```

3. **Restart MicroK8s Control Plane Services:**
   After rotating certificates, restart the MicroK8s snap services to load the new keypairs:
   ```bash
   ssh ramjac@thinker "microk8s stop && microk8s start"
   ```

4. **Verify Control Plane Health & Node Registration:**
   ```bash
   ssh ramjac@thinker "microk8s status --wait-ready"
   ssh ramjac@thinker "microk8s kubectl get nodes -o wide"
   ssh ramjac@thinker "microk8s kubectl get pods -A"
   ```

5. **Update Workstation Kubeconfig:**
   If the cluster CA or API server certificate changed, export the refreshed client config:
   ```bash
   ssh ramjac@thinker "microk8s config" > ~/.kube/config
   kubectl get nodes
   ```

---

### 2.2 K3s Clusters (Multi-Node HA Deployments)

For K3s environments managed via systemd:

1. **Check Certificate Expiration:**
   ```bash
   k3s certificate check
   ```

2. **Rotate Control Plane Certificates:**
   Stop the server, rotate certificates, and restart:
   ```bash
   sudo systemctl stop k3s
   sudo k3s certificate rotate
   sudo systemctl start k3s
   ```

3. **Verify Cluster Readiness:**
   ```bash
   k3s kubectl get nodes
   ```

---

## 3. Internal Application PKI & Automated Trust Distribution

OCI enforces zero-static certificates and dynamic trust bundle propagation across all application microservices:

```
+-----------------------------------------------------------------------------------+
| cert-manager Namespace                                                           |
|                                                                                   |
|  [ ClusterIssuer: oci-lan-issuer ] ---> [ Secret: oci-lan-root-ca-key-pair ]      |
|                                                      |                            |
|                                            (trust-manager Bundle)                 |
+------------------------------------------------------|----------------------------+
                                                       |
                             +-------------------------+-------------------------+
                             |                                                   |
                             v                                                   v
                +--------------------------+                        +--------------------------+
                | oci-core Namespace       |                        | oci-internal Namespace   |
                | ConfigMap: oci-ca-bundle |                        | ConfigMap: oci-ca-bundle |
                +--------------------------+                        +--------------------------+
                             |                                                   |
                             v                                                   v
                +--------------------------+                        +--------------------------+
                | Go Microservices         |                        | Nextcloud & COTS         |
                | pkg/mtls                 |                        | Internal CA trust store  |
                | DynamicCAPoolReloader    |                        |                          |
                +--------------------------+                        +--------------------------+
```

1. **trust-manager Bundle Synchronization:**
   The declarative manifest [`k8s/ha/lan/tls/oci-ca-bundle.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/lan/tls/oci-ca-bundle.yaml) synchronizes `secret/oci-lan-root-ca-key-pair` in namespace `cert-manager` to `ConfigMap/oci-ca-bundle` across target namespaces (`oci-core`, `oci-internal`).
   Verify sync status with:
   ```bash
   kubectl get bundle oci-ca-bundle
   ```

2. **Zero-Downtime Hot-Swapping (`pkg/mtls`):**
   Go microservices utilize [`pkg/mtls.DynamicCAPoolReloader`](file:///home/ramjac/orange-cat-investments/pkg/mtls/mtls.go) which watches the mounted `ca.crt` file. When the CA rotates on disk, running gRPC and HTTP servers hot-swap the active `ClientCAs` and `RootCAs` pools on every handshake, requiring **zero pod restarts**.

3. **Automated Background Reconciler (`cmd/worker`):**
   The Go background worker runs [`CARotationManager`](file:///home/ramjac/orange-cat-investments/internal/service/pki/ca_rotation.go) on an hourly ticker. When the intermediate CA approaches 30 days before expiration:
   - Issues a fresh ECDSA P-256 certificate and key pair.
   - Synchronizes `secret/oci-intermediate-ca-key-pair` via the Kubernetes API.
   - Verifies `trust-manager` bundle propagation.
   - Publishes an `ops.ca_rotated.v1` event envelope across RabbitMQ.

---

## 4. Prometheus Monitoring & Alerting Rules

Prometheus continuously evaluates certificate expiration metrics configured in [`k8s/monitoring/prometheus.yaml`](file:///home/ramjac/orange-cat-investments/k8s/monitoring/prometheus.yaml):

| Alert Rule | Metric Expression | Threshold | Severity | Action |
| :--- | :--- | :--- | :--- | :--- |
| `CertManagerCertExpiringSoon` | `certmanager_certificate_expiration_timestamp_seconds - time()` | `< 30 days` | `warning` | Verify cert-manager and worker auto-rotation. |
| `BlackboxSSLCertExpiringSoon` | `probe_ssl_earliest_cert_expiry - time()` | `< 30 days` | `critical` | Execute Section 2 control plane cert rotation SOP. |

---

## 5. Emergency Recovery (Post-Expiration Recovery)

If control plane certificates have already expired:
1. `kubectl` will return: `Unable to connect to the server: x509: certificate has expired`.
2. Do **not** reinstall the cluster.
3. On MicroK8s, force certificate renewal via root shell:
   ```bash
   sudo microk8s refresh-certs -e server.crt
   sudo microk8s refresh-certs -e front-proxy-client.crt
   sudo snap restart microk8s
   ```
4. On K3s:
   ```bash
   sudo systemctl stop k3s
   sudo k3s certificate rotate
   sudo systemctl start k3s
   ```
5. Confirm cluster connectivity is restored:
   ```bash
   microk8s kubectl get nodes
   ```
