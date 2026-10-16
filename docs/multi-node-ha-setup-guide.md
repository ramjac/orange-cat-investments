# Multi-Node High Availability Setup & Operations Runbook

This runbook provides complete, step-by-step instructions for deploying, verifying, and operating Orange Cat Investments (OCI) on a resilient 3-node physical home or office lab cluster.

---

## 1. Hardware Architecture & Bill of Materials (BOM)

To achieve true high availability and tolerate single-node hardware failure, the recommended topology utilizes **three identical small-form-factor PCs** (e.g., Intel NUCs, Minisforum, Beelink, or Lenovo ThinkCentre Tiny):

| Component | Minimum Specification | Recommended Specification |
| :--- | :--- | :--- |
| **Nodes** | 3x Physical Machines | 3x Mini PCs (Intel Core i5 / AMD Ryzen 5+) |
| **CPU** | 4 Cores per node | 6-8 Cores per node |
| **RAM** | 16 GB DDR4/DDR5 per node | 32 GB DDR4/DDR5 per node |
| **Storage** | 256 GB NVMe SSD | 1 TB PCIe 4.0 NVMe SSD (M.2) |
| **Network** | 1 GbE RJ45 Ethernet | 2.5 GbE Ethernet (Dedicated Switch) |
| **Power Protection**| Wall Outlet | Mini UPS (e.g. APC Back-UPS 650VA) |

> [!TIP]
> Connect all 3 nodes to a dedicated unmanaged Gigabit or 2.5G network switch rather than relying on Wi-Fi. etcd latency requires stable, jitter-free ping times (< 10ms round-trip).

---

## 2. Network Plan & Subnet Allocations

Configure your router / DHCP server with static DHCP reservations or static IPs:

```text
Local Subnet: 192.168.1.0/24
Gateway:      192.168.1.1
DNS Server:   192.168.1.250 (Cluster VIP) or 192.168.1.5 (Pi-hole)

Node 1 (Primary Server):      192.168.1.201  (Hostname: oci-node1)
Node 2 (Secondary Server):    192.168.1.202  (Hostname: oci-node2)
Node 3 (Secondary Server):    192.168.1.203  (Hostname: oci-node3)

Virtual IP (VIP - kube-vip):  192.168.1.250  (Shared HA Address)
```

---

## 3. Host Operating System Preparation

Install **Ubuntu Server 24.04 / 26.04 LTS** or **Debian 12** on all 3 nodes:
1. Set hostnames: `sudo hostnamectl set-hostname oci-node1` (and `oci-node2`, `oci-node3`).
2. Add your SSH public key to `~/.ssh/authorized_keys` for non-interactive Ansible access.
3. Verify passwordless `sudo` is configured for your deployment user.

---

## 4. Automated Cluster Provisioning via Ansible

