# Cloudflare Tunnel Setup Guide for Customer Portal

Cloudflare Tunnel (`cloudflared`) is the **strongly recommended** ingress solution for exposing the Customer Portal (`web/customer-portal`) and Customer BFF (`cmd/customer-bff`) to retail investors over the public internet.

---

## 1. Architectural Advantages

* **Zero Open Inbound Ports:** No port forwarding on your office or home gateway (ports 80 and 443 remain completely closed).
* **Carrier-Grade NAT (CGNAT) Resilient:** Works even when your ISP does not assign a routable public IPv4 address (common on residential fiber, 5G home internet, or Starlink).
* **DDoS & WAF Protection:** Cloudflare absorbs SYN floods, HTTP floods, and volumetric Layer 7 attacks before traffic ever touches your home servers.
* **Automated Edge TLS:** Cloudflare handles global edge TLS certificates automatically without requiring local ACME challenges.
* **Origin Shielding:** Your physical residential/office IP address is never revealed in public DNS A/AAAA records.

---

## 2. Setup Prerequisites

1. A domain name active on Cloudflare DNS (e.g. `orangecatinvestments.com`).
2. Cloudflare Zero Trust account (free tier supports up to 50 users and unlimited tunnels).
3. The `cloudflared` CLI installed on your management workstation:
   ```bash
   # Debian / Ubuntu
   sudo apt-get install cloudflared
   # macOS
   brew install cloudflared
   ```

---

## 3. Provisioning the Tunnel

### Step 3.1: Authenticate with Cloudflare
```bash
cloudflared tunnel login
```
This opens your browser. Select your domain zone to authorize `cloudflared`.

### Step 3.2: Create the Tunnel
```bash
cloudflared tunnel create oci-customer-tunnel
```
Output will display the **Tunnel ID** (UUID) and path to the credentials JSON file:
```text
Created tunnel oci-customer-tunnel with id 018f0000-0000-7000-8000-000000000001
Path: /home/user/.cloudflared/018f0000-0000-7000-8000-000000000001.json
```

### Step 3.3: Configure DNS Records in Cloudflare
Route your public customer hostnames through the tunnel:
```bash
cloudflared tunnel route dns oci-customer-tunnel customer.orangecatinvestments.com
cloudflared tunnel route dns oci-customer-tunnel customer-api.orangecatinvestments.com
cloudflared tunnel route dns oci-customer-tunnel auth.orangecatinvestments.com
```

---

## 4. Deploying to the Kubernetes Cluster

### Step 4.1: Create the Secret
Create the Kubernetes secret from the generated JSON credentials file:
```bash
kubectl create secret generic cloudflare-tunnel-credentials \
  --namespace oci-core \
  --from-file=credentials.json=$HOME/.cloudflared/<tunnel-id>.json
```

### Step 4.2: Customize Domain Names
Edit [`cloudflare-tunnel.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/public-ingress/cloudflare/cloudflare-tunnel.yaml) to replace `orangecatinvestments.com` with your actual domain name and tunnel name.

### Step 4.3: Apply the Manifest
```bash
kubectl apply -f k8s/ha/public-ingress/cloudflare/cloudflare-tunnel.yaml
```

### Step 4.4: Verify Tunnel Health
```bash
kubectl -n oci-core get pods -l app=cloudflare-tunnel
kubectl -n oci-core logs -l app=cloudflare-tunnel --tail=50
```
Look for `Connection established with Cloudflare edge`.

Navigate to `https://customer.orangecatinvestments.com` from an external device (cellular network) to verify end-to-end connectivity.
