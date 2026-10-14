# AGENTS.md - Developer & Agent Instructions for Orange Cat Investments (OCI)

Welcome! This repository hosts the software application platform for **Orange Cat Investments (OCI)**. This document serves as instructions and conventions for autonomous software agents (and human engineers) modifying or expanding this codebase.

---

## 1. High-Level Architecture Overview

OCI is organized as a single **Monorepo** following **Domain-Driven Design (DDD)** and microservices architecture principles.

### Key Operational Planes & Tech Stack:
* **Orchestration & Infrastructure:** K3s Kubernetes, Ansible, Cert-Manager, Trust-Manager, Valkey, RabbitMQ.
* **Database Layer:** PostgreSQL 16 using schema isolation (`workforce`, `facilities`, `core_invest`, `ops`). Supports partitioned modular schemas (`01-workforce-core.sql`, `02-facilities-core.sql`, etc.) for white-labeled SMB deployments.
* **Custom Backend Services:**
  * **Unified API Server (`cmd/server`):** Serves domain REST APIs across `core-invest`, `facilities`, `ops`, and `workforce`. Supports runtime domain feature flags (`ENABLE_CORE_INVEST`, `ENABLE_FELINE_WORKFORCE`, `ENABLE_PEBBLE_GATEWAY`).
  * **Go REST BFFs (`cmd/customer-bff` & `cmd/employee-bff`):** Connect web/mobile frontends using `net/http`. Uses server-side Valkey session management and double-submit CSRF protection.
  * **Wearable Gateway Proxy (`cmd/pebble-proxy`):** Bridges Bluetooth/HTTP Pebble watch companion app requests with IT helpdesk operations (`ops.it_tickets`).
  * **Go Background Worker (`cmd/worker`):** Event subscriber (Watermill / RabbitMQ) for observation triggers, backtesting jobs, edge device OTA firmware rollout pipelines, and automated cert-manager CA rotation tickers.
* **Client Applications:**
  * **Customer Web Portal (`web/customer-portal`):** Vue 3 SPA for retail investors and customers (Identity Provider: **ZITADEL OIDC**). Features portfolio analytics, WebRTC habitat stream player, trading terminal, and ESG transparency dashboard.
  * **Employee Web Portal (`web/employee-portal`):** Vue 3 SPA for corporate staff, employers, caretakers, and facilities engineers (Identity Provider: **Forgejo OAuth2**). Includes interactive Care Schedule, Emergency Medical Hold, VHR charting, leave requests, and performance review cycles.
  * **Hugo Marketing Site (`web/marketing`):** Static public marketing website.
  * **Flutter Mobile App (`mobile/flutter_app`):** Field maintenance mobile app with offline SQLite sync engine, QR scanning, and firmware OTA rollout tracking.
  * **Pebble Watch App (`embedded/pebble`):** On-call alert response watch app (Embedded JavaScript / RePebble Alloy framework).
* **Simulation Driver:**
  * **OCI Platform Simulator (`simulation/cmd/simulator`):** CLI and automation engine to exercise all 12 personas across corporate staff, feline executives, and retail investors in mock or live execution modes.

---

## 2. Directory Structure Conventions

```text
.
├── api/                   # OpenAPI 3.1 YAML specifications (`openapi.yaml`)
├── config/                # Platform branding & white-label design token manifest (`branding.yaml`)
├── docs/                  # Architecture Decision Records (`adr/`) and specifications
├── personas/              # Profile specifications for all 12 corporate & customer personas
├── proto/                 # Protobuf definitions (`oci/core/`, `oci/ops/`)
├── pkg/                   # Shared Go packages (`branding`, `envelope`, `mtls`, `gen/go`)
│   ├── branding/          # Design token & white-label configuration loader
│   └── envelope/          # Standard EventEnvelope[T] message wrapper
├── scripts/               # SQL scripts (`init.sql`, modular schemas) & bootstrapping
├── cmd/                   # Go application entrypoints
│   ├── server/            # Unified API server hosting all 4 domains
│   ├── customer-bff/      # Customer portal backend-for-frontend (:8083)
│   ├── employee-bff/      # Employee portal backend-for-frontend (:8081)
│   ├── pebble-proxy/      # Pebble wearable companion app gateway proxy (:8082)
│   └── worker/            # Background worker (Watermill, OTA, CA rotation)
├── simulation/            # Full platform simulation driver & persona action suites
│   └── cmd/simulator/     # Standalone simulation CLI (`-mode live|mock`)
├── web/                   # Frontend applications
│   ├── customer-portal/   # Vue 3 SPA for retail investors and customers (:5174)
│   ├── employee-portal/   # Vue 3 SPA for employee and feline care management (:5173)
│   └── marketing/         # Hugo static marketing website
├── mobile/                # Mobile applications
│   └── flutter_app/       # Flutter offline-first field maintenance application
├── embedded/              # Embedded firmware & wearable applications
│   └── pebble/            # Pebble watch companion app (Pebble Alloy / JS / C)
├── k8s/                   # Kubernetes deployment manifests (PostgreSQL, Valkey, RabbitMQ, COTS)
├── ansible/               # Ansible playbooks for host & cluster provisioning
└── internal/              # Core domain logic, handlers, repositories, services, and sagas
    ├── handler/           # HTTP handlers (core_invest, facilities, ops, workforce)
    ├── repository/        # pgxpool database repositories & mock implementations
    ├── service/           # Domain business logic services & PKI rotation manager
    └── saga/              # Distributed sagas (OnboardingSaga, OffboardingEngine)
```

