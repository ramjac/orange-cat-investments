# AGENTS.md - Developer & Agent Instructions for Orange Cat Investments (OCI)

Welcome! This repository hosts the software application platform for **Orange Cat Investments (OCI)**. This document serves as instructions and conventions for autonomous software agents (and human engineers) modifying or expanding this codebase.

---

## 1. High-Level Architecture Overview

OCI is organized as a single **Monorepo** following **Domain-Driven Design (DDD)** and microservices architecture principles.

### Key Operational Planes & Tech Stack:
* **Orchestration & Infrastructure:** MicroK8s on Ubuntu Server, Ansible, Cert-Manager, Trust-Manager, Valkey, RabbitMQ.
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
    * *Invariants: Domain APIs NEVER call each other. Authorized callers are strictly BLL APIs and authorized background/batch jobs.*
  * **Tier 5: Event Bus, Async Workers & Batch Schedulers:**
    * **Go Background Worker (`cmd/worker`):** Event subscriber (Watermill / RabbitMQ) for observation triggers, backtesting jobs, edge device OTA firmware rollout pipelines, async ACH bank transfer settlements, and automated cert-manager CA rotation tickers.
    * **Scheduled Batch Schedulers:** Cron runners for recurring ACH DCA deposits and automated end-of-month PDF statements.
    * **Wearable Gateway Proxy (`cmd/pebble-proxy` :8082):** Bridges Bluetooth/HTTP Pebble watch companion app requests with IT helpdesk operations and pub/sub real-time incident alerting.
* **Simulation Driver:**
  * **OCI Platform Simulator (`simulation/cmd/simulator`):** CLI and automation engine to exercise all personas across corporate staff, feline executives, and retail investors against the running OCI stack.

### Kubernetes Namespace Architecture:
* **`oci-core`:** Collocates all custom OCI applications (BFFs, BLLs, Domain APIs, `oci-pebble-proxy`, `oci-worker`, `oci-customer-portal`, `oci-employee-portal`) together with external customer identity (**ZITADEL**) and core operational infrastructure (`valkey`, `rabbitmq`, `meilisearch`, and dedicated database **`postgres-core`**).
* **`oci-internal`:** Dedicated namespace isolating off-the-shelf commercial/open-source internal business applications (COTS): Nextcloud, Forgejo, ERPNext/Frappe HR, and Snipe-IT, powered by their own dedicated internal database (**`postgres-internal`**).
* **`external-sim`:** Dedicated namespace hosting simulated external 3rd-party SaaS / financial APIs (`mock-broker`, `mock-bank`) completely isolated from production systems.
* **`monitoring` & `logging`:** Cluster observability and telemetry stacks (Prometheus, Grafana, Loki, Promtail).

---

## 2. Directory Structure Conventions

