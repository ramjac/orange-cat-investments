# ADR 0002: Multi-Node High Availability & Resilient SMB Cluster Topology

* **Status:** Accepted
* **Date:** 2026-10-15
* **Author:** Principal Enterprise Architect
* **Context:** Architectural blueprint for evolving Orange Cat Investments from a single-node deployment into a resilient 3-node physical home/office server cluster.

---

## 1. Context and Problem Statement

Orange Cat Investments operates revenue-generating automated trading engines alongside real-time feline behavioral telemetry and internal business ERP platforms. In a single-node host deployment (e.g., a single workstation running MicroK8s or Docker):
1. **Single Point of Hardware Failure:** A motherboard failure, power supply interruption, kernel panic, or disk degradation results in complete operational downtime.
2. **Database Non-Portability:** Local hostpath volumes lock PostgreSQL database state (`postgres-core` and `postgres-internal`) to a single physical disk, preventing automated container failover across nodes.
3. **Internal Security Friction:** Intranet browser access to `*.oci.local` endpoints triggers severe red HTTPS certificate warnings (`NET::ERR_CERT_AUTHORITY_INVALID`), complicating day-to-day employee operations.
4. **Public Exposure Vulnerabilities:** Exposing customer-facing investment terminals to the public internet via direct home router port forwarding exposes residential/office WAN IPs to DDoS attacks, port scans, and ISP Carrier-Grade NAT (CGNAT) restrictions.

We require a resilient, enterprise-grade architecture capable of running on 2 to 3 low-power physical computers (such as mini PCs or Intel NUCs) with automated failover, distributed replicated block storage, clean intranet PKI, and zero-trust public ingress.

---

## 2. Decision Drivers

* **Cost & Power Efficiency:** Low operational overhead suitable for home/SMB offices (mini PCs consuming < 35W per node).
* **High Availability & Quorum:** Zero control-plane downtime during single-node reboots or hardware faults.
* **Storage Resilience:** Synchronous block replication across physical machines so stateful services (PostgreSQL, Nextcloud) can survive the sudden death of any physical host.
* **Zero Browser Friction:** Valid, trusted TLS certificates for internal staff without paying for public domain certificates on internal names.
* **Perimeter Defense:** Customer access must be shielded behind carrier-grade DDoS mitigation without opening inbound firewall ports.

---

## 3. Considered Options

### 3.1 Kubernetes Clustering Engine
* **Option A: MicroK8s Clustering (Dqlite):** Simple to add nodes, but Dqlite has historically shown higher edge-case fragility during abrupt power loss compared to Raft-backed etcd.
* **Option B: Full Kubernetes (kubeadm):** Requires dedicated master nodes separate from worker nodes, ballooning hardware requirements to 5+ nodes for true HA.
* **Option C: K3s with Embedded etcd (Selected):** Lightweight, certified Kubernetes distribution. Embedded etcd eliminates external etcd management overhead while supporting 3-node hyperconverged control-plane/worker topologies.

### 3.2 Distributed Replicated Storage Engine
* **Option A: Rook-Ceph:** Highly mature and enterprise-tested, but memory and CPU footprint (Ceph OSDs, MONs, MGRs) requires 4GB+ RAM per node solely for storage overhead.
* **Option B: NFS Server / Shared NAS:** Introduces a centralized single point of failure (the NAS) and NFS file locking issues with PostgreSQL write-ahead logs (`WAL`).
* **Option C: Longhorn (Selected):** CNCF incubated distributed block storage. Minimal CPU/RAM overhead (~500MB per node), synchronous block replication over iSCSI, native Kubernetes CSI driver, encrypted volume support (`dm-crypt`), and automated multi-replica snapshot/backup capabilities.

### 3.3 Virtual IP (VIP) Management for Control Plane & LAN Services
* **Option A: External Hardware Load Balancer (F5, Kemp):** Unjustified hardware expenditure and maintenance overhead for SMB.
* **Option B: HAProxy + Keepalived VMs:** Requires separate infrastructure outside the Kubernetes cluster.
* **Option C: kube-vip (Selected):** Kubernetes-native static pod and DaemonSet utilizing Gratuitous ARP (GARP) broadcasts. Provides sub-second IP failover for control-plane API (:6443) and LAN Service Ingress without requiring BGP routers.

### 3.4 Public Customer Ingress Architecture
* **Option A: Direct Router Port Forwarding (80/443) + Let's Encrypt:** Exposes local WAN IP; blocked if ISP uses CGNAT; vulnerable to bandwidth-exhaustion DDoS.
* **Option B: Reverse Proxy VPS WireGuard Tunnel:** Requires leasing and maintaining a cloud VPS.
* **Option C: Cloudflare Tunnel (cloudflared) (Selected):** Zero inbound open router ports; outbound HTTPS connections to Cloudflare edge; automated DDoS filtering; edge SSL termination; immune to CGNAT.

