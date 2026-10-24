# Control Plane & Host Certificate Lifecycle Runbook

**Audience:** Infrastructure & Systems Engineers (`emp-human-frank`)  
**Scope:** Kubernetes control plane certificates, cert-manager internal PKI, and automated trust distribution for Orange Cat Investments (OCI).

---

## 1. Overview & Threat Model

Kubernetes control planes generate internal TLS certificates during initial cluster bootstrapping for:
- API server (`kube-apiserver`)
- Kubelet node agents and client credentials
- Datastore peer and client communication (Dqlite / Raft / etcd)
- Aggregation layer front-proxy client
- Controller-manager and scheduler client certificates

In **MicroK8s on Ubuntu Server**, control plane certificates are stored under `/var/snap/microk8s/current/certs` and are valid for **365 days (1 year)** by default. Infrastructure engineers must execute scheduled annual audits and managed rotations to prevent uncoordinated expiration, certificate drift, or communication failures across cluster nodes.

If control plane certificates expire:
- Inter-node communication fails.
- Node status drops to `NotReady`.
- `kubectl` commands fail with `x509: certificate has expired or is not yet valid`.
- Automated CI/CD deployments and scheduled cronjobs fail.

> [!NOTE]
> Internal application mutual TLS (mTLS) is completely decoupled from the Kubernetes control plane certificates. Application mTLS is managed via `cert-manager` and the in-cluster `CARotationManager` background worker (see Section 3).

---

## 2. Annual Control Plane Certificate Renewal SOP (MicroK8s Standard)

Infrastructure engineers must execute the annual certificate inspection and refresh cycle at least 30 days prior to expiration.

### 2.1 Inspect Certificate Expiration

Log into the Ubuntu Server MicroK8s control plane node via SSH and check the expiration status of all control plane certificates:

```bash
ssh ramjac@thinker "microk8s refresh-certs --check"
```

The output enumerates all managed TLS certificates, their file paths under `/var/snap/microk8s/current/certs`, and remaining validity. Verify that all certificates have more than 30 days remaining. If any certificate is approaching expiration, proceed to the rotation steps below.

---

### 2.2 Execute Managed Certificate Rotation

MicroK8s provides the native `microk8s refresh-certs` tool. When executed, previous certificates are automatically backed up to `/var/snap/microk8s/current/certs/backup/` before issuing fresh certificates.

1. **Rotate Control Plane Certificates:**
   Rotate expiring certificates (e.g. `server.crt` and `front-proxy-client.crt`):
   ```bash
   ssh ramjac@thinker "sudo microk8s refresh-certs -e server.crt && sudo microk8s refresh-certs -e front-proxy-client.crt"
   ```
   *(Or refresh all certificates: `ssh ramjac@thinker "sudo microk8s refresh-certs"`)*.

2. **Restart MicroK8s Services:**
   Restart the services to reload the renewed certificates:
   ```bash
   ssh ramjac@thinker "sudo microk8s stop && sudo microk8s start"
   ```

3. **Multi-Node HA Considerations:**
   In an HA multi-node cluster, refresh certificates across nodes sequentially:
   1. Refresh certificates and restart services on node 1 (`thinker`).
   2. Verify cluster status: `microk8s status`.
   3. Repeat for secondary nodes.

---

### 2.3 Update Client Kubeconfig & Verify Cluster Health

Rotating the control plane certificates regenerates client administrative credentials. Workstations and CI/CD runners must update their `kubeconfig`:

1. **Export Refreshed Kubeconfig:**
   ```bash
   ssh ramjac@thinker "microk8s config" > thinker-kube.config
   ```

2. **Verify Cluster Health & Pod Status:**
   ```bash
   export KUBECONFIG="$(pwd)/thinker-kube.config"
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
           +-----------------------------------+               +-----------------------------------+
           | Namespace: oci-core               |               | Namespace: oci-internal           |
           | ConfigMap: oci-ca-bundle          |               | ConfigMap: oci-ca-bundle          |
           |   (ca.crt)                        |               |   (ca.crt)                        |
           +-----------------------------------+               +-----------------------------------+
                             |                                                   |
                             +-------------------------+-------------------------+
                                                       |
                                                       v
                                     +-----------------------------------+
                                     | Go Microservices (pkg/mtls)       |
                                     | DynamicCAPoolReloader             |
                                     | - Hot-reloads ca.crt via inotify  |
                                     | - Zero pod restarts on rotation   |
                                     +-----------------------------------+
```

1. **Automated Intermediate CA Rotation:**
   The Go background worker runs [`CARotationManager`](../../internal/service/pki/ca_rotation.go) on an hourly ticker. When the intermediate CA approaches 30 days before expiration:
   - Issues a fresh ECDSA P-256 certificate and key pair.
   - Synchronizes `secret/oci-intermediate-ca-key-pair` via the Kubernetes API.
   - Verifies `trust-manager` bundle propagation.
   - Emits an `ops.ca_rotated.v1` domain event.

2. **Dynamic In-Cluster Hot-Reloading (`pkg/mtls`):**
   Go gRPC services mount `ConfigMap/oci-ca-bundle` to `/etc/ssl/certs/oci`. The [`DynamicCAPoolReloader`](../../pkg/mtls/mtls.go) detects filesystem timestamp updates on `ca.crt` and dynamically reloads the server and client certificate verification pools in memory without dropping in-flight connections or restarting pods.

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
2. **Do not reinstall the cluster.** MicroK8s retains all underlying state, persistent volume data, and database transactions.
3. Access the control plane host via SSH or console:
   ```bash
   ssh ramjac@thinker
   ```
4. Force certificate renewal via root shell:
   ```bash
   sudo microk8s refresh-certs
   ```
5. Restart the MicroK8s services:
   ```bash
   sudo microk8s stop && sudo microk8s start
   ```
6. Update the workstation kubeconfig with the renewed client certificate:
   ```bash
   microk8s config > thinker-kube.config
   ```
7. Confirm cluster connectivity is restored:
   ```bash
   microk8s kubectl get nodes
   ```