```text
.
├── api/                   # OpenAPI 3.1 YAML specifications (`openapi.yaml`)
├── config/                # Platform branding & white-label design token manifest (`branding.yaml`)
├── docs/                  # Architecture & security manuals, ADRs (`adr/`), reports & runbooks (`runbooks/`)
├── personas/              # Profile specifications for all 13 corporate & customer personas
├── proto/                 # Protobuf definitions (`oci/workforce/`, `oci/facilities/`, etc.)
├── pkg/                   # Shared Go packages (`authz`, `branding`, `cache`, `cosignverify`, `cots`, `dbpool`, `envelope`, `grpcutil`, `httputil`, `mtls`, `outbox`, `reporting`, `search`, `session`, `telemetry`, `waf`, `gen/go`)
│   ├── authz/policy/      # Embedded OPA/Rego v1 authorization engine (`workforce.rego`, `trading.rego`)
│   ├── bank/plaid/        # Encapsulated bank adapter driver
│   ├── branding/          # Design token & white-label configuration loader
│   ├── broker/alpaca/     # Encapsulated broker adapter driver
│   ├── cache/valkey/      # Distributed cache, ETags, sliding rate limiter, idempotency, AMQP invalidation
│   ├── cosignverify/      # SHA-256 & Cosign ECDSA/Ed25519 firmware verification
│   ├── cots/              # REST/WebDAV clients for Forgejo, Frappe HR, Nextcloud, Snipe-IT
│   ├── dbpool/            # File-reloaded dynamic PostgreSQL pgxpool credential manager
│   ├── envelope/          # Standard EventEnvelope[T] message wrapper & W3C TraceContext propagator
│   ├── grpcutil/          # Dynamic mTLS builders, SPIFFE URI SAN interceptors & actor metadata
│   ├── httputil/          # Shared HTTP helpers and router utilities
│   ├── mtls/              # Dynamic mTLS certificate (`tmpfs`) and CA pool reloaders
│   ├── outbox/            # Transactional SQL outbox (`InsertTx`, `ClaimPending`, `Forwarder`)
│   ├── reporting/         # PDF/CSV executive reports, statements, 1099-B & GDPR certificates
│   ├── search/meilisearch/# Typo-tolerant Meilisearch client
│   ├── session/valkey/    # Valkey distributed session stores with Lua bulk revocation & CSRF
│   ├── telemetry/         # OpenTelemetry tracer/meter setup & custom business/WAF/upstream metrics
│   └── waf/               # Embedded Coraza OWASP CRS HTTP middleware
├── scripts/               # SQL scripts (`init.sql`, `migrations/`, modular schemas), CI contracts & deploy scripts
├── cmd/                   # Go application entrypoints
│   ├── customer-bff/      # Customer portal backend-for-frontend (:8083)
│   ├── employee-bff/      # Employee portal backend-for-frontend (:8081)
│   ├── customer-bll/      # Customer Business Logic Layer API (:8085)
│   ├── internal-bll/      # Internal Business Logic Layer API (:8086)
│   ├── workforce-api/     # Workforce Domain gRPC API (:50051)
│   ├── facilities-api/    # Facilities Domain gRPC API (:50052)
│   ├── core-invest-api/   # Core Invest Domain gRPC API (:50053)
│   ├── ops-api/           # Ops Domain gRPC API (:50054)
│   ├── pebble-proxy/      # Pebble wearable companion app gateway proxy (:8082)
│   ├── portal-server/     # Static SPA/marketing file server container with /healthz
│   ├── worker/            # Background worker (Outbox Forwarder, Watermill, OTA, CA rotation, ACH/NAV batch)
│   ├── mock-bank/         # Isolated mock bank API (external-sim)
│   ├── mock-broker/       # Isolated mock broker API (external-sim)
│   └── mock-hardware/     # Isolated mock habitat camera/telemetry/OTA client (external-sim)
├── simulation/            # Full platform simulation driver & 13-persona action suites
│   └── cmd/simulator/     # Standalone simulation CLI (`-mode live|step|continuous`)
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
│   ├── oci-core/          # Production custom OCI apps, ZITADEL, OpenBao, coturn, Valkey, RabbitMQ, Meilisearch, postgres-core
│   ├── oci-dev/           # Staging mirror of custom OCI workloads + daily retention CronJob
│   ├── oci-internal/      # Off-the-shelf internal COTS apps (Nextcloud, Forgejo, Runner, ERPNext, Snipe-IT, postgres-internal)
│   ├── external-sim/      # External simulation microservices (prod & -dev Mock Bank, Mock Broker, Mock Hardware)
│   ├── registry/          # Harbor Enterprise Container Registry (`harbor.yaml` on :32000)
│   ├── security/          # Kyverno admission controller & Enforce ClusterPolicies
│   ├── ha/                # Multi-node HA configs (Gateway API `gateway.yaml`, LAN DNS/TLS, Longhorn, Cloudflare Tunnel)
│   ├── monitoring/        # Prometheus, Grafana, Alloy, Tempo, kube-state-metrics, Gatus
│   └── logging/           # Loki & Promtail CRI pod-logging stack
├── ansible/               # Ansible playbooks for host & cluster provisioning
└── internal/              # Core domain logic, app routers, handlers, repositories, services, and sagas
    ├── app/               # Modular service runtime packages (customerbff, employeebff, customerbll, internalbll, worker)
    ├── handler/           # HTTP handlers (workforce, ops, cots, reporting)
    ├── repository/        # pgxpool database repositories, audit hash-chain & mock implementations
    ├── service/           # Domain business logic services & PKI rotation manager
    └── saga/              # Distributed sagas (OnboardingSaga, OffboardingEngine)
```

