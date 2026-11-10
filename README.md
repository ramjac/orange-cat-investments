# Orange Cat Investments (OCI) - SMB Self-Hosted Platform

An end-to-end architecture specification, domain-driven design blueprint, and production-grade implementation for a self-hosted small-to-medium business (SMB) platform.

---

## 1. Business & Application Domains (Domain-Driven Design)

Orange Cat Investments (OCI) operates across two primary operational planes:

### Core Business Subdomain (External & Revenue-Generating)
The primary business engine that ingests feline behavioral telemetry and automates algorithmic investment portfolios:
* **Orange Observer (`core_invest`):** Ingests live optical camera streams (RTSP/WebRTC), detects feline activities (zooming, napping, perched, eating, grooming, playful pouncing), and publishes observation events.
* **Orange Investor & Investment Management API:** Consumes observation events via event queues, calculates algorithmic portfolio allocations, and executes automated brokerage orders.
* **Backtesting Workbench:** Dynamic backtesting engine comparing historical feline behavioral telemetry against equity asset performance.
* **Surfaces:** Public marketing site (Hugo), Customer Portal (Vue SPA via Go BFF).

### Supporting & Operations Subdomains (Internal Business Operations)
Internal operational infrastructure supporting the business, its personnel, and physical habitat hardware:
* **Facilities & Hardware Assets (`facilities`):** Tracks physical hardware (edge cameras, observation perches, smart collars, feeders, gateways) and cat habitat maintenance. Includes an offline-first SQLite sync engine for field technicians and an Over-the-Air (OTA) firmware release pipeline.
* **Workforce Management (`workforce`):** Unified management for both human staff and feline executives (`emp-feline-*`). Implements automated onboarding/offboarding sagas, dietary protocols, feeding times, and emergency medical trading holds.
* **Internal Operations & Helpdesk (`ops`):** IT ticket ingestion from Forgejo repository issues, on-call alert routing, and Pebble smartwatch companion wearable proxy.
* **PKI & CA Key Rotation (`internal/service/pki`):** Automated intermediate CA renewal and certificate re-issuance worker through cert-manager without human intervention.
* **Surfaces:** Employee Web Portal (Vue 3 SPA), Field Maintenance Mobile App (Flutter), Wearable Gateway Proxy (Pebble).

---

### Off-the-Shelf OSS vs. Custom Code Boundary Rules

1. **System of Record for Generic Operations:** ERPNext and Frappe HR are the single source of truth for double-entry bookkeeping, tax accounting, legal payroll processing, standard customer billing, and core HR workflows. Snipe-IT is the source of record for non-networked office physical inventory.
2. **Custom Go Domain Responsibilities:** Custom Go microservices are reserved strictly for:
   * **OCI Core Domain:** Camera stream processing, observation event ingestion, automated trading algorithms, and real-time investor fund balancing.
   * **OCI Hardware Operations:** Edge camera rigs, observation perches, and custom telemetry that off-the-shelf software cannot model natively.
3. **Integration Mechanism:** Custom services never duplicate ERP database tables. Go services communicate with ERPNext/Frappe HR via REST APIs or asynchronous webhooks (e.g., an onboarding event in Frappe HR emits a webhook to `/api/v1/webhooks/frappe-hr` that executes the distributed `OnboardingSaga`).

---

## 2. Application Architecture & Communication Boundaries

The application architecture follows Domain-Driven Design (DDD) principles with clean schema and microservice separation:

* **Domain APIs & Persistence:** Each domain accesses its own designated PostgreSQL schema (`core_invest`, `facilities`, `workforce`, `ops`). Direct cross-schema table queries or cross-Domain API calls are prohibited.
* **Business Logic Layer (BLL):** BLL services and asynchronous background workers (`cmd/worker`) orchestrate interactions across multiple Domain APIs into cohesive workflows, enforcing business rules and cross-domain transactions.
* **Backend-For-Frontend (BFF):** Web and mobile frontends connect strictly to dedicated Go BFF APIs (`cmd/customer-bff` and `cmd/employee-bff`). The BFF manages user authentication (OIDC/sessions), authorization, request aggregation, and API response shaping tailored to frontend requirements.
* **Inter-Service Protocols & Security:**
  * **Frontend $\rightarrow$ BFF:** HTTPS REST with secure, HTTP-only opaque session cookies and double-submit CSRF protection (`X-CSRF-Token`). No raw JWTs in SPAs.
  * **BFF $\rightarrow$ BLL $\rightarrow$ Domain APIs:** gRPC over Mutual TLS (mTLS) with dynamic certificate reloading via standard library `crypto/tls`.
  * **Asynchronous Messaging:** RabbitMQ event stream using standardized `envelope.EventEnvelope[T]` wrapping UUIDv7 `EventID`, UTC timestamp `OccurredAt`, and OTel `CorrelationID`. CWE-532 log hygiene ensures raw payloads are never logged.

---

## 3. Core Technology Stack

