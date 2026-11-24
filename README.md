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
* **Daily NAV Portfolio Snapshots & BLL History (`7.10`, `7.11`):** Extended `core_invest.portfolio_snapshots` with `snapshot_type`, `snapshot_date`, `cash_usd`, `equity_usd`, `daily_return_pct`, and partial unique index `uq_portfolio_daily_eod_snapshot` (synchronized across `scripts/init.sql`, both Postgres configmaps, and `scripts/migrations/001-portfolio-snapshot-daily-nav.sql`), with 90-day dev seed data and `RecordDailySnapshot` / `ListPortfolioSnapshots` gRPC endpoints in `core-invest-api`. `CustomerBLLService` exposes `CaptureDailySnapshots` and snapshot-backed `GetPortfolioHistory` (`1M|3M|6M|1Y|YTD`) using a consumer-side `historyProvider` interface in `brokerage.go`, driven by `RunDailyNAVSnapshotBatch` (`BATCH_MODE=nav-snapshot`) and the `nav-snapshot` CronJob (`0 21 * * 1-5` UTC).

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
* **Edge Device Heartbeats & Connectivity Derivation (`7.8`, `7.9`):** `FacilitiesService` exposes `RecordAssetHeartbeat` (updating `facilities.hardware_assets.last_ping_at`, returning `NOT_FOUND` for unknown serials) and `ListAssetConnectivity` (deriving `online` at read time from the 180s staleness threshold without background sweeps); `CoreInvestService` exposes `UpdateCameraStreamStatus` to update `core_invest.camera_streams.status`. `InternalBLLService` exposes `ReportDeviceHeartbeat` (fanning out to `facilities-api` and `core-invest-api`) and `ListDeviceConnectivity`, `mock-hardware` reports heartbeats every 60s using `MOCK_CAMERA_SERIAL`, and `employee-bff` exposes session-authenticated `GET /facilities/devices/connectivity`.

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

### 5.21 CI/CD Supply Chain Hardening (Roadmap Phase 1)
* **Toolchain Single Source of Truth:** Root `.go-version` drives `ci.yaml` Go installs and every Go Dockerfile via `ARG GO_VERSION`; `simulation/cmd/simulator -validate-flags` and `scripts/verify-ci-contracts.sh` fail fast on toolchain drift or renamed simulator flags.
* **Forgejo Persistent Storage & Eviction Protection:** `forgejo-data-pvc` at `/var/lib/gitea`, `system-cluster-critical` runner with a 20Gi `forgejo-runner-docker-pvc`, daily `registry-gc` CronJob with `sha-*` tag retention, 24h image pruning in the build job, and the `NodeFilesystemLowSpace` alert.
* **Immutable-SHA Promotion & Post-Deploy Verification:** `ci.yaml` deploys exact `:sha-<COMMIT>` images to `oci-dev` and promotes those exact immutable tags to `oci-core`, validated post-rollout by `scripts/verify-deployed-images.sh`.
* **Registry Pre-Flight & Single-Command Deploy:** `deploy-k8s.sh` pre-flights all referenced images against the registry before applying, `ci-status.sh [--wait]` surfaces Forgejo Actions run/job state with failed-log tailing, and `deploy-latest.sh` chains test, sync, pipeline wait, image verification, and the live simulator in one command.
* **Dedicated Dev Mock Instances:** `mock-bank-dev` / `mock-broker-dev` / `mock-hardware-dev` serve only `oci-dev` (own cert-manager certificate, own RabbitMQ/`oci-dev` BLL wiring) so fault injection and chaos can never reach `oci-core`; contract checks enforce the environment-to-mock wiring.

### 5.22 Edge Device Heartbeats & Daily NAV Portfolio History
* **Edge Device Heartbeats & Connectivity Tracking:** `facilities.proto` (`RecordAssetHeartbeat`, `ListAssetConnectivity`), `core_invest.proto` (`UpdateCameraStreamStatus`), and `internal_bll.proto` (`ReportDeviceHeartbeat`, `ListDeviceConnectivity`) track live hardware connectivity (`last_ping_at >= now - stale_after_seconds`, default 180s). `cmd/mock-hardware` reports 60s heartbeats for `QR-CAM-ORANGE-01`, exposed via session-authenticated `GET /facilities/devices/connectivity` on Employee BFF and documented in `api/openapi.yaml`.
* **Daily NAV Snapshot Engine & Portfolio History:** `core_invest.portfolio_snapshots` stores idempotent `daily_eod` valuations (`uq_portfolio_daily_eod_snapshot`, `scripts/migrations/001-portfolio-snapshot-daily-nav.sql`) captured by the `nav-snapshot` CronJob (`0 21 * * 1-5` UTC, `BATCH_MODE=nav-snapshot`). Customer BLL (`GetPortfolioHistory`), Customer BFF, and the Customer Portal expose snapshot-backed `1M|3M|6M|1Y|YTD` performance history.

