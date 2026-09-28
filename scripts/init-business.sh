#!/usr/bin/env bash
set -euo pipefail

# OCI Platform Genericization & Small Business Blueprint Generator (`init-business.sh`)
# Interactive / automated CLI tool to bootstrap white-labeled small business platforms and custom Go microservices.

echo "=========================================================================="
echo "  OCI Small Business Platform Generator & Microservice Scaffolder"
echo "=========================================================================="

BUSINESS_NAME="${1:-"Acme Global Enterprises"}"
PRIMARY_COLOR="${2:-"#2563eb"}"
BASE_DOMAIN="${3:-"acme.local"}"

echo "Configuring white-labeled platform for: ${BUSINESS_NAME}"
echo "Primary Brand Color: ${PRIMARY_COLOR}"
echo "Base Domain: ${BASE_DOMAIN}"

CONFIG_FILE="config/branding.yaml"

if [ -f "${CONFIG_FILE}" ]; then
    echo "Updating ${CONFIG_FILE} with custom business parameters..."
    sed -i "s/Legal Name:.*/Legal Name: \"${BUSINESS_NAME} LLC\"/g" "${CONFIG_FILE}" 2>/dev/null || true
    sed -i "s/trading_name:.*/trading_name: \"${BUSINESS_NAME}\"/g" "${CONFIG_FILE}" 2>/dev/null || true
    sed -i "s/primary_color:.*/primary_color: \"${PRIMARY_COLOR}\"/g" "${CONFIG_FILE}" 2>/dev/null || true
    sed -i "s/base_domain:.*/base_domain: \"${BASE_DOMAIN}\"/g" "${CONFIG_FILE}" 2>/dev/null || true
fi

echo ""
echo "✅ Business configuration generated successfully!"
echo "Generated branding manifest: ${CONFIG_FILE}"
echo ""
echo "Quickstart to launch your small business suite:"
echo "  1. Docker Compose mode: docker compose -f docker-compose.smb.yaml up -d"
echo "  2. Kubernetes mode:     ./scripts/deploy-k8s.sh --profile small-business"
echo "=========================================================================="
