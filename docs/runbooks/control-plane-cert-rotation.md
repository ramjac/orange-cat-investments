# Control Plane & Host Certificate Lifecycle Runbook

**Audience:** Infrastructure & Systems Engineers (`emp-human-frank`)  
**Scope:** Kubernetes control plane certificates, cert-manager internal PKI, and automated trust distribution for Orange Cat Investments (OCI).

---

## 1. Overview & Threat Model

Kubernetes control planes generate internal TLS certificates during initial cluster bootstrapping for:
- API server (`kube-apiserver`)
- Kubelet node agents and client credentials
- etcd peer and client communication (embedded etcd or external datastore)
- Aggregation layer front-proxy client
- Controller-manager and scheduler client certificates

In K3s, control plane certificates are stored under `/var/lib/rancher/k3s/server/tls` and are valid for **365 days (1 year)** by default. While K3s will automatically rotate certificates upon restart if they are within 90 days of expiration, infrastructure engineers must execute scheduled annual audits and managed rotations to prevent uncoordinated restarts, certificate drift, or split-brain conditions across HA control planes.

If control plane certificates expire:
- Inter-node communication fails.
- Node status drops to `NotReady`.
- `kubectl` commands fail with `x509: certificate has expired or is not yet valid`.
- Automated CI/CD deployments and scheduled cronjobs fail.

> [!NOTE]
> Internal application mutual TLS (mTLS) is completely decoupled from the Kubernetes control plane certificates. Application mTLS is managed via `cert-manager` and the in-cluster `CARotationManager` background worker (see Section 3).

---

## 2. Annual Control Plane Certificate Renewal SOP (K3s Standard)

Infrastructure engineers must execute the annual certificate inspection and refresh cycle at least 30 days prior to expiration.

### 2.1 Inspect Certificate Expiration

Log into the K3s control plane node via SSH and check the expiration status of all control plane certificates:

```bash
ssh admin@k3s-control-plane-01 "sudo k3s certificate check"
```

The output enumerates all managed TLS certificates, their validity windows, and days remaining:
```text
CERTIFICATE                           EXPIRES
/var/lib/rancher/k3s/server/tls/client-admin.crt                  Jan 15, 2027 10:00 UTC (358 days)
/var/lib/rancher/k3s/server/tls/client-auth-proxy.crt            Jan 15, 2027 10:00 UTC (358 days)
/var/lib/rancher/k3s/server/tls/client-controller.crt            Jan 15, 2027 10:00 UTC (358 days)
/var/lib/rancher/k3s/server/tls/client-kube-apiserver.crt        Jan 15, 2027 10:00 UTC (358 days)
/var/lib/rancher/k3s/server/tls/client-kube-proxy.crt            Jan 15, 2027 10:00 UTC (358 days)
/var/lib/rancher/k3s/server/tls/client-k3s-controller.crt        Jan 15, 2027 10:00 UTC (358 days)
/var/lib/rancher/k3s/server/tls/server-ca.crt                    Jan 12, 2036 10:00 UTC (3645 days)
/var/lib/rancher/k3s/server/tls/serving-kube-apiserver.crt       Jan 15, 2027 10:00 UTC (358 days)
```

Verify that all certificates have more than 30 days remaining. If any certificate is approaching expiration, proceed to the rotation steps below.

---

### 2.2 Execute Managed Certificate Rotation

K3s provides the native `k3s certificate rotate` subcommand. When executed, previous certificates are automatically backed up to `/var/lib/rancher/k3s/server/tls-backup-<timestamp>/` before issuing fresh 365-day certificates.

1. **Stop the K3s Service:**
   ```bash
   ssh admin@k3s-control-plane-01 "sudo systemctl stop k3s"
   ```

2. **Rotate Control Plane Certificates:**
   Rotate all leaf and server certificates:
   ```bash
   ssh admin@k3s-control-plane-01 "sudo k3s certificate rotate"
   ```
   *(Optional: To rotate a specific component only, use `--service <service>`, e.g., `sudo k3s certificate rotate --service kube-apiserver`)*.

3. **Start the K3s Service:**
   ```bash
   ssh admin@k3s-control-plane-01 "sudo systemctl start k3s"
   ```