### 5.23 Dev-Only Fault Injection, Upstream Resilience & Sandboxed Chaos Suite
* **Dev-Only Fault Controllers (`cmd/mock-bank`, `cmd/mock-broker`):** `GET/PUT/DELETE /sim/faults` registered strictly when `ENABLE_FAULT_INJECTION=true` (configured only in `mock-*-dev.yaml`, enforced by `scripts/verify-ci-contracts.sh` so it never appears in unsuffixed mock manifests or `k8s/oci-core/`).
* **Bounded Upstream Deadlines & Vendor-Free Error Mapping (`internal/app/customerbll`, `internal/app/customerbff`):** Structured `APIError` types in `pkg/broker/alpaca` and `pkg/bank/plaid`, 5s default bounded context deadlines (`UPSTREAM_TIMEOUT`) on all broker/bank calls, gRPC `Unavailable` / `DeadlineExceeded` / `FailedPrecondition` error mapping without leaking vendor names, and corresponding `503`/`504`/`422` structured JSON responses in `customer-bff`.
* **Simulator Fault Verification & Sandboxed Chaos Suite (`simulation/driver_fault_patch.go`, `scripts/run-chaos-suite.sh`, `scripts/run-chaos-audit.sh`):** Opt-in `-fault-scenarios` mode injects outage (`5xx`) and halted-symbol (`422`) faults on `-dev` mocks and resets `/sim/faults` via `defer`; sandboxed chaos scripts exercise `oci-dev` workloads and `-dev` mocks without ever mutating `oci-core`.

### 5.24 RabbitMQ Poison-Pill & Dead-Letter Handling
* **Watermill Resilience Middleware (`internal/app/worker/poison_middleware.go`, `router.go`):** Every worker handler is wrapped with per-topic `middleware.PoisonQueue` (`<topic>.dlq`), CWE-532-safe poison logging (`event_id`, `topic`, `correlation_id`, and error only — never payload bytes), pre-retry `EnvelopeValidationMiddleware` (rejecting malformed JSON or missing `EventID`/`OccurredAt` straight to `.dlq`), and 3-attempt exponential-backoff `middleware.Retry`, with upfront AMQP `.dlq` queue declaration via `SubscribeInitialize` so dead-lettered messages trigger Prometheus `RabbitMQDeadLetterQueueNotEmpty`.

### 5.25 Stateful Store Exporters, Alert Rules & OpenTelemetry Business Metrics
* **Stateful Store Exporter Hardening & Alert Rules (`k8s/monitoring/prometheus.yaml`, `docs/runbooks/stateful-store-alerts.md`):** Hardened PostgreSQL, Valkey, and Meilisearch exporters and added `valkey_alerts` (`ValkeyMemorySaturation`, `ValkeyKeyEvictions`), `postgresql_alerts` (`PostgresConnectionSaturation`, `PostgresDeadTupleBloat`, `PostgresXIDWraparoundRisk`, `PostgresLowCacheHitRatio`), and `meilisearch_alerts` (`MeilisearchDown`, `MeilisearchDiskUsageHigh`) with `severity`/`tier` labels, runbook remediation, and CI contract validation in `scripts/verify-ci-contracts.sh`.
* **OpenTelemetry Business Metrics (`pkg/telemetry/metrics.go`, `internal/app/...`, `cmd/pebble-proxy`):** Eliminated Grafana Beyla across manifests and source code (enforced by `scripts/verify-ci-contracts.sh`) and wired all 10 custom OTel metrics helpers (`RecordOnboardingAttempt`, `RecordBankLinkingAttempt`, `RecordOrderExecutionLatency`, `RecordBracketOrderFill`, `RecordFelineBehavioralEvent`, `RecordFelineInferenceConfidence`, `RecordCareScheduleUpdate`, `RecordEmergencyHoldToggle`, `RecordPebbleMTTA`, `RecordUpstreamRequest`) to domain callers with `sdkmetric.NewManualReader()` unit tests.

