# Direct Public Ingress Setup Guide (NGINX + Let's Encrypt + DDNS)

This guide documents the alternative method for exposing customer services (`customer.orangecatinvestments.com`) directly via router port forwarding and cert-manager ACME certificates.

---

## 1. Network & Router Prerequisites

1. **Static Ingress LoadBalancer IP:**
   * Ensure `MetalLB` is running and serving a dedicated ingress IP from its pool (e.g., `192.168.1.251`).
   * *Note:* Keep the control plane virtual IP (`192.168.1.250` managed by `kube-vip`) dedicated to Kubernetes API traffic; do not forward public web ports to the control-plane VIP.
2. **Router Port Forwarding (NAT):**
   * Log into your gateway / firewall (pfSense, UniFi, OPNsense, ASUS, etc.).
   * Configure Port Forwarding rules:
     * **HTTP:** External Port `80` (TCP) -> Target Ingress IP `192.168.1.251` Port `80`
     * **HTTPS:** External Port `443` (TCP) -> Target Ingress IP `192.168.1.251` Port `443`
3. **NAT Hairpinning / Loopback:**
   * Enable "NAT Reflection" or "Hairpin NAT" so devices inside the local network can resolve the external domain to the internal cluster without timeout.

> [!CAUTION] Security & Ingress Controller Isolation
> Forwarding WAN ports 80/443 directly to an ingress controller routes external internet traffic into the cluster. Internal services (Employee Portal, Forgejo, Nextcloud, Domain APIs, Longhorn dashboard) MUST use an isolated internal ingress class (`ingressClassName: internal`) or enforce CIDR IP whitelisting (`nginx.ingress.kubernetes.io/whitelist-source-range: "192.168.0.0/16,10.0.0.0/8,172.16.0.0/12"`). Only customer-facing public routes (`customer.orangecatinvestments.com`) should be exposed to public traffic.

---

## 2. Dynamic DNS (DDNS) Configuration

If your ISP provides a dynamic WAN IPv4 address:
1. Update `ZONE_ID`, `RECORD_NAME`, and `API_TOKEN` in [`ddns-updater.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/public-ingress/nginx-letsencrypt/ddns-updater.yaml).
2. Deploy the CronJob:
   ```bash
   kubectl apply -f k8s/ha/public-ingress/nginx-letsencrypt/ddns-updater.yaml
   ```
3. Test a manual run:
   ```bash
   kubectl -n oci-core create job --from=cronjob/cloudflare-ddns-updater test-ddns-01
   kubectl -n oci-core logs job/test-ddns-01
   ```

---

## 3. Deploying Let's Encrypt ClusterIssuers

1. Edit [`letsencrypt-issuer.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/public-ingress/nginx-letsencrypt/letsencrypt-issuer.yaml) with your admin email.
2. Apply the issuers:
   ```bash
   kubectl apply -f k8s/ha/public-ingress/nginx-letsencrypt/letsencrypt-issuer.yaml
   ```
3. Verify issuer status:
   ```bash
   kubectl get clusterissuer
   ```
   Status must be `READY: True`.

---

## 4. Deploying Public Ingress

1. Update hostnames in [`public-customer-ingress.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/public-ingress/nginx-letsencrypt/public-customer-ingress.yaml) to match your public DNS.
2. Apply the ingress:
   ```bash
   kubectl apply -f k8s/ha/public-ingress/nginx-letsencrypt/public-customer-ingress.yaml
   ```
3. Watch certificate challenge and issuance:
   ```bash
   kubectl -n oci-core get certificate customer-portal-public-tls -w
   kubectl -n oci-core describe challenge
   ```
4. Once `READY: True`, verify HTTPS with:
   ```bash
   curl -Iv https://customer.orangecatinvestments.com
   ```