* **Orchestration & Hosting:** Kubernetes (MicroK8s on Ubuntu Server). Adheres strictly to [12-Factor App](https://12factor.net/) principles.
* **Infrastructure Provisioning:** Ansible for initial host setup and Kubernetes cluster bootstrapping (`ansible/`).
* **Source Control & CI/CD:** [Forgejo](https://forgejo.org) for Git repository hosting, issue tracking, Kanban boards, CI/CD pipelines via Forgejo Actions, internal OAuth2 authentication, and package registry.
* **Customer Identity Provider:** [ZITADEL](https://github.com/zitadel/zitadel) for customer OIDC/SSO authentication.
* **Office Productivity:** Nextcloud for documents, calendars, contacts, chat, and team collaboration.
* **ERP & HR System:** ERPNext and Frappe HR for accounting, double-entry bookkeeping, inventory, CRM, HR, payroll, asset tracking, and billing.
* **Office Asset Tracking:** [Snipe-IT](https://snipeitapp.com/) for non-networked office inventory.
* **Databases:** PostgreSQL 16 (Relational DB with isolated schemas `workforce`, `facilities`, `core_invest`, `ops`).
* **Caching & Session Storage:** Valkey (Redis-compatible).
* **Event Streaming & Queuing:** RabbitMQ with Watermill pub/sub router engine.
* **Search Engine:** Meilisearch.
* **Metrics & Monitoring:** Prometheus + Grafana.
* **Log Aggregation:** Loki + Promtail.
* **Frontend Applications:** Vue 3 SPAs (Employee Portal `web/employee-portal` & Customer Portal `web/customer-portal`), Hugo (Static Marketing Site).
* **Mobile Applications:** Flutter SDK (Field Maintenance App with SQLite offline sync).
* **Wearables & Embedded:** Pebble Alloy Embedded JS Companion App (`embedded/pebble`) and Go Gateway Proxy (`cmd/pebble-proxy`).
* **Backend Services & Tooling:** Go 1.22+.

### Kubernetes Namespace Architecture
The cluster workload topology is structured into dedicated namespaces for isolation and operational clarity:
* **`oci-core`:** Collocates all custom OCI applications (BFFs, BLLs, Domain APIs, `oci-pebble-proxy`, `oci-worker`, `oci-customer-portal`, `oci-employee-portal`) together with external customer identity (**ZITADEL**), core operational infrastructure (`valkey`, `rabbitmq`, `meilisearch`), and dedicated proprietary database **`postgres-core`** (`workforce`, `facilities`, `core_invest`, `ops`).
* **`oci-internal`:** Isolates off-the-shelf commercial and open-source internal business applications (COTS): Nextcloud, Forgejo, ERPNext/Frappe HR, and Snipe-IT, backed by their own isolated internal database **`postgres-internal`** (`nextcloud`, `forgejo`, `snipeit`, `erpnext`, `frappe_hr`).
* **`external-sim`:** Dedicated namespace hosting simulated 3rd-party SaaS / financial APIs (`mock-broker` on `:8089`, `mock-bank` on `:8088`), maintaining complete air-gap isolation from production persistence.
* **`monitoring` & `logging`:** Cluster observability stacks (Prometheus, Grafana, Loki, Promtail).

---

## 4. Employee & Customer Personas (Workflow Simulation)

To exercise and simulate the entire software stack using autonomous drivers and AI agents, OCI defines 12 distinct personas across corporate staff, feline executives, and retail investors:

### Corporate Staff & Feline Executives
1. **Garfield (`emp-feline-garfield`)** — Chief Observation Officer (COO)
2. **Barneby (`emp-feline-barneby`)** — Senior Alpha Perch Analyst
3. **Alice Vance (`emp-human-alice`)** — Head of Human & Feline Resources
4. **Bob Builder (`emp-human-bob`)** — Lead Facilities & Edge Telemetry Engineer
5. **Carol Danvers (`emp-human-carol`)** — Chief Investment Officer (CIO)
6. **David Quant (`emp-human-david`)** — Feline Behavioral Data Scientist
7. **Dr. Elena Rostova (`emp-human-elena`)** — Chief Veterinary Officer & Habitat Specialist
8. **Frank Operations (`emp-human-frank`)** — Platform Security & K8s Infrastructure Lead
9. **Rick Newhire (`emp-human-rick`)** — Junior Operations Associate (exercising new hire and onboarding workflows)
10. **Elise Dev (`emp-human-elise`)** — Software Developer (exercising software dev, CI/CD pipelines, and Forgejo workflows)

### Retail Investor Customers
11. **Arthur Pendelton (`cust-longterm-arthur`)** — Long-Term Value Investor (passive regular fund deposits, quarterly performance reviews)
12. **Chloe Spark (`cust-active-chloe`)** — Short-Term Momentum Alpha Trader (active tactical buy/sell orders on feline zoomies activity spikes)

Detailed profile specifications are located in the `personas/` directory.

---

## 5. Implemented Capabilities & Completed Milestones

All core functional domains, distributed sagas, client surfaces, and background pipelines have been implemented and verified:

### 5.1 Core Investment Engine & Quantitative Telemetry (`core_invest`)
* **Real-time WebRTC/RTSP Stream Ingestion:** Ingests live feline habitat video feeds registered via `/core-invest/streams` and `/api/v1/core-invest/streams`.
* **Automated Brokerage Execution Integrations:** Automated trade placement and execution logging via `/core-invest/brokerage/orders`.
* **Dynamic Backtesting Workbench:** Historical backtesting engine (`/core-invest/backtesting/runs`) evaluating feline behavioral telemetry against equity asset performance metrics.
* **Embedded Real-Time Computer Vision Inference Engine:** Frame-level activity classification scoring feeding into `core_invest.observation_events`.
* **ML Model Registry & Confidence Drift Tracking:** Model registry schema `core_invest.ml_models` and drift evaluation endpoints (`/core-invest/models`, `/core-invest/models/drift`).
* **Algorithmic Order Types & Pre-Trade Risk Engine:** Smart Order Router supporting Bracket orders, Trailing Stop-Loss, OCO, IOC, GTC, and risk management guardrails.
* **Automated Investor Statements & Tax Document Generation:** Statement generator (`/core-invest/statements/generate`) producing Sharpe/Sortino ratios and Form 1099-B tax documentation.

### 5.2 Customer Experience & Retail Investor Surfaces (`web/customer-portal` & `cmd/customer-bff`)
* **Customer Web Portal & Dedicated Go BFF:** Vue 3 Single Page Application (`web/customer-portal` on `:5174`) and dedicated Go BFF server (`cmd/customer-bff` on `:8083`) utilizing ZITADEL OIDC authentication, Valkey server-side sessions, and double-submit CSRF protection.
* **Low-Latency In-Browser WebRTC Habitat Video Player:** Embedded WebRTC stream player component with real-time AI vision detection overlays for Garfield and Barneby.
* **Active Trading Terminal & Real-Time Price Ticker:** High-velocity execution terminal in `web/customer-portal` and live price ticker endpoint (`/api/v1/customer/ticker`).
* **Live "Zoomie Index" & Behavioral Momentum Alerts:** Real-time Zoomie Index scoring endpoint (`/api/v1/customer/zoomie-index`) and 1-click tactical pounce trade execution.
* **Automated Recurring Deposits & Capital Allocations:** Schema `core_invest.recurring_deposits`, Go BFF endpoints (`/api/v1/customer/deposits`), and Vue deposit management UI.
* **Customer Developer API Keys & Webhook Subscriptions:** Developer API key creation/revocation (`/api/v1/customer/api-keys`) and event webhook subscription management (`/api/v1/customer/webhooks`).
* **Feline Welfare & ESG Transparency Dashboard:** Transparency endpoint (`/api/v1/customer/transparency/welfare`) and ESG dashboard displaying veterinary adherence, daily nap hours, and habitat comfort ratings backed by PostgreSQL queries.

### 5.3 Facilities & Mobile Field Maintenance (`facilities` & `mobile/flutter_app`)
* **Flutter Mobile Offline Database Sync:** Full local SQLite persistence (`sqflite`), QR scanner camera view, and offline queue sync engine (`SyncEngine`) posting batched logs to `/facilities/sync/maintenance-logs`.
* **Hardware Asset Management:** Keyset-paginated hardware inventory (`/facilities/assets`) supporting high-throughput edge camera fleets, habitat gateways, network switches, and mobile devices.
* **Edge Device Firmware OTA Pipeline:** Firmware release catalog (`/facilities/firmware`) and OTA job scheduling (`/facilities/firmware/ota-jobs`), executed by the background worker Watermill router (`ota_pipeline_handler`).
* **Electronic Veterinary Health Records (VHR) & Clinical Charting:** Schema `workforce.vhr_records`, Go service layer, and Employee Portal VHR charting tab.
* **Remote PTZ & Camera Lens Calibration:** Pan-Tilt-Zoom (PTZ) control and lens calibration REST endpoints (`/facilities/cameras/{id}/ptz`, `/facilities/cameras/{id}/calibrate`).
* **Habitat Environmental Telemetry (HVAC, Lux, Decibels):** Schema `facilities.environmental_telemetry` and telemetry ingestion endpoint (`/facilities/environment/telemetry`).

### 5.4 Workforce Operations & Automated Sagas (`workforce` & `web/employee-portal`)
* **Frappe HR Webhook Ingestion:** Webhook endpoint `/api/v1/webhooks/frappe-hr` triggering distributed `OnboardingSaga` and `OffboardingEngine` flows across workforce, Forgejo, and internal services.
* **Interactive Care Schedule UI:** Vue 3 Employee Portal interface for Dr. Elena Rostova to dynamically edit feline dietary protocols, feeding times, and toggle Emergency Medical Trading Holds (`/api/v1/workforce/care-schedules/{felineId}/medical-hold`).
* **Leave Requests & Catnip Break Workflow UI:** End-to-end leave management (`/api/v1/workforce/leave-requests`) and Employee Portal management UI for Alice Vance (`emp-human-alice`) supporting leave requests, manager approval/rejection workflows, and feline catnip break compliance tracking.
* **Review Cycles & Health Assessment Workflow UI:** End-to-end review management (`/api/v1/workforce/review-cycles`), clinical scoring, and Employee Portal UI tab for Alice Vance and Dr. Elena Rostova to schedule, evaluate, and record joint staff performance reviews and monthly feline health assessments.
* **Cross-Species Workplace Incident & Conflict Logging:** Schema `workforce.incidents`, Go service layer (`/api/v1/workforce/incidents`), and Employee Portal incident reporting tab.

### 5.5 Operations, Security & Wearables (`ops` & `cmd/pebble-proxy`)
* **Pebble Companion App Gateway Proxy:** Dedicated proxy server (`cmd/pebble-proxy` on `:8082`) translating active IT tickets into Pebble watch AppMessage dictionary payloads (`/ops/pebble/alerts`) and receiving ACK button clicks (`/ops/pebble/ack`).
* **Automated Root/Intermediate CA Key Rotation:** Background ticker worker (`cmd/worker` + `internal/service/pki`) continuously evaluating certificate expiration against a 30-day threshold and automating certificate re-issuance via `cert-manager`.
* **Active Push Notification Services (`ntfy` / WebPush):** Push notification dispatch service (`/ops/notifications/push`) routing critical cluster and habitat alerts to on-call engineers.
* **Automated Database Backup & Disaster Recovery:** Kubernetes backup CronJob (`k8s/postgres/backup-cronjob.yaml`) and database backup script (`scripts/backup-db.sh`).

### 5.6 Platform Genericization & Small Business Blueprint (White-Labeling & Modularization)
* **Modular Domain Decoupling & Optional Feline Extensions:** Feature flag toggles (`ENABLE_CORE_INVEST`, `ENABLE_FELINE_WORKFORCE`, `ENABLE_PEBBLE_GATEWAY`) in Go servers and partitioned SQL scripts (`scripts/00-bootstrap.sql`, `01-workforce-core.sql`, `02-facilities-core.sql`, `03-ops-core.sql`, `04-oci-feline-investments.sql`).
* **Centralized Brand Tokenization & UI White-Labeling:** Centralized design token manifest (`config/branding.yaml`), Go branding loader (`pkg/branding/config.go`), and CSS compilation tool (`scripts/generate-theme.js`).
* **Turnkey Business COTS Suite Fast-Path:** Standalone COTS-first stack via `docker-compose.smb.yaml` and `./scripts/deploy-k8s.sh --profile business-only`.
* **CLI Project Generator & New Custom Domain Scaffolding:** CLI generator script `scripts/init-business.sh` for bootstrapping custom white-labeled small business platforms.

### 5.7 Enterprise Multi-Tier Architecture & Microservices Decomposition
Orange Cat Investments implements all architectural tiers across an enterprise-grade multi-tier system with strict tier isolation: Frontends $\rightarrow$ Backend-for-Frontend (BFF) $\rightarrow$ Business Logic Layer (BLL) $\rightarrow$ Domain APIs (isolated SQL databases) $\rightarrow$ Async Workers & Batch Schedulers.

* **Client Applications:**
  * **Customer Web Portal (`web/customer-portal`):** Vue 3 SPA for retail investors and customers with ZITADEL OIDC auth, portfolio analytics, WebRTC stream player with AI overlays, trading terminal, and ESG dashboard.
  * **Employee Web Portal (`web/employee-portal`):** Vue 3 SPA for corporate staff with Forgejo OAuth2 auth, COTS application launcher, interactive Care Schedule editor, Emergency Medical Hold, VHR charting, leave requests, and performance review cycles.
  * **Static Marketing Website (`web/marketing`):** Fast Hugo static marketing site compiled into a lightweight `cmd/portal-server` container (`oci-marketing:latest`) bound to `https://oci.local`.
  * **Field Maintenance Mobile App (`mobile/flutter_app`):** Flutter mobile app with offline SQLite sync engine, QR code scanning, and firmware rollout tracking.
  * **Pebble Watch Companion App (`embedded/pebble`):** On-call alert response watch app in RePebble Alloy JavaScript.
* **Backend-for-Frontend (BFF) Layer:**
  * **Customer BFF (`cmd/customer-bff` :8083):** Manages ZITADEL OIDC sessions (`__Host-customer-session`), double-submit CSRF tokens, SSE price ticker streaming, and presentation aggregation. *Invariant: Never connects directly to Domain APIs or DB schemas.*
  * **Employee BFF (`cmd/employee-bff` :8081):** Manages Forgejo OAuth2 sessions (`__Host-employee-session`), CSRF tokens, COTS integrations, and internal presentation models. *Invariant: Never connects directly to Domain APIs or DB schemas.*
* **Business Logic Layer (BLL):**
  * **Customer BLL (`cmd/customer-bll` :8085):** Middle-tier orchestrator for customer business rules, trade order execution, Plaid bank linking, broker adapter coordination, and async event publishing.
  * **Internal BLL (`cmd/internal-bll` :8086):** Middle-tier orchestrator for corporate sagas (`OnboardingSaga`, `OffboardingEngine`), veterinary care directives, executive BI reporting, and incident event publishing.
* **Domain APIs (Isolated PostgreSQL Schemas):**
  * **Workforce Domain API (`cmd/workforce-api` :50051):** Owns PostgreSQL `workforce` schema.
  * **Facilities Domain API (`cmd/facilities-api` :50052):** Owns PostgreSQL `facilities` schema.
  * **Core Invest Domain API (`cmd/core-invest-api` :50053):** Owns PostgreSQL `core_invest` schema.
  * **Ops Domain API (`cmd/ops-api` :50054):** Owns PostgreSQL `ops` schema.
  * *Invariants:* Domain APIs never call each other. Direct cross-domain database queries and inter-domain API calls are strictly forbidden. Ingress is restricted via Kubernetes NetworkPolicies strictly to authorized BLL and worker pods.
* **Async Workers & Batch Schedulers:**
  * **Go Background Worker (`cmd/worker`):** Event subscriber (Watermill / RabbitMQ) for observation triggers, backtesting jobs, edge device OTA firmware rollout pipelines, async ACH bank transfer settlements, and automated cert-manager CA rotation.
  * **Scheduled Batch Schedulers:** Cron runners for daily recurring ACH DCA deposits (00:00 UTC) and automated end-of-month PDF statements (1st of month at 01:00 UTC).
  * **Wearable Gateway Proxy (`cmd/pebble-proxy` :8082):** Bridges Bluetooth/HTTP Pebble watch companion app requests with IT helpdesk operations and pub/sub real-time incident alerting.
* **Pub/Sub Real-Time Alerting Pipeline:**
  * Safety incidents (`events.workforce.incident_created.v1`) fan out asynchronously from RabbitMQ to web portal banners via Server-Sent Events (SSE) and Pebble smartwatch alerts, guaranteeing emergency responsiveness without synchronous coupling.

### 5.8 Zero-Downtime PKI Infrastructure & Long-Running Cluster Resilience
Developed and deployed to guarantee uninterrupted multi-day operations, zero-downtime certificate rollovers, and stateful application stability:

* **Automated Cross-Namespace CA Trust Distribution (`trust-manager` & `Bundle` CRD):**
  * Uses cert-manager's `trust-manager` addon and [`k8s/ha/lan/tls/oci-ca-bundle.yaml`](k8s/ha/lan/tls/oci-ca-bundle.yaml) to continuously sync `secret/oci-lan-root-ca-key-pair` in namespace `cert-manager` to `ConfigMap/oci-ca-bundle` across all application namespaces (`oci-core`, `oci-internal`, `monitoring`).
* **Zero-Downtime Dynamic CA Pool Hot-Swapping (`pkg/mtls`):**
  * [`pkg/mtls.DynamicCAPoolReloader`](pkg/mtls/mtls.go) watches file modification timestamps on mounted `ca.crt` bundles and hot-swaps active `ClientCAs` and `RootCAs` pools dynamically under RWMutex lock via `tls.Config.GetConfigForClient` and `tls.Config.VerifyConnection`, accepting new CA roots without connection drops or pod restarts.
* **Decoupled Transport & Probing Architecture:**
  * Domain APIs (`cmd/*-api`) and BLL services (`cmd/*-bll`) operate as pure gRPC services implementing standard `grpc_health_v1` and probed via Kubernetes `tcpSocket` directly on their service ports. Frontend BFFs, web portals, and background worker provide unauthenticated plaintext HTTP `/healthz` endpoints.
* **Active Kubernetes Cert-Manager CA Reconciler in `cmd/worker`:**
  * [`internal/service/pki/ca_rotation.go`](internal/service/pki/ca_rotation.go) interfaces directly with Kubernetes API using standard library crypto and in-cluster credentials to inspect expiration metrics, trigger automated reissuances 30 days prior to expiry, verify `trust-manager` bundle synchronization, and emit `ops.ca_rotated.v1` events.
* **COTS Persistent Volume Architecture & Idempotency (Nextcloud):**
  * Mandated PVCs (`nextcloud-data-pvc`) mounted directly to `/var/www/html` to persist configuration and data. Enforced unprivileged database users (`nextcloud_user` without `CREATEROLE` or superuser privileges) across COTS manifests to prevent role explosion and table ownership corruption across pod restarts.
* **Identity & Session Store Resiliency (ZITADEL & Valkey):**
  * Dedicated PersistentVolumeClaim on Valkey (`valkey-data-pvc` on `/data`) with `valkey-server --appendonly yes --dir /data` enabled. ZITADEL database connection limits and dedicated `startupProbe`/`livenessProbe` configured for DNS/CNI startup tolerance.
* **Control Plane Certificate Expiration Runbooks & Alerts:**
  * Documented annual control plane cert rotation SOPs in [`docs/runbooks/control-plane-cert-rotation.md`](docs/runbooks/control-plane-cert-rotation.md) and Prometheus Alertmanager rules in [`k8s/monitoring/prometheus.yaml`](k8s/monitoring/prometheus.yaml) (`CertManagerCertExpiringSoon` and `BlackboxSSLCertExpiringSoon`).
* **COTS Application Lifecycle Management & Upstream Patching Runbooks:**
  * Documented pre-flight database snapshots, sequential Nextcloud upgrades, Forgejo OAuth2 validation, Snipe-IT key preservation, ERPNext/Frappe bench migrations, and single-command rollbacks in [`docs/runbooks/cots-patching-lifecycle.md`](docs/runbooks/cots-patching-lifecycle.md) and [`scripts/cots-maintenance.sh`](scripts/cots-maintenance.sh).

### 5.9 Distributed Caching, Valkey Session Management & Rate Limiting (BFF Layer)
Orange Cat Investments leverages Valkey 7.2 in namespace `oci-core` (`valkey-service.oci-core.svc.cluster.local:6379`) backed by persistent storage (`valkey-data-pvc`) and AOF persistence:

* **Distributed Valkey Session Store for BFF Multi-Pod Scaling:**
  * Implemented in [`pkg/session/valkey`](pkg/session/valkey/store.go) supporting both Valkey distributed storage and thread-safe in-memory fallback for local dev and offline tests.
  * Customer sessions (`session:customer:{token}`) and Employee sessions (`session:employee:{token}`) with 24-hour TTL.
  * Multi-device session revocation via Redis Sets (`user:sessions:customer:{id}`, `user:sessions:employee:{id}`) enabling atomic bulk purges on logout, account closure, or GDPR erasure.
* **Cache-Aside & Read-Through Query Caching:**
  * Implemented in [`pkg/cache/valkey`](pkg/cache/valkey/cache.go) with raw-byte caching (`GetRaw` / `SetRaw`) for zero-allocation conditional HTTP responses.
  * Customer Portal: Portfolio Valuation (`cache:portfolio:{customer_id}`, 30s TTL), Quotes (`cache:quotes:{symbol}`, 5s TTL), Habitat Telemetry (5s TTL), ESG Welfare Metrics (15m TTL).
  * Employee Portal: Feline Care Schedules (`cache:care_schedule:{feline_id}`, 2m TTL), Nextcloud metadata (`cache:cots:nextcloud:folders`, 60s TTL).
* **Event-Driven & Eager Cache Invalidation:**
  * [`InvalidationListener`](pkg/cache/valkey/invalidation.go) subscribing via Watermill / RabbitMQ to `trade_executed`, `order_placed`, `care_schedule_updated`, and `incident_created` events.
  * Synchronous eager invalidation across mutation paths (orders, deposits, care schedule updates, medical hold toggles, GDPR erasures).
* **HTTP Conditional Validation (`ETag` & `304 Not Modified`):**
  * Payloads are hashed with SHA-256 (`ComputeETag` / `HandleConditionalETag`). Matching `If-None-Match` requests return `304 Not Modified` with zero response body.
* **Distributed Sliding-Window Rate Limiting:**
  * [`RateLimiter`](pkg/cache/valkey/ratelimit.go) using Valkey sorted sets (`ZREMRANGEBYSCORE`, `ZCARD`, `ZADD`) with in-memory fallback and standard `X-RateLimit-*` and `Retry-After` headers.
  * Applied to login (5 req/15min), orders (10 req/s), deposits (5 req/15min), and Pebble Watch acknowledgments (10 req/min).

### 5.10 Pure gRPC Wire Transport & Decoupled BFF Layer
* **Pure gRPC Wire Transport:** Transitioned inter-service communication to pure gRPC over dynamic mTLS with standard Protobuf RPCs (`pkg/grpcutil`). Domain APIs (`workforce-api` :50051, `facilities-api` :50052, `core-invest-api` :50053, `ops-api` :50054) serve compiled Protobuf stubs, standard gRPC health checks (`grpc.health.v1`), and streaming RPCs (`StreamPortfolioValuation`, `StreamEnvironmentalTelemetry`), completely removing HTTP fallback and `h2c` multiplexing.
* **Zero Direct Database Dependencies in BFFs:** Completely eliminated `pgxpool` and direct SQL repository access from `cmd/customer-bff` and `cmd/employee-bff`. BFFs function strictly as presentation and session proxies delegating all domain logic and data fetching to Customer BLL and Internal BLL.

### 5.11 Fast Typo-Tolerant Full-Text Search via Meilisearch
* **Meilisearch Client Engine:** Implemented [`pkg/search/meilisearch`](pkg/search/meilisearch/client.go) providing resilient search, index provisioning, document indexing, and in-memory fallback for local development.
* **Real-Time CDC Event Ingestion:** Background worker (`cmd/worker`) subscribes to domain events (`incident_created`, etc.) and automatically indexes searchable entities into Meilisearch.
* **Employee Command+K Search Palette:** Global `⌘K` / `Ctrl+K` modal in `web/employee-portal` searching across Clinical VHRs, IT tickets, SOPs, and hardware assets.
* **Customer Catalog Search:** Instant search bar in `web/customer-portal` searching funds, tickers, ESG feline welfare transparency, and tax statements.

### 5.12 Automated Health Dashboard & Multi-Tenant Uptime Monitoring via Gatus
* **Unified Status Page (`status.oci.local`):** Declarative deployment of Gatus (`k8s/monitoring/gatus.yaml`) monitoring uptime percentages, response latencies, and TLS certificate expiration across Customer Planes, Employee Planes, Domain APIs, and COTS enterprise applications.
* **Automated Webhook Incident Fanout:** Configured custom webhook alert dispatcher triggering workplace incident broadcast alerts to Employee BFF and Pebble watch companion apps upon consecutive health probe failures.
* **Portal Integration:** Integrated real-time status indicators and status page links directly in the headers of both the Customer Portal and Employee Portal.

### 5.13 COTS & Message Queue Prometheus Telemetry
* **RabbitMQ Built-in Prometheus Telemetry (`rabbitmq_prometheus`):** Enabled native Prometheus plugin with `prometheus.return_per_object_metrics = true` exposing per-object queue telemetry on port `15692`. Configured Prometheus scraping and Alertmanager rules for Dead-Letter Queue (DLQ) backlogs, queue spikes, stagnant consumers, and memory limits in `k8s/monitoring/prometheus.yaml`.
* **COTS Application Telemetry:** Enabled native Prometheus metrics in Forgejo (`/metrics`), deployed `nextcloud-exporter` sidecar on port `9205` in `k8s/oci-internal/nextcloud.yaml`, and instrumented ZITADEL customer identity Prometheus metrics on `/debug/metrics` with public Ingress route protection.

### 5.14 Modular Application Architecture & Local 5-Tier Parity
* **Modular BFF, BLL & Worker Packages (`internal/app/`):** Decomposed monolithic `cmd/*/main.go` entrypoints into focused internal packages (`< 450` lines per file) enforcing strict 5-tier isolation: Chi-based HTTP BFFs (`customer-bff`, `employee-bff`, `pebble-proxy`) with fail-fast mTLS startup, pure gRPC BLLs (`customer-bll`, `internal-bll`), isolated gRPC Domain APIs (`workforce-api`, `facilities-api`, `core-invest-api`, `ops-api`), and BLL-routed background workers/batch jobs (`cmd/worker`).
* **Production-Aligned Local Topology (`scripts/run-local-dev.sh`):** Single-command local launcher running all 10 tiered services on production-equivalent ports with signal trapping and graceful child shutdown, eliminating legacy unified `:8080` HTTP server drift.

### 5.15 Stateful Persona Simulation & Continuous Load Test Hardening
* **Stateful Cookie & Double-Submit CSRF Simulation (`simulation/`):** Per-persona `http.CookieJar` session persistence, automated pre-action login handshakes, transparent `X-CSRF-Token` injection on state-changing requests, mid-iteration `401` re-authentication resilience, and negative security assertions (session purge on offboarding and GDPR erasure).
* **Load Test Readiness & Runbook (`docs/runbooks/cluster-load-testing.md`):** Removed retired in-process mock routing, promoted `512Mi` Domain API memory limits, wired configurable Forgejo developer persona tokens, implemented self-replenishing `mock-broker` cash balances for continuous runs, completed Barneby care schedule seeds, and documented cAdvisor/metrics-server telemetry verification.

### 5.16 Self-Hosted In-Cluster Forgejo Actions CI/CD Pipeline
* **In-Cluster Source Sync & Runner (`scripts/sync-repo-to-forgejo.sh`, `k8s/oci-internal/forgejo-runner.yaml`):** Automated Forgejo repository synchronization and `gitea/act_runner` + Docker-in-Docker builds pushing multi-stage images directly to `localhost:32000`.
* **Gated Staging-to-Production Promotion (`.forgejo/workflows/ci.yaml`):** Automated multi-stage pipeline running OpenAPI/SQL/Go/Vue test gates, deploying immutable `sha-<COMMIT>` images to `oci-dev`, validating all 13 personas via live simulation, and gating promotion to `oci-core`.

### 5.17 Self-Hosted Kubernetes Infrastructure Blind-Spot Telemetry
* **Storage, Pod State & Backup Alerting (`k8s/monitoring/`):** Configured kubelet volume capacity scraping (`PVCCapacityWarning` `< 15%` free and `PVCCapacityPrediction` 24h linear exhaustion), `kube-state-metrics` (`PodCrashLoopBackOff`, `PodImagePullBackOff`, `PodPending`, `OOMKilled`), and `BackupFailedOrStale` (`> 26h`) rules alongside operational runbooks in [`docs/runbooks/k8s-monitoring-blind-spots.md`](docs/runbooks/k8s-monitoring-blind-spots.md).

### 5.18 Container Builds & Image Promotion for All External-Sim Mocks
* **External-Sim Dockerfiles (`cmd/mock-bank/Dockerfile`, `cmd/mock-broker/Dockerfile`):** Added container build definitions consistent with `mock-hardware` for external mock integration testing.
* **Pipeline Integration (`ci.yaml` & `verify-ci-contracts.sh`):** Integrated mock container builds into deployment pipelines ensuring complete integration tracking in `oci-dev`.

### 5.19 Lifelike Mock Hardware Behavior
* **BehaviorModel (`cmd/mock-hardware/behavior.go`):** Replaced uniform random generator with a realistic Markov chain activity model for Mock Hardware to simulate lifelike transitions between activities (`sleeping`, `zooming`, etc.), and added bounding and directional drift to mock environmental telemetry correlating directly with current activities.

### 5.20 Trace Context Propagation Over RabbitMQ & Grafana Correlation
* **Tracing (`pkg/envelope/tracing.go`, `internal/app/worker/tracing_middleware.go`):** Propagates OpenTelemetry context via Watermill `message.Metadata` and adds Loki/Prometheus cross-datasource links to Tempo.

---

## 6. Home Server Self-Hosting & Quickstart Guide

This guide walks you through setting up and running the full Orange Cat Investments platform on your home server (or local Linux/macOS machine) to try it out.

### 6.1 Prerequisites

Ensure your host machine has the following tools installed:

| Tool | Recommended Version | Purpose |
|---|---|---|
| **Git** | 2.40+ | Source code checkout |
| **Go** | 1.22+ (tested on 1.25) | Backend compilation & running |
| **PostgreSQL** | 16+ (or Docker / Podman) | Multi-schema relational database |
| **RabbitMQ** | 3.12+ (or Docker / Podman) | Distributed event messaging broker |
| **Valkey / Redis** | 7.2+ (or Docker / Podman) | Session storage & cache |
| **Node.js & npm** | Node 18+ / npm 9+ | Building the Vue 3 Employee Portal |
| **Flutter SDK** *(Optional)* | 3.19+ | Running the Field Maintenance mobile app |
| **MicroK8s on Ubuntu Server / Ansible** *(Optional)* | v1.28+ | Full Kubernetes cluster deployment |

---

### 6.2 Architecture & Port Topology

When running locally on a home server, the services bind to the following ports:

```text
       ┌─────────────────────────────────────────────────────────────┐
       │                       Home Server Host                      │
       ├─────────────────────────┬───────────────────────────────────┤
       │ Service                 │ Port / Endpoint                   │
       ├─────────────────────────┼───────────────────────────────────┤
       │ PostgreSQL 16           │ localhost:5432                    │
       │ Valkey (Redis)          │ localhost:6379                    │
       │ RabbitMQ                │ localhost:5672 (AMQP) / 15672 (UI)│
       │ Workforce Domain API    │ localhost:50051 (gRPC)            │
       │ Facilities Domain API   │ localhost:50052 (gRPC)            │
       │ Core Invest Domain API  │ localhost:50053 (gRPC)            │
       │ Ops Domain API          │ localhost:50054 (gRPC)            │
       │ Customer BLL API        │ localhost:8085 (gRPC)             │
       │ Internal BLL API        │ localhost:8086 (gRPC)             │
       │ Employee BFF            │ http://localhost:8081             │
       │ Pebble Gateway Proxy    │ http://localhost:8082             │
       │ Customer BFF            │ http://localhost:8083             │
       │ Background Worker       │ http://localhost:8080/healthz     │
       │ Vue 3 Employee Portal   │ http://localhost:5173             │
       │ Vue 3 Customer Portal   │ http://localhost:5174             │
       └─────────────────────────┴───────────────────────────────────┘
```

---

### 6.3 Step-by-Step Standalone Setup (Fastest / Docker Compose)

#### Step 1: Clone the Repository
```bash
git clone https://github.com/ramjac/orange-cat-investments.git
cd orange-cat-investments
```

#### Step 2: Start Infrastructure (PostgreSQL, Valkey, RabbitMQ)
If you already have Docker or Podman installed, launch the core infrastructure containers:

```bash
# 1. Run PostgreSQL 16
docker run -d --name oci-postgres \
  -e POSTGRES_USER=oci \
  -e POSTGRES_PASSWORD=catnip \
  -e POSTGRES_DB=oci \
  -p 5432:5432 \
  postgres:16-alpine

# 2. Run Valkey (Redis-compatible session store)
docker run -d --name oci-valkey \
  -p 6379:6379 \
  valkey/valkey:7.2-alpine

# 3. Run RabbitMQ with Management UI
docker run -d --name oci-rabbitmq \
  -p 5672:5672 -p 15672:15672 \
  rabbitmq:3.13-management-alpine
```

#### Step 3: Initialize Database Schemas & Production Starter Admin

The OCI platform strictly segregates **production schemas** from **development seed data**:

* **Clean Production Database (`scripts/init.sql` & `k8s/oci-core/`):**
  Defines the core database schemas (`workforce`, `facilities`, `core_invest`, `ops`), indexes, and table constraints. It is kept completely clean of mock personas, demo portfolios, test hardware assets, and simulated tickets. The production database is provisioned with **ONLY** a single starter administrator account:
  - **Email:** `admin@oci.local`
  - **Employee ID:** `018f0000-0000-7000-8000-000000000001`
  - **Role:** `hr` (Platform Administrator & HR Lead)
  - **Purpose:** Enables the first corporate employee to log in and begin onboarding the rest of the workforce.

* **Starter Admin Bootstrap Password (`starter-admin-credentials.txt`):**
  The initial bootstrap password for the starter admin is autogenerated using a CSPRNG and stored in a local file:
  ```text
  starter-admin-credentials.txt
  ```
  *(Also available in JSON format at `.starter-admin-credentials.json`)*.
  This file is strictly listed in `.gitignore` and must never be committed to version control. To generate or inspect the credentials:
  ```bash
  # Generate local bootstrap credentials (if not already created):
  ./scripts/generate-starter-admin.sh
  ```

  > [!IMPORTANT]
  > **MANDATORY SECURITY DIRECTIVE:**
  > The starter platform administrator password located in `starter-admin-credentials.txt` **MUST BE CHANGED IMMEDIATELY** upon first login. After initial authentication, navigate to account settings to set a permanent, organization-managed password before onboarding corporate staff or connecting retail customer accounts.

* **Development & Simulation Seed Data (`scripts/seed-dev.sql` & `k8s/oci-dev/`):**
  Mock personas (Garfield, Barneby, Dr. Elena, Alice, Elise, Marcus, etc.), demo retail portfolios (Arthur Pendelton, Chloe Spark), habitat zones, edge hardware, and test review cycles belong exclusively to the **`oci-dev`** namespace. To populate development fixtures:
  ```bash
  # Apply seed fixtures strictly to the oci-dev development environment:
  ./scripts/seed-dev.sh
  ```
  *(This script enforces strict safety guards and is hard-coded to refuse execution against production `oci-core`)*.

```bash
# Apply clean production schema and starter admin using psql:
PGPASSWORD=catnip psql -h localhost -U oci -d oci -f scripts/init.sql
```

> **Note:** If testing without a live database, all Go services automatically fall back to built-in in-memory mock persistence when `DATABASE_URL` is omitted.

#### Step 4: Launch Backend Services (5-Tier Topology)

You can launch the complete 5-tier backend suite with a single command, or run each tier independently in separate terminal tabs or `tmux` panes.

##### Option A: Single-Command Local Topology Launcher (Recommended)
Launch all 10 backend services with proper inter-tier network wiring, environment configuration, and clean shutdown handling:

```bash
./scripts/run-local-dev.sh
```

##### Option B: Manual Multi-Tab Service Launch
To run each tier and service in individual terminal tabs or `tmux` panes:

**Tier 4 — Domain APIs (Pure gRPC over PostgreSQL Schemas):**
```bash
export DATABASE_URL="postgres://oci:catnip@localhost:5432/oci?sslmode=disable"

# 1. Workforce Domain API (:50051)
PORT=50051 go run ./cmd/workforce-api

# 2. Facilities Domain API (:50052)
PORT=50052 go run ./cmd/facilities-api

# 3. Core Invest Domain API (:50053)
PORT=50053 go run ./cmd/core-invest-api

# 4. Ops Domain API (:50054)
PORT=50054 go run ./cmd/ops-api
```

**Tier 3 — Business Logic Layer (BLL Services):**
```bash
export WORKFORCE_API_URL="localhost:50051"
export FACILITIES_API_URL="localhost:50052"
export CORE_INVEST_API_URL="localhost:50053"
export OPS_API_URL="localhost:50054"
export RABBITMQ_URL="amqp://guest:guest@localhost:5672/"

# 5. Customer BLL (:8085)
PORT=8085 go run ./cmd/customer-bll

# 6. Internal BLL (:8086)
PORT=8086 go run ./cmd/internal-bll
```

**Tier 2 — Backend-for-Frontend (BFF Proxies):**
```bash
export VALKEY_ADDR="localhost:6379"
export CUSTOMER_BLL_URL="localhost:8085"
export INTERNAL_BLL_URL="localhost:8086"

# 7. Employee BFF (:8081)
PORT=8081 go run ./cmd/employee-bff

# 8. Pebble Watch Companion Gateway Proxy (:8082)
PORT=8082 go run ./cmd/pebble-proxy

# 9. Customer BFF (:8083)
PORT=8083 go run ./cmd/customer-bff
```

**Tier 5 — Async Background Worker:**
```bash
export CUSTOMER_BLL_URL="localhost:8085"
export INTERNAL_BLL_URL="localhost:8086"
export RABBITMQ_URL="amqp://guest:guest@localhost:5672/"

# 10. Background Event Worker (Watermill, OTA, CA Rotation, ACH queue)
HEALTH_PORT=8080 go run ./cmd/worker
```

#### Step 5: Launch the Web Portals (Employee & Customer)

In new terminal tabs, launch the Vue 3 single-page applications:

**1. Employee Web Portal (`:5173`)**
```bash
cd web/employee-portal
npm install
npm run dev
```
Open **`http://localhost:5173`** to manage care schedules, feline dietary plans, VHR records, leave requests, and emergency medical trading holds.

**2. Customer Web Portal (`:5174`)**
```bash
cd web/customer-portal
npm install
npm run dev
```
Open **`http://localhost:5174`** to view retail portfolio balances, watch the live habitat WebRTC stream with AI detection overlays, monitor the Zoomie Index, and configure recurring deposits.

#### Step 6: Launch the Flutter Mobile App (Field Maintenance)

If you have Flutter installed, you can launch the mobile maintenance app on desktop Linux/macOS, Android emulator, or connected phone:
```bash
cd mobile/flutter_app
flutter pub get
flutter run -d linux   # or -d macos / -d chrome / -d <device-id>
```
Technicians can scan simulated QR codes (`QR-CAM-ORANGE-01`), record offline maintenance actions, and trigger the offline synchronization engine.

#### Step 7: White-Label Branding & Turnkey Small Business Mode (Optional)

To test white-labeling or spin up the generic small-business ecosystem:
```bash
# 1. Compile CSS design tokens from config/branding.yaml:
node scripts/generate-theme.js

# 2. Or initialize custom company branding parameters:
./scripts/init-business.sh "Acme Logistics" "#2563eb" "acme.local"

# 3. Launch turnkey lightweight SMB stack (Postgres, Valkey, RabbitMQ, Forgejo):
docker compose -f docker-compose.smb.yaml up -d
```

---

### 6.4 Kubernetes Cluster Deployment (Self-Hosted MicroK8s on Ubuntu Server Standard)

Orange Cat Investments is designed to run natively on **MicroK8s on Ubuntu Server**, providing a fast, lightweight, and robust Kubernetes foundation ideal for edge hardware, bare-metal mini PCs, or small-business server nodes.

#### Step 1: Ubuntu Server & MicroK8s Host Setup

1. **Install MicroK8s on Ubuntu Server:**
   ```bash
   sudo snap install microk8s --classic --channel=1.34/stable
   sudo usermod -a -G microk8s $USER
   mkdir -p ~/.kube
   sudo chown -f -R $USER ~/.kube
   newgrp microk8s
   ```

2. **Enable Core Cluster Addons:**
   MicroK8s provides turn-key addons for DNS, storage, ingress, registry, and PKI:
   ```bash
   microk8s enable dns storage ingress registry cert-manager metrics-server
   ```

3. **Configure Workstation Kubeconfig:**
   Export the cluster configuration to your local workstation:
   ```bash
   microk8s config > thinker-kube.config # or ~/.kube/config
   export KUBECONFIG="$(pwd)/thinker-kube.config"
   kubectl get nodes -o wide
   ```

#### Step 2: Storage, Registry & Ingress Conventions

1. **Storage Provisioner (`microk8s-hostpath`):**
   MicroK8s defaults to the `microk8s-hostpath` StorageClass. On hosts with attached secondary drives (e.g. SATA/NVMe SSD mounted at `/mnt/data`), persistent volumes can be directed to the dedicated storage partition.

2. **In-Cluster Container Registry (`localhost:32000`):**
   MicroK8s provides a built-in container registry (`microk8s enable registry`) serving on port `:32000` / `localhost:32000`, matching all OCI container deployment manifests. Alternatively, the manifest in `k8s/registry/registry.yaml` can be applied.

3. **Ingress & Routing (NGINX Ingress Standard):**
   MicroK8s includes the official NGINX Ingress controller (`microk8s enable ingress`) listening on host ports `80` and `443`. All OCI Ingress manifests utilize standard Ingress rules adopted by the controller.

#### Step 3: Deploy Platform Workloads

Use `./scripts/deploy-k8s.sh` to deploy the desired deployment profile:

* **All-In-One Platform (`--profile all` or default):**
  Deploys the full platform suite including core applications (`oci-core`), internal business applications & COTS (`oci-internal`), external simulation mocks (`external-sim`), cluster monitoring & logging (`monitoring`, `logging`), and development staging (`oci-dev`):
  ```bash
  ./scripts/deploy-k8s.sh
  # or explicitly:
  ./scripts/deploy-k8s.sh --profile all
  ```

* **Business-Only COTS Suite (`--profile business-only`):**
  Deploys strictly internal business applications (`oci-internal`: Nextcloud, Snipe-IT, ERPNext, Frappe HR, Forgejo, and internal PostgreSQL) without demo data or proprietary OCI applications, provisioning only the initial admin account:
  ```bash
  ./scripts/deploy-k8s.sh --profile business-only
  ```

* **Development & Staging Simulation Suite (`--profile dev`):**
  Deploys an isolated staging environment (`oci-dev`) alongside external financial mocks (`external-sim`) and populates development personas:
  ```bash
  ./scripts/deploy-k8s.sh --profile dev
  ./scripts/seed-dev.sh
  ```

#### Step 4: Accessing Cluster Services & Running Simulations

To interact with deployed cluster services from your local workstation:

```bash
# 1. Port-forward core API and BFF services:
kubectl port-forward -n oci-core svc/workforce-api-service 50051:50051 &
kubectl port-forward -n oci-core svc/oci-employee-bff-service 8081:8081 &
kubectl port-forward -n oci-core svc/oci-customer-bff-service 8083:8083 &
kubectl port-forward -n oci-internal svc/forgejo-service 3000:3000 &

# 2. Run the 12-persona platform simulation driver against the live cluster:
go run ./simulation/cmd/simulator -mode live -bff-url http://localhost:8081 -cust-bff-url http://localhost:8083

# 3. Execute concurrent load testing (e.g. 10 parallel simulation runs):
go build -o /tmp/simulator ./simulation/cmd/simulator
for i in {1..10}; do
  /tmp/simulator -mode live -bff-url http://localhost:8081 -cust-bff-url http://localhost:8083 -delay 5 &
done
wait
```

> **Clean Reset:** To tear down all OCI workloads and reset the cluster back to a clean state at any time, run `./scripts/cleanup-k8s.sh` (see [Section 6.8](#68-cluster-cleanup--environment-reset)).


---

### 6.5 Bare-Metal MicroK8s Cluster Provisioning via Ansible

For users provisioning a fresh, unconfigured bare-metal Ubuntu Server (e.g. mini PC, Intel NUC, or dedicated home lab node) from scratch:

1. **Configure Ansible Inventory:**
   Edit `ansible/inventory/hosts.ini` with your Ubuntu Server's IP address and SSH user:
   ```ini
   [microk8s_servers]
   192.168.1.246 ansible_user=ramjac ansible_ssh_private_key_file=~/.ssh/id_ed25519
   ```

2. **Run Infrastructure Deployment Script:**
   ```bash
   chmod +x scripts/deploy-infrastructure.sh
   ./scripts/deploy-infrastructure.sh
   ```

---

### 6.6 OCI Platform Simulation Driver

The repository includes a comprehensive simulation driver (`simulation/cmd/simulator`) that exercises all 13 OCI personas (Garfield, Barneby, Alice, Bob, Carol, David, Dr. Elena, Frank, Arthur, Chloe, Rick, Elise, Marcus) through their full daily operational routines:

> [!NOTE]
> **STRICT TARGETING INVARIANT:**
> The simulation engine and synthetic personas are strictly restricted to the **`oci-dev`** namespace (or local development stack). The production environment (`oci-core`) is safeguarded: any attempt to target production domains (`api.oci.local`) or production cluster services (`*.oci-core.*`) is automatically rejected by built-in configuration guards. The simulator runs exclusively in live mode against running HTTP service endpoints to catch authentic runtime issues.

```bash
# 1. Run against Kubernetes oci-dev Staging (smart URL routing):
go run ./simulation/cmd/simulator -bff-url https://dev.employee-bff.oci.local -cust-bff-url https://dev.customer-bff.oci.local

# 2. Run against local test services:
go run ./simulation/cmd/simulator \
  -bff-url http://localhost:8081 \
  -cust-bff-url http://localhost:8083 \
  -broker-url http://localhost:8089 \
  -bank-url http://localhost:8088
```
```

---

### 6.7 Functional Verification & Health Checks

Once services are running, verify system operation using `curl`:

**1. Health Check:**
```bash
curl -i http://localhost:8081/healthz
# Response: HTTP/200 {"status":"ok"}
```

**2. List Hardware Assets (Keyset Pagination):**
```bash
curl -s http://localhost:8081/facilities/assets | jq .
```

**3. Submit Offline Maintenance Logs (Field Sync):**
```bash
curl -s -X POST http://localhost:8081/facilities/sync/maintenance-logs \
  -H "Content-Type: application/json" \
  -d '[{
    "asset_id": "018e0000-0000-7000-8000-000000000001",
    "qr_code_scanned": "QR-CAM-ORANGE-01",
    "action_taken": "Cleaned optical lens and recalibrated zoom sensor"
  }]' | jq .
```

**4. Trigger Edge Device Firmware OTA Update:**
```bash
curl -s -X POST http://localhost:8081/facilities/firmware/ota-jobs \
  -H "Content-Type: application/json" \
  -d '{
    "asset_ids": ["018e0000-0000-7000-8000-000000000001"],
    "release_id": "018e0000-0000-7000-8000-000000000002"
  }' | jq .
```

**5. Toggle Feline Emergency Medical Hold:**
```bash
curl -s -X POST http://localhost:8081/api/v1/workforce/care-schedules/emp-feline-garfield/medical-hold \
  -H "Content-Type: application/json" \
  -d '{"emergency_medical_hold": true}' | jq .
```

**6. Pebble Watch Companion Gateway Alert & ACK:**
```bash
# Check pending watch alerts:
curl -s http://localhost:8082/ops/pebble/alerts | jq .

# Submit Pebble watch physical button ACK:
curl -s -X POST http://localhost:8082/ops/pebble/ack \
  -H "Content-Type: application/json" \
  -d '{
    "ticket_id": "018e0000-0000-7000-8000-000000000003",
    "acknowledged_by": "frank_ops",
    "button_id": 1
  }' | jq .
```

**7. Run All Verification Tests:**
```bash
go test -v -race ./...
```
All unit, integration, and race detection test suites will report passing!

---

### 6.8 Cluster Cleanup & Environment Reset

When you finish testing or want to reset your home server cluster to a clean slate (such that you can start testing from scratch the next day):

```bash
chmod +x scripts/cleanup-k8s.sh
./scripts/cleanup-k8s.sh
```

This automated cleanup routine:
* Safely deletes all OCI domain applications, databases, logging, monitoring, and namespaces (`oci-core`, `oci-internal`, `external-sim`, `postgres`, `monitoring`, `logging`).
* Deletes and releases OCI persistent volume claims (`postgres-data-pvc`).
* **Preserves System Infrastructure:** Keeps all cluster system services, storage classes, ingress controllers (e.g. MicroK8s NGINX ingress), container registries, and `kubectl` control plane connectivity intact.

To also stop and remove standalone Docker containers from Section 6.3:
```bash
docker stop oci-postgres oci-valkey oci-rabbitmq && docker rm oci-postgres oci-valkey oci-rabbitmq
```


---

## 7. Future Work & PR-Scoped Technical Roadmap

The remaining roadmap is an ordered sequence of **Pull-Request-sized steps** (`7.1`–`7.19`), followed by an intentionally undecided exploratory item (`7.20`). Steps are numbered in execution order; each lists **Depends On**, **Current Codebase State**, **PR Scope**, and **Verification Gate**. An agent should take the lowest-numbered step whose dependencies are complete.

### 7.0 Rules That Apply to Every Step

1. **Demo Mocks in Every Environment:** OCI is a demo platform. Both `oci-core` ("prod") and `oci-dev` ("dev") are fed by mock external systems in `external-sim` (bank, broker, edge hardware) that pretend to be real third parties. Mocks feeding `oci-core` are expected and correct. What is forbidden is (a) the *simulation driver* (`simulation/`) targeting `oci-core`, and (b) fault injection or chaos affecting anything `oci-core` depends on (see Phase 4).
2. **Baseline Definition of Done:** In addition to each step's own Verification Gate, every PR must pass the AGENTS.md §4 baseline: `go vet ./...`, `go test -race ./...`, the `scripts/init.sql` assertion check, and `python3 scripts/validate-openapi.py api/openapi.yaml`. PRs touching `web/*` must also pass `npm test` in each changed portal.
3. **Offline vs. Cluster Gates:** Gates marked **[offline]** must pass on any workstation. Gates marked **[cluster]** require the `thinker` test cluster. If the agent has no cluster access, it must complete all offline gates, leave the cluster gates unchecked, and state that explicitly in the PR description. It must never claim a cluster gate passed without running it.
4. **Schema Change Protocol:** The database bootstrap SQL exists in **three copies** that must stay in sync: `scripts/init.sql`, `k8s/oci-core/postgres-core-configmap.yaml`, and `k8s/oci-dev/postgres-dev-configmap.yaml` (seed rows: `scripts/seed-dev.sql` and the `oci-dev` configmap). Bootstrap scripts only run against an empty data volume, so every schema change must be written idempotently (`ALTER TABLE ... ADD COLUMN IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`) and also added to a numbered migration file `scripts/migrations/NNN-<name>.sql` that can be applied to existing databases with `psql -f`. The first step needing this (`7.8`) creates `scripts/migrations/`.
5. **Completion Protocol:** When a step merges, the same PR replaces that step's body with a one-paragraph `> [!NOTE] **Status: Completed**` summary and adds a matching bullet to §5. Steps are never renumbered after work begins.

### Execution Phases

1. **Phase 1 — CI/CD Supply Chain (`7.1`–`7.5`):** toolchain alignment, Forgejo/runner storage hygiene, container builds for every mock, honest immutable-SHA promotion, and registry pre-flight checks. This phase comes first so every later PR is built and promoted reliably.
2. **Phase 2 — Core Domain Completion (`7.6`–`7.12`):** dedicated dev instances of the mocks, lifelike mock hardware, edge device heartbeats/discovery, and the daily NAV snapshot engine.
3. **Phase 3 — Observability (`7.13`–`7.16`):** stateful store exporters and alerts, custom OpenTelemetry business metrics (Grafana Beyla dropped), and trace propagation over RabbitMQ.
   - [x] **7.13** Stateful Store Exporter Hardening
4. **Phase 4 — Fault Injection & Chaos (`7.17`–`7.19`):** fault injection on dev-only mocks, poison-pill/DLQ handling, and a chaos suite confined to `oci-dev` and the dev mocks in `external-sim`.
5. **Phase 5 — Exploratory (`7.20`):** zero-trust secrets and workload identity, undecided.

---

### Phase 1: CI/CD Supply Chain & Deployment Pipeline Hardening

#### 7.1 Toolchain Version Single Source of Truth & Fail-Fast CI Contract Gate
* **Depends On:** None.
* **Current Codebase State:** `go.mod` declares `go 1.26.0`. The 11 Go Dockerfiles under `cmd/*/Dockerfile` and `web/marketing/Dockerfile` (Hugo build stage) use `FROM golang:1.26-alpine`. `.forgejo/workflows/ci.yaml` (`validate-and-test`, `integration-test-gate`) downloads `go1.23.6` when Go is absent on the runner. The simulator CLI (`simulation/cmd/simulator/main.go`) has no way to validate flags without running, so a renamed flag in `ci.yaml` is only caught at the integration gate. No `web/*/package.json` declares `engines`.
* **PR Scope:**
  1. Add a root `.go-version` (`1.26.0`). Make `ci.yaml` install exactly that version. Add `ARG GO_VERSION=1.26` and `FROM golang:${GO_VERSION}-alpine AS builder` to every Go Dockerfile, including `web/marketing/Dockerfile`.
  2. Add a `-validate-flags` boolean to `simulation/cmd/simulator/main.go`: after `flag.Parse()`, exit `0` immediately without network calls.
  3. Add `scripts/verify-ci-contracts.sh` as the first step of `validate-and-test`. It asserts (a) every Dockerfile `GO_VERSION` default matches the `go.mod` major.minor, and (b) every `go run ./simulation/cmd/simulator ...` command in `ci.yaml`, run with `-validate-flags` appended, exits `0`.
* **Verification Gate:**
  * [offline] `./scripts/verify-ci-contracts.sh` passes, and fails when a Dockerfile's `GO_VERSION` is temporarily changed to `1.25`.
  * [offline] `go run ./simulation/cmd/simulator -validate-flags -bff-url x` exits `0`; `go run ./simulation/cmd/simulator -validate-flags -url x` exits non-zero.

#### 7.2 Forgejo Persistent Storage, Runner Disk Hygiene & Eviction Protection
* **Depends On:** None.
* **Current Codebase State:** `k8s/oci-internal/forgejo.yaml` declares no volumes, so repositories and Actions logs under `/var/lib/gitea` are lost on pod restart. `k8s/oci-internal/forgejo-runner.yaml` backs Docker-in-Docker storage with `emptyDir: {}` and has no `priorityClassName`. `k8s/registry/` contains only `registry.yaml` (no tag retention).
* **PR Scope:**
  1. Add a `forgejo-data-pvc` PersistentVolumeClaim (10Gi, default StorageClass) mounted at `/var/lib/gitea` in `forgejo.yaml`.
  2. In `forgejo-runner.yaml`, set `priorityClassName: system-cluster-critical` and replace the `docker-storage` `emptyDir` with a `forgejo-runner-docker-pvc` (20Gi). Add a `docker system prune -af --filter "until=24h"` step at the start of the image build job in `ci.yaml`.
  3. Add `k8s/registry/registry-gc-cronjob.yaml`: a daily CronJob that deletes `sha-*` tags beyond the 10 newest per repository (preserving `latest`, `dev-latest`, `prod-stable`) and then runs `registry garbage-collect`.
  4. Add a `NodeFilesystemLowSpace` alert (node filesystem available `< 20%`, from the existing `kubernetes-nodes` cAdvisor scrape) to `k8s/monitoring/prometheus.yaml`. Document runner eviction recovery (empty-commit re-trigger, inspecting `action_task.runner_id`, reading logs from `/var/lib/gitea/actions_log/`) in `docs/runbooks/cluster-load-testing.md`.
* **Verification Gate:**
  * [offline] All `k8s/**/*.yaml` parse with `yaml.safe_load_all`.
  * [cluster] Deleting the Forgejo pod preserves repositories and prior Actions logs.

#### 7.3 Container Builds & Image Promotion for All External-Sim Mocks
> [!NOTE] **Status: Completed**

#### 7.4 Honest Immutable-SHA Promotion & Post-Deploy Image Verification
* **Depends On:** `7.1`, `7.3`.
* **Current Codebase State:** `promote-oci-core` in `ci.yaml` retags images to `:latest`/`:prod-stable` and runs `kubectl apply -f k8s/oci-core/`. That is a no-op when the manifests already say `:latest`, so `oci-core` keeps running the previous build. Its 13 application Deployments are: `workforce-api`, `facilities-api`, `core-invest-api`, `ops-api`, `customer-bll`, `internal-bll`, `oci-customer-bff`, `oci-employee-bff`, `oci-customer-portal`, `oci-employee-portal`, `oci-pebble-proxy`, `oci-worker`, `oci-marketing`. Promotion also re-applies `k8s/external-sim/mock-hardware.yaml`. That is correct, because prod is fed by mocks (§7.0 rule 1), but it is done without pinning a SHA.
* **PR Scope:**
  1. Keep the `:prod-stable` retag. Replace the `kubectl apply` rollout with `kubectl set image deployment/<app> <container>=localhost:32000/<image>:sha-${COMMIT_SHA} -n oci-core` for all 13 Deployments, followed by `rollout status`.
  2. Pin the prod-serving mocks in `external-sim` (`mock-hardware`, `mock-bank`, `mock-broker`) to the same SHA in `promote-oci-core`.
  3. Add `scripts/verify-deployed-images.sh <namespace> <sha-tag> [deployment...]`. It reads each Deployment's container images via `kubectl get deployment -o jsonpath` and exits non-zero on any image not ending in `:<sha-tag>`. Call it at the end of `deploy-oci-dev` (for `oci-dev`) and `promote-oci-core` (for `oci-core` and the `external-sim` mocks).
* **Verification Gate:**
  * [offline] A test with a stub `kubectl` on `PATH` shows the script exits `0` for matching tags and `1` for `:latest` or a different SHA.
  * [cluster] After a pipeline run, `kubectl get deploy -n oci-core -o jsonpath='{..image}'` shows only `:sha-<commit>` tags.

#### 7.5 Registry Pre-Flight, Pipeline Status Helper & Single-Command Deploy
* **Depends On:** `7.4`.
* **Current Codebase State:** `scripts/deploy-k8s.sh` (`set -euo pipefail`) applies manifests without checking that the images they reference exist in `localhost:32000`. Most rollout waits already end in `|| true`, but the two `mock-hardware` waits (lines 94 and 113) abort the script on failure, silently skipping all later steps. There is no `scripts/ci-status.sh` or `scripts/deploy-latest.sh`.
* **PR Scope:**
  1. Add a registry pre-flight to `deploy-k8s.sh`: collect every `localhost:32000/<repo>:<tag>` in the selected profile's manifests and query `http://<registry>:32000/v2/<repo>/tags/list` (registry host from `REGISTRY_HOST`, default `localhost`). On any miss, print the missing list plus a hint to run `scripts/sync-repo-to-forgejo.sh`, then exit `1` before applying anything.
  2. Replace the aborting `mock-hardware` waits with a `wait_rollout` helper that records failures. At the end, print a summary and exit non-zero if anything failed.
  3. Add `scripts/ci-status.sh [--wait]`, which prints the latest Forgejo Actions run and job statuses via the Forgejo API (enabled by `7.2`) and tails a failed job's log.
  4. Add `scripts/deploy-latest.sh`: `go test -race ./...` → `sync-repo-to-forgejo.sh` → `ci-status.sh --wait` → `verify-deployed-images.sh` → run the simulator against `oci-dev` from the workstation.
* **Verification Gate:**
  * [offline] `bash -n` on all three scripts. A test pointing the pre-flight at a local `python3 -m http.server` stub that serves a fake `tags/list` detects a missing image and exits `1`.
  * [cluster] `./scripts/deploy-latest.sh` completes end-to-end.

---

### Phase 2: Core Domain Completion

#### 7.6 Dedicated Dev Instances of External-Sim Mocks
* **Depends On:** `7.4`.
* **Current Codebase State:** `external-sim` runs a single `mock-bank`, `mock-broker`, and `mock-hardware`. `oci-core` and `oci-dev` `customer-bll` both point at the same `mock-broker-service` / `mock-bank-service`, and `mock-hardware` publishes only to `oci-core` RabbitMQ and `internal-bll`. As a result, `oci-dev` gets no hardware events, and any fault injected into a mock (Phase 4) would also hit prod. In addition, `mock-bank`'s `OCI_WEBHOOK_URL` points at `unified-api.oci-core.svc.cluster.local:8080`, a service that no longer exists. No Go code receives `/api/v1/bank/webhooks`.
* **PR Scope:**
  1. Add `mock-bank-dev`, `mock-broker-dev`, and `mock-hardware-dev` Deployments and Services in `external-sim` (new files `k8s/external-sim/mock-*-dev.yaml`), with their own cert-manager `Certificate` for `mock-hardware-dev`. The existing unsuffixed instances remain the prod-serving mocks.
  2. Point `oci-dev` at the dev instances: `BROKER_API_BASE_URL` and `BANK_API_BASE_URL` in `k8s/oci-dev/bll-services.yaml` → `mock-*-dev-service`. `mock-hardware-dev` uses `RABBITMQ_URL` → `rabbitmq-service.oci-dev` and `INTERNAL_BLL_ADDR` → `internal-bll-service.oci-dev:8086`. In `k8s/oci-dev/bll-services.yaml`, change the NetworkPolicy's mock-hardware ingress selector to `app: mock-hardware-dev`. `oci-core` keeps `app: mock-hardware`.
  3. Set `OCI_WEBHOOK_URL` to empty on both bank instances (no receiver exists in the 5-tier architecture). Delete the stale `unified-api` reference.
  4. In `ci.yaml`, `deploy-oci-dev` pins the `-dev` mocks to `:sha-${COMMIT_SHA}`, and `promote-oci-core` keeps pinning the unsuffixed ones (`7.4`). In `scripts/deploy-k8s.sh`, the `dev` profile applies only the `-dev` mock manifests.
  5. Extend `scripts/verify-ci-contracts.sh` with an environment-wiring check: `k8s/oci-dev/` may reference only `mock-*-dev-service` in `external-sim`; `k8s/oci-core/` may reference only unsuffixed `mock-*-service`; `mock-*-dev.yaml` may reference only `.oci-dev.` hosts; and unsuffixed mock manifests only `.oci-core.` hosts.
* **Verification Gate:**
  * [offline] `./scripts/verify-ci-contracts.sh` passes and `! grep -rn "unified-api" k8s/`.
  * [cluster] `oci-dev` RabbitMQ shows `events.observation.cat_spotted.v1` traffic from `mock-hardware-dev`, and `oci-core` still receives events from `mock-hardware`.

#### 7.7 Lifelike Mock Hardware Behavior
* **Depends On:** `7.6`.
* **Current Codebase State:** `cmd/mock-hardware/main.go` (single file, no tests) picks activities uniformly with `rand.Intn` every 15s and environmental readings independently every 20s. It emits `envelope.EventEnvelope[envelope.CatSpottedPayload]` on `events.observation.cat_spotted.v1` (`pkg/envelope/events.go`), which the worker consumes via `cat_spotted_handler`.
* **PR Scope:**
  1. Move generation logic into `cmd/mock-hardware/behavior.go`: a Markov activity model with a configurable transition matrix (e.g. `sleeping` mostly stays `sleeping`; `zooming` comes in short bursts), and environmental readings that drift from the previous value and correlate with activity (e.g. `zooming` raises `noise_db`). Inject a `*rand.Rand` so tests are deterministic.
  2. Keep the topic, envelope type, and payload fields unchanged.
  3. Add `cmd/mock-hardware/behavior_test.go`: a seeded run produces only valid transitions, the long-run activity distribution is within tolerance of the matrix's stationary distribution, environmental values stay in range, and envelopes carry a UUIDv7 `EventID` and UTC `OccurredAt`.
* **Verification Gate:**
  * [offline] `go test -race ./cmd/mock-hardware/...`

#### 7.8 Edge Device Heartbeats — Domain Layer
* **Depends On:** `7.4`.
* **Current Codebase State:** `facilities.hardware_assets` already has `status` (`active`, `maintenance_required`, `offline`, ...) and `last_ping_at`. `core_invest.camera_streams` already has `status` (`active`, `idle`, `error`). Nothing updates these from device activity.
* **Design Decision (fixed):** **No new tables or columns.** Heartbeats update `hardware_assets.last_ping_at`. A device counts as offline when `last_ping_at` is older than `3 ×` the heartbeat interval (60s interval → 180s). That is derived at read time, never written by a background sweep. Camera stream `status` is updated only on explicit device reports.
* **PR Scope:**
  1. `proto/oci/facilities/v1/facilities.proto`: add `RecordAssetHeartbeat(serial_number, observed_at)` (sets `last_ping_at`, returns `NOT_FOUND` for unknown serials) and `ListAssetConnectivity(zone_id optional, stale_after_seconds)`, which returns assets with a computed `online` boolean. `proto/oci/core_invest/v1/core_invest.proto`: add `UpdateCameraStreamStatus(camera_asset_id, status)`. Regenerate `pkg/gen/go/`.
  2. Implement in `internal/repository/facilities/` and `internal/repository/core_invest/` (pgx + mock), and wire the handlers in `cmd/facilities-api` and `cmd/core-invest-api`.
  3. Create `scripts/migrations/` with a `README.md` documenting rule 4 of §7.0. No migration file is needed for this step.
* **Verification Gate:**
  * [offline] Repository and handler unit tests cover heartbeat update, unknown serial, and online/offline derivation at the staleness boundary.

#### 7.9 Edge Device Heartbeats — BLL, Mock Wiring & BFF Exposure
* **Depends On:** `7.7`, `7.8`.
* **Current Codebase State:** `internal-bll` exposes `IngestEnvironmentalTelemetry` and `GetFelineTarget` to `mock-hardware`. No heartbeat RPC exists, and the portals cannot see device connectivity.
* **PR Scope:**
  1. `proto/oci/internal_bll/v1/internal_bll.proto`: add `ReportDeviceHeartbeat(serial_number, camera_status optional)`, which fans out to `facilities-api` (`RecordAssetHeartbeat`) and, when `camera_status` is set, `core-invest-api` (`UpdateCameraStreamStatus`). Add `ListDeviceConnectivity(zone_id optional)`. Implement both in `internal/app/internalbll/`. The BLL performs the cross-domain fan-out; Domain APIs never call each other.
  2. `cmd/mock-hardware`: send a heartbeat every 60s for its camera serial, taken from a new `MOCK_CAMERA_SERIAL` env var set in both mock-hardware manifests. Fall back to the existing fixture constant. Add the serial to the seed data in both configmaps if it is not already present.
  3. `internal/app/employeebff/`: add `GET /facilities/devices/connectivity` (session-authenticated), and document it in `api/openapi.yaml`.
* **Verification Gate:**
  * [offline] BLL unit tests with mock Domain clients verify fan-out and error mapping (unknown serial → `NOT_FOUND` → HTTP 404 at the BFF).
  * [cluster] In `oci-dev`, the connectivity endpoint shows the mock camera online, and it shows offline about 3 minutes after `kubectl scale deploy/mock-hardware-dev --replicas=0 -n external-sim`.

#### 7.10 Extend `core_invest.portfolio_snapshots` for Daily NAV
* **Depends On:** `7.8` (for `scripts/migrations/`).
* **Current Codebase State:** `core_invest.portfolio_snapshots` (`snapshot_id`, `portfolio_id`, `valuation_usd`, `holdings_summary`, `triggered_by_event_id`, `snapshot_at`) exists in `scripts/init.sql`, `k8s/oci-core/postgres-core-configmap.yaml`, and `k8s/oci-dev/postgres-dev-configmap.yaml`. No Go code reads or writes it.
* **Design Decision (fixed):** **Extend the existing table** rather than creating a new one. `valuation_usd` is the NAV.
* **PR Scope:**
  1. In all three SQL copies, and in `scripts/migrations/001-portfolio-snapshot-daily-nav.sql`, add idempotently: `snapshot_type VARCHAR(20) NOT NULL DEFAULT 'observation' CHECK (snapshot_type IN ('observation','daily_eod'))`, `snapshot_date DATE`, `cash_usd DECIMAL(14,2) NOT NULL DEFAULT 0.00`, `equity_usd DECIMAL(14,2) NOT NULL DEFAULT 0.00`, `daily_return_pct DECIMAL(8,4) NOT NULL DEFAULT 0.0000`, plus `CREATE UNIQUE INDEX IF NOT EXISTS uq_portfolio_daily_eod_snapshot ON core_invest.portfolio_snapshots (portfolio_id, snapshot_date) WHERE snapshot_type = 'daily_eod'`.
  2. Add about 90 days of `daily_eod` seed rows for the dev customer portfolios to `scripts/seed-dev.sql` and the `oci-dev` configmap.
  3. `internal/repository/core_invest/`: add `UpsertDailySnapshot` (`ON CONFLICT ... DO UPDATE`) and `ListSnapshots(portfolioID, from, to, snapshotType)` (pgx + mock).
  4. `core_invest.proto`: add `RecordDailySnapshot` and `ListPortfolioSnapshots`. Regenerate stubs and wire the handlers in `cmd/core-invest-api`.
* **Verification Gate:**
  * [offline] Repository tests cover upsert idempotency (two writes for the same day → one row) and date-range filtering. Extend the `init.sql` assertion to require `uq_portfolio_daily_eod_snapshot`.
  * [cluster] `psql -f scripts/migrations/001-*.sql` runs twice cleanly against the existing `oci-dev` database.

#### 7.11 Daily NAV Snapshot Batch & BLL History Service
* **Depends On:** `7.10`.
* **Current Codebase State:** `getHistoricalValuationPoints` in `internal/app/customerbll/brokerage.go` takes a `*alpaca.Client` parameter and falls back to hardcoded `0.78 / 0.85 / 0.92` multipliers. Scheduled jobs exist in two forms: one-shot CronJobs that run `oci-worker` with `BATCH_MODE` (`ach-recurring`, `eom-statements` in `cmd/worker/main.go` and `k8s/*/batch-jobs.yaml`), **and** hourly tickers in `internal/app/worker/schedulers.go` that the long-running worker also starts. Recurring ACH is therefore triggered by both today.
* **Design Decision (fixed):** The NAV job uses **only** the CronJob + `BATCH_MODE` pattern. Do **not** add a ticker to `schedulers.go`. (Removing the existing duplicate ACH/EOM tickers is out of scope; leave a `TODO` referencing this step.)
* **PR Scope:**
  1. `customer_bll.proto`: add `CaptureDailySnapshots()` and `GetPortfolioHistory(portfolio_id, period)` with `period ∈ {1M, 3M, 6M, 1Y, YTD}`. Implement both in `internal/app/customerbll/`. Capture computes cash, equity, and daily return per active portfolio from the broker client and calls `RecordDailySnapshot`. History reads `ListPortfolioSnapshots`.
  2. In `brokerage.go`, change `getHistoricalValuationPoints` to read from `core-invest-api` snapshots and return an empty series when there are none. **Delete the multiplier fallback.** Change its broker parameter from `*alpaca.Client` to a small consumer-side interface (`historyProvider`) declared in `brokerage.go`. Other files in `customerbll` that import the broker driver are out of scope.
  3. `internal/app/worker/batch_jobs.go`: add `RunDailyNAVSnapshotBatch`, and dispatch `BATCH_MODE=nav-snapshot` in `cmd/worker/main.go`. Add a `nav-snapshot` CronJob (`0 21 * * 1-5`, UTC) to `k8s/oci-core/batch-jobs.yaml` and `k8s/oci-dev/batch-jobs.yaml`.
* **Verification Gate:**
  * [offline] `! grep -n "0\.78\|0\.85\|0\.92\|alpaca" internal/app/customerbll/brokerage.go`
  * [offline] BLL tests: history for each period returns seeded points in date order; capture writes one snapshot per active portfolio; an empty history returns no synthetic points.

#### 7.12 Portfolio History in Customer BFF & Portal
> [!NOTE] **Status: Completed**

---

### Phase 3: Stateful Store & OpenTelemetry Observability

#### 7.13 Stateful Store Exporter Hardening
> [!NOTE] **Status: Completed**

#### 7.14 Stateful Store Alert Rules
* **Depends On:** `7.13`.
* **Current Codebase State:** `alert_rules.yml` in `k8s/monitoring/prometheus.yaml` has groups for RabbitMQ, COTS, PVCs, Kubernetes object state, backups, and PKI, but none for Valkey, PostgreSQL, or Meilisearch.
* **PR Scope:**
  1. Add a `valkey_alerts` group (`ValkeyMemorySaturation` `> 0.85` of `redis_memory_max_bytes` when that metric is non-zero; `ValkeyKeyEvictions` `increase(redis_evicted_keys_total[5m]) > 0`).
  2. Add a `postgresql_alerts` group (`PostgresConnectionSaturation` `> 0.80` of `pg_settings_max_connections`; `PostgresDeadTupleBloat`; `PostgresXIDWraparoundRisk` on `pg_database_age`; `PostgresLowCacheHitRatio` `< 0.99`).
  3. Add a `meilisearch_alerts` group (`MeilisearchDown` `up{job="meilisearch"} == 0`; `MeilisearchDiskUsageHigh` on the `meilisearch-data-pvc` kubelet volume stats).
  4. Give every rule `severity` and `tier` labels, and document thresholds and remediation in `docs/runbooks/stateful-store-alerts.md`.
* **Verification Gate:**
  * [offline] A Python check parses the embedded `alert_rules.yml` and asserts each named alert exists with `severity` and `tier`. If `promtool` is available, `promtool check rules` passes.

#### 7.15 Drop Grafana Beyla & Wire OpenTelemetry Business Metrics
* **Depends On:** `7.11`, `7.13`.
* **Current Codebase State:** `pkg/telemetry` (`InitProvider`, OTLP to `alloy.monitoring.svc.cluster.local:4317`), `otelhttp`, and `otelgrpc` are wired into every service, so eBPF auto-instrumentation (Grafana Beyla) is redundant. Beyla appears only in historical README prose, with no manifests. Six of the eight helpers in `pkg/telemetry/metrics.go` (`RecordOrderExecutionLatency`, `RecordBracketOrderFill`, `RecordFelineBehavioralEvent`, `RecordCareScheduleUpdate`, `RecordEmergencyHoldToggle`, `RecordPebbleMTTA`) have no callers.
* **Design Decision (fixed):** Grafana Beyla is dropped. Telemetry uses only the OpenTelemetry Go SDK.
* **PR Scope:**
  1. Confirm and enforce Beyla removal: add a check to `scripts/verify-ci-contracts.sh` that fails if `beyla` appears under `k8s/`, `cmd/`, `internal/`, `pkg/`, `scripts/`, or `docs/`.
  2. Add `RecordFelineInferenceConfidence` (histogram `oci.feline.inference.confidence`) to `metrics.go`.
  3. Wire the callers:
     * `internal/app/customerbll/orders.go`: order execution latency and bracket fills.
     * `internal/app/worker/consumers_invest.go`: behavioral event and inference confidence when handling `cat_spotted`.
     * `internal/app/internalbll/`: care schedule update and medical-hold toggle handlers.
     * `cmd/pebble-proxy/alerts_handlers.go`: MTTA on ACK, measured from the alert's creation time.
  4. Add tests using `sdkmetric.NewManualReader()` asserting each helper records a data point.
* **Verification Gate:**
  * [offline] `go test -race ./pkg/telemetry/... ./internal/app/... ./cmd/pebble-proxy/...`. Every `Record*` function in `metrics.go` has at least one non-test caller (checked with `grep`).

#### 7.16 Trace Context Propagation Over RabbitMQ & Grafana Correlation
> [!NOTE]
> **Status: Completed**
> OpenTelemetry `TextMapCarrier` implemented for Watermill metadata. Publisher context injection and router trace extraction middleware implemented with `correlation_id` injection from spans. Grafana tempo datasource updated for cross-datasource correlation to Loki and Prometheus.

---

### Phase 4: Fault Injection & Chaos Engineering

> [!IMPORTANT]
> **Chaos Targeting Invariant:** Fault injection and chaos may only affect `oci-dev` and the **dev mock instances** (`mock-*-dev`) in `external-sim`. The unsuffixed mocks in `external-sim` serve `oci-core` and must never be fault-injected or disrupted. Nothing in `oci-core` or `oci-internal` may be stopped, scaled, or mutated. To simulate an `oci-internal` COTS system being "down", apply a temporary `NetworkPolicy` **in `oci-dev`** that blocks egress from `oci-dev` workloads to `oci-internal`.

#### 7.17 Dev-Only Fault Injection in Mock Bank & Mock Broker
* **Depends On:** `7.3`, `7.6`.
* **Current Codebase State:** Neither mock has fault-injection controls. After `7.6`, dedicated `mock-bank-dev` and `mock-broker-dev` instances serve only `oci-dev`.
* **PR Scope:**
  1. Add a fault controller to each mock, registered **only when `ENABLE_FAULT_INJECTION=true`**. Set that variable only in the `mock-*-dev.yaml` manifests, and add a `verify-ci-contracts.sh` check that it never appears in the unsuffixed mock manifests.
     * `cmd/mock-bank`: `GET/PUT/DELETE /sim/faults` with `{outage: bool, latency_ms: int, reject_transfers: bool}`.
     * `cmd/mock-broker`: `GET/PUT/DELETE /sim/faults` with `{outage, latency_ms, reject_orders: bool, halted_symbols: []string}`.
     * Fault state is in-memory; `DELETE` resets to healthy. Return 404 when disabled.
  2. In `internal/app/customerbll/`, make sure broker/bank calls use bounded context deadlines and map upstream failures to gRPC `Unavailable`, `DeadlineExceeded`, or `FailedPrecondition` (rejections), without leaking vendor names in messages.
  3. In `simulation/`, add an opt-in `-fault-scenarios` flag that sets faults on the `-dev` mocks, asserts the BFF returns structured 5xx/422 responses, then always resets with `defer`.
* **Verification Gate:**
  * [offline] Mock handler tests cover enabled/disabled registration and each fault type. BLL tests check error mapping. `./scripts/verify-ci-contracts.sh`.
  * [cluster] `-fault-scenarios` against `oci-dev` passes and leaves `/sim/faults` healthy.

#### 7.18 RabbitMQ Poison-Pill & Dead-Letter Handling
* **Depends On:** `7.16`.
* **Current Codebase State:** `internal/app/worker/router.go` uses no `PoisonQueue` or `Retry` middleware. `RabbitMQDeadLetterQueueNotEmpty` exists in Prometheus and expects queues matching `.*dlq.*`.
* **PR Scope:**
  1. Add Watermill `middleware.Retry` (3 attempts, exponential backoff) followed by `middleware.PoisonQueue` publishing to `<topic>.dlq` in `router.go`. Add an envelope validation middleware that rejects (sends straight to the poison queue without retry) messages that fail to unmarshal or lack `EventID`/`OccurredAt`.
  2. Poison log lines contain only `event_id` (if parseable), `topic`, `correlation_id`, and the validation error, never the payload (CWE-532).
* **Verification Gate:**
  * [offline] GoChannel router tests: malformed, missing-field, and handler-error messages land in `.dlq`; a following valid message is still processed; and captured log output contains no payload bytes.

#### 7.19 Sandboxed Chaos Suite for `oci-dev`
* **Depends On:** `7.14`, `7.17`, `7.18`.
* **Current Codebase State:** No chaos tooling exists. In `oci-dev`, only `internal-bll` calls `oci-internal` (Nextcloud, Forgejo, ERPNext, Snipe-IT URLs in `k8s/oci-dev/bll-services.yaml`).
* **PR Scope:**
  1. Add `scripts/run-chaos-suite.sh`. It wraps every mutating `kubectl` call in a guard function that refuses any namespace other than `oci-dev`, and refuses `external-sim` resources not suffixed `-dev`. It installs `trap` cleanup on `EXIT`/`INT`/`ERR` that deletes every applied chaos policy and restores replica counts. Scenarios:
     * **COTS outage:** apply `k8s/chaos/oci-dev-deny-internal-egress.yaml` (selects `app: internal-bll` in `oci-dev`, allows egress everywhere except the `oci-internal` namespace) and verify that employee-portal COTS features degrade with clean errors.
     * **Tier partitions:** temporary `NetworkPolicy`s in `oci-dev` blocking BFF→BLL and BLL→Domain API traffic; verify bounded gRPC errors and recovery after removal.
     * **Database restart:** scale `statefulset/postgres-dev` to 0 and back to 1; verify the Domain APIs reconnect without crash loops.
     * **Pod kills:** delete one pod each of a Domain API, a BLL, and a BFF in `oci-dev`; verify they are replaced and become Ready.
     * **External outage:** use `7.17` faults on the `-dev` mocks.
     * After all scenarios, run the simulator against `oci-dev` and require it to pass.
  2. Document scenarios, guards, and expected alerts in `docs/runbooks/chaos-engineering.md`.
* **Verification Gate:**
  * [offline] `bash -n`. A test with a stub `kubectl` proves the guard rejects `-n oci-core`, `-n oci-internal`, and `-n external-sim` resources without the `-dev` suffix, and that cleanup runs on simulated failure.
  * [cluster] A full suite run against `oci-dev` ends with the simulator passing and no chaos `NetworkPolicy` left behind.

---

### Phase 5: Exploratory (Architecture Undecided)

#### 7.20 Zero-Trust Workload Identity & Dynamic Secret Management
* **Depends On:** Phases 1–4, plus an ADR in `docs/adr/` choosing the tooling.
* **Status:** *Intentionally undecided.* No agent should start this until the ADR exists and this item has been rewritten into PR-sized steps.
* **Exploration Areas:**
  * **Dynamic secrets:** replace static Kubernetes `Secret` manifests with short-lived credentials (candidates: HashiCorp Vault, OpenBao, External Secrets Operator, CSI Secret Store).
  * **Workload identity:** cryptographically attested workload identity (candidates: SPIFFE/SPIRE, cert-manager `csi-driver-spiffe`, OpenZiti) alongside or replacing the current `pkg/mtls` dynamic CA reloading.
