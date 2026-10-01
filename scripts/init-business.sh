#!/usr/bin/env bash
set -euo pipefail

# OCI Platform Branding Configurator & Small Business Blueprint Initializer (`init-business.sh`)
# CLI tool to configure white-labeled branding parameters, module toggles, and design tokens for SMB deployments.

echo "=========================================================================="
echo "  OCI Small Business Platform Branding Configurator & Blueprint Initializer"
echo "=========================================================================="

BUSINESS_NAME="${1:-"Acme Global Enterprises"}"
PRIMARY_COLOR="${2:-"#2563eb"}"
BASE_DOMAIN="${3:-"acme.local"}"
ENABLE_CORE_INVEST="${4:-false}"
ENABLE_FELINE="${5:-false}"
ENABLE_PEBBLE="${6:-false}"

echo "Configuring white-labeled platform branding for: ${BUSINESS_NAME}"
echo "Primary Brand Color:      ${PRIMARY_COLOR}"
echo "Base Domain:              ${BASE_DOMAIN}"
echo "Enable Core Invest:       ${ENABLE_CORE_INVEST}"
echo "Enable Feline Workforce:  ${ENABLE_FELINE}"
echo "Enable Pebble Gateway:    ${ENABLE_PEBBLE}"

CONFIG_FILE="config/branding.yaml"

if [ -f "${CONFIG_FILE}" ]; then
    echo "Updating ${CONFIG_FILE} with custom business parameters..."
    sed -i "s/legal_name:.*/legal_name: \"${BUSINESS_NAME} LLC\"/g" "${CONFIG_FILE}"
    sed -i "s/trading_name:.*/trading_name: \"${BUSINESS_NAME}\"/g" "${CONFIG_FILE}"
    sed -i "s/primary_color:.*/primary_color: \"${PRIMARY_COLOR}\"/g" "${CONFIG_FILE}"
    sed -i "s/base_domain:.*/base_domain: \"${BASE_DOMAIN}\"/g" "${CONFIG_FILE}"
    sed -i "s/enable_core_invest:.*/enable_core_invest: ${ENABLE_CORE_INVEST}/g" "${CONFIG_FILE}"
    sed -i "s/enable_feline_workforce:.*/enable_feline_workforce: ${ENABLE_FELINE}/g" "${CONFIG_FILE}"
    sed -i "s/enable_pebble_gateway:.*/enable_pebble_gateway: ${ENABLE_PEBBLE}/g" "${CONFIG_FILE}"
fi

# Regenerate frontend CSS theme tokens from updated config
if command -v node >/dev/null 2>&1 && [ -f "scripts/generate-theme.js" ]; then
    echo "Compiling frontend design tokens (theme.css)..."
    node scripts/generate-theme.js
fi

echo ""
echo "✅ Business branding and module configuration initialized successfully!"
echo "Generated branding manifest: ${CONFIG_FILE}"
echo ""
echo "Quickstart to launch your small business suite:"
echo "  1. Docker Compose mode: docker compose -f docker-compose.smb.yaml up -d"
echo "  2. Kubernetes mode:     ./scripts/deploy-k8s.sh --profile small-business"
echo "=========================================================================="