### 5.26 Operational Hardening & Staging Hygiene
* **Prometheus TSDB Sizing & Retention Contract:** Enforced `requests.memory: 512Mi`, `limits.memory: 2Gi`, `--storage.tsdb.retention.time=3d`, and `--storage.tsdb.retention.size=2GB` in `scripts/verify-ci-contracts.sh` (Check #10) and documented Prometheus TSDB sizing and WAL recovery procedures in `docs/runbooks/cluster-load-testing.md`.
* **Configurable BFF Rate-Limit Policies:** Added `RATE_LIMIT_ENABLED` (default `true`) and `RATE_LIMIT_MULTIPLIER` (default `1`) support in `pkg/cache/valkey/ratelimit.go` and `internal/app/customerbff/router.go`; kept strict production defaults in `k8s/oci-core/`, configured `RATE_LIMIT_ENABLED=false` strictly in `k8s/oci-dev/oci-customer-bff.yaml` and `k8s/oci-dev/oci-employee-bff.yaml` (enforced by Check #11 in `scripts/verify-ci-contracts.sh`), and removed the background `kubectl exec` `ratelimit:*` pruning loop from `scripts/run-load-test.sh`.
* **Live Nextcloud COTS Credentials & WebDAV Hardening:** Provisioned `NEXTCLOUD_ADMIN_USER` and `NEXTCLOUD_ADMIN_PASS` in `k8s/oci-core/oci-secrets.yaml`, `k8s/oci-dev/oci-secrets.yaml`, and `scripts/deploy-k8s.sh`; wired `secretKeyRef` on `internal-bll` in both `bll-services.yaml` manifests; added CWE-532-safe startup logging in `cmd/internal-bll/main.go`; and hardened `pkg/cots/nextcloud/client.go` with 100-document result capping, bounded 16 MiB body reads, WebDAV parent-directory `MKCOL` creation on `409`/`404`/`403`, and graceful HTML-redirect fallback for Nextcloud 28 Apache.
* **Automated `oci-dev` Database Retention & Seed Reset:** Added `scripts/prune-dev-db.sh` (with strict `oci-core`/prod guards and `--dry-run` seed-ID verification) and `k8s/oci-dev/dev-db-retention-cronjob.yaml` (`30 3 * * *` UTC) to retain baseline seed rows plus the latest 500 synthetic rows per high-churn table and execute `VACUUM ANALYZE`, documented in `docs/runbooks/cluster-load-testing.md`.

### 5.27 Chaos-Driven Observability, Alerting, Logging & Resilience Hardening
* **Centralized Kubernetes Pod Log Shipping:** Added `ServiceAccount`/`ClusterRole`/`ClusterRoleBinding`, Downward API `HOSTNAME` (`spec.nodeName`), read-only `/var/log/pods` and `/var/log/containers` mounts, and a `kubernetes-pods` CRI scrape job in `k8s/logging/promtail.yaml` indexing `namespace`, `pod`, `container`, and `app` labels in Loki (enforced by Check #14 in `scripts/verify-ci-contracts.sh`).
* **Prometheus Availability, Error-Rate, Upstream-Fault & `oci-dev` Alerting:** Enabled `rabbitmq_prometheus` (`:15692`, `/metrics/per-object`) in `k8s/oci-dev/rabbitmq.yaml`, added `oci-dev` RabbitMQ and ZITADEL scrape targets to `k8s/monitoring/prometheus.yaml`, added `RecordUpstreamRequest` (`oci.upstream.requests.total`) in `pkg/telemetry/metrics.go` + `internal/app/customerbll/`, and added `DeploymentZeroAvailableReplicas`, `StatefulSetReplicasUnavailable`, `ScrapeTargetDown`, updated `MeilisearchDown`, `HighHTTP5xxErrorRate`, `UpstreamBankOrBrokerErrorSpike`, and `UpstreamBankOrBrokerTimeoutSpike` alert rules (enforced by Check #15 in `scripts/verify-ci-contracts.sh`).
* **Baseline-Green Gatus Monitoring:** Switched the 6 mTLS gRPC probes in `k8s/monitoring/gatus.yaml` to `tcp://` connectivity checks, added `monitoring/gatus` ingress rules to the BLL and Domain API `NetworkPolicy` manifests, switched the 3 portal/marketing checks to in-cluster `/healthz` URLs, and added TCP checks for `erpnext-service` and `frappe-hr-service` — achieving **19 / 19 (100%) healthy** at baseline (enforced by Check #16 in `scripts/verify-ci-contracts.sh`).
* **BFF/BLL Caller Error Logging, Sanitization & Latency Cascade Hardening:** Added structured `slog.Error` caller logging, `503`/`504` gRPC mapping, and internal IP/DB/vendor sanitization across `customerbff`, `employeebff`, `internal/handler`, `customerbll`, and `internalbll`; skipped `GetPortfolio` during `handleLogin` when `name != ""`; and short-circuited sequential broker timeouts in `customerbll.GetPortfolio` to eliminate the `C6` login cascade.
* **Mock `/sim/faults` Auth/Transport & ACH Clearing Fixes:** Exempted `/sim/faults` from API auth in `cmd/mock-bank/auth.go` and `cmd/mock-broker/auth.go`, reused `d.client.transport` (`OCI_INGRESS_IP`-aware) in `simulation/driver_fault_patch.go`, forwarded `BankAccount`/`AccountID`/`DayOfMonth` in `handleCreateDeposit`, skipped `CreateRecurringDeposit` for `one_time` deposits (eliminating `SQLSTATE 23514`), added bank account resolution + NSF R01 pre-authorization in `InitiateACHDeposit`, and passed resolved `accountID` into `achPayload["bank_account"]` (reducing `events.banking.ach_transfer_requested.v1.dlq` from 4,200 messages to **0**).

### 5.28 Zero-Trust Workload Identity & Dynamic Secret Management
* **SPIFFE URI SAN Workload Authorization (`7.20a`):** Added `VerifyServerIdentity` (`spiffe://oci.local/ns/<ns>/sa/<sa>` and DNS/IP SAN validation) in `pkg/mtls/mtls.go` and unary/stream gRPC SPIFFE authorization interceptors (`WithAllowedPeerSPIFFEIDs`, `AllowedDomainAPICallers`, `AllowedCustomerBLLCallers`, `AllowedInternalBLLCallers`) in `pkg/grpcutil/interceptors.go`. Provisioned dedicated `ServiceAccount`s and `POD_NAMESPACE` Downward API injection across Domain APIs, BLLs, BFFs, Worker, Pebble Proxy, and `mock-hardware`, and enforced fail-closed mTLS startup in `cmd/internal-bll/main.go`.
* **Ephemeral In-Memory X.509-SVID Mounts (`7.20b`):** Deployed `cert-manager-csi-driver-spiffe` (`k8s/ha/lan/tls/csi-driver-spiffe.yaml`, `driver: spiffe.csi.cert-manager.io` with `spiffe.csi.cert-manager.io/fs-group: "65532"` and pod `supplementalGroups: [65532]`) mounting 1-hour TTL certificates on in-memory `tmpfs` at `/etc/tls/certs` hot-reloaded by `pkg/mtls.DynamicCertReloader`, removing static `kind: Certificate` CRs and `*-tls-certs` Secret volumes (enforced by Check #17 in `scripts/verify-ci-contracts.sh`).
* **OpenBao & Dynamic Schema-Isolated PostgreSQL Credentials (`7.20c`):** Deployed OpenBao `2.1.0` (`k8s/oci-core/openbao.yaml`) and automated idempotent Kubernetes Auth and Database Secrets Engine provisioning in `scripts/init-openbao.sh` issuing 24-hour dynamic PostgreSQL roles strictly scoped to each Domain API's schema (`workforce`, `facilities`, `core_invest`, `ops`), removed cross-schema `ops.*` queries from `internal/repository/core_invest/gdpr_pgx.go`, and added file-backed `pgxpool` credential hot-reloading (`BeforeConnect`, `MaxConnLifetime = 30m`) in `pkg/dbpool/dbpool.go`.
* **External Secrets Operator (ESO) & Static Secret Elimination (`7.20d`):** Installed External Secrets Operator (`v2.15.0`) with `ClusterSecretStore/openbao-kv-store` (`k8s/oci-core/external-secrets-stores.yaml`), `ExternalSecret` (`external-secrets.io/v1`), and per-domain `VaultDynamicSecret` (`generators.external-secrets.io/v1alpha1`) manifests in `k8s/oci-core/oci-secrets.yaml`, `k8s/oci-dev/oci-secrets.yaml`, and `k8s/oci-internal/internal-secrets.yaml`, eliminating raw static `kind: Secret` credential manifests (enforced by Check #18 in `scripts/verify-ci-contracts.sh`).

### 5.29 Artifact Management & Software Supply Chain Security
* **Architectural Invariant SAST, Linter, Vulnerability & Secret Gates (`7.30`):** Added `.semgrep.yml` mechanically enforcing `AGENTS.md` §3 domain/tier/vendor/hardcoding invariants, `.golangci.yml` (v2 with `gosec`), and `.gitleaks.toml`, wired alongside `govulncheck ./...` into `.forgejo/workflows/ci.yaml` and Check #19 in `scripts/verify-ci-contracts.sh`.
* **Distroless Runtime Containers & Restricted Pod Security Standards (`7.31`):** Migrated all 13 Go service Dockerfiles (`cmd/*/Dockerfile`) and `web/marketing/Dockerfile` (`containerPort: 8080`) to `gcr.io/distroless/static-debian12:nonroot` (`USER nonroot:nonroot`) and enforced restricted `securityContext` (`runAsNonRoot: true`, `runAsUser: 65532`, `readOnlyRootFilesystem: true`, `allowPrivilegeEscalation: false`, `capabilities.drop: ["ALL"]`) across all Go Deployments and CronJobs in `k8s/oci-core/`, `k8s/oci-dev/`, and `k8s/external-sim/` (enforced by Check #20 in `scripts/verify-ci-contracts.sh`).
* **Forgejo Package Registry & Firmware Artifact Publishing (`7.32`):** Enabled `FORGEJO__packages__ENABLED=true` in `k8s/oci-internal/forgejo.yaml`, added `scripts/publish-firmware-package.sh` and CI packaging steps uploading generic OTA `.bin` firmware and Pebble `.pbw` bundles, and updated `internal/app/worker` (`ota_pipeline_handler`) and `simulation/actions.go` (`bob_builder`) to publish, fetch, and verify SHA-256 checksums against Forgejo Packages (enforced by Check #21 in `scripts/verify-ci-contracts.sh`).
* **Harbor Enterprise Container Registry, Pull-Through Proxy Caches & Trivy Gate (`7.33`):** Replaced barebones `registry:2` (`k8s/registry/registry.yaml`, `registry-gc-cronjob.yaml`) with Harbor (`k8s/registry/harbor.yaml`: Core, Portal, Registry, Jobservice, and embedded Trivy scanner on `:32000` / `registry.oci.local` with Forgejo OIDC auth and `proxy-dockerhub`/`proxy-ghcr`/`proxy-gcr` pull-through caches) and added a `trivy image --exit-code 1 --severity CRITICAL,HIGH` gate in `.forgejo/workflows/ci.yaml` (enforced by Check #22 in `scripts/verify-ci-contracts.sh`).
* **CycloneDX SBOMs & Cryptographic Cosign Signing (`7.34`):** Provisioned `ConfigMap/oci-cosign-pub` across `k8s/oci-core/`, `k8s/oci-dev/`, and `k8s/external-sim/`, wired `syft` CycloneDX SBOM generation and `cosign sign` / `cosign attest` into `.forgejo/workflows/ci.yaml`, added `cosign_signature` to `facilities.firmware_releases` (synced across `scripts/init.sql`, both Postgres configmaps, `scripts/migrations/002-firmware-cosign-signature.sql`, and `facilities.proto`), and added Ed25519/ECDSA Cosign signature verification (`pkg/cosignverify`, `cmd/mock-hardware/ota_verify.go`, `internal/app/worker`) rejecting unsigned or tampered firmware (enforced by Check #23 in `scripts/verify-ci-contracts.sh`).
* **Kyverno Cluster Admission Control Policies (`7.35`):** Deployed Kyverno (`k8s/security/kyverno.yaml`) and four `Enforce`-mode `ClusterPolicy` manifests in `k8s/security/kyverno-policies.yaml` (`verify-oci-image-signatures`, `disallow-latest-tag`, `enforce-restricted-pss`, `forbid-tls-subpath-mounts`) validated against all `oci-core`/`oci-dev` workloads and negative test fixtures in `k8s/security/fixtures/invalid-manifests.yaml` (enforced by Check #24 in `scripts/verify-ci-contracts.sh`).

### 5.30 Enterprise Zero-Trust Networking & Edge Security
* **Default-Deny Zero-Trust `NetworkPolicy` Microsegmentation Matrix (`7.36`):** Deployed baseline `default-deny-all` (`policyTypes: [Ingress, Egress]`) and explicit 5-tier Layer-3/Layer-4 Ingress + Egress allowlists in `k8s/oci-core/network-policies.yaml`, `k8s/oci-dev/network-policies.yaml`, `k8s/oci-internal/network-policies.yaml`, and `k8s/external-sim/network-policies.yaml` guaranteeing BFFs have zero network path to `:5432` or `:50051–50054`, Domain APIs have zero east-west paths, and `external-sim` has zero database egress (enforced by Check #25 in `scripts/verify-ci-contracts.sh`).
* **Kubernetes Gateway API Migration (`7.37`):** Replaced all legacy `networking.k8s.io/v1` `Ingress` resources with Gateway API (`gateway.networking.k8s.io/v1`) `GatewayClass/oci-gateway-class` and `Gateway/oci-edge-gateway` (`k8s/ha/lan/tls/gateway.yaml`) and namespace-scoped `HTTPRoute` manifests (`k8s/oci-core/http-routes.yaml`, `k8s/oci-dev/http-routes.yaml`, `k8s/oci-internal/http-routes.yaml`, `k8s/external-sim/http-routes.yaml`), verified by the `frank_audit_gateway_routes` simulator action and Check #26 in `scripts/verify-ci-contracts.sh`.
* **Edge Web Application Firewall — Coraza OWASP CRS Middleware (`7.38`):** Implemented `pkg/waf/coraza.go` (`waf.Middleware`) with OWASP CRS SQLi, XSS, path-traversal, RCE, and scanner rules tuned for JSON REST and SSE endpoints, wired `RecordWAFBlockedRequest` (`oci.waf.blocked.requests.total`) into `pkg/telemetry/metrics.go`, attached the middleware to `customer-bff` and `employee-bff`, and added simulator WAF attack probe verification (`HTTP 403 Forbidden`) for `Frank Operations` (enforced by Check #27 in `scripts/verify-ci-contracts.sh`).
* **In-Cluster STUN/TURN Media Relay (`coturn`) & WebRTC ICE Discovery (`7.39`):** Deployed `coturn` (`k8s/oci-core/coturn.yaml`, `k8s/oci-dev/coturn.yaml`) with ephemeral HMAC-SHA1 TURN REST API credentials (`--use-auth-secret`) and exposed `GET /api/v1/customer/webrtc/ice-servers` on `customer-bff` (documented in `api/openapi.yaml` and wired into `web/customer-portal/src/App.vue`) returning 1-hour time-bound `stun:`/`turn:` ICE server configurations (enforced by Check #28 in `scripts/verify-ci-contracts.sh`).

### 5.31 Financial & Transactional Integrity, Auditability & Policy-as-Code
* **Transactional SQL Outbox Pattern & Valkey `Idempotency-Key` Middleware (`7.40`):** Added schema-scoped `outbox_events` tables (`workforce.outbox_events`, `facilities.outbox_events`, `core_invest.outbox_events`, `ops.outbox_events`) across `scripts/init.sql`, both Postgres ConfigMaps, and `scripts/migrations/003-transactional-outbox.sql`. Domain API repositories atomically persist `envelope.EventEnvelope[T]` records inside the same `pgx.Tx` transaction as domain mutations, and `oci-worker` runs a gRPC-bridged SQL Outbox Forwarder (`pkg/outbox`, `internal/app/worker/outbox_relay.go`) polling via `customer-bll`/`internal-bll` using `FOR UPDATE SKIP LOCKED` while preserving Phase 9 Zero-Trust `NetworkPolicy` isolation. Added `pkg/cache/valkey/idempotency.go` (`IdempotencyMiddleware`) enforcing client-generated UUID `Idempotency-Key` headers on `POST /api/v1/customer/trading/orders` and `POST /api/v1/customer/deposits` (`X-Idempotent-Replayed: true` on cached replay, `409 Conflict` on concurrent in-flight duplicates), wired into `web/customer-portal/src/App.vue`, `api/openapi.yaml`, and the `chloe_idempotent_order_replay` simulator action (enforced by Check #29 in `scripts/verify-ci-contracts.sh`).
* **Tamper-Evident Cryptographic Hash-Chain Audit Log (`7.41`):** Added append-only `ops.audit_logs` with SHA-256 hash chaining (`ops.compute_audit_log_hash()` under advisory transaction lock) and `BEFORE UPDATE OR DELETE` / `BEFORE TRUNCATE` immutability triggers (`ops.prevent_audit_log_mutation()`) across `scripts/init.sql`, both Postgres ConfigMaps, and `scripts/migrations/004-tamper-evident-audit-log.sql`. Exposed `AppendAuditLog`, `VerifyAuditChain`, and `ListAuditLogs` across `ops-api` and `internal-bll`, wired high-sensitivity mutations (medical holds, employee offboarding, leave status approvals, quant orders, API key revocation, and GDPR erasure with hashed non-PII subjects) to emit audit entries, exposed `GET /api/v1/ops/audit-logs/verify` & `GET /ops/audit-logs/verify` on `employee-bff`, and added the `frank_verify_audit_chain` simulator action (enforced by Check #30 in `scripts/verify-ci-contracts.sh`).
* **Policy-as-Code Authorization Engine — Embedded OPA / Rego Go SDK (`7.42`):** Created `pkg/authz/policy` embedding `github.com/open-policy-agent/opa/v1/rego` with compiled embedded Rego v1 policies (`policies/workforce.rego`, `policies/trading.rego`), propagated authenticated employee actor metadata across BFF $\rightarrow$ BLL gRPC calls (`pkg/grpcutil/actor.go`), wired fail-closed policy enforcement into `internal-bll` and `customer-bll` logging every policy denial (`authz.policy.denied`) to `ops.audit_logs`, and added the negative authorization simulator check `rick_unauthorized_medical_hold_denied` (`HTTP 403 Forbidden`) (enforced by Check #31 in `scripts/verify-ci-contracts.sh`).

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

## 7. Future Work & Technical Roadmap

Completed roadmap work (`Phase 1`–`Phase 9`: steps `7.1`–`7.39`) has been promoted into Section 5 above. Only open future work phases remain below.

### 7.0 Rules That Apply to Future Work

1. **Demo Mocks in Every Environment:** OCI is a demo platform. Both `oci-core` ("prod") and `oci-dev` ("dev") are fed by mock external systems in `external-sim` (bank, broker, edge hardware) that pretend to be real third parties. Mocks feeding `oci-core` are expected and correct. What is forbidden is (a) the *simulation driver* (`simulation/`) targeting `oci-core`, and (b) fault injection or chaos affecting anything `oci-core` depends on.
2. **Baseline Definition of Done:** Every PR must pass the AGENTS.md §4 baseline: `go vet ./...`, `go test -race ./...`, the `scripts/init.sql` assertion check, and `python3 scripts/validate-openapi.py api/openapi.yaml`. PRs touching `web/*` must also pass `npm test` in each changed portal.
3. **Offline vs. Cluster Gates:** Gates marked **[offline]** must pass on any workstation. Gates marked **[cluster]** require the `thinker` test cluster. Never claim a cluster gate passed without running it.
4. **Schema Change Protocol:** The database bootstrap SQL exists in **three copies** that must stay in sync: `scripts/init.sql`, `k8s/oci-core/postgres-core-configmap.yaml`, and `k8s/oci-dev/postgres-dev-configmap.yaml` (seed rows: `scripts/seed-dev.sql` and the `oci-dev` configmap). Bootstrap scripts only run against an empty data volume, so every schema change must be written idempotently (`ALTER TABLE ... ADD COLUMN IF NOT EXISTS`, `CREATE INDEX IF NOT EXISTS`) and also added to a numbered migration file `scripts/migrations/NNN-<name>.sql` that can be applied to existing databases with `psql -f`.
5. **Completion Protocol:** When a future work item merges, document the shipped capability in Section 5 and remove the completed item from Section 7.

---

### Execution Phases

1. **Phase 11 — Enterprise Data Lifecycle, Object Storage, Messaging Sandbox & Contract Testing (`7.43`–`7.47`):** S3-compatible object storage (`Garage`) for statements/1099-Bs/firmware, continuous WAL archiving & PITR via `PgBackRest`, in-cluster `Mailpit` SMTP verification, automated `goose` migrations + `PgBouncer` transaction pooling, and `buf breaking` + `oasdiff` contract gates.
   - [ ] **7.43** S3-Compatible Object Storage (`Garage`) for PDF Statements, 1099-Bs & Firmware Blobs
   - [ ] **7.44** Continuous PostgreSQL WAL Archiving & Point-In-Time Recovery (`PgBackRest` to `Garage` S3)
   - [ ] **7.45** In-Cluster SMTP / Email Sandbox (`Mailpit`) & Simulator Email Receipt Verification
   - [ ] **7.46** Automated Database Migration Runner (`goose`) & `PgBouncer` Transaction Pooling
   - [ ] **7.47** Automated Wire & REST Contract Testing (`buf breaking` & `oasdiff`)

---

### Phase 11: Enterprise Data Lifecycle, Object Storage, Messaging Sandbox & Contract Testing (`7.43`–`7.47`)

Completes the self-hosted enterprise infrastructure with S3 blob storage, point-in-time database recovery, an in-cluster SMTP verification server, automated schema migrations + connection pooling, and wire-level contract breaking-change gates.

#### 7.43 S3-Compatible Object Storage (`Garage`) for PDF Statements, 1099-Bs & Firmware Blobs
* **Depends On:** None.
* **Current Codebase State:** Generated end-of-month investor statements (`core_invest.investment_statements`), Form 1099-B tax documents, and firmware blobs store only metadata rows in PostgreSQL without an underlying S3 object store for binary blobs.
* **PR Scope:**
  1. Deploy **Garage** (lightweight Rust S3-compatible object store, `< 64 MiB` idle RAM) in `k8s/oci-core/garage.yaml` and `k8s/oci-dev/garage.yaml` with dedicated PVCs and buckets `oci-statements`, `oci-firmware`, and `oci-backups`.
  2. Implement a vendor-neutral S3 storage driver in `pkg/blobstore/s3` (using standard AWS S3 v2 SDK pointed at Garage's cluster endpoint, consumed via generic `blobStore` variables per `AGENTS.md` §3 Rule 6).
  3. Update `customer-bll` (`GenerateStatement` / EOM PDF batch runner) and `internal-bll` (`facilities` firmware catalog) to upload generated PDF/1099-B and firmware blobs to Garage and return time-limited pre-signed S3 download URLs.
* **Verification Gate:**
  * [offline] Unit tests with an in-memory/httptest S3 server verify blob upload, SHA-256 content integrity, and pre-signed URL generation.
  * [cluster] Arthur Pendelton's statement generation action uploads a PDF to Garage in `oci-dev` and downloads it via the returned pre-signed URL (`HTTP 200`, `%PDF-` header).

#### 7.44 Continuous PostgreSQL WAL Archiving & Point-In-Time Recovery (`PgBackRest` to `Garage` S3)
* **Depends On:** `7.43`.
* **Current Codebase State:** Database backups rely solely on a daily `pg_dump` CronJob (`scripts/backup-db.sh`), offering a 24-hour Recovery Point Objective (RPO) and no Point-In-Time Recovery (PITR).
* **PR Scope:**
  1. Configure `postgres-core` and `postgres-internal` with `wal_level = replica`, `archive_mode = on`, and a **PgBackRest** sidecar streaming compressed, encrypted Write-Ahead Logs (WAL) and daily differential backups directly to the `oci-backups` bucket in `Garage` (`7.43`).
  2. Add `scripts/verify-pitr-restore.sh` and update `docs/runbooks/stateful-store-alerts.md` with a step-by-step PITR restore drill (`--type=time "--target=YYYY-MM-DD HH:MM:SS"`).
  3. Export PgBackRest backup lag metrics to Prometheus (`PostgresWALArchiveFailing` alert rule in `k8s/monitoring/prometheus.yaml`).
* **Verification Gate:**
  * [offline] `scripts/verify-ci-contracts.sh` validates the `PostgresWALArchiveFailing` alert rule and PgBackRest configuration.
  * [cluster] Executing a PITR drill on a test table in `oci-dev` restores the database state to a target timestamp prior to a synthetic `DROP TABLE`.

#### 7.45 In-Cluster SMTP / Email Sandbox (`Mailpit`) & Simulator Email Receipt Verification
* **Depends On:** None.
* **Current Codebase State:** Onboarding sagas, trade executions, ZITADEL registrations, Forgejo PR notifications, and Nextcloud shares lack an in-cluster SMTP server to deliver and verify transactional emails.
* **PR Scope:**
  1. Deploy **Mailpit** (`k8s/oci-internal/mailpit.yaml`) exposing SMTP on `mailpit-service.oci-internal.svc.cluster.local:1025` and its REST inspection API/UI on `:8025` (`mail.oci.local`).
  2. Wire `SMTP_HOST=mailpit-service.oci-internal.svc.cluster.local:1025` into `oci-worker` (sending trade confirmation, ACH settlement, and onboarding welcome emails), Forgejo, Nextcloud, and ZITADEL.
  3. Extend the OCI Platform Simulator (`simulation/actions.go`) to query Mailpit's native REST API (`GET /api/v1/messages?query=to:rick@oci.local` and `to:chloe@example.com`) after `OnboardingSaga` and trade execution to assert that expected transactional emails were delivered.
* **Verification Gate:**
  * [offline] Unit tests verify worker SMTP notification formatting and CWE-532 redaction in logs.
  * [cluster] Simulator verifies live email delivery in Mailpit for `Rick Newhire` onboarding and `Chloe Spark` order execution.

#### 7.46 Automated Database Migration Runner (`goose`) & `PgBouncer` Transaction Pooling
* **Depends On:** None.
* **Current Codebase State:** Per `scripts/migrations/README.md`, numbered migrations in `scripts/migrations/NNN-*.sql` must be applied manually with `psql -f` on existing volumes, and 10-worker concurrent simulation runs open direct PostgreSQL connections from every microservice and COTS pod.
* **PR Scope:**
  1. Build a lightweight migration runner (`cmd/db-migrate` using `pressly/goose/v3` or a Kubernetes Job `k8s/oci-core/db-migrate-job.yaml` and `k8s/oci-dev/db-migrate-job.yaml`) that automatically applies `scripts/migrations/*.sql` in sequence and tracks versions in `public.goose_db_version` during `deploy-oci-dev` and `promote-oci-core` in `.forgejo/workflows/ci.yaml`.
  2. Add a CI contract check in `scripts/verify-ci-contracts.sh` verifying that every table/column in `scripts/migrations/*.sql` is also present in `scripts/init.sql`, `k8s/oci-core/postgres-core-configmap.yaml`, and `k8s/oci-dev/postgres-dev-configmap.yaml`.
  3. Deploy **PgBouncer** (`pool_mode = transaction`, `max_client_conn = 500`, `default_pool_size = 25`) in front of `postgres-core`, `postgres-dev`, and `postgres-internal`, routing all Domain APIs and COTS apps through the pooled service port.
* **Verification Gate:**
  * [offline] Unit test applies `scripts/init.sql` followed by all `scripts/migrations/*.sql` idempotently; `./scripts/verify-ci-contracts.sh` enforces 3-copy SQL sync.
  * [cluster] 10-worker concurrent load test (`scripts/run-load-test.sh`) runs through PgBouncer with active PostgreSQL backend connections (`pg_stat_activity`) staying below 35.

#### 7.47 Automated Wire & REST Contract Testing (`buf breaking` & `oasdiff`)
* **Depends On:** None.
* **Current Codebase State:** `AGENTS.md` §4 validates `api/openapi.yaml` syntax via `scripts/validate-openapi.py`, and `.proto` files are compiled manually without automated breaking-change detection against `main`.
* **PR Scope:**
  1. Add `buf.yaml` and `buf.gen.yaml` at the repository root configuring `buf lint` (`STANDARD` category) and `buf breaking` (`FILE` / `WIRE_JSON` categories) across `proto/oci/**`.
  2. Add `buf lint`, `buf breaking --against '.git#branch=main'`, and `oasdiff breaking main:api/openapi.yaml api/openapi.yaml` to `scripts/verify-ci-contracts.sh` and `.forgejo/workflows/ci.yaml` so any wire-breaking Protobuf change (between BLLs and Domain APIs) or breaking OpenAPI 3.1 REST change (between BFFs and Vue/Flutter/Pebble clients) fails CI automatically.
* **Verification Gate:**
  * [offline] `./scripts/verify-ci-contracts.sh` executes `buf lint`, `buf breaking`, and `oasdiff` cleanly, and negative test cases confirm renaming a proto field number or removing a required OpenAPI response field triggers a non-zero exit code.
  * [cluster] Forgejo Actions `validate-and-test` runs contract diffing on every PR.
