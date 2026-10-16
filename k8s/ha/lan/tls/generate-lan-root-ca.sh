#!/usr/bin/env bash
# ==============================================================================
# Generate Orange Cat Investments Intranet Root Certificate Authority (CA)
# ==============================================================================
set -euo pipefail

OUT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CA_KEY="${OUT_DIR}/oci-lan-root-ca.key"
CA_CRT="${OUT_DIR}/oci-lan-root-ca.crt"
CA_CONF="${OUT_DIR}/ca-openssl.cnf"
DAYS_VALID=3650 # 10 Years

echo "==> Generating OpenSSL configuration..."
cat <<EOF > "${CA_CONF}"
[ req ]
default_bits        = 4096
default_md          = sha256
distinguished_name  = req_distinguished_name
x509_extensions     = v3_ca
prompt              = no

[ req_distinguished_name ]
C                   = US
ST                  = California
L                   = San Francisco
O                   = Orange Cat Investments
OU                  = Security & Infrastructure
CN                  = Orange Cat Investments Intranet Root CA

[ v3_ca ]
subjectKeyIdentifier   = hash
authorityKeyIdentifier = keyid:always,issuer
basicConstraints       = critical, CA:true, pathlen:1
keyUsage               = critical, digitalSignature, cRLSign, keyCertSign
EOF

echo "==> Generating 4096-bit RSA Root CA private key..."
openssl genrsa -out "${CA_KEY}" 4096
chmod 400 "${CA_KEY}"

echo "==> Generating self-signed Root CA certificate (valid for ${DAYS_VALID} days)..."
openssl req -new -x509 \
  -days "${DAYS_VALID}" \
  -key "${CA_KEY}" \
  -out "${CA_CRT}" \
  -config "${CA_CONF}"

echo "==> Verifying generated Root CA certificate..."
openssl x509 -in "${CA_CRT}" -noout -text | grep -E "(Issuer|Subject|Not After|CA:TRUE)"

echo "==> Creating or updating cert-manager Secret in Kubernetes..."
if command -v kubectl >/dev/null 2>&1; then
  kubectl create namespace cert-manager --dry-run=client -o yaml | kubectl apply -f -
  kubectl create secret tls oci-lan-root-ca-key-pair \
    --namespace cert-manager \
    --cert="${CA_CRT}" \
    --key="${CA_KEY}" \
    --dry-run=client -o yaml | kubectl apply -f -
  echo "==> Secret 'oci-lan-root-ca-key-pair' successfully provisioned in 'cert-manager' namespace."
else
  echo "==> Note: kubectl not detected. To install this secret on your cluster, run:"
  echo "    kubectl create secret tls oci-lan-root-ca-key-pair --namespace cert-manager --cert=${CA_CRT} --key=${CA_KEY}"
fi

echo "==> Root CA Certificate saved to: ${CA_CRT}"
echo "    Distribute this file to employee workstations and devices per 'client-ca-trust-installation.md'."
