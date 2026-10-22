# AGENTS.md - Developer & Agent Instructions for Orange Cat Investments (OCI)

Welcome! This repository hosts the software application platform for **Orange Cat Investments (OCI)**. This document serves as instructions and conventions for autonomous software agents (and human engineers) modifying or expanding this codebase.

---

## 1. High-Level Architecture Overview

OCI is organized as a single **Monorepo** following **Domain-Driven Design (DDD)** and microservices architecture principles.

### Key Operational Planes & Tech Stack:
* **Orchestration & Infrastructure:** K3s Kubernetes, Ansible, Cert-Manager, Trust-Manager, Valkey, RabbitMQ.
* **Database Layer:** PostgreSQL 16 using schema isolation (`workforce`, `facilities`, `core_invest`, `ops`). Supports partitioned modular schemas (`01-workforce-core.sql`, `02-facilities-core.sql`, etc.) for white-labeled SMB deployments.
* **Enterprise Multi-Tier Architecture & Custom Services:**
  * **Tier 1: Client Applications:**
    * **Customer Web Portal (`web/customer-portal`):** Vue 3 SPA for retail investors and customers (Identity Provider: **ZITADEL OIDC**). Features portfolio analytics, WebRTC habitat stream player, trading terminal, and ESG transparency dashboard.
    * **Employee Web Portal (`web/employee-portal`):** Vue 3 SPA for corporate staff, employers, caretakers, and facilities engineers (Identity Provider: **Forgejo OAuth2**). Includes interactive Care Schedule, Emergency Medical Hold, VHR charting, leave requests, and performance review cycles.
    * **Hugo Marketing Site (`web/marketing`):** Static public marketing website.
    * **Flutter Mobile App (`mobile/flutter_app`):** Field maintenance mobile app with offline SQLite sync engine, QR scanning, and firmware OTA rollout tracking.
    * **Pebble Watch App (`embedded/pebble`):** On-call alert response watch app (Embedded JavaScript / RePebble Alloy framework).
  * **Tier 2: Backend-for-Frontend (BFF) Layer:**
    * **Customer BFF (`cmd/customer-bff` :8083):** Dedicated presentation proxy managing ZITADEL OIDC sessions (`__Host-customer-session`), CSRF tokens, and SSE ticker streaming. *Invariant: NEVER connects directly to Domain APIs or DB schemas.*
    * **Employee BFF (`cmd/employee-bff` :8081):** Dedicated presentation proxy managing Forgejo OAuth2 sessions (`__Host-employee-session`), CSRF tokens, and COTS integrations. *Invariant: NEVER connects directly to Domain APIs or DB schemas.*
  * **Tier 3: Business Logic Layer (BLL):**
    * **Customer BLL (`cmd/customer-bll`):** Middle-tier orchestrator for customer business rules, trade order execution, Plaid bank linking, broker adapter coordination, and async event publishing.
    * **Internal BLL (`cmd/internal-bll`):** Middle-tier orchestrator for corporate sagas (`OnboardingSaga`, `OffboardingEngine`), veterinary care directives, executive BI reporting, and incident event publishing.
  * **Tier 4: Domain APIs (gRPC on PostgreSQL):**
    * **Workforce Domain API (`cmd/workforce-api` :50051):** Owns `workforce` schema.
    * **Facilities Domain API (`cmd/facilities-api` :50052):** Owns `facilities` schema.
    * **Core Invest Domain API (`cmd/core-invest-api` :50053):** Owns `core_invest` schema.
    * **Ops Domain API (`cmd/ops-api` :50054):** Owns `ops` schema.
    * *(Note: Currently hosted in unified server `cmd/server` during active microservice decomposition).*
    * *Invariants: Domain APIs NEVER call each other. Authorized callers are strictly BLL APIs and authorized background/batch jobs.*
  * **Tier 5: Event Bus, Async Workers & Batch Schedulers:**
    * **Go Background Worker (`cmd/worker`):** Event subscriber (Watermill / RabbitMQ) for observation triggers, backtesting jobs, edge device OTA firmware rollout pipelines, async ACH bank transfer settlements, and automated cert-manager CA rotation tickers.
    * **Scheduled Batch Schedulers:** Cron runners for recurring ACH DCA deposits and automated end-of-month PDF statements.
    * **Wearable Gateway Proxy (`cmd/pebble-proxy` :8082):** Bridges Bluetooth/HTTP Pebble watch companion app requests with IT helpdesk operations and pub/sub real-time incident alerting.
