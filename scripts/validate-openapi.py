#!/usr/bin/env python3
"""
OpenAPI 3.1 Specification Contract Validator & Linting Script
Validates syntax, structure, versioning compliance, and domain entity consistency.
"""

import sys
import yaml

FORBIDDEN_KEYWORDS = ["collar", "perch", "feeder", "acoustic_events"]

def validate_openapi(filepath="api/openapi.yaml"):
    print(f"--> Validating OpenAPI specification: {filepath}")
    try:
        with open(filepath, "r", encoding="utf-8") as f:
            spec = yaml.safe_load(f)
    except Exception as e:
        print(f"FATAL: Failed to parse YAML: {e}", file=sys.stderr)
        return False

    # 1. Versioning check
    openapi_ver = spec.get("openapi", "")
    if not openapi_ver.startswith("3.1"):
        print(f"ERROR: Expected OpenAPI 3.1.x, got '{openapi_ver}'", file=sys.stderr)
        return False
    print(f"  [✓] OpenAPI version: {openapi_ver}")

    # 2. Info check
    info = spec.get("info", {})
    if not info.get("title") or not info.get("version"):
        print("ERROR: Missing info.title or info.version", file=sys.stderr)
        return False
    print(f"  [✓] Title: {info.get('title')}, Version: {info.get('version')}")

    # 3. Server base URLs check (must enforce /api/v1)
    servers = spec.get("servers", [])
    if not servers:
        print("ERROR: No servers defined", file=sys.stderr)
        return False
    for s in servers:
        url = s.get("url", "")
        if not url.endswith("/api/v1"):
            print(f"ERROR: Server URL '{url}' does not enforce standardized '/api/v1' prefix", file=sys.stderr)
            return False
    print(f"  [✓] Standardized /api/v1 server prefixes verified across {len(servers)} servers")

    # 4. Paths validation
    paths = spec.get("paths", {})
    if not paths:
        print("ERROR: No paths defined in specification", file=sys.stderr)
        return False

    valid_methods = {"get", "post", "put", "delete", "patch"}
    total_endpoints = 0

    for path, path_item in paths.items():
        if not path.startswith("/"):
            print(f"ERROR: Path '{path}' must begin with '/'", file=sys.stderr)
            return False

        # Verify no obsolete hardware paths exist
        for kw in FORBIDDEN_KEYWORDS:
            if kw in path.lower():
                print(f"ERROR: Obsolete hardware keyword '{kw}' found in path '{path}'", file=sys.stderr)
                return False

        methods_found = 0
        for method, op in path_item.items():
            if method.lower() in valid_methods:
                methods_found += 1
                total_endpoints += 1
                op_id = op.get("operationId")
                if not op_id:
                    print(f"WARNING: Operation {method.upper()} {path} is missing operationId")
                responses = op.get("responses", {})
                if not responses:
                    print(f"ERROR: Operation {method.upper()} {path} has no responses defined", file=sys.stderr)
                    return False

        if methods_found == 0:
            print(f"ERROR: Path '{path}' has no valid HTTP methods", file=sys.stderr)
            return False

    print(f"  [✓] Successfully validated {len(paths)} unique paths with {total_endpoints} operations")
    print("--> OpenAPI 3.1 specification contract validation: PASSED")
    return True

if __name__ == "__main__":
    filepath = sys.argv[1] if len(sys.argv) > 1 else "api/openapi.yaml"
    success = validate_openapi(filepath)
    sys.exit(0 if success else 1)
