#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
MARKETING_DIR="$REPO_ROOT/web/marketing"
PUBLIC_DIR="$MARKETING_DIR/public"

echo "=== Building OCI Marketing Static Site ==="

mkdir -p "$PUBLIC_DIR"
mkdir -p "$PUBLIC_DIR/css"

if command -v hugo &> /dev/null; then
    echo "Hugo binary detected. Compiling site with Hugo..."
    (cd "$MARKETING_DIR" && hugo --minify -d "$PUBLIC_DIR")
else
    echo "Hugo not found in PATH; compiling template directly to $PUBLIC_DIR..."
    # Copy static assets
    cp -r "$MARKETING_DIR/static/"* "$PUBLIC_DIR/" 2>/dev/null || true

    # Render template values from branding tokens
    LEGAL_NAME="Orange Cat Investments LLC"
    TRADING_NAME="Orange Cat Investments"
    TAGLINE="Feline-Driven Wealth Management & Algorithmic Quantitative Telemetry"
    BASE_DOMAIN="oci.local"
    CUSTOMER_PORTAL="https://customer.oci.local"
    REGISTER_URL="https://customer.oci.local/#register"
    LOGO_ICON="🐈"

    sed \
        -e "s|{{ \.Site\.Title }}|$TRADING_NAME|g" \
        -e "s|{{ \.Site\.Params\.legalName }}|$LEGAL_NAME|g" \
        -e "s|{{ \.Site\.Params\.tradingName }}|$TRADING_NAME|g" \
        -e "s|{{ \.Site\.Params\.tagline }}|${TAGLINE//&/\\&}|g" \
        -e "s|{{ \.Site\.Params\.baseDomain }}|$BASE_DOMAIN|g" \
        -e "s|{{ \.Site\.Params\.customerPortalURL }}|$CUSTOMER_PORTAL|g" \
        -e "s|{{ \.Site\.Params\.registerURL }}|$REGISTER_URL|g" \
        -e "s|{{ \.Site\.Params\.logoIcon }}|$LOGO_ICON|g" \
        "$MARKETING_DIR/layouts/index.html" > "$PUBLIC_DIR/index.html"
fi

echo "Compiled static marketing assets successfully into $PUBLIC_DIR"

if command -v docker &> /dev/null; then
    echo "Docker detected. Building oci-marketing:latest container image..."
    docker build -t oci-marketing:latest -f "$MARKETING_DIR/Dockerfile" "$REPO_ROOT" || {
        echo "Docker build encountered an issue (daemon might not be running). Skipping image build."
    }
fi

echo "=== Marketing Site Build Completed Successfully ==="