---

## 3. Mandatory Coding Standards & Principles
 
### Domain Boundary & Dual Identity Rules

1. **Strict Prohibition on Hardcoded Data in Core Apps (`oci-core`):**
   * **Forbidden in Production Apps:** NEVER hardcode domain entity IDs, customer names, customer IDs, employee names, employee IDs, portfolio IDs, bank account numbers, or persona names in code deployed to `oci-core` applications (`cmd/*-bll`, `cmd/*-api`, `cmd/*-bff`, `cmd/pebble-proxy`, `cmd/worker`, `internal/...`, `pkg/...`).
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
   * Production applications (`cmd/*-api`, `cmd/*-bll`, `cmd/*-bff`, `internal/handler/...`, `internal/service/...`) must NEVER contain synthetic endpoints, mock routes, or proxy helpers designed to facilitate simulations or tests.
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
     - **End-User/Caretaker Persona:** Interacts with legitimate OCI endpoints on the BFFs (`customer-bff`, `employee-bff`) as an authentic user would.
   * **Purpose of Simulation:** The objective is to verify that the persona has sufficient permissions and that the underlying infrastructure (e.g., Forgejo, Nextcloud, K8s runners) correctly supports the complete end-to-end workflow (e.g., from Kanban card to feature branch to code review and merge), without requiring OCI production services to act as an intermediary.

3. **Simulation Execution Architecture:**
   * The platform simulation driver acts as a true end-to-end integration test driving persona workflows against an actual running instance of the OCI stack (e.g. dev staging at `https://dev.api.oci.local` or a running local development stack).
   * In-process mock simulation mode is eliminated: the simulator communicates exclusively over network HTTP boundaries to authentic services, verifying end-to-end permissions, routing, session stores, CSRF enforcement, and business workflows to ensure runtime defects are reliably caught. Simulations must NEVER target the production `oci-core` namespace.

4. **Simulation Execution Location Standard (External Workstation vs. Test Server Host):**
   * The platform simulation driver (`simulation/cmd/simulator`) must **NOT typically be run directly on the test server host** (e.g. `thinker`).
   * The simulator acts as an external client driving synthetic persona workflows; running it on the test server consumes target server CPU, RAM, and scheduler capacity with the test generator itself, which skews cluster telemetry and defeats the purpose of evaluating how the server handles external load.
   * Simulations and load tests must be executed **externally** (from the developer workstation, client machines, or dedicated CI/CD runners) across the network, generating authentic ingress network traffic into the test server.
   * Executing the simulator directly on the test server host is strictly restricted to rare network debugging tasks (e.g. diagnosing local host network interfaces, firewall rules, or container socket bindings).

### Security Invariants
1. **Zero Long-Lived / Static Certs or Static Secrets:** All inter-service gRPC calls require mTLS via `cert-manager-csi-driver-spiffe` (ephemeral 1-hour X.509-SVIDs mounted on `tmpfs` with SPIFFE URI SAN authorization in `pkg/grpcutil`). Certificates and CA pools (`trust-manager` `oci-ca-bundle`) must be reloaded dynamically in Go standard library `crypto/tls` (`GetCertificate` / `GetClientCertificate`). Never use `subPath` secret mounts, and never commit static `kind: Secret` credential manifests—use OpenBao + External Secrets Operator (`ExternalSecret` / `VaultDynamicSecret`).
2. **No Sensitive Payload Logging (CWE-532):** Background event workers, WAF middleware, and HTTP handlers must never log unredacted message payloads, exploit strings, secret keys, or credentials.
3. **No JWTs in Web SPAs:** Web browser clients authenticate via OIDC/OAuth2 handled at the Go BFF layer. Access tokens are stored exclusively in Valkey. Browsers receive only opaque session cookies (`__Host-customer-session` or `__Host-employee-session`, both `HttpOnly`, `Secure`, `SameSite=Lax`).
4. **Double-Submit CSRF & Edge WAF:** All state-changing HTTP requests (`POST`, `PUT`, `DELETE`) to the BFFs must include the `X-CSRF-Token` header matching the CSRF cookie. All BFF HTTP routes are protected by embedded Coraza OWASP CRS WAF middleware (`pkg/waf`).
5. **Policy-as-Code & Audit Integrity:** Sensitive workforce and pre-trade authorization checks are evaluated fail-closed via embedded OPA/Rego (`pkg/authz/policy`) in the BLL tier, with denials and sensitive mutations recorded in the tamper-evident SHA-256 hash-chain ledger (`ops.audit_logs`).

