# Employee Device Trust Store Installation Guide

To eliminate browser TLS warnings (e.g., "Your connection is not private" / `NET::ERR_CERT_AUTHORITY_INVALID`) when navigating to intranet portals (`https://employee.oci.local`, `https://forgejo.oci.local`, etc.), install the **Orange Cat Investments Intranet Root CA** certificate on employee client devices.

* **Certificate File:** [`oci-lan-root-ca.crt`](file:///home/ramjac/orange-cat-investments/k8s/ha/lan/tls/oci-lan-root-ca.crt) (generated via [`generate-lan-root-ca.sh`](file:///home/ramjac/orange-cat-investments/k8s/ha/lan/tls/generate-lan-root-ca.sh))

---

## 1. Ubuntu / Debian Linux

1. Copy the Root CA certificate to the system trust directory:
   ```bash
   sudo cp oci-lan-root-ca.crt /usr/local/share/ca-certificates/oci-lan-root-ca.crt
   ```
2. Update the system CA trust store:
   ```bash
   sudo update-ca-certificates
   ```
3. Verify certificate is trusted:
   ```bash
   openssl verify -CAfile /etc/ssl/certs/ca-certificates.crt oci-lan-root-ca.crt
   ```

---

## 2. Fedora / Red Hat Enterprise Linux (RHEL) / CentOS

1. Copy the Root CA certificate to anchors directory:
   ```bash
   sudo cp oci-lan-root-ca.crt /etc/pki/ca-trust/source/anchors/
   ```
2. Update CA trust:
   ```bash
   sudo update-ca-trust extract
   ```

---

## 3. macOS

1. Add the certificate to the System Keychain:
   ```bash
   sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain oci-lan-root-ca.crt
   ```
2. Alternatively via GUI:
   * Double-click `oci-lan-root-ca.crt` to open **Keychain Access**.
   * Drag the certificate into **System** -> **Certificates**.
   * Double-click the certificate, expand **Trust**, and set **When using this certificate** to **Always Trust**.

---

## 4. Microsoft Windows 11 / 10

1. Open PowerShell or Command Prompt as **Administrator**:
   ```powershell
   certutil -addstore -f "ROOT" oci-lan-root-ca.crt
   ```
2. Alternatively via GUI:
   * Press `Win + R`, type `certmgr.msc`, and press Enter.
   * Right-click **Trusted Root Certification Authorities** -> **All Tasks** -> **Import**.
   * Browse and select `oci-lan-root-ca.crt`, placing it into the **Trusted Root Certification Authorities** store.

---

## 5. Apple iOS / iPadOS

1. Send `oci-lan-root-ca.crt` to the device via AirDrop, email, or download from an internal link.
2. Tap **Settings** -> **Profile Downloaded** -> tap **Install** and enter passcode.
3. Enable full SSL trust:
   * Go to **Settings** -> **General** -> **About** -> **Certificate Trust Settings**.
   * Under **Enable full trust for root certificates**, toggle on **Orange Cat Investments Intranet Root CA**.

---

## 6. Android (11+)

1. Go to **Settings** -> **Security** (or **Biometrics and security**).
2. Tap **More security settings** -> **Encryption & credentials** -> **Install a certificate** -> **CA certificate**.
3. Confirm the security warning and select `oci-lan-root-ca.crt`.
4. Name the certificate `OCI Intranet Root CA` and save.

---

## 7. Mozilla Firefox (All Platforms)

Firefox maintains its own independent certificate trust store by default.
1. In Firefox, navigate to `about:preferences#privacy`.
2. Scroll down to **Security** -> **Certificates** -> click **View Certificates...**.
3. Under the **Authorities** tab, click **Import...**.
4. Select `oci-lan-root-ca.crt`.
5. Check the box: `[x] Trust this CA to identify websites` and click **OK**.
