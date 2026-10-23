# Runbook: Cluster Load Testing & High-Concurrency Performance Verification

This runbook documents the procedures, execution guidelines, and host system tuning required to execute high-concurrency load tests against an Orange Cat Investments (OCI) Kubernetes deployment.

---

## 1. Architectural Principles & Rules

1. **Strictly External Execution:**
   * In accordance with [AGENTS.md](AGENTS.md) Section 3, the simulation driver ([`simulation/cmd/simulator`](simulation/cmd/simulator)) must **NEVER be run directly on the test server host** (`thinker`).
   * Running the load generator on the target server consumes host CPU, RAM, and scheduler capacity, invalidating cluster telemetry.
   * Tests must be dispatched from an external client workstation across the local network, generating authentic ingress network traffic into the cluster.

2. **Zero-Configuration Dynamic Ingress Dialing (`OCI_INGRESS_IP`):**
   * The Go simulation client implements a custom `net.Dialer` in [`simulation/client.go`](simulation/client.go) that intercepts requests to `*.oci.local` and routes them directly to `OCI_INGRESS_IP:80` (NGINX / MicroK8s Ingress).
   * **Do not edit `/etc/hosts` or configure local DNS resolvers.** Passing `OCI_INGRESS_IP` automatically handles ingress routing.