### Kubernetes Namespace & Database Isolation Standards
1. **Core Workload & Customer Plane Collocation (`oci-core`):** All custom OCI applications (BFFs, BLLs, Domain APIs, web portals, worker, pebble-proxy, `coturn`), customer identity provider (ZITADEL), secret manager (OpenBao), and their direct operational dependencies (Valkey, RabbitMQ, Meilisearch, and `postgres-core`) must reside together in the `oci-core` namespace. Never place internal employee COTS tools here.
2. **COTS Isolation (`oci-internal`):** Commercial and open-source off-the-shelf business platforms (Nextcloud, Forgejo, ERPNext, Frappe HR, Snipe-IT) must be deployed exclusively in `oci-internal` and connect to their dedicated `postgres-internal-service` database. They must connect to shared core resources (like Valkey) via full cluster FQDNs (`valkey-service.oci-core.svc.cluster.local:6379`).
3. **Database Tier Separation (Dual PostgreSQL):** Proprietary business schemas (`workforce`, `facilities`, `core_invest`, `ops`) and customer auth are isolated to `postgres-core` in `oci-core` (accessed via OpenBao schema-scoped dynamic PostgreSQL roles). COTS third-party applications use independent databases (`forgejo`, `nextcloud`, `snipeit`, `erpnext`) on `postgres-internal` in `oci-internal`. COTS applications must never share or connect to `postgres-core`.
4. **Simulation Isolation (`external-sim`):** Mock 3rd-party services (e.g., Mock Broker API, Mock Bank API, Mock Hardware) must run strictly in `external-sim`. They use stateless in-memory persistence and must never connect directly to either PostgreSQL database or access internal application secrets.
5. **No Legacy `apps` or Standalone `postgres` Namespaces:** The legacy `apps` and `postgres` namespaces are deprecated and deleted. Never create or reintroduce Kubernetes manifests targeting `namespace: apps` or `namespace: postgres`.
6. **Gateway API, Registry & Cluster Engine Conventions (MicroK8s on Ubuntu Server Reference Standard):**
   * **MicroK8s on Ubuntu Server as Primary Reference:** Orange Cat Investments uses **MicroK8s on Ubuntu Server** as the primary Kubernetes reference engine.
   * **Kubernetes Gateway API (`HTTPRoute` / `GRPCRoute`):** Legacy `networking.k8s.io/v1` `Ingress` manifests are forbidden. All edge routing uses Envoy Gateway (`GatewayClass/oci-gateway-class`, `Gateway/oci-edge-gateway` in `k8s/ha/lan/tls/gateway.yaml`) with namespace-scoped `HTTPRoute` / `GRPCRoute` manifests (`k8s/*/http-routes.yaml`).
   * **In-Cluster Enterprise Container Registry (`:32000`):** Images are hosted via Harbor (`k8s/registry/harbor.yaml`) on NodePort `32000` / `registry.oci.local` (preserving `localhost:32000/<image>:<tag>`), or the MicroK8s registry addon when active, with Trivy vulnerability scanning, CycloneDX SBOMs, and Cosign signatures enforced by Kyverno in `oci-core`.
   * **Dedicated Workload Storage:** Workloads default to standard persistent storage (e.g., MicroK8s hostpath storage `microk8s enable hostpath-storage` / `microk8s-hostpath` StorageClass, or Longhorn `longhorn` in multi-node HA). When available, configure storage paths so persistent volume claims land on dedicated persistent storage.

### Event Protocol Standard
All domain messages emitted across RabbitMQ must be wrapped in the standard Go `envelope.EventEnvelope[T]` with UUIDv7 `EventID`, UTC timestamp `OccurredAt`, and OTel context `CorrelationID`.

