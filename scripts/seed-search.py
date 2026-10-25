#!/usr/bin/env python3
"""
Seed Meilisearch search indexes (customer_catalog, employee_knowledge) with development fixtures.
This script is intended strictly for development and simulation testing (e.g. oci-dev, local simulator).
"""

import argparse
import json
import os
import sys
import urllib.error
import urllib.request

CUSTOMER_CATALOG_DOCS = [
    {
        "id": "fund-zoomie",
        "category": "fund",
        "symbol": "ZOOM",
        "title": "OCI Feline Behavioral Alpha Fund",
        "description": "Dynamic quantitative weighting based on active feline zoomie intervals.",
    },
    {
        "id": "fund-catnip",
        "category": "fund",
        "symbol": "NIP",
        "title": "Organic Catnip Agriculture ESG Fund",
        "description": "Sustainable feline agricultural infrastructure and cat grass ventures.",
    },
    {
        "id": "esg-welfare-2026",
        "category": "transparency",
        "title": "2026 Feline Workplace Welfare & Habitat Transparency",
        "description": "OSHA-compliant scratch post maintenance and continuous perch monitoring.",
    },
    {
        "id": "stmt-tax-1099b",
        "category": "statement",
        "title": "Form 1099-B Proceeds From Broker and Barter Exchange",
        "description": "Annual consolidated tax statement for realized capital gains and DCA distributions.",
    },
]

EMPLOYEE_KNOWLEDGE_DOCS = [
    {
        "id": "sop-laser-01",
        "category": "sop",
        "title": "SOP-014: Laser Pointer Telemetry & Biometrics",
        "snippet": "Standard clinical operating procedure for automated optical tracking and zoomie measurement.",
    },
    {
        "id": "sop-nip-02",
        "category": "sop",
        "title": "SOP-089: Catnip Dispensary Quality Assurance",
        "snippet": "Guidelines for testing organic potency, THC-free feline compliance, and batch seals.",
    },
    {
        "id": "vhr-garfield",
        "category": "vhr",
        "title": "VHR: Garfield Clinical Notes (Feline Alpha)",
        "snippet": "Dr. Sarah Chen: Weight 14.2 kg. Prescribed 15-minute scheduled cardboard box play sessions.",
    },
    {
        "id": "vhr-chloe",
        "category": "vhr",
        "title": "VHR: Chloe Biometric & Activity Log",
        "snippet": "Active zoomies observed in Perch Zone Bravo. Resting heart rate nominal at 135 bpm.",
    },
    {
        "id": "ticket-it-101",
        "category": "ticket",
        "title": "Switch Port Error on Edge Switch Habitat Alpha",
        "snippet": "High packet error rate on port ge-0/0/4 connected to high-speed feline camera 1.",
    },
    {
        "id": "asset-cam-4k",
        "category": "asset",
        "title": "Edge 4K Habitat Camera (Firmware v2.1)",
        "snippet": "Sony IMX sensor edge vision node with real-time pose estimation inference.",
    },
]


def post_json(url, path, data, api_key=None):
    target = f"{url.rstrip('/')}{path}"
    headers = {"Content-Type": "application/json"}
    if api_key:
        headers["Authorization"] = f"Bearer {api_key}"
    payload = json.dumps(data).encode("utf-8")
    req = urllib.request.Request(target, data=payload, headers=headers, method="POST")
    with urllib.request.urlopen(req, timeout=5) as resp:
        return json.loads(resp.read().decode("utf-8"))


def seed(meili_url, api_key):
    print(f"Connecting to Meilisearch at {meili_url}...")
    try:
        health_req = urllib.request.Request(f"{meili_url.rstrip('/')}/health")
        with urllib.request.urlopen(health_req, timeout=3):
            pass
    except Exception as e:
        print(f"ℹ️  Meilisearch at {meili_url} is not reachable ({e}). Skipping search seeding.")
        return False

    try:
        post_json(meili_url, "/indexes", {"uid": "customer_catalog", "primaryKey": "id"}, api_key)
    except Exception:
        pass

    try:
        post_json(meili_url, "/indexes", {"uid": "employee_knowledge", "primaryKey": "id"}, api_key)
    except Exception:
        pass

    try:
        post_json(meili_url, "/indexes/customer_catalog/documents", CUSTOMER_CATALOG_DOCS, api_key)
        print("  ✓ Seeded customer_catalog documents into Meilisearch")
        post_json(meili_url, "/indexes/employee_knowledge/documents", EMPLOYEE_KNOWLEDGE_DOCS, api_key)
        print("  ✓ Seeded employee_knowledge documents into Meilisearch")
        return True
    except Exception as e:
        print(f"❌ Failed to index documents in Meilisearch: {e}")
        return False


def main():
    parser = argparse.ArgumentParser(description="Seed Meilisearch fixtures for dev & simulation")
    parser.add_argument("--url", default=os.getenv("MEILISEARCH_URL", "http://localhost:7700"), help="Meilisearch Base URL")
    parser.add_argument("--key", default=os.getenv("MEILI_MASTER_KEY", ""), help="Meilisearch Master Key")
    args = parser.parse_args()

    seed(args.url, args.key)


if __name__ == "__main__":
    main()