3. **External Mock Service Resolution:**
   * Mock bank and broker services running in the `external-sim` namespace are reached via `-broker-url` and `-bank-url` or dynamically resolved through [`simulation/config.go`](file:///home/ramjac/orange-cat-investments/simulation/config.go) (`broker-api.sim.oci.local`, `plaid-api.sim.oci.local`).

---

## 2. Host Operating System Tuning (Ubuntu Server & MicroK8s)

During high-concurrency testing (e.g. 32+ workers sending ~500 req/sec), thousands of ephemeral TCP connections are established. Default Linux kernel network parameters can quickly saturate connection tracking tables (`nf_conntrack`), leading to dropped packets and ARP resolution failures ("no route to host").

To ensure resilience, configure the following kernel sysctl settings on the Ubuntu Server host:

### 2.1 Sysctl Network Tuning (`/etc/sysctl.d/99-microk8s-loadtest.conf`)

Create `/etc/sysctl.d/99-microk8s-loadtest.conf` on the host:

```ini
# Prevent connection tracking table exhaustion
net.netfilter.nf_conntrack_max = 262144
net.nf_conntrack_max = 262144

# Expand ephemeral port range & enable rapid socket recycling
net.ipv4.ip_local_port_range = 10240 65535
net.ipv4.tcp_tw_reuse = 1
net.ipv4.tcp_fin_timeout = 15

# Increase connection backlog queues
net.core.somaxconn = 32768
net.core.netdev_max_backlog = 16384
net.ipv4.tcp_max_syn_backlog = 16384

# Expand open file descriptors and inotify limits
fs.file-max = 2097152
fs.inotify.max_user_watches = 524288
fs.inotify.max_user_instances = 8192
```

Apply immediately:
```bash
sudo sysctl --system
```

### 2.2 MicroK8s Service Persistence
Ensure MicroK8s snap services are enabled and running across reboots on Ubuntu Server:
```bash
sudo systemctl enable --now snap.microk8s.daemon-containerd
```

---

## 3. Executing a Load Test

The repository includes an automated concurrency runner: [`scripts/run-load-test.sh`](../../scripts/run-load-test.sh).

### 3.1 Syntax
```bash
./scripts/run-load-test.sh [TARGET_HOST] [CONCURRENCY] [DURATION_SECONDS]
```

* `TARGET_HOST`: IP or hostname of the Ingress controller (default: `192.168.1.246`).
* `CONCURRENCY`: Number of parallel simulator worker instances (default: `32`).
* `DURATION_SECONDS`: Total test execution time in seconds (default: `600`, 10 minutes).

### 3.2 Quick Baseline Run (5 Minutes, 16 Workers)
```bash
./scripts/run-load-test.sh 192.168.1.246 16 300
```

### 3.3 Heavy Stress Run (10 Minutes, 32 Workers)
```bash
./scripts/run-load-test.sh 192.168.1.246 32 600
```

---

## 4. Monitoring & Telemetry During Execution

While the load test runs in the background, monitor both the client workstation and the target server:

### 4.1 On Target Server (`thinker`)
```bash
# Monitor CPU & thread distribution
htop

# Monitor active network connections and socket states
ss -s

# Monitor MicroK8s pod health and resource usage
microk8s kubectl top nodes
microk8s kubectl top pods -A
kubectl top nodes
kubectl top pods -A
```

### 4.2 On Client Workstation
```bash
# Watch total log volume generated
tail -f /tmp/oci_load_test/worker_1.log
```

---

## 5. Post-Test Analysis & Latency Metrics

Once the test completes, worker logs reside in `/tmp/oci_load_test/worker_*.log`.

Run the automated Python analysis script to calculate percentile latencies (P50, P90, P95, P99), throughput rates, and endpoint breakdowns:

```bash
python3 -c "
import glob, json, datetime
from collections import defaultdict, Counter

files = sorted(glob.glob('/tmp/oci_load_test/worker_*.log'))
action_latencies = defaultdict(list)
domain_latencies = defaultdict(list)

domain_map = {
    'arthur': 'Customer BFF (Passive Retail Invest)',
    'chloe': 'Customer BFF (Active Day Trading)',
    'alice': 'Employee BFF (HR & Sagas)',
    'bob': 'Facilities API (IoT & Devices)',
    'carol': 'Core Invest API (Execution)',
    'david': 'Core Invest API (Market Data)',
    'elena': 'Workforce API (Veterinary Care)',
    'frank': 'Ops API & Wearables',
    'garfield': 'Facilities API (Habitats)',
    'barneby': 'Pebble Proxy (Alerts)',
    'rick': 'Employee BFF (Onboarding)',
}

total = 0
for f in files:
    prev_time = None
    with open(f) as fp:
        for line in fp:
            if not line.startswith('{'): continue
            try: data = json.loads(line)
            except: continue
            t_str = data.get('time')
            if not t_str: continue
            dt = datetime.datetime.fromisoformat(t_str)
            action = data.get('action')
            if action and prev_time is not None:
                delta_ms = (dt - prev_time).total_seconds() * 1000.0 - 20.0
                if 0 <= delta_ms < 5000:
                    action_latencies[action].append(delta_ms)
                    total += 1
                    p = action.split('_')[0].lower()
                    domain_latencies[domain_map.get(p, 'Other')].append(delta_ms)
            prev_time = dt

def stats(vals):
    s = sorted(vals); n = len(s)
    return f'P50: {s[n//2]:5.2f}ms | Avg: {sum(s)/n:5.2f}ms | P95: {s[int(n*0.95)]:5.2f}ms | P99: {s[int(n*0.99)]:5.2f}ms'

print(f'Total Dispatched Actions: {total:,}\n')
for dom in sorted(domain_latencies.keys()):
    print(f'{dom:35s}: {stats(domain_latencies[dom])}')
"
```

---

## 6. Understanding Response Codes & Concurrency Behavior

* **200 / 201 OK:** Expected response for valid, authenticated read and write operations.
* **401 Unauthorized:** Occurs on endpoints requiring authentication when simulation threads test unauthenticated or expired session behavior.
* **409 Conflict / Constraint Violation:** Expected during multi-worker concurrent runs when multiple parallel threads attempt to submit identical timesheets for the same persona on the same date. This verifies that PostgreSQL row-level locks and unique constraints are functioning correctly.
* **404 Not Found:** Occurs for endpoints deliberately targeting mocked external third-party systems or non-API static paths.
