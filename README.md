# Orange Cat Investments (OCI) - SMB Self-Hosted Platform

An end-to-end architecture specification, domain-driven design blueprint, and structured prompting framework for building a production-grade, self-hosted small-to-medium business platform.

---

## 1. Business & Application Domains (Domain-Driven Design)

Orange Cat Investments (OCI) operates across two primary operational planes:

### Core Business Subdomain (External & Revenue-Generating)
The primary business engine that ingests feline behavioral data and automates investment portfolios:
* **Orange Observer:** Ingests camera streams, detects orange cat activities, and publishes observation events.
* **Orange Investor & Investment Management API:** Consumes observation events, executes investment/divestment strategies, and handles fund allocations.
* **Customer Accounts & Brokerage API:** Handles customer deposits, withdrawals, balances, and customer-facing portfolio readouts.
* **Surfaces:** Public marketing site (Hugo), Customer Portal (Vue SPA via Go BFF).

### Supporting & Operations Subdomains (Internal Business Operations)
Internal operational infrastructure supporting the business and its physical assets:
* **Facilities & Hardware Assets (Supporting):** Tracks physical hardware (edge cameras, observation perches, networking) and cat habitat maintenance. Field maintenance staff scan and service these devices.
* **Workforce Management (Generic):** Unified management for both human staff and feline employees (onboarding, perks, care schedules, performance/behavioral reviews, role-based access).
* **Internal Operations & Helpdesk (Generic):** Ticket ingestion from Forgejo issues, internal alerts, and staff CLI/wearable tools.
* **Surfaces:** Employee Access CLI (Cobra), Field Maintenance Mobile App (Flutter), On-Call Watch App (Pebble).

---

### Off-the-Shelf OSS vs. Custom Code Boundary Rules

1. **System of Record for Generic Operations:** ERPNext and Frappe HR are the single source of truth for double-entry bookkeeping, tax accounting, legal payroll processing, standard customer billing, and core HR workflows. Homebox is the source of record for non-networked office physical inventory.
2. **Custom Go Domain Responsibilities:** Custom Go microservices are reserved strictly for:
   * **OCI Core Domain:** Camera stream processing, observation event ingestion, automated trading algorithms, and real-time investor fund balancing.
   * **OCI Hardware Operations:** Edge camera rigs, observation perches, and custom telemetry that off-the-shelf software cannot model natively.
3. **Integration Mechanism:** Custom services never duplicate ERP database tables. Go services communicate with ERPNext/Frappe HR via REST APIs or asynchronous webhooks (e.g., an investment dividend event in the custom engine emits a webhook that logs an accounting ledger entry in ERPNext).

---

## 2. Application Architecture & Communication Boundaries

The application architecture follows Domain-Driven Design (DDD) principles with a clean microservice separation:

* **Domain APIs & Persistence:** Each domain is represented by a single Domain API sitting in front of its own persistence layer (typically PostgreSQL schemas). Domain APIs serve persisted data and enforce entity integrity. Domain APIs **never** call each other directly.
* **Business Logic Layer (BLL):** BLL services and asynchronous workers orchestrate interactions across multiple Domain APIs into cohesive workflows, enforcing business rules and cross-domain transactions.
* **Backend-For-Frontend (BFF):** Web and mobile frontends connect strictly to a dedicated Go BFF API. The BFF manages user authentication (OIDC/sessions), authorization, request aggregation, and API response shaping tailored to frontend requirements. The BFF can only invoke BLL services.
* **Inter-Service Protocols & Security:**
  * **Frontend $\rightarrow$ BFF:** HTTPS REST with secure, HTTP-only opaque session cookies and CSRF protection.
  * **BFF $\rightarrow$ BLL $\rightarrow$ Domain APIs:** gRPC over Mutual TLS (mTLS). Client whitelists and role-based interceptors restrict call permissions at each layer.