---

## 4. Decision Outcome

We adopt the **3-Node Hyperconverged Resilient Topology**:
1. **Control Plane & Clustering:** 3 physical nodes running K3s with embedded etcd (`--cluster-init`). All 3 nodes act as schedulable control-plane servers.
2. **Control Plane & Ingress VIP:** `kube-vip` manages virtual IP `192.168.1.250` across the nodes.
3. **Storage Tier:** Longhorn v1.7+ deployed across all 3 nodes, providing StorageClass `longhorn` (3 replicas) for PostgreSQL and Nextcloud volumes.
4. **Intranet PKI & DNS:** Internal Root CA (`oci-lan-root-ca`) managed via cert-manager `ClusterIssuer`, paired with split-horizon DNS (Pi-hole / router) resolving `*.oci.local` to the VIP.
5. **Customer Ingress:** High-availability `cloudflared` deployment (2 replicas) routing public traffic from `customer.orangecatinvestments.com` directly to `oci-customer-portal` and `oci-customer-bff` with zero open firewall ports.

---

## 5. Architectural Specifications

### 5.1 Quorum & Etcd Math

In a Raft-based etcd cluster:
$$\text{Quorum} = \left\lfloor \frac{N}{2} \right\rfloor + 1$$

| Node Count ($N$) | Quorum Needed | Tolerated Node Failures | Split-Brain Risk |
| :--- | :--- | :--- | :--- |
| **1 Node** | 1 | 0 | None (Total Loss) |
| **2 Nodes** | 2 | 0 | Severe (Network partition causes quorum loss on both sides) |
| **3 Nodes (Selected)** | **2** | **1** | **Resilient (1 node can die or reboot with zero disruption)** |
| **4 Nodes** | 3 | 1 | Same tolerance as 3 nodes, higher latency overhead |
| **5 Nodes** | 3 | 2 | Ideal for large data centers; overkill for SMB office |

> [!IMPORTANT]
> A 2-node cluster does NOT provide high availability because losing 1 node drops available votes to 1/2, losing quorum and freezing etcd into read-only mode. A **3-node minimum** is mandatory for automated failover.

### 5.2 Storage Failover Architecture

```text
 Physical Node 1                 Physical Node 2                 Physical Node 3
+-----------------------------+ +-----------------------------+ +-----------------------------+
| [Active Postgres Pod]       | |                             | |                             |
|          |                  | |                             | |                             |
|  Longhorn Volume Engine     | |                             | |                             |
|    |           |            | |                             | |                             |
|    v           +------------+-+------------+                | |                             |
| [Replica 1]                 | | [Replica 2] |                | | [Replica 3]                 |
| (Local NVMe)                | | (Remote NVMe via iSCSI)     | | (Remote NVMe via iSCSI)     |
+-----------------------------+ +-----------------------------+ +-----------------------------+
                                       |
                   [Simulate Node 1 Hardware Failure / Power Loss]
                                       |
                                       v
 Physical Node 1 (DOWN)          Physical Node 2 (ACTIVE)        Physical Node 3
+-----------------------------+ +-----------------------------+ +-----------------------------+
|          OFFLINE            | | [Rescheduled Postgres Pod]  | |                             |
|                             | |          |                  | |                             |
|                             | |  Longhorn Volume Engine     | |                             |
|                             | |    |           |            | |                             |
|                             | |    v           +------------+-+------------+                |
|                             | | [Replica 2]                 | | [Replica 3]                 |
+-----------------------------+ +-----------------------------+ +-----------------------------+
```

When Node 1 fails:
1. Kubernetes node controller marks Node 1 `NotReady` after 40 seconds.
2. The StatefulSet pod (`postgres-core-0`) is evicted and scheduled onto Node 2.
3. Longhorn detaches the volume from the dead node and attaches the volume engine to Node 2 using existing Replicas 2 and 3.
4. PostgreSQL performs standard WAL recovery and resumes serving transactions in < 45 seconds total.
5. When Node 1 boots back up, Longhorn automatically re-syncs Replica 1 in the background.

---

## 6. Consequences & Operational Impact

### Positive:
* **Zero Downtime Maintenance:** Nodes can be drained, updated, rebooted, and rejoined sequentially with zero interruption to trading algorithms or staff portals.
* **Data Durability:** Three independent copies of all database writes exist across distinct physical hardware.
* **Complete Security Posture:** Eliminates browser certificate warnings across employee devices and shields public portals behind Cloudflare edge security.

### Negative / Trade-offs:
* **Hardware Footprint:** Requires 3 physical machines (minimum 8GB RAM, 4-core CPU, 256GB NVMe each).
* **Storage Write Amplification:** 3x replication consumes 3x raw disk space on local SSDs and introduces a minor network latency penalty on disk writes (sub-millisecond over Gigabit/2.5G LAN).