### Go Monorepo & gopls MCP Toolchain Standards
When developing, refactoring, or navigating Go code across the OCI monorepo, autonomous agents should leverage the connected `gopls` Model Context Protocol (MCP) server tools:

1. **Absolute Paths Mandatory:**
   All gopls tools accepting file or directory arguments (`file`, `files`, `dir`) **must** receive absolute paths (e.g., `/home/ramjac/orange-cat-investments/cmd/worker/main.go`). Relative paths will cause resolution errors in `gopls`.

2. **Workspace & Dependency Inspection:**
   - Use `go_workspace` to identify active module roots and workspace layout when starting work in the Go workspace.
   - Run `go_vulncheck` whenever `go.mod` or `go.sum` dependencies are modified, or during security evaluations, to identify known vulnerabilities.

3. **Symbol Navigation & Package Understanding:**
   - **Fuzzy Symbol Search (`go_search`):** Search for types, structs, interfaces, methods, or functions across the entire monorepo without guessing file locations.
   - **Intra-Package Dependencies (`go_file_context`):** When opening an unfamiliar Go file, use `go_file_context` to inspect declarations and intra-package cross-file relationships.
   - **Exported Package Contracts (`go_package_api`):** Query public package APIs (both internal packages and third-party dependencies) to understand their exported types and signatures without reading every source implementation.

4. **Refactoring & Symbol Integrity:**
   - **Call Site Discovery (`go_symbol_references`):** Before changing or deleting any function signature, struct field, or interface method, use `go_symbol_references` to inspect all call sites and evaluate blast radius.
   - **Workspace Renames (`go_rename_symbol`):** For renaming identifiers across packages, use `go_rename_symbol` to compute safe workspace-wide edits rather than manual find-and-replace.
   - **Protobuf Boundary Reminder:** In accordance with Rule 7, never edit generated `*.pb.go` or `*.pb.gw.go` files even if gopls locates symbols in them. Make changes to the relevant `.proto` files and regenerate code.

5. **Iterative Diagnostics & Rapid Test Inner Loop:**
   - Immediately after editing any Go file, call `go_diagnostics` passing the absolute paths of modified files. Resolve any compilation, type, or lint errors reported before running tests.
   - During iterative development, execute tests only for the impacted packages (`go test -race ./pkg/<name>/...`) to maintain fast feedback loops.
   - Monorepo-wide test runs (`go test -v -race ./...`) should be executed during final verification prior to creating pull requests.

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
3. **Go Code Quality, Diagnostics & Race Detection:**
   - Run `go_diagnostics` across all modified Go files to verify zero compilation, type check, or linter errors.
   - Run full test suite with race detector enabled:
   ```bash
   go test -v -race ./...
   go vet ./...
   ```
   - If dependencies in `go.mod` or `go.sum` were touched, run vulnerability scanning:
   ```json
   go_vulncheck({"pattern": "./..."})
   ```
4. **Platform Simulation Driver Validation:**
   Run the simulation test suite:
   ```bash
   go test -v ./simulation/...
   ```
5. **Frontend Theme & Design Token Compilation:**
   Verify `scripts/generate-theme.js` compiles CSS variables from `config/branding.yaml`:
   ```bash
   node scripts/generate-theme.js
   ```
6. **General Rule:** Always inspect modified files after editing to verify formatting, content accuracy, and absence of syntax errors.
7. **Protocol Buffers** Only use the .proto files. Ignore the generated *.pb.go and *.pb.gw.go files.
8. **Pull Requests** If a pull request is merged, then delete the branch. If it is closed, leave the branch.
9. **Git branch protection** Never commit changes on main. Changes should only happen on main by merging Pull Requests. If you have changes and main is checked out, then make a new feature branch.

---

## 5. Local Server & Target Testing Environments

* **Local Private Server Notes (`local-server-notes.md`):**
  When performing integration tests, cluster provisioning, or remote verification on dedicated local testing hardware, always check for the presence of `local-server-notes.md` in the repository root.
  * This file is gitignored to keep private hostnames, SSH user details, network specifics, and cluster configurations private.
  * Agents should inspect `local-server-notes.md` first to obtain connectivity details, active container/Kubernetes runtime specifications (e.g. MicroK8s, Docker), and local execution parameters before executing commands or tests against local infrastructure.