4. **Multi-Node HA Considerations:**
   In an HA multi-server topology (e.g., 3-node embedded etcd cluster), rotate server nodes **one at a time**:
   1. Stop service and rotate node 1 (`k3s-master-01`), start service, and verify etcd quorum.
   2. Repeat for node 2 (`k3s-master-02`).
   3. Repeat for node 3 (`k3s-master-03`).
   Worker / agent nodes (`k3s-agent`) automatically receive renewed client certificates upon reconnecting to the server. If an agent node loses connection, restart its agent service (`systemctl restart k3s-agent`).

---

### 2.3 Update Client Kubeconfig & Verify Cluster Health

Rotating the control plane certificates regenerates the client admin certificate in `/etc/rancher/k3s/k3s.yaml`. Workstations and CI/CD runners must update their `kubeconfig`:

1. **Export Refreshed Kubeconfig:**
   ```bash
   ssh admin@k3s-control-plane-01 "sudo cat /etc/rancher/k3s/k3s.yaml" > ~/.kube/config
   ```
   *(Ensure the `server:` URL points to the control plane IP or DNS VIP rather than `127.0.0.1`)*.

2. **Verify Cluster Health & Pod Status:**
   ```bash
   kubectl get nodes -o wide
   kubectl get pods -A
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
   The declarative manifest [`k8s/ha/lan/tls/oci-ca-bundle.yaml`](../../k8s/ha/lan/tls/oci-ca-bundle.yaml) synchronizes `secret/oci-lan-root-ca-key-pair` in namespace `cert-manager` to `ConfigMap/oci-ca-bundle` across target namespaces (`oci-core`, `oci-internal`).
   Verify sync status with:
   ```bash
   kubectl get bundle oci-ca-bundle
   ```

2. **Zero-Downtime Hot-Swapping (`pkg/mtls`):**
   Go microservices utilize [`pkg/mtls.DynamicCAPoolReloader`](../../pkg/mtls/mtls.go) which watches the mounted `ca.crt` file. When the CA rotates on disk, running gRPC and HTTP servers hot-swap the active `ClientCAs` and `RootCAs` pools on every handshake, requiring **zero pod restarts**.

3. **Automated Background Reconciler (`cmd/worker`):**
   The Go background worker runs [`CARotationManager`](../../internal/service/pki/ca_rotation.go) on an hourly ticker. When the intermediate CA approaches 30 days before expiration:
   - Issues a fresh ECDSA P-256 certificate and key pair.
   - Synchronizes `secret/oci-intermediate-ca-key-pair` via the Kubernetes API.
   - Verifies `trust-manager` bundle propagation.
   - Publishes an `ops.ca_rotated.v1` event envelope across RabbitMQ.

---

## 4. Prometheus Monitoring & Alerting Rules

Prometheus continuously evaluates certificate expiration metrics configured in [`k8s/monitoring/prometheus.yaml`](../../k8s/monitoring/prometheus.yaml):

| Alert Rule | Metric Expression | Threshold | Severity | Action |
| :--- | :--- | :--- | :--- | :--- |
| `CertManagerCertExpiringSoon` | `certmanager_certificate_expiration_timestamp_seconds - time()` | `< 30 days` | `warning` | Verify cert-manager and worker auto-rotation. |
| `BlackboxSSLCertExpiringSoon` | `probe_ssl_earliest_cert_expiry - time()` | `< 30 days` | `critical` | Execute Section 2 control plane cert rotation SOP. |

---

## 5. Emergency Recovery (Post-Expiration Recovery)

If control plane certificates have expired prior to manual rotation:
1. `kubectl` commands fail with:
   ```text
   Unable to connect to the server: x509: certificate has expired or is not yet valid
   ```
2. **Do not reinstall the cluster.** K3s retains all underlying state, persistent volume data, and SQLite/etcd database transactions.
3. Access the control plane host via SSH or out-of-band console:
   ```bash
   ssh admin@k3s-control-plane-01
   ```
4. Stop the K3s daemon:
   - Systemd: `sudo systemctl stop k3s`
   - OpenRC: `sudo rc-service k3s stop`
5. Force certificate renewal via root shell:
   ```bash
   sudo k3s certificate rotate
   ```
6. Start the K3s service:
   - Systemd: `sudo systemctl start k3s`
   - OpenRC: `sudo rc-service k3s start`
7. Update the workstation kubeconfig with the renewed client certificate:
   ```bash
   sudo cat /etc/rancher/k3s/k3s.yaml > ~/.kube/config
   ```
8. Confirm cluster connectivity is restored:
   ```bash
   kubectl get nodes
   ```
