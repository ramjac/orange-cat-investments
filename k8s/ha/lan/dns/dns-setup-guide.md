# Local Network Intranet DNS Configuration Guide

This guide describes how to configure Local Network (LAN) split-horizon DNS for the Orange Cat Investments 3-node HA cluster.

## Cluster Virtual IP (VIP) Target
* **VIP Address:** `192.168.1.250` (managed by `kube-vip`)
* **Intranet Domain:** `*.oci.local`

---

## Option 1: Pi-hole (Recommended for Home Labs)

1. **Via Web Admin Console:**
   * Navigate to **Local DNS** -> **DNS Records**.
   * Add domain: `employee.oci.local` -> IP: `192.168.1.250`.
   * Repeat for `api.oci.local`, `forgejo.oci.local`, `nextcloud.oci.local`, `erpnext.oci.local`, `auth.oci.local`, `longhorn.oci.local`.

2. **Via Command Line / Automation:**
   * Copy [`pihole-custom.list`](file:///home/ramjac/orange-cat-investments/k8s/ha/lan/dns/pihole-custom.list) to `/etc/pihole/custom.list` on your Pi-hole host:
     ```bash
     cat pihole-custom.list >> /etc/pihole/custom.list
     pihole restartdns
     ```

3. **Wildcard Domain via dnsmasq in Pi-hole:**
   * Create `/etc/dnsmasq.d/02-oci.conf` on your Pi-hole:
     ```text
     address=/oci.local/192.168.1.250
     ```
   * Run `pihole restartdns`. Any subdomain `*.oci.local` will now resolve to `192.168.1.250`.

---

## Option 2: Router Split-Horizon DNS (pfSense / OPNsense)

1. Open pfSense/OPNsense Web GUI -> **Services** -> **DNS Resolver** (Unbound).
2. Scroll down to **Host Overrides** and click **Add**:
   * **Host:** `employee`
   * **Domain:** `oci.local`
   * **IP Address:** `192.168.1.250`
   * **Description:** `OCI HA Cluster Employee Portal VIP`
3. Under **Domain Overrides** (for wildcards):
   * **Domain:** `oci.local`
   * **IP Address:** `192.168.1.250`
4. Click **Save** and **Apply Changes**.

---

## Option 3: UniFi Gateway (UDM / UCG / UXG)

1. In UniFi Network Controller, go to **Settings** -> **Routing** -> **DNS**.
2. Click **Create Entry**:
   * **Domain Name:** `*.oci.local` (or individual hosts like `employee.oci.local`)
   * **IP Address:** `192.168.1.250`
3. Click **Add** to apply across all LAN clients via DHCP.

---

## Option 4: Local Workstation `/etc/hosts` (Quick Testing)

If centralized router DNS is not yet configured, add the following to `/etc/hosts` (Linux/macOS) or `C:\Windows\System32\drivers\etc\hosts`:

```text
192.168.1.250  employee.oci.local employee-bff.oci.local api.oci.local auth.oci.local pebble-proxy.oci.local forgejo.oci.local nextcloud.oci.local erpnext.oci.local hr.oci.local snipeit.oci.local longhorn.oci.local
```

Verify with:
```bash
ping -c 2 employee.oci.local
```