### Reference Implementations
* [ThreeDotsLabs - Wild Workouts Go DDD Example](https://github.com/ThreeDotsLabs/wild-workouts-go-ddd-example)
* [ThreeDotsLabs - Practical DDD in Go](https://threedots.tech/post/)

---

## 3. Core Technology Stack

* **Orchestration & Hosting:** Kubernetes (K3s). Adheres strictly to [12-Factor App](https://12factor.net/) principles.
* **Infrastructure Provisioning:** Ansible for initial host setup and Kubernetes cluster bootstrapping.
* **Source Control & CI/CD:** [Forgejo](https://forgejo.org) for Git repository hosting, issue tracking, Kanban boards, CI/CD pipelines via Forgejo Actions, internal OAuth2 authentication, and package registry. [Renovate](https://github.com/renovatebot/renovate) for dependency pinning.
* **Customer Identity Provider:** [ZITADEL](https://github.com/zitadel/zitadel) for customer OIDC/SSO authentication.
* **Office Productivity:** Nextcloud for documents, calendars, contacts, chat, and team collaboration.
* **ERP & HR System:** ERPNext and Frappe HR for accounting, double-entry bookkeeping, inventory, CRM, HR, payroll, asset tracking, and billing.
* **Office Asset Tracking:** [Homebox](https://homebox.software/en/) for non-networked office inventory.
* **Databases:** PostgreSQL 16 (Relational DB with JSONB for document-store needs).
* **Caching & Session Storage:** Valkey.
* **Event Streaming & Queuing:** RabbitMQ for asynchronous messaging and pub/sub.
* **Search Engine:** Meilisearch.
* **Metrics & Monitoring:** Prometheus + Grafana.
* **Log Aggregation:** Loki + Promtail.
* **Uptime Monitoring:** Uptime Kuma.
* **Alerting:** Alertmanager integrated with `ntfy` for push notifications.
* **Frontend Applications:** Vue.js (SPA), Hugo (Static Marketing Site).
* **Backend Services & Tooling:** Go (1.22+). Go with [cdk8s](https://cdk8s.io/) for Helm and Kubernetes manifest generation.
* **Configuration & Glue:** YAML for Kubernetes manifests and Forgejo Actions; Bash for setup scripts; HCL Terraform for cloud infrastructure fallbacks.
* **Repository Strategy:** Single Monorepo pattern.

---

### Go Application Architecture & Libraries

* **BFF Framework:** `net/http` + [`go-chi/chi`](https://github.com/go-chi/chi).
* **Internal Microservices:** gRPC via [`grpc-go`](https://github.com/grpc/grpc-go).
* **Validation:** [`go-playground/validator`](https://github.com/go-playground/validator).
* **Testing:** [`testify`](https://github.com/stretchr/testify) for assertions/mocks and [`mockery`](https://github.com/vektra/mockery) for interface mocking.
* **HTTP Client:** [`resty`](https://resty.dev/) with retry policies.
* **Observability & Tracing:** [OpenTelemetry Go SDK](https://github.com/open-telemetry/opentelemetry-go).
* **Event Messaging:** [`watermill`](https://github.com/ThreeDotsLabs/watermill) for event routing over RabbitMQ.
* **Database Access:** [`sqlc`](https://github.com/sqlc-dev/sqlc) with [`pgx/v5`](https://github.com/jackc/pgx) driver and [`goose`](https://github.com/pressly/goose) for migrations.
* **Configuration:** [`koanf`](https://github.com/knadh/koanf).
* **CLI Applications:** [`cobra`](https://github.com/spf13/cobra) and [`color`](https://github.com/fatih/color).
* **Certificate Lifecycle:** [`cert-manager`](https://cert-manager.io) and [`trust-manager`](https://cert-manager.io/docs/projects/trust-manager/).

#### Standard Event Envelope
```go
type EventEnvelope[T any] struct {
    EventID       string    `json:"event_id"`       // UUIDv7
    EventType     string    `json:"event_type"`     // e.g. "observation.cat_spotted.v1"
    OccurredAt    time.Time `json:"occurred_at"`    // UTC timestamp
    CorrelationID string    `json:"correlation_id"` // Tracing ID passed from context
    Payload       T         `json:"payload"`        // Typed domain payload
}
```

#### Protocol Buffer Contract Management
* **Tooling:** `buf` (v1/v2) for Protobuf schema linting, breaking change detection, and stub generation using `protoc-gen-go` and `protoc-gen-go-grpc`.
* **Directory Layout:**
```text
proto/
└── oci/
    ├── core/
    │   ├── observation/v1/observation.proto
    │   └── investment/v1/investment.proto
    └── ops/
        ├── facilities/v1/facilities.proto
        └── workforce/v1/workforce.proto
```
* **Generated Output:** Compiled stubs are placed in `pkg/gen/go/...`.

---

## 4. Authentication & Security Patterns

### PKI Architecture

* **Cert-Manager & Trust-Manager:** `cert-manager` manages X.509 certificate issuance; `trust-manager` distributes public CA bundles across all Kubernetes namespaces via ConfigMaps.
* **Two-Tier PKI:** Offline Root CA (generated once manually, stored off-cluster) signs an online intermediate signing CA managed in-cluster by `cert-manager`.
* **Leaf Workload Certificates:** 7-day lifetime, 3-day `renewBefore`, `rotationPolicy: Always`, ECDSA P-256 keys, SPIFFE ID in URI SAN (e.g., `spiffe://oci.local/ns/{namespace}/sa/{serviceaccount}`).
* **Mounting Standards:** Mounted as full Kubernetes Secret directory mounts. `subPath` mounts are strictly forbidden for TLS materials.
* **Cluster & Public Certificates:**
  * K3s cluster internal certs: Auto-renewed on restart. Recommended node reboot schedule: every 60 days.
  * Ingress TLS (Portal, Forgejo, API hostnames): ACME / Let's Encrypt or internal CA with automatic 90-day renewals.

#### Workload PKI Invariants
1. No service ever holds a static, long-lived certificate.
2. Every service reloads certificates in-process using `crypto/tls` callbacks (`GetCertificate` / `GetClientCertificate`) without requiring pod restarts.
3. Certificate mounts must be full Secret directory mounts (no `subPath`).
4. Fail fast: services must refuse to boot if valid, unexpired certificate material cannot be loaded.
5. Any rotation step requiring human action must be explicitly documented with its operational cadence (once-ever, once-per-environment, or recurring).

---

### Customer-Facing Auth (OIDC + BFF)

```text
Browser (Vue)              BFF (Go)                 ZITADEL (OIDC)
    │                          │                          │
    │ 1. GET /                 │                          │
    ├─────────────────────────►│                          │
    │ 401 + no session cookie  │                          │
    │◄─────────────────────────┤                          │
    │                          │                          │
    │ 2. navigate to /auth/login (top-level, not fetch)   │
    ├─────────────────────────►│                          │
    │ 3. 302 → /authorize?     │                          │
    │    response_type=code    │                          │
    │    code_challenge=S256   │                          │
    │    state, nonce          │                          │
    │◄──────────────────────────┼──────────────────────────┤
    ├───────────────────────────┼─────────────────────────►│
    │ 4. user logs in (password / TOTP / passkey)          │
    │◄──────────────────────────┼──────────────────────────┤
    │ 5. 302 → /auth/callback?code=...&state=...           │
    ├─────────────────────────►│                          │
    │                          │ 6. POST /token           │
    │                          │    code + code_verifier  │
    │                          ├─────────────────────────►│
    │                          │    id_token, access_token│
    │                          │◄─────────────────────────┤
    │                          │ 7. validate id_token     │
    │                          │    (JWKS, iss, aud, exp) │
    │                          │ 8. create server session │
    │                          │    in Valkey             │
    │ 9. Set-Cookie: session=opaque; HttpOnly; Secure;    │
    │    SameSite=Lax; Path=/                             │
    │◄─────────────────────────┤                          │
    │                          │                          │
    │ 10. GET /api/v1/me (cookie auto-sent)               │
    ├─────────────────────────►│ 11. lookup session in Valkey
    │                          │ 12. gRPC + mTLS, downscoped
    │                          │     context → BLL        │
    │ 13. shaped JSON          │                          │
    │◄─────────────────────────┤                          │
```

#### Key Implementation Details
* **Top-Level Redirection:** Authentication initiation (Step 2) must be a top-level window navigation, never an XHR/fetch request.
* **No JWTs in SPA:** Access/ID tokens are stored exclusively server-side in Valkey. The browser receives only an opaque session cookie (`__Host-session`), eliminating XSS token theft vectors.
* **Go BFF Libraries:** `coreos/go-oidc/v3` for OIDC verification, `golang.org/x/oauth2` for authorization code exchange, and `alexedwards/scs/v2` (Valkey store) for session management.
* **BFF Middleware Chain:** `RequestID` $\rightarrow$ `RealIP` $\rightarrow$ `Recoverer` $\rightarrow$ `Logger(slog)` $\rightarrow$ `OTel` $\rightarrow$ `SessionLoad` $\rightarrow$ `CSRF` $\rightarrow$ `RequireAuth` $\rightarrow$ `Authz` $\rightarrow$ `Handler`.
* **Vue Frontend Integration:** Pinia auth store hydrated from `GET /api/v1/me`. Standard HTTP client with `credentials: 'include'` and CSRF header. No OAuth JavaScript libraries in the frontend.

#### Hardening Checklist
| Security Concern | Mitigation Action |
| :--- | :--- |
| **CSRF** | Double-submit cookie + `X-CSRF-Token` header validation; `SameSite=Lax` cookie attribute. |
| **Session Fixation** | Rotate session ID on privilege level changes (login, MFA step-up, password change). |
| **Cookie Scope** | `__Host-session` prefix, `Path=/`, `HttpOnly`, `Secure`, no `Domain` attribute. |
| **Session Lifetime** | 30-minute idle timeout, 12-hour absolute maximum TTL enforced in Valkey. |
| **Token Refresh** | Handled silently on the server between BFF and ZITADEL. Never exposed to browser. |
| **Logout** | Destroy Valkey session $\rightarrow$ revoke ZITADEL tokens $\rightarrow$ perform OIDC RP-Initiated Logout. |
| **Content Security Policy** | Strict nonce-based CSP headers on all HTML responses. |
| **Rate Limiting** | Strict IP and session rate limiting on `/auth/*` endpoints to prevent brute force attacks. |
| **Audit Logging** | All session creations, refreshes, and revocations emitted as structured logs to Loki. |

---

### Internal Inter-Service Auth

* **Workload Identity:** Every service runs under a dedicated Kubernetes ServiceAccount mapped to a SPIFFE ID (`spiffe://oci.local/ns/{namespace}/sa/{serviceaccount}`).
* **Transport Encryption:** All gRPC connections require mTLS with `ClientAuth: tls.RequireAndVerifyClientCert`.
* **Authorization Interceptor:** A gRPC unary/stream interceptor extracts the caller's SPIFFE ID from the peer certificate URI SAN and validates it against an allowed client whitelist and role-policy matrix.

---

## 5. Operations & Human Intervention Guidelines

When an operational or setup procedure requires human intervention, it must be documented with explicit instructions specifying:
1. **Pre-requisites:** Credentials, access roles, and tools required before starting.
2. **Frequency Classification:**
   * **Once-Ever:** Initial cluster setup or root CA key generation.
   * **Once-Per-Environment:** Provisioning a new staging/production cluster or external IdP tenant.
   * **Recurring:** Annual root CA renewals, manual emergency cert rotations, or annual audit reviews.
3. **Step-by-Step Verification:** Commands to verify that the operation succeeded without breaking running workloads.

---

# OCI Platform Prompting Framework & Implementation Plan

A multi-tier platform spanning infrastructure, Go domain services, background workers, web portals, mobile apps, and wearable devices cannot be generated in a single AI prompt. To prevent context truncation and architectural drift, use this **Contract-First, Phased Prompting** sequence.

---

## Architecture Flow Map

```text
[Phase 1: Architecture & Contracts]
       │
       ├──► [Phase 2: Data & Core Infrastructure (K8s/Postgres)]
       │
       ├──► [Phase 2.5: Workload PKI & Certificate Lifecycle]
       │
       ├──► [Phase 3: Go Domain Engine (APIs)]
       │
       ├──► [Phase 4: Go Async Worker Engine (Queues/Timers)]
       │
       ├──► [Phase 5: Web Surfaces (Public Static + Logged-in Portals)]
       │
       ├──► [Phase 6: External Clients (Mobile & Pebble App)]
       │
       ├──► [Phase 7: OSS Integration (Forgejo, SSO, Webhooks)]
       │
       └──► [Phase 8: Automation of Business Processes (Workforce, Onboarding, Separation)]
```

---

## Master Prompt Sequences

### Phase 1: Master Specification & API Contracts

#### Prompt 1.1: System Architecture Design
> **Prompt:**
> "Act as a Principal Enterprise Architect. I am designing a self-hosted business application platform for Orange Cat Investments (OCI).
> - **Existing OSS:** Forgejo (source/CI), PostgreSQL 16, ZITADEL (OIDC/SSO), RabbitMQ (event bus/queue), K3s (orchestration), Frappe HR / ERPNext (ERP/Payroll).
> - **Custom Go Services:** Go REST BFF API for web/mobile, gRPC Domain/BLL microservices, Go asynchronous background worker service.
> - **Client Surfaces:** Static marketing site (Hugo), secure authenticated customer/employee web portal (Vue SPA), field maintenance mobile app (Flutter), and an on-call watch app (Pebble).
> - **Business Domains:** Core Investment Engine, Orange Observer (Cat activity ingestion), Facilities & Hardware Assets, Workforce Management (Human & Feline), IT Helpdesk.
>
> Produce an Architectural Decision Record (ADR) detailing:
> 1. Domain boundary decomposition mapping domains to PostgreSQL schemas.
> 2. Event-driven architecture between Go REST BFF, gRPC BLL services, and Go Worker service.
> 3. Unified authentication and authorization model leveraging OIDC tokens, opaque server-side Valkey sessions, and SPIFFE mTLS identities across internal services."

#### Prompt 1.2: OpenAPI & Database Schema Definitions
> **Prompt:**
> "Based on our architecture, write a PostgreSQL 16 initialization script (`init.sql`) defining schemas and tables for:
> 1. `workforce` (human staff and feline employees, onboarding status, care schedules, review cycles)
> 2. `facilities` (hardware assets, edge cameras, observation perches, maintenance tickets, locations)
> 3. `core_invest` (cat observation events, investment allocation strategies, portfolio snapshots)
> Include foreign key constraints, `UUIDv7` primary keys, `TIMESTAMPTZ` audit timestamps, and row-level status constraints. Next, generate a corresponding OpenAPI 3.1 YAML specification covering the CRUD endpoints for Asset Tracking and Employee Onboarding/Leave Requests."

---

### Phase 2: Infrastructure & Foundation Setup

#### Prompt 2.1: Kubernetes Local Environment Setup
> **Prompt:**
> "Provide a set of clean, modular Kubernetes manifests (or a `kustomization.yaml` layout) targeted for a local K8s cluster (K3s) containing:
> 1. PostgreSQL 16 StatefulSet with PersistentVolumeClaim and an init configmap running our schema.
> 2. RabbitMQ deployment for message queuing and pub/sub.
> 3. Forgejo deployment integrated with PostgreSQL backend.
> 4. Ingress configuration routing traffic to services based on hostnames (`forgejo.oci.local`, `api.oci.local`, `portal.oci.local`)."

---

### Phase 2.5: Workload PKI & Certificate Lifecycle

#### Prompt 2.5: Workload PKI and Certificate Lifecycle
> **Prompt:**
> "Act as a Platform Security Engineer. We run a self-hosted K3s cluster hosting Go microservices communicating over gRPC. Implement internal mTLS using cert-manager and trust-manager.
> Requirements:
> 1. **PKI Manifests:** Create a ClusterIssuer for an internal Intermediate CA (signed by an offline Root CA). Use trust-manager to distribute the public CA bundle ConfigMap (`oci-ca-bundle`) across all application namespaces.
> 2. **Service Certificates:** Provide a baseline Certificate resource template for Go workloads (7-day duration, 3-day renewal window, ECDSA P-256, URI SAN formatted as `spiffe://oci.local/ns/{namespace}/sa/{serviceaccount}`).
> 3. **Go TLS Configuration:** Write a reusable Go helper package `pkg/mtls` that configures standard library `crypto/tls` for gRPC servers and clients:
>    - Configure `tls.Config.GetCertificate` and `tls.Config.GetClientCertificate` to reload certificate material dynamically from mounted directory paths without pod restarts.
>    - Load the trust-manager root CA bundle to enforce `ClientAuth: tls.RequireAndVerifyClientCert`.
> 4. **Interceptors:** Provide a gRPC unary interceptor that extracts the SPIFFE ID from the peer certificate's URI SAN and validates it against an allowed client caller whitelist."

---

### Phase 3: Custom Go Domain Services (REST BFF & gRPC BLL)

#### Prompt 3.1: API Boilerplate & Domain Layer
> **Prompt:**
> "You are a Senior Go Engineer. Build the Go REST BFF service (`cmd/bff`) using `net/http` and `go-chi/chi`, and internal gRPC services (`cmd/domain-facilities`).
> Requirements:
> - Follow clean architecture (`cmd/`, `internal/domain/`, `internal/service/`, `internal/repository/`, `internal/handler/`).
> - Implement the `facilities` domain: Hardware Asset Tracking (`Asset` entity, `CreateAsset`, `GetAssetByID`, `ListAssets`, `ReportMaintenance`).
> - Use `pgx/v5` for PostgreSQL connection pooling and `sqlc` generated queries.
> - Implement OIDC authentication middleware validating sessions stored in Valkey (issued via ZITADEL).
> - Configure gRPC clients using the `pkg/mtls` package for mTLS inter-service calls.
> - Provide unit tests for the service layer using `testify` and standard interfaces."

---

### Phase 4: Custom Go Async Workers & Schedulers

#### Prompt 4.1: Queue Worker & Timer-Triggered Engine
> **Prompt:**
> "Create a standalone Go background worker service (`cmd/worker`) designed to run alongside domain services.
> It must demonstrate two patterns:
> 1. **Message Consumer:** Subscribe to a RabbitMQ topic (`events.observation.cat_spotted.v1`) using Watermill to trigger automated investment allocation checks.
> 2. **Cron/Timer Runner:** Use Go ticker pools or Kubernetes CronJobs to scan the `facilities` table every midnight for hardware edge cameras requiring preventive maintenance and publish notification events.
> Include graceful shutdown handling (`os.Interrupt`, `SIGTERM`), structured logging using `log/slog`, and worker concurrency limiting."

---

### Phase 5: Web Surfaces

#### Prompt 5.1: Static Marketing Website
> **Prompt:**
> "Generate a static marketing landing page for Orange Cat Investments using Hugo. Showcase OCI investment philosophy based on feline activity observation, public performance metrics, announcements, uptime indicators, and a 'Customer Portal Sign In' link pointing to `/auth/login`."

#### Prompt 5.2: Authenticated Web Portal (Vue SPA)
> **Prompt:**
> "Build an authenticated web portal SPA in Vue 3 (Composition API, TypeScript, Pinia) for Customer Accounts and Staff Workforce Management.
> - Connect to the Go BFF API with `credentials: 'include'` and CSRF headers.
> - Implement Pinia stores for user state and facilities asset management.
> - Include route guards based on user roles and handle loading, error, and optimistic UI updates."

---

### Phase 6: Mobile & Pebble Watch Applications

#### Prompt 6.1: Mobile Application (Flutter)
> **Prompt:**
> "Write a Flutter (Dart) mobile application for field technicians performing hardware asset maintenance in cat habitats.
> - Features: Barcode/QR code camera scanner for asset serial numbers, asset detail screen fetching from `GET /api/v1/facilities/assets/{id}`, and an offline-first action queue to record maintenance logs when connectivity is restored."

#### Prompt 6.2: Pebble Watch App (C / Pebble C SDK)
> **Prompt:**
> "You are an embedded developer targeting Pebble OS using the Pebble C SDK.
> Create a lightweight Pebble watch app for on-call facilities engineers that:
> 1. Displays an alert summary window: 'Urgent Habitat Alerts: X'.
> 2. Uses the Pebble AppMessage API to request updated alert counts from a companion mobile proxy endpoint (`GET /api/v1/ops/alerts/summary`).
> 3. Configures the 'Select' button handler to send an ACK ('Acknowledge Alert') back through AppMessage to the Go API.
> Provide complete C code with event handlers for `app_message` and window lifecycle management."

---

### Phase 7: OSS Integration & Event Hooks

#### Prompt 7.1: Forgejo Webhook Receiver in Go
> **Prompt:**
> "Write an HTTP handler in the Go REST API that validates and processes incoming Forgejo Webhooks (`push`, `issues`, `pull_request`).
> - When an issue with label `bug` or `internal-it` is opened in Forgejo, ingest it into the custom IT helpdesk database table (`ops.it_tickets`).
> - Verify the Forgejo HMAC signature header (`X-Forgejo-Signature`) using a configured secret before processing."

---

### Phase 8: Automation of Business Processes

#### Prompt 8.1: Automated Onboarding Workflow (Human & Feline Employees)
> **Prompt:**
> "Act as a Backend Workflow Architect. Design an automated employee onboarding worker in Go that orchestrates new hires across Frappe HR, ZITADEL, Go Workforce Domain API, and Forgejo.
> Requirements:
> 1. **Trigger:** Listen for `events.workforce.employee_created.v1` published when a candidate is marked 'Hired' in Frappe HR.
> 2. **Orchestration Steps:**
>    - Create user identity in ZITADEL via REST API with initial role assignment (`role:employee` for humans, `role:feline_asset` for cats).
>    - Provision account in Forgejo with team membership based on department.
>    - Create record in Go `workforce` schema with onboarding checklist items (hardware allocation, dietary schedule for felines, access badge issuance).
>    - Send welcome email / notification payload via `ntfy` to HR managers.
> 3. **Error Handling & Compensation:** Implement saga pattern with retry logic and compensating actions if provisioning in ZITADEL or Forgejo fails."

#### Prompt 8.2: Automated Performance & Care Review Cycles
> **Prompt:**
> "Write a scheduled Go background task (`cmd/worker`) that manages recurring performance reviews for human staff and health/care reviews for feline employees.
> Requirements:
> 1. **Cron Schedule:** Runs on the 1st of every month at 01:00 UTC.
> 2. **Logic:**
>    - Query `workforce.employees` for records where `next_review_due <= NOW()`.
>    - Distinguish between human employees (trigger Frappe HR Appraisal Form creation via REST API) and feline employees (create Habitat Health & Care Assessment ticket in Go `facilities` schema).
>    - Generate Forgejo issues in the `workforce-operations` repository assigned to department managers or veterinary caretakers.
>    - Publish `events.workforce.review_scheduled.v1` event to RabbitMQ."

#### Prompt 8.3: Employee Separation & Asset Revocation Workflow
> **Prompt:**
> "Design an offboarding and access revocation engine in Go triggered upon employee termination or feline retirement.
> Requirements:
> 1. **Trigger:** Webhook or event `events.workforce.employee_separated.v1` originating from Frappe HR.
> 2. **Immediate Execution Actions (Atomic Saga):**
>    - Suspend ZITADEL user account and invalidate all active Valkey sessions immediately.
>    - Remove user SSH/GPG keys and disable account in Forgejo.
>    - Query `facilities.assets` for all assigned physical devices (laptops, edge cameras, smart collars) and set asset status to `pending_return`.
>    - Create an urgent return task for facilities staff in the mobile app queue.
> 3. **Audit Trail:** Write an immutable audit log entry to `workforce.separation_audit_logs` capturing timestamp, executor ID, and status of every revoked credential."

---

## 6. Best Practices for Google AI Tools

| Tool | Recommended Workflow |
| :--- | :--- |
| **Project IDX** | Open the repo root. Enable workspace context in the AI chat pane so Gemini reads `go.mod`, SQL schema files, and Protobuf specs directly. |
| **Gemini Code Assist** | Highlight gRPC interface declarations or Go struct definitions in your IDE and request handlers or unit tests that conform strictly to the interfaces. |
| **Vertex AI Studio / Gemini 1.5 Pro** | Provide the full OpenAPI specification and PostgreSQL schema in the system prompt to maintain a unified context window when generating complex cross-domain code. |