* **Simulation Driver:**
  * **OCI Platform Simulator (`simulation/cmd/simulator`):** CLI and automation engine to exercise all 12 personas across corporate staff, feline executives, and retail investors in mock or live execution modes.

### Kubernetes Namespace Architecture:
* **`oci-core`:** Collocates all custom OCI applications (`oci-server`, `oci-customer-bff`, `oci-employee-bff`, `oci-pebble-proxy`, `oci-worker`, `oci-customer-portal`, `oci-employee-portal`) together with external customer identity (**ZITADEL**) and core operational infrastructure (`valkey`, `rabbitmq`, `meilisearch`, and dedicated database **`postgres-core`**).
* **`oci-internal`:** Dedicated namespace isolating off-the-shelf commercial/open-source internal business applications (COTS): Nextcloud, Forgejo, ERPNext/Frappe HR, and Snipe-IT, powered by their own dedicated internal database (**`postgres-internal`**).
* **`external-sim`:** Dedicated namespace hosting simulated external 3rd-party SaaS / financial APIs (`mock-broker`, `mock-bank`) completely isolated from production systems.
* **`monitoring` & `logging`:** Cluster observability and telemetry stacks (Prometheus, Grafana, Loki, Promtail).

---

## 2. Directory Structure Conventions

```text
.
├── api/                   # OpenAPI 3.1 YAML specifications (`openapi.yaml`)
├── config/                # Platform branding & white-label design token manifest (`branding.yaml`)
├── docs/                  # Architecture Decision Records (`adr/`) and specifications
├── personas/              # Profile specifications for all 12 corporate & customer personas
├── proto/                 # Protobuf definitions (`oci/workforce/`, `oci/facilities/`, etc.)
├── pkg/                   # Shared Go packages (`branding`, `envelope`, `mtls`, `cache`, `session`, `gen/go`)
│   ├── branding/          # Design token & white-label configuration loader
│   ├── cache/             # Valkey distributed cache, conditional ETags, and sliding rate limiter
│   ├── envelope/          # Standard EventEnvelope[T] message wrapper
│   ├── mtls/              # Dynamic mTLS certificate and CA pool reloaders
│   └── session/           # Valkey distributed session stores with memory fallback
├── scripts/               # SQL scripts (`init.sql`, modular schemas) & bootstrapping
├── cmd/                   # Go application entrypoints
│   ├── customer-bff/      # Customer portal backend-for-frontend (:8083)
│   ├── employee-bff/      # Employee portal backend-for-frontend (:8081)
│   ├── customer-bll/      # Customer Business Logic Layer API (planned/target)
│   ├── internal-bll/      # Internal Business Logic Layer API (planned/target)
│   ├── workforce-api/     # Workforce Domain gRPC API (:50051, planned/target)
│   ├── facilities-api/    # Facilities Domain gRPC API (:50052, planned/target)
│   ├── core-invest-api/   # Core Invest Domain gRPC API (:50053, planned/target)
│   ├── ops-api/           # Ops Domain gRPC API (:50054, planned/target)
│   ├── server/            # Unified API server hosting all 4 domains (active refactoring)
│   ├── pebble-proxy/      # Pebble wearable companion app gateway proxy (:8082)
│   ├── portal-server/     # Static SPA file server container with /healthz
│   ├── worker/            # Background worker (Watermill, OTA, CA rotation, ACH queue)
│   ├── mock-bank/         # Isolated mock Plaid/bank API (external-sim)
│   └── mock-broker/       # Isolated mock Alpaca/broker API (external-sim)
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
├── k8s/                   # Kubernetes deployment manifests
│   ├── namespaces/        # Core namespace definitions
│   ├── oci-core/          # Custom OCI apps, Zitadel IdP, core infra (Valkey, RabbitMQ, Meilisearch, postgres-core)
│   ├── oci-internal/      # Off-the-shelf internal business apps (Nextcloud, Forgejo, ERPNext, Snipe-IT, postgres-internal)
│   ├── external-sim/      # External simulation microservices (Mock Broker, Mock Bank)
│   ├── monitoring/        # Prometheus & Grafana stack
│   └── logging/           # Loki & Promtail aggregation stack
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

1. **Strict Prohibition on Hardcoded Data in Core Apps (`oci-core`):**
   * **Forbidden in Production Apps:** NEVER hardcode domain entity IDs, customer names, customer IDs, employee names, employee IDs, portfolio IDs, bank account numbers, or persona names in code deployed to `oci-core` applications (`cmd/server`, `cmd/customer-bff`, `cmd/employee-bff`, `cmd/pebble-proxy`, `cmd/worker`, `internal/...`, `pkg/...`).
   * **Dynamic Context Extraction:** All IDs, portfolio references, account identifiers, and customer/employee metadata must be dynamically extracted from HTTP request parameters, JSON payloads, authenticated session context (e.g., `CustomerFromContext` or `EmployeeFromContext`), database repositories (`pgxpool`), or external client lookups.
   * **Allowed Hardcoding Zones:** Hardcoding of test personas, seed accounts, or fixture entities is permitted ONLY in:
     - Database seed scripts (`scripts/init.sql`, `scripts/0*-*.sql`) for populating initial demo environments.
     - Unit and integration tests (`*_test.go`).
     - Platform simulation driver and persona action suites (`simulation/...`).
     - Isolated external simulation / mock microservices running in `external-sim` (`cmd/mock-broker/...`, `cmd/mock-bank/...`).
   * Hardcoding is also permissible for external unintegrated market ticker symbols where no market data feed is connected.
2. **Schema & Multi-Tier Layer Isolation Invariants:**
   * **Tier Hierarchy:** The platform enforces strict tier isolation: Frontends $\rightarrow$ Backend-for-Frontend (BFF) $\rightarrow$ Business Logic Layer (BLL) $\rightarrow$ Domain APIs (gRPC over isolated SQL databases) $\rightarrow$ Async Workers & Batch Schedulers.
   * **BFF Isolation Invariant:** Frontend BFF applications (`customer-bff`, `employee-bff`) must **NEVER connect directly to Domain APIs, database schemas (`pgxpool`), or async queue consumers**. They must exclusively interface with client frontends, manage session cookies/CSRF tokens, and delegate all business operations to the Business Logic Layer (BLL) or COTS APIs.
   * **Domain API Isolation Invariant:** Domain APIs (`workforce-api`, `facilities-api`, `core-invest-api`, `ops-api`) encapsulate their isolated PostgreSQL schemas over gRPC mTLS. **Domain APIs NEVER call each other.** Direct cross-domain database queries and inter-domain API calls are strictly forbidden. The ONLY authorized clients of Domain APIs are BLL APIs and authorized background worker/batch jobs.
   * **Business Logic Layer (BLL) Role:** BLL services (`customer-bll`, `internal-bll`) contain all business orchestration, financial calculations, sagas, external provider clients (broker/bank), and event emissions. They sit strictly between BFFs and Domain APIs.
   * **Async & Batch Processing Invariant:** Financial clearing operations (ACH settlement), batch generation (PDF reports), or periodic DCA execution run asynchronously via Watermill queue consumers or scheduled batch runners, never in synchronous request-response loops of BFFs.
   * **Pub/Sub Real-Time Alerting Standard:** Safety incidents (`events.workforce.incident_created.v1`) must fan out asynchronously to web portal banners (SSE/WebSocket) and Pebble watch alerts (`cmd/pebble-proxy`), ensuring emergency responsiveness without synchronous coupling.
3. **Dual Identity Provider Model:**
   * **ZITADEL:** Strictly reserved for external customer identity management (`web/customer-portal`).
   * **Forgejo:** System of record for internal employee, employer, staff, and developer identity management (`web/employee-portal`).
4. **COTS vs. Custom Isolation:** ERPNext/Frappe HR remain systems of record for generic accounting and payroll. Snipe-IT is the source of record for non-networked office inventory. Custom Go code is reserved for OCI core domain algorithms and hardware asset telemetry. Go microservices interface with COTS via REST/webhooks, never by sharing database tables.
5. **Platform Modularization & White-Labeling:** When developing or modifying domain features, respect domain feature flags (`ENABLE_CORE_INVEST`, `ENABLE_FELINE_WORKFORCE`, `ENABLE_PEBBLE_GATEWAY`) and utilize design tokens from `config/branding.yaml` / `pkg/branding` rather than hardcoding static brand colors or organizational names.
6. **Vendor & Product Name Encapsulation (Generic Domain Variable Naming):**
   * **Driver Package Encapsulation:** It is fine, appropriate, and expected that dedicated driver/adapter client packages (e.g., `pkg/broker/alpaca/client.go` or `pkg/bank/plaid/client.go`) have vendor product names throughout their internal implementation, comments, and package/type definitions.
   * **Domain-Centric Consumption:** Beyond those specialized driver packages, vendor product names must NOT spread further across the codebase. Consuming application entrypoints (`cmd/...`), services (`internal/service/...`), handlers (`internal/handler/...`), and repositories must use generic, domain-centric variable and field names. For example, use `broker` or `brokerClient` (instead of `alpaca` or `alpacaClient`), and `bank` or `bankClient` (instead of `plaid` or `plaidClient`).
7. **Separation of Simulation Code and Production Applications:**
   * Production applications (`cmd/server`, `cmd/*-bff`, `internal/handler/...`, `internal/service/...`) must NEVER contain synthetic endpoints, mock routes, or proxy helpers designed to facilitate simulations or tests.
   * All persona workflow simulations must reside strictly within `simulation/` and interact directly with the appropriate native system APIs (see detailed standard below).

### Simulation vs. Production Application Boundary (Employee & Persona Workflows)

1. **No Synthetic Facilitation Endpoints in Core Apps:**
   * **The Anti-Pattern:** When designing simulation scenarios for an employee (e.g., a software developer creating branches, committing code, or submitting pull requests; or an HR manager adjusting records), it is tempting to create proxy endpoints in `internal/handler/handler.go` or the web portals (`web/employee-portal`) so the simulation can interact exclusively within the custom OCI codebase. This is strictly prohibited.
   * **Why it is forbidden:** In reality, employees do not use custom internal website wrappers for standard toolchains (e.g., developers do not use a corporate dashboard to commit git code). Adding synthetic endpoints pollutes the production API surface, blurs domain boundaries, and produces false confidence by testing artificial proxy code rather than actual employee workflows and permissions.

2. **Simulating Employee Workflows Directly at the Target System's API Boundary:**
   * All API interactions simulating an employee's workflow must be placed directly in the simulation driver (`simulation/actions.go`, `simulation/client.go`).
   * The simulation client should directly target the native APIs of the relevant platform using the persona's credentials:
     - **Developer Persona (e.g., Elise, Marcus):** Interacts directly with Forgejo's REST APIs (`/api/v1/repos/{owner}/{repo}/...` for issues/Kanban, branch creation, file commits via contents API, pull requests, reviews, and merges).
     - **Corporate/Executive Persona:** Interacts directly with Nextcloud (WebDAV/OCS APIs), ERPNext, or Snipe-IT native APIs.
     - **End-User/Caretaker Persona:** Interacts with legitimate OCI endpoints on `oci-server` or the BFFs (`customer-bff`, `employee-bff`) as an authentic user would.
   * **Purpose of Simulation:** The objective is to verify that the persona has sufficient permissions and that the underlying infrastructure (e.g., Forgejo, Nextcloud, K8s runners) correctly supports the complete end-to-end workflow (e.g., from Kanban card to feature branch to code review and merge), without requiring OCI production services to act as an intermediary.

3. **Dual-Mode Simulation Architecture (`-mode mock` vs `-mode live`):**
   * **Mock Mode (`simulation/mock_*.go`):** Fast, standalone local testing without external dependencies. Standalone in-memory HTTP servers (e.g., `MockForgejoServer` in `simulation/mock_forgejo.go`, or stateless mock broker/bank in `external-sim`) emulate the third-party or COTS API contracts.
   * **Live Mode:** The simulation client routes HTTP calls directly to the live internal COTS service (e.g., `http://forgejo-service.oci-internal.svc:3000` or `https://forgejo.oci.local`) or development endpoints in `oci-dev`. Simulations must NEVER target the production `oci-core` namespace.

### Security Invariants
1. **Zero Long-Lived / Static Certs:** All inter-service gRPC calls require mTLS via `cert-manager`. Certificates must be reloaded dynamically in Go standard library `crypto/tls` (`GetCertificate` / `GetClientCertificate`). Never use `subPath` secret mounts.
2. **No Sensitive Payload Logging (CWE-532):** Background event workers and HTTP handlers must never log unredacted message payloads, secret keys, or credentials.
3. **No JWTs in Web SPAs:** Web browser clients authenticate via OIDC/OAuth2 handled at the Go BFF layer. Access tokens are stored exclusively in Valkey. Browsers receive only opaque session cookies (`__Host-customer-session` or `__Host-employee-session`, both `HttpOnly`, `Secure`, `SameSite=Lax`).
4. **Double-Submit CSRF:** All state-changing HTTP requests (`POST`, `PUT`, `DELETE`) to the BFFs must include the `X-CSRF-Token` header matching the CSRF cookie.

### Kubernetes Namespace & Database Isolation Standards
1. **Core Workload & Customer Plane Collocation (`oci-core`):** All custom OCI applications (`oci-server`, BFFs, web portals, worker, pebble-proxy), customer identity provider (ZITADEL), and their direct operational dependencies (Valkey, RabbitMQ, Meilisearch, and `postgres-core`) must reside together in the `oci-core` namespace. Never place internal employee COTS tools here.
2. **COTS Isolation (`oci-internal`):** Commercial and open-source off-the-shelf business platforms (Nextcloud, Forgejo, ERPNext, Frappe HR, Snipe-IT) must be deployed exclusively in `oci-internal` and connect to their dedicated `postgres-internal-service` database. They must connect to shared core resources (like Valkey) via full cluster FQDNs (`valkey-service.oci-core.svc.cluster.local:6379`).
3. **Database Tier Separation (Dual PostgreSQL):** Proprietary business schemas (`workforce`, `facilities`, `core_invest`, `ops`) and customer auth are isolated to `postgres-core` in `oci-core`. COTS third-party applications use independent databases (`forgejo`, `nextcloud`, `snipeit`, `erpnext`) on `postgres-internal` in `oci-internal`. COTS applications must never share or connect to `postgres-core`.
4. **Simulation Isolation (`external-sim`):** Mock 3rd-party services (e.g., Alpaca Broker API, Plaid / Banking API) must run strictly in `external-sim`. They use stateless in-memory persistence and must never connect directly to either PostgreSQL database or access internal application secrets.
5. **No Legacy `apps` or Standalone `postgres` Namespaces:** The legacy `apps` and `postgres` namespaces are deprecated and deleted. Never create or reintroduce Kubernetes manifests targeting `namespace: apps` or `namespace: postgres`.

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
