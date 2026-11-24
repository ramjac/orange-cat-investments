#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# Forgejo Generic Package Registry Firmware Publisher (Phase 8 / 7.32 & 7.34)
# ==============================================================================
# Packages edge camera (.bin) and Pebble watch (.pbw) firmware artifacts,
# computes SHA-256 digests, signs with Cosign (or ECDSA OpenSSL fallback),
# uploads to Forgejo Package Registry, and optionally registers the release
# with Employee BFF (/api/v1/facilities/firmware/releases).
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

FORGEJO_URL="${FORGEJO_URL:-http://forgejo-service.oci-internal.svc.cluster.local:3000}"
FORGEJO_OWNER="${FORGEJO_OWNER:-oci}"
PACKAGE_NAME="${PACKAGE_NAME:-edge-firmware}"
VERSION="${VERSION:-2.2.0}"
DEVICE_TYPE="${DEVICE_TYPE:-edge_camera}"
ARTIFACT_NAME="${ARTIFACT_NAME:-}"
INPUT_FILE="${INPUT_FILE:-}"
FORGEJO_TOKEN="${FORGEJO_TOKEN:-}"
COSIGN_KEY_FILE="${COSIGN_KEY_FILE:-}"
BFF_URL="${BFF_URL:-}"
OUT_DIR="${OUT_DIR:-${REPO_ROOT}/dist/firmware}"

usage() {
    cat <<EOF
Usage: $(basename "$0") [OPTIONS]

Options:
  --forgejo-url URL      Forgejo base URL (default: http://forgejo-service.oci-internal.svc.cluster.local:3000)
  --owner OWNER          Forgejo package owner/org (default: oci)
  --package NAME         Package name (default: edge-firmware)
  --version VER          Semantic firmware version (default: 2.2.0)
  --device-type TYPE     Device type: edge_camera|pebble_watch|gateway (default: edge_camera)
  --file PATH            Existing binary file to publish (optional; generates deterministic bundle if omitted)
  --token TOKEN          Forgejo access token with package write scope
  --cosign-key PATH      Cosign/ECDSA private key PEM path for blob signing
  --bff-url URL          Optional Employee BFF URL to register firmware release metadata
  -h, --help             Show help
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --forgejo-url) FORGEJO_URL="$2"; shift 2 ;;
        --owner) FORGEJO_OWNER="$2"; shift 2 ;;
        --package) PACKAGE_NAME="$2"; shift 2 ;;
        --version) VERSION="$2"; shift 2 ;;
        --device-type) DEVICE_TYPE="$2"; shift 2 ;;
        --file) INPUT_FILE="$2"; shift 2 ;;
        --token) FORGEJO_TOKEN="$2"; shift 2 ;;
        --cosign-key) COSIGN_KEY_FILE="$2"; shift 2 ;;
        --bff-url) BFF_URL="$2"; shift 2 ;;
        -h|--help) usage; exit 0 ;;
        *) echo "ERROR: Unknown argument: $1" >&2; usage; exit 1 ;;
    esac
done

if [[ -z "${ARTIFACT_NAME}" ]]; then
    if [[ "${DEVICE_TYPE}" == "pebble_watch" ]]; then
        ARTIFACT_NAME="pebble-oci-${VERSION}.pbw"
    else
        ARTIFACT_NAME="${DEVICE_TYPE}-${VERSION}.bin"
    fi
fi

mkdir -p "${OUT_DIR}"
ARTIFACT_PATH="${OUT_DIR}/${ARTIFACT_NAME}"

if [[ -n "${INPUT_FILE}" ]]; then
    cp "${INPUT_FILE}" "${ARTIFACT_PATH}"
else
    # Build deterministic firmware bundle manifest + payload for reproducible builds
    python3 - "${ARTIFACT_PATH}" "${DEVICE_TYPE}" "${VERSION}" <<'PY'
import hashlib, json, sys

out_path, dev_type, version = sys.argv[1], sys.argv[2], sys.argv[3]
header = json.dumps({
    "vendor": "Orange Cat Investments",
    "device_type": dev_type,
    "version": version,
    "format": "oci-ota-v2",
}, sort_keys=True).encode("utf-8")
digest = hashlib.sha256(header).hexdigest().encode("utf-8")
with open(out_path, "wb") as f:
    f.write(b"OCI-FW-V2\n" + header + b"\nSHA256:" + digest + b"\n")
PY
fi

SHA256_SUM="$(sha256sum "${ARTIFACT_PATH}" | awk '{print $1}')"
COSIGN_SIG=""

if [[ -n "${COSIGN_KEY_FILE}" && -f "${COSIGN_KEY_FILE}" ]]; then
    if command -v cosign &>/dev/null; then
        COSIGN_SIG="$(COSIGN_PASSWORD="${COSIGN_PASSWORD:-}" cosign sign-blob --yes --key "${COSIGN_KEY_FILE}" --b64=true "${ARTIFACT_PATH}" 2>/dev/null | tail -n1 | tr -d '\r\n' || true)"
    fi
    if [[ -z "${COSIGN_SIG}" ]] && command -v openssl &>/dev/null; then
        COSIGN_SIG="$(openssl dgst -sha256 -sign "${COSIGN_KEY_FILE}" "${ARTIFACT_PATH}" | base64 | tr -d '\r\n')"
    fi
fi

PACKAGE_URL="${FORGEJO_URL%/}/api/packages/${FORGEJO_OWNER}/generic/${PACKAGE_NAME}/${VERSION}/${ARTIFACT_NAME}"
echo "Firmware Artifact : ${ARTIFACT_PATH}"
echo "SHA-256 Checksum  : ${SHA256_SUM}"
echo "Package Target URL: ${PACKAGE_URL}"

if [[ -n "${FORGEJO_TOKEN}" ]]; then
    HTTP_CODE=$(curl -sS -o /tmp/forgejo_pkg_upload.out -w "%{http_code}" \
        -X PUT \
        -H "Authorization: token ${FORGEJO_TOKEN}" \
        --upload-file "${ARTIFACT_PATH}" \
        "${PACKAGE_URL}")
    if [[ "${HTTP_CODE}" != "201" && "${HTTP_CODE}" != "200" && "${HTTP_CODE}" != "409" ]]; then
        echo "ERROR: Forgejo package upload failed (HTTP ${HTTP_CODE}):" >&2
        cat /tmp/forgejo_pkg_upload.out >&2 || true
        exit 1
    fi
    echo "Uploaded firmware package to Forgejo Package Registry (HTTP ${HTTP_CODE})."
else
    echo "FORGEJO_TOKEN not set; skipping remote upload (dry package build complete)."
fi

if [[ -n "${BFF_URL}" ]]; then
    PAYLOAD=$(python3 - "${DEVICE_TYPE}" "${VERSION}" "${PACKAGE_URL}" "${SHA256_SUM}" "${COSIGN_SIG}" <<'PY'
import json, sys
print(json.dumps({
    "target_hardware": sys.argv[1],
    "version": sys.argv[2],
    "binary_url": sys.argv[3],
    "checksum_sha256": sys.argv[4],
    "cosign_signature": sys.argv[5],
    "status": "published",
}))
PY
)
    curl -sf -X POST "${BFF_URL%/}/api/v1/facilities/firmware/releases" \
        -H "Content-Type: application/json" \
        -d "${PAYLOAD}" >/dev/null
    echo "Registered firmware release ${VERSION} with ${BFF_URL}."
fi