---

## 3. Mandatory Coding Standards & Principles

### Domain Boundary & Dual Identity Rules
1. **Database System of Record (No Hardcoded Data):** Do not hardcode domain entity lists, employee rosters, or welfare reports in APIs, services, or web/mobile frontends when that data can be stored in and fetched from the PostgreSQL database. Hardcoding is permitted only for unintegrated external market feeds (e.g. stock tickers) where no database source currently exists.
2. **Schema Isolation:** Custom Go services strictly access their designated database schema (`workforce`, `facilities`, `core_invest`, `ops`). Direct cross-schema table queries or cross-Domain API calls are prohibited.
3. **Dual Identity Provider Model:**
   * **ZITADEL:** Strictly reserved for external customer identity management (`web/customer-portal`).
   * **Forgejo:** System of record for internal employee, employer, staff, and developer identity management (`web/employee-portal`).
4. **COTS vs. Custom Isolation:** ERPNext/Frappe HR remain systems of record for generic accounting and payroll. Snipe-IT is the source of record for non-networked office inventory. Custom Go code is reserved for OCI core domain algorithms and hardware asset telemetry. Go microservices interface with COTS via REST/webhooks, never by sharing database tables.
5. **Platform Modularization & White-Labeling:** When developing or modifying domain features, respect domain feature flags (`ENABLE_CORE_INVEST`, `ENABLE_FELINE_WORKFORCE`, `ENABLE_PEBBLE_GATEWAY`) and utilize design tokens from `config/branding.yaml` / `pkg/branding` rather than hardcoding static brand colors or organizational names.

### Security Invariants
1. **Zero Long-Lived / Static Certs:** All inter-service gRPC calls require mTLS via `cert-manager`. Certificates must be reloaded dynamically in Go standard library `crypto/tls` (`GetCertificate` / `GetClientCertificate`). Never use `subPath` secret mounts.
2. **No Sensitive Payload Logging (CWE-532):** Background event workers and HTTP handlers must never log unredacted message payloads, secret keys, or credentials.
3. **No JWTs in Web SPAs:** Web browser clients authenticate via OIDC/OAuth2 handled at the Go BFF layer. Access tokens are stored exclusively in Valkey. Browsers receive only opaque session cookies (`__Host-customer-session` or `__Host-employee-session`, both `HttpOnly`, `Secure`, `SameSite=Lax`).
4. **Double-Submit CSRF:** All state-changing HTTP requests (`POST`, `PUT`, `DELETE`) to the BFFs must include the `X-CSRF-Token` header matching the CSRF cookie.

### Event Protocol Standard
All domain messages emitted across RabbitMQ must be wrapped in the standard Go `envelope.EventEnvelope[T]` with UUIDv7 `EventID`, UTC timestamp `OccurredAt`, and OTel context `CorrelationID`.

---

## 4. Verification & Testing Procedures

Before marking any task as complete or submitting code changes, agents must verify their work using appropriate validation methods:

1. **SQL Validation:**
   Ensure `scripts/init.sql` parses correctly and enforces PostgreSQL constraints:
   ```bash
   python3 -c "
   with open('scripts/init.sql') as f:
       sql = f.read()
   assert 'facilities.firmware_releases' in sql
   assert 'facilities.maintenance_logs' in sql
   assert 'facilities.device_ota_jobs' in sql
   print('SQL syntax & constraints validated successfully')
   "
   ```
2. **OpenAPI Specification Validation:**
   Verify `api/openapi.yaml` for YAML correctness and OpenAPI 3.1 compliance:
   ```bash
   python3 -c "import yaml; yaml.safe_load(open('api/openapi.yaml'))"
   ```
3. **Go Code Quality & Race Detection:**
   Run full test suite with race detector enabled:
   ```bash
   go test -v -race ./...
   go vet ./...
   ```
4. **Platform Simulation Driver Validation:**
   Run the 12-persona simulation test suite and driver to verify end-to-end integration:
   ```bash
   go test -v ./simulation/...
   go run ./simulation/cmd/simulator -mode mock
   ```
5. **Frontend Theme & Design Token Compilation:**
   Verify `scripts/generate-theme.js` compiles CSS variables from `config/branding.yaml`:
   ```bash
   node scripts/generate-theme.js
   ```
6. **General Rule:** Always inspect modified files after editing to verify formatting, content accuracy, and absence of syntax errors.

---

## 5. Local Server & Target Testing Environments

* **Local Private Server Notes (`local-server-notes.md`):**
  When performing integration tests, cluster provisioning, or remote verification on dedicated local testing hardware, always check for the presence of `local-server-notes.md` in the repository root.
  * This file is gitignored to keep private hostnames, SSH user details, network specifics, and cluster configurations private.
  * Agents should inspect `local-server-notes.md` first to obtain connectivity details, active container/Kubernetes runtime specifications (e.g. MicroK8s, K3s, Docker), and local execution parameters before executing commands or tests against local infrastructure.