### Step 4.1: Prepare Inventory
Copy the example inventory template and update IP addresses:
```bash
cp ansible/inventory/ha-cluster.ini.example ansible/inventory/ha-cluster.ini
```
Verify the contents in [`ansible/inventory/ha-cluster.ini`](file:///home/ramjac/orange-cat-investments/ansible/inventory/ha-cluster.ini):
* Ensure `node1`, `node2`, and `node3` match your physical machine IPs.
* Ensure `k3s_vip=192.168.1.250` is an unused IP on your local subnet.
* Update `k3s_cluster_token` with a strong cryptographic passphrase.

### Step 4.2: Execute Ansible Provisioning Playbook
Run the multi-node HA cluster setup:
```bash
ansible-playbook -i ansible/inventory/ha-cluster.ini ansible/playbooks/k3s_ha_cluster.yml
```

**What the automation performs:**
1. Installs Longhorn prerequisites (`open-iscsi`, `nfs-common`, `cryptsetup`) across all 3 nodes.
2. Generates the `kube-vip` static pod manifest to bind virtual IP `192.168.1.250` via ARP.
3. Initializes Node 1 with `--cluster-init` embedded etcd.
4. Concurrently joins Node 2 and Node 3 to the embedded etcd quorum.
5. Extracts the administrative `kubeconfig` to `~/.kube/config-oci-ha` pointed directly at `https://192.168.1.250:6443`.

### Step 4.3: Verify Cluster Status
Set your local environment to use the new cluster kubeconfig:
```bash
export KUBECONFIG=~/.kube/config-oci-ha
kubectl get nodes -o wide
```
Expected output:
```text
NAME         STATUS   ROLES                       AGE   VERSION        INTERNAL-IP
oci-node1    Ready    control-plane,etcd,master   5m    v1.36.4+k3s1   192.168.1.201
oci-node2    Ready    control-plane,etcd,master   3m    v1.36.4+k3s1   192.168.1.202
oci-node3    Ready    control-plane,etcd,master   2m    v1.36.4+k3s1   192.168.1.203
```

Check etcd quorum:
```bash
kubectl -n kube-system get pods -l app=kube-vip
```

---

## 5. Distributed Replicated Storage Deployment (Longhorn)

Once the 3 nodes are active, deploy Longhorn to establish cross-node replicated block storage.

### Step 5.1: Deploy Longhorn Storage Engine
Execute the automated deploy script:
```bash
./k8s/ha/storage/deploy-longhorn.sh
```
This applies:
* StorageClasses in [`longhorn-storageclasses.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/storage/longhorn-storageclasses.yaml):
  * `longhorn`: Default StorageClass with 3-node block replication.
  * `longhorn-crypto`: Encrypted storage class utilizing LUKS / dm-crypt.
* Automated snapshots and backup schedules in [`longhorn-backup-recurring.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/storage/longhorn-backup-recurring.yaml).
* Longhorn UI Ingress in [`longhorn-ui-ingress.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/storage/longhorn-ui-ingress.yaml).

### Step 5.2: Verify Storage Health
```bash
kubectl -n longhorn-system get pods
kubectl get storageclass
```
Ensure `longhorn` is marked `(default)`.

---

## 6. Local Network (LAN) Employee Portal & Intranet TLS

### Step 6.1: Intranet DNS Setup
Configure your router (pfSense/UniFi) or Pi-hole to resolve `*.oci.local` to `192.168.1.250`.
Follow instructions in [`k8s/ha/lan/dns/dns-setup-guide.md`](file:///home/ramjac/orange-cat-investments/k8s/ha/lan/dns/dns-setup-guide.md).

### Step 6.2: Issue Intranet TLS Certificates
1. Generate the local Root CA:
   ```bash
   ./k8s/ha/lan/tls/generate-lan-root-ca.sh
   ```
2. Apply the cert-manager `ClusterIssuer`:
   ```bash
   kubectl apply -f k8s/ha/lan/tls/intranet-cluster-issuer.yaml
   ```
3. Apply the HA Intranet Ingress:
   ```bash
   kubectl apply -f k8s/ha/lan/tls/intranet-ingress-ha.yaml
   ```
4. Verify certificates are issued:
   ```bash
   kubectl -n oci-core get certificate
   kubectl -n oci-internal get certificate
   ```
5. Install `oci-lan-root-ca.crt` on employee laptops and devices following [`client-ca-trust-installation.md`](file:///home/ramjac/orange-cat-investments/k8s/ha/lan/tls/client-ca-trust-installation.md).

---

## 7. Exposing Customer Portal to Public Internet

### Recommended Method: Cloudflare Tunnel (Zero Port Forwarding)
1. Install `cloudflared` and create tunnel per [`setup-cloudflare-tunnel.md`](file:///home/ramjac/orange-cat-investments/k8s/ha/public-ingress/cloudflare/setup-cloudflare-tunnel.md).
2. Create the secret:
   ```bash
   kubectl create secret generic cloudflare-tunnel-credentials \
     --namespace oci-core \
     --from-file=credentials.json=$HOME/.cloudflared/<tunnel-id>.json
   ```
3. Deploy the tunnel pods:
   ```bash
   kubectl apply -f k8s/ha/public-ingress/cloudflare/cloudflare-tunnel.yaml
   ```
4. Verify both tunnel replicas are running on separate physical nodes:
   ```bash
   kubectl -n oci-core get pods -l app=cloudflare-tunnel -o wide
   ```

---

## 8. Chaos & High-Availability Validation Tests

Once physical hardware is operational, conduct the following validation tests to certify the cluster before placing customer funds or production workloads on it:

### Test 1: Control Plane VIP Failover
1. Ping the cluster VIP: `ping 192.168.1.250`.
2. Locate the node currently hosting the VIP:
   ```bash
   kubectl -n kube-system logs daemonset/kube-vip-ds | grep "promoted to master"
   ```
3. Physically disconnect the Ethernet cable or power cord from that node.
4. **Validation:**
   * Ping should drop at most 1 packet (< 1 second interruption).
   * `kubectl get nodes` continues to respond without timing out.

### Test 2: Database Pod & Replicated Storage Failover
1. Check which node hosts `postgres-core-0`:
   ```bash
   kubectl -n oci-core get pod postgres-core-0 -o wide
   ```
2. Hard reboot that host (`sudo reboot -f`).
3. Observe Kubernetes rescheduling:
   ```bash
   kubectl -n oci-core get pod postgres-core-0 -w
   ```
4. **Validation:**
   * Within 40-50 seconds, `postgres-core-0` transitions to `Terminating` on the dead node and schedules onto one of the two surviving nodes.
   * Longhorn attaches the volume to the new node and transitions to `Running`.
   * Log entries confirm database recovery:
     ```bash
     kubectl -n oci-core logs postgres-core-0 --tail=20
     ```
     `database system was not properly shut down; automatic recovery in progress... database system is ready to accept connections`.

### Test 3: Zero Quorum Loss
1. Verify surviving 2 nodes maintain full etcd quorum:
   ```bash
   kubectl get pods -A
   ```
   All API operations succeed normally.
2. Power the dead node back on.
3. Observe node rejoining the cluster as `Ready` and Longhorn re-syncing the third storage replica automatically.
