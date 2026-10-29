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
       │ Unified API Server      │ http://localhost:8080             │
       │ Employee BFF            │ http://localhost:8081             │
       │ Pebble Gateway Proxy    │ http://localhost:8082             │
       │ Customer BFF            │ http://localhost:8083             │
       │ Background Worker       │ (Event subscriber daemon)         │
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

#### Step 4: Launch Backend Services

You can run each service in separate terminal tabs, in `tmux`/`screen`, or as `systemd` user services.

**1. Unified API Server (`:8080`)**
Hosts all 4 domains (`core-invest`, `facilities`, `ops`, `workforce`):
```bash
export DATABASE_URL="postgres://oci:catnip@localhost:5432/oci?sslmode=disable"
export PORT="8080"
go run ./cmd/workforce-api
```

**2. Background Event Worker (Watermill & CA Rotation)**
Handles cat observation triggers, trading execution events, firmware OTA pipelines, and intermediate CA renewal:
```bash
go run ./cmd/worker
```

**3. Employee Portal Backend-for-Frontend (`:8081`)**
Provides session management, CSRF validation, and employee management endpoints:
```bash
export DATABASE_URL="postgres://oci:catnip@localhost:5432/oci?sslmode=disable"
export PORT="8081"
go run ./cmd/employee-bff
```

**4. Pebble Companion Watch Gateway Proxy (`:8082`)**
Translates IT tickets into smartwatch AppMessage alert packets and processes physical ACK button presses:
```bash
export DATABASE_URL="postgres://oci:catnip@localhost:5432/oci?sslmode=disable"
export PORT="8082"
go run ./cmd/pebble-proxy
```

**5. Customer Portal Backend-for-Frontend (`:8083`)**
Provides investor portfolio analytics, trading terminal feeds, recurring deposits, and ESG transparency endpoints:
```bash
export DATABASE_URL="postgres://oci:catnip@localhost:5432/oci?sslmode=disable"
export PORT="8083"
go run ./cmd/customer-bff
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
go run ./simulation/cmd/simulator -mode live -server-url http://localhost:8080 -bff-url http://localhost:8081 -cust-bff-url http://localhost:8083

# 3. Execute concurrent load testing (e.g. 10 parallel simulation runs):
go build -o /tmp/simulator ./simulation/cmd/simulator
for i in {1..10}; do
  /tmp/simulator -mode live -server-url http://localhost:8080 -bff-url http://localhost:8081 -cust-bff-url http://localhost:8083 -delay 5 &
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
go run ./simulation/cmd/simulator -url https://dev.api.oci.local

# 2. Run against local test services:
go run ./simulation/cmd/simulator \
  -server-url http://localhost:8080 \
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
curl -i http://localhost:8080/healthz
# Response: HTTP/200 {"status":"ok"}
```

**2. List Hardware Assets (Keyset Pagination):**
```bash
curl -s http://localhost:8080/facilities/assets | jq .
```

**3. Submit Offline Maintenance Logs (Field Sync):**
```bash
curl -s -X POST http://localhost:8080/facilities/sync/maintenance-logs \
  -H "Content-Type: application/json" \
  -d '[{
    "asset_id": "018e0000-0000-7000-8000-000000000001",
    "qr_code_scanned": "QR-CAM-ORANGE-01",
    "action_taken": "Cleaned optical lens and recalibrated zoom sensor"
  }]' | jq .
```

**4. Trigger Edge Device Firmware OTA Update:**
```bash
curl -s -X POST http://localhost:8080/facilities/firmware/ota-jobs \
  -H "Content-Type: application/json" \
  -d '{
    "asset_ids": ["018e0000-0000-7000-8000-000000000001"],
    "release_id": "018e0000-0000-7000-8000-000000000002"
  }' | jq .
```

**5. Toggle Feline Emergency Medical Hold:**
```bash
curl -s -X POST http://localhost:8080/api/v1/workforce/care-schedules/emp-feline-garfield/medical-hold \
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

The following roadmap outlines planned technical debt remediation, architectural decomposition, observability expansion, and security hardening to evolve Orange Cat Investments into an authentic, production-grade self-hosted small-to-medium business (SMB) platform.

The remaining items are prioritized in the execution order that maximizes engineering velocity and minimizes architectural risk:
1. **Phase 1 — End-to-End Verification, Codebase Modularity & Self-Hosted CI/CD (`7.1`–`7.3`):** Establish stateful authenticated persona simulation (`7.1`), decompose monolithic `main.go` entrypoints into modular handler packages while protected by that test suite (`7.2`), and wire automated in-cluster Forgejo Actions builds and live simulation promotion gates (`7.3`).
2. **Phase 2 — Core Domain Completion (`7.4`, `7.10`):** Implement dynamic habitat camera provisioning and computer vision stream lifecycle management (`7.4`) and dynamic historical portfolio valuation with daily NAV snapshots (`7.10`) cleanly atop the modular BFF/BLL architecture.
3. **Phase 3 — Full-Stack Infrastructure, Database & Application Observability (`7.5`–`7.7`):** Instrument self-hosted Kubernetes blind spots (`7.5`), stateful data stores (`7.6`), and OpenTelemetry distributed tracing/metrics (`7.7`) across all completed domain services.
4. **Phase 4 — Chaos Engineering & Zero-Trust Hardening (`7.8`–`7.9`):** Execute fault injection and chaos resilience verification against the fully instrumented alerting stack (`7.8`), followed by dynamic secret management and workload identity mesh hardening (`7.9`).

---

### 7.1 Stateful Persona Session Management & Cookie/CSRF Persistence in Platform Simulation

> [!NOTE]
> **Status: Completed**
> The OCI Platform Simulator (`simulation/client.go`, `simulation/driver.go`, `simulation/context.go`, `simulation/session_test.go`) manages authentic stateful sessions with per-persona `http.CookieJar` instances, automated pre-action authentication handshakes, transparent double-submit CSRF header injection, and negative security testing (session purge on offboarding & GDPR erasure).

Enhance the OCI Platform Simulator (`simulation/cmd/simulator`, `simulation/client.go`, `simulation/driver.go`) to manage authentic stateful sessions, persisting session cookies and double-submit CSRF tokens across consecutive actions executed by the same persona:

* **Why First:** The simulation driver is the primary end-to-end verification gate for refactoring (`7.2`) and automated CI/CD cluster promotion (`7.3`). Adding stateful cookie and CSRF persistence first ensures all subsequent architectural changes are validated against authentic browser session flows.
* **The Problem:**
  The simulation HTTP client (`simulation.Client`) previously lacked persistent cookie and CSRF handling, which caused authentic persona workflows calling protected BFF endpoints (`oci-customer-bff` and `oci-employee-bff`) to fail with `401 Unauthorized` or `403 Forbidden` in live environments where Valkey sessions (`__Host-customer-session`, `__Host-employee-session`) and double-submit CSRF verification (`X-CSRF-Token`) are strictly enforced. Furthermore, stateless execution cannot test real-world user lifecycles (e.g. login $\rightarrow$ authenticated operations $\rightarrow$ session revocation / logout), multi-step trading sessions, or session expiration.
* **Architecture & Objectives:**
  * **Per-Persona Cookie Jar & Session Scope:** Equip the simulation client with per-persona `http.CookieJar` instances (`net/http/cookiejar.New(nil)`). When executing a persona's action list during a simulation iteration, all HTTP requests dispatched by that persona reuse the persona's cookie jar, automatically preserving session cookies (`__Host-customer-session`, `__Host-employee-session`, `customer_session`, `employee_session`) and CSRF cookies across calls.
  * **Automated Persona Authentication Handshake:** Introduce an automated pre-action authentication hook in `simulation.Driver`:
    * *Customer Personas (Arthur Pendelton, Chloe Spark):* Prior to running daily customer actions, perform an authentication handshake against `POST /api/v1/customer/auth/login` (or ZITADEL OIDC flow in live mode), populating the cookie jar and extracting the active CSRF token.
    * *Employee & Staff Personas (Bob Builder, Alice Vance, Dr. Elena Rostova, Frank Operations, etc.):* Authenticate via `POST /api/v1/employee/auth/login` (in development mode) or Forgejo OAuth2 handshake (`/api/v1/employee/auth/forgejo/login`), establishing an authentic session in Valkey.
  * **Transparent Double-Submit CSRF Header Injection:** Automatically extract the active CSRF token from the persona's cookie jar and populate the `X-CSRF-Token` header on state-changing requests (`POST`, `PUT`, `PATCH`, `DELETE`) to BFF endpoints, eliminating boilerplate in individual action definitions.
  * **Stateful Lifecycle & Security Testing:**
    * Enable personas to execute realistic multi-step journeys: login $\rightarrow$ browse directory / portfolio $\rightarrow$ submit timesheet / order $\rightarrow$ view paystub / statement $\rightarrow$ explicit logout.
    * Enable negative security testing: verify that session invalidation during offboarding (e.g. Alice triggering employee offboarding) immediately invalidates the persona's session in Valkey, causing subsequent calls with the cached cookie to correctly fail with `401 Unauthorized`.
  * **Live Verification Paradigm:** The simulation engine exercises authentic live HTTP endpoints, ensuring all authentication handshakes, cookies, CSRF tokens, and middle-tier layers are verified against real network boundaries.

### 7.2 Modular BFF & BLL Application Architecture (Decomposition of Monolithic main.go Files)

> [!NOTE]
> **Status: Completed**
> The monolithic application entrypoints (`cmd/customer-bff`, `cmd/employee-bff`, `cmd/customer-bll`, `cmd/internal-bll`, `cmd/worker`, `cmd/pebble-proxy`) have been decomposed into modular packages under `internal/app/` (< 450 lines per file). Architecture strictly enforces 5-Tier layer isolation:
> 1. **BFF Tier (`customer-bff`, `employee-bff`, `pebble-proxy`):** Chi routers (`chi/v5`), Valkey session/CSRF middleware, Meilisearch client; dials BLL services over gRPC with zero database or Domain API imports. Implements fail-fast startup (exiting with code 1 on dial failure or mTLS initialization error, eliminating plaintext fallbacks). Serves quant routes (`/core-invest/backtesting/runs/{id}`, `/core-invest/streams/{id}`, `/core-invest/brokerage/orders/{id}`) and customer webhook deletion (`DELETE /api/v1/customer/webhooks/{id}`).
> 2. **BLL Tier (`customer-bll`, `internal-bll`):** Pure gRPC microservices implementing business orchestrations, external bank/broker adapters, dynamic active employee resolution for payroll runs, Meilisearch knowledge indexing, and sagas; dials Domain APIs over gRPC mTLS with zero HTTP endpoints (probed via `tcpSocket` / gRPC health without inert HTTP listeners).
> 3. **Domain APIs & Database:** Pure gRPC over PostgreSQL schemas (`workforce`, `facilities`, `core_invest`, `ops`) with zero cross-domain calls.
> 4. **Worker & Batch CronJobs:** Dial customer and internal BLL services over gRPC mTLS with zero direct database or Domain API connections. OTA triggers (`HandleOTATriggered`) and backtest runs (`HandleBacktestRun`) actively persist results via BLL RPCs (`UpdateOTAJobStatus` and `CompleteBacktestRun`). ACH transfer settlements return explicit errors if the BLL client is absent rather than synthesizing fake settlements. Network policies restrict Domain APIs exclusively to `tier: bll`, and dev BLL network policies admit `tier: worker` and `tier: batch`.

Refactor the OCI BFF and BLL service entrypoints (`cmd/customer-bff/main.go`, `cmd/employee-bff/main.go`, `cmd/customer-bll/main.go`, `cmd/internal-bll/main.go`, `cmd/worker/main.go`) from monolithic single-file implementations into a structured, modular internal package architecture:

* **Why Second:** Decomposing the monolithic `main.go` files before adding new stream management routes (`7.4`), OpenTelemetry instrumentation (`7.7`), or fault-tolerance middleware (`7.8`) ensures new features land cleanly in dedicated controller packages rather than compounding 1,000–2,200+ line files.
* **The Problem:**
  Several critical application entrypoints currently span over 1,000–2,200+ lines within a single `main.go` file (e.g. `cmd/customer-bff/main.go` at 2,200+ lines, `cmd/worker/main.go` at 1,300+ lines, `cmd/customer-bll/main.go` at 1,100+ lines). These files combine CLI configuration parsing, server bootstrapping, middleware chains, route multiplexing, and dozens of discrete domain endpoint handlers in one place. This monolithic structure hinders readability, complicates code reviews, increases git merge conflict probability, and makes targeted unit testing more cumbersome.
* **Architecture & Objectives:**
  * **Formalized Layered Application Layout:**
    * `cmd/<service>/main.go`: Retain as an ultra-lean bootstrap entrypoint responsible only for environment flag parsing, logging setup, dependency injection wiring, OS signal handling, and server lifecycle (`ListenAndServe` / graceful `Shutdown`).
    * `internal/app/<service>/`: House the server application struct, configuration loader, and dependency container.
    * `internal/app/<service>/router.go`: Define HTTP mux routing tables and middleware chains (session authentication, CSRF validation, rate limiting, request tracing, versioning).
    * `internal/app/<service>/handlers/`: Group related HTTP handlers into focused controller modules:
      * For `customer-bff`: `auth_handlers.go`, `portfolio_handlers.go`, `trading_handlers.go`, `stream_handlers.go`, `ticket_handlers.go`, `audit_handlers.go`.
      * For `employee-bff`: `auth_handlers.go`, `workforce_handlers.go`, `facilities_handlers.go`, `review_handlers.go`, `incident_handlers.go`, `cots_handlers.go`.
      * For `customer-bll`: `orders.go`, `banking.go`, `brokerage.go`, `streams.go`.
      * For `internal-bll`: `incidents.go`, `firmware.go`, `reports.go`, `veterinary.go`, `sagas.go`.
  * **Controller & Service Interface Segregation:**
    * Decouple HTTP serialization/deserialization logic from domain business rules by injecting domain service interfaces rather than concrete clients.
    * Standardize API error responses via structured error helpers (`apierror.Render(w, err)`).
  * **Isolated Sub-Module Testing:**
    * Enable unit testing of individual handler packages (`handlers/*_test.go`) without spinning up the entire server multiplexer or instantiating all domain clients.

### 7.3 Automated In-Cluster Forgejo Runner CI/CD Pipeline & Test Cluster Promotion

> [!NOTE]
> **Status: Completed**
> The self-hosted in-cluster Forgejo runner CI/CD supply chain and promotion pipeline are established and verified:
> 1. **In-Cluster Source Synchronization (`scripts/sync-repo-to-forgejo.sh`):** Automates organization provisioning (`oci`), repository synchronization (`monorepo`, `orange-cat-investments`), Actions enablement, and backlog issue seeding (#42) during platform deployments (`scripts/deploy-k8s.sh`, Ansible).
> 2. **In-Cluster Runner Daemon & RBAC (`k8s/oci-internal/forgejo-runner.yaml`):** Deploys `gitea/act_runner` paired with a privileged Docker-in-Docker sidecar (`docker:26-dind`), cluster deployer RBAC (`forgejo-runner-sa`), host networking, automatic in-cluster `/data/kubeconfig` generation, and insecure registry access for `localhost:32000`.
> 3. **Self-Hosted Container Builds & Local Registry Push:** Builds all multi-stage microservices and web portals inside the runner and pushes directly to `localhost:32000` without requiring external developer workstation compilation.
> 4. **Gated Promotion Pipeline (`.forgejo/workflows/ci.yaml`):** Implements automated multi-stage gates: lint/unit tests $\rightarrow$ staging deployment (`oci-dev`) $\rightarrow$ live simulation validation gate $\rightarrow$ production promotion rolling updates (`oci-core`).
> 5. **End-to-End Simulation Verification:** Verified via the 13-persona simulation suite (73 persona action workflows targeting live service endpoints).

Refactor the test cluster verification and deployment workflow so that test clusters (e.g. `thinker`) are tested as authentic, self-contained deployment targets rather than relying on manual image builds pushed from external developer workstations:

* **Why Third:** With stateful live simulation (`7.1`) and modularized service packages (`7.2`) in place, automating the in-cluster Forgejo CI/CD pipeline and live simulation promotion gate allows all subsequent feature and infrastructure changes (`7.4`–`7.9`) to be built, tested, and promoted through the cluster's native GitOps supply chain.
* **The Problem:**
  Building OCI microservices and frontend SPAs locally on a developer or agent workstation and pushing them to the test cluster's registry (`localhost:32000` / `thinker:32000`) bypasses the cluster's internal deployment machinery. It exercises the workstation's local environment, Go compiler, and network rather than validating the deployed cluster's own self-hosted software supply chain.
* **Architecture & Objectives:**
  * **In-Cluster Source Synchronization:** Automatically sync or bootstrap the `orange-cat-investments` repository into the in-cluster **Forgejo** instance (`http://forgejo-service.oci-internal.svc:3000`) during platform deployment.
  * **In-Cluster Container Builds via Forgejo Actions:** Utilize the deployed in-cluster runner ([`k8s/oci-internal/forgejo-runner.yaml`](k8s/oci-internal/forgejo-runner.yaml)) running `gitea/act_runner` with Docker-in-Docker (DinD). When code changes land on Forgejo, [`.forgejo/workflows/ci.yaml`](.forgejo/workflows/ci.yaml) executes inside the cluster node.
  * **Direct Push to In-Cluster Registry (`localhost:32000`):** The runner builds the multi-stage container images (Domain APIs, BLLs, BFFs, `oci-worker`, `oci-pebble-proxy`, `oci-marketing`, portals) and pushes directly to `localhost:32000` over the in-cluster network.
  * **Gated Promotion to Staging (`oci-dev`) & Production (`oci-core`):** Once unit tests and builds succeed inside the cluster runner, the workflow triggers Kubernetes rolling updates in `oci-dev` and subsequent promotion to `oci-core`.
  * **Automated End-to-End Simulation Verification:** The runner triggers an automated post-deployment simulation run (`simulation/cmd/simulator -mode live`) to verify that all personas function correctly on the freshly deployed cluster without requiring external CLI intervention.

### 7.4 Mock Hardware Asset Data Generation for Platform Workflows
Design and implement mock hardware assets that generate data, such as interpreted video feed events, to trigger downstream OCI applications and simulate realistic physical environments.

* **Why Fourth:** Completes the remaining functional domain surface across `facilities-api`, `core-invest-api`, `internal-bll`, `customer-bll`, and `customer-bff` before layering full-stack observability (`7.5`–`7.7`) and chaos testing (`7.8`).
* **The Problem:**
  Habitat camera streams and physical assets are currently stubbed out via static seed scripts (`scripts/seed-dev.sql`, `scripts/init.sql`). Since OCI applications require active edge data (e.g., habitat motion events, temperature readings) to exercise platform logic (like trading decisions based on feline zoomies), we need a dynamic mechanism to mock this physical edge. Without a real video ingest layer, we must simulate the output of computer vision models directly to trigger business logic.
* **Architecture & Objectives:**
  * **Mock Event Generators:**
    * Create dedicated Go binaries in `cmd/mock-hardware` to simulate various edge devices (e.g., smart cameras, IoT climate sensors).
    * Configure these mock devices to emit high-level interpreted events rather than raw data (e.g., publishing `events.investment.observation_ingested.v1` for a detected "zoomie" instead of streaming RTSP video).
  * **Dynamic Hardware Discovery & Catalog API:**
    * Implement gRPC RPCs in `CoreInvestService` and `FacilitiesService` to track and discover these virtual hardware assets.
    * Allow the platform to query the status of mock devices and subscribe to their event streams via RabbitMQ.
  * **Simulated Environment Responses:**
    * Wire the mock event streams into `oci-worker` and internal services to trigger system actions (e.g., altering asset allocation when an alpha perch posture is detected).
    * Introduce realistic variance and probabilistic behavior into the mock generators to ensure unpredictable, lifelike event distributions for the simulation engine.

### 7.5 Common Self-Hosted K8s Monitoring Blind Spots (Infrastructure Catch-All)
Address critical Kubernetes infrastructure failure modes commonly overlooked when transitioning from managed cloud platforms (EKS, GKE, Cloud SQL) to bare-metal / self-hosted environments. While application telemetry surfaces service-level issues, self-hosted clusters require explicit instrumentation for the silent infrastructure failures that managed cloud providers automatically abstract:

* **Why Fifth:** Protects the underlying bare-metal Kubernetes host and storage volumes first—serving as the foundation of the observability phase before database exporters (`7.6`), application tracing (`7.7`), and chaos engineering (`7.8`).
* **Persistent Volume (PVC) Disk Saturation (The #1 Silent Killer):**
  * *The Vulnerability:* In managed cloud environments, managed database services and CSI drivers can automatically expand underlying block storage. In self-hosted Kubernetes, if a PersistentVolumeClaim backing PostgreSQL (`postgres-data-pvc`, `postgres-internal-data-pvc`), Nextcloud (`nextcloud-data-pvc`), RabbitMQ, Valkey, or Prometheus itself hits 100% capacity, write-ahead logs fail, databases crash, tables corrupt, and pods enter unrecoverable CrashLoopBackOff loops.
  * *Implementation Plan:* Ensure Kubelet volume metrics (`kubelet_volume_stats_available_bytes` and `kubelet_volume_stats_capacity_bytes`) are scraped by Prometheus.
  * *Alert Rules:* Configure alerts to trigger whenever any PVC drops below 15% free capacity, or if linear disk consumption predicts volume exhaustion within 24 hours (`predict_linear(kubelet_volume_stats_available_bytes[4h], 86400) < 0`).
* **Kubernetes Object State & OOMKills (`kube-state-metrics`):**
  * *The Vulnerability:* Standard CPU and memory graphs return to healthy, baseline levels immediately after a crashed pod restarts, masking memory leaks, thrashing restart loops, or image pull regressions.
  * *Implementation Plan:* Deploy `kube-state-metrics` (standard within `kube-prometheus-stack`) to track and alert on Kubernetes control plane object state anomalies.
  * *Key Alerts:*
    * Pods stuck in `CrashLoopBackOff` or `ImagePullBackOff` for more than 5 minutes.
    * Pods stuck in `Pending` (unschedulable due to resource requests, node affinity, or unbound PVCs).
    * Container terminations caused by Out-Of-Memory evictions (`kube_pod_container_status_last_terminated_reason{reason="OOMKilled"}`).
* **Kubernetes API Event Shipping to Long-Term Logs:**
  * *The Vulnerability:* By default, the Kubernetes API server purges cluster events (`kubectl get events`) after 60 minutes. If a pod fails to mount a volume, fails liveness probes, or is evicted by the node at 2:00 AM, all forensic evidence is deleted by morning.
  * *Implementation Plan:* Configure Grafana Alloy (or Promtail) with the `loki.source.kubernetes_events` component to tail Kubernetes API events continuously and ship them to Grafana Loki alongside container stdout/stderr log streams, ensuring full historical auditability.
* **Automated Backup Verification Metrics:**
  * *The Vulnerability:* Running stateful apps (Postgres, Nextcloud, Forgejo) requires automated daily backups (via CloudNativePG Barman, Velero, or custom K8s backup CronJobs). Backups that silently fail due to storage exhaustion, network timeouts, or permission errors leave the cluster vulnerable to total data loss.
  * *Implementation Plan:* Instrument and scrape backup controller metrics (e.g. backup completion timestamps, backup size, and exit status).
  * *Alert Rule:* Alert immediately if the timestamp of the last successful backup is older than 26 hours (`time() - backup_last_successful_timestamp_seconds > 26 * 3600`), guaranteeing daily backup verification.
* **"Watching the Watchers" (Dead Man's Switch / External Heartbeat Instructions):**
  * *Operational Scope:* Instructions only (no implementation in repo).
  * *The Problem:* If your self-hosted Kubernetes node loses electrical power, experiences hardware motherboard failure, or loses ISP internet connectivity, in-cluster Prometheus and Alertmanager instances are completely offline and physically incapable of firing webhook alerts or sending push notifications.
  * *Setup Instructions:*
    1. Utilize Prometheus Alertmanager's built-in `Watchdog` alert (a synthetic alert configured to always fire continuously with severity `none`).
    2. In Alertmanager configuration, configure a dedicated route forwarding the `Watchdog` notification via an HTTP webhook every 5–10 minutes to a free external dead man's switch service (e.g. [Healthchecks.io](https://healthchecks.io), Better Stack, or Cronitor).
    3. If the external heartbeat monitor stops receiving the expected ping within the configured grace period (e.g. 15 minutes), the external service immediately pages the on-call engineer via SMS, email, or mobile push notification, providing external out-of-band alerting for complete site, power, or internet failures.

### 7.6 Infrastructure Application Monitoring (Valkey, PostgreSQL, & Meilisearch)
Implement comprehensive Prometheus telemetry, health probing, and alerting strategies for the remaining stateful data stores backing Orange Cat Investments (`valkey`, `meilisearch`, and dual `postgres-core`/`postgres-internal` databases), complementing the already-implemented RabbitMQ and COTS telemetry. Because databases and persistence stores generally do not expose Prometheus metrics natively, Kubernetes architectures must utilize the **Exporter Sidecar Pattern**, dedicated operators, or native internal plugin modules:

* **Why Sixth:** Extends cluster observability (`7.5`) into the persistence and caching tiers before instrumenting application-level distributed traces (`7.7`) and running database/cache chaos failure tests (`7.8`).
* **Valkey (In-Memory Cache & Session Store):**
  * **Redis Wire Compatibility:** Because Valkey maintains wire and command compatibility with Redis, the standard `oliver006/redis_exporter` works seamlessly without custom forks.
  * **Setup:**
    * In native manifests (`k8s/oci-core/valkey.yaml` and `k8s/oci-dev/valkey.yaml`), deploy `oliver006/redis_exporter:latest` as a sidecar container pointing to `redis://localhost:6379`.
  * **Key Metrics & Alerts:**
    * *Memory Saturation:* Compare `redis_memory_used_bytes` against `redis_memory_max_bytes` (alert above 85% to prevent sudden Linux OOMKilled pod evictions).
    * *Key Evictions:* `increase(redis_evicted_keys_total[5m]) > 0` indicates Valkey is dropping keys under memory pressure, directly impacting active BFF sessions (`pkg/session/valkey`) and cached API query results (`pkg/cache/valkey`).
    * *Cache Efficiency:* Calculate hit rate: `redis_keyspace_hits_total / (redis_keyspace_hits_total + redis_keyspace_misses_total)` (alert when hit rate drops below baseline thresholds).
    * *Connected Clients & Command Latency:* Monitor `redis_connected_clients` against connection caps and track slow operations via `redis_slowlog_length`.
* **PostgreSQL (Dual-Instance RDBMS - `postgres-core` & `postgres-internal`):**
  * **Setup:**
    * *Exporter Sidecar:* Run `prometheus-community/postgres_exporter` as a sidecar in `k8s/oci-core/postgres-core.yaml`, `k8s/oci-dev/postgres-dev.yaml`, and `k8s/oci-internal/postgres-internal.yaml` authenticated via a read-only monitoring database role.
    * *Query Profiling Extension:* Ensure the `pg_stat_statements` extension is enabled in `postgresql.conf` and database bootstrap scripts to monitor slow queries, execution counts, and buffer hit cache ratios.
  * **Key Metrics & Alerts:**
    * *Connection Saturation:* Active connections (`pg_stat_activity_count`) approaching `pg_settings_max_connections` (alert at 80% to prevent rejecting connection requests from application pools in `pgxpool`).
    * *Dead Tuples & Autovacuum Lag:* Monitor `pg_stat_user_tables_n_dead_tup` to detect autovacuum starvation, which leads to severe table bloat and degraded query performance over time.
    * *Transaction ID Wraparound & Replication Lag:* Monitor `pg_database_age` against emergency vacuum limits, and track WAL replication lag (`pg_stat_replication_bytes_lag`) if read replicas are deployed.
    * *Buffer Cache Hit Ratio:* Alert if `sum(blks_hit) / (sum(blks_hit) + sum(blks_read))` drops below 99%.
* **Meilisearch (Full-Text Search Engine):**
  * **Setup & Exposition:**
    * Enable experimental native Prometheus telemetry by adding the `MEILI_EXPERIMENTAL_PROMETHEUS_METRICS=true` environment variable to `k8s/oci-core/meilisearch.yaml` and `k8s/oci-dev/meilisearch.yaml` on port `7700`.
  * **Key Metrics & Alerts:**
    * *Task & Indexing Queue Backlog:* Monitor pending task count (`meilisearch_task_queue_depth` or pending tasks in `/tasks?statuses=enqueued,processing`) to detect asynchronous indexing lag when updating knowledge base documents, SOPs, or VHR health records.
    * *LMDB Disk & Memory Saturation:* Monitor `meilisearch_database_size_bytes` against the persistent volume capacity (`meilisearch-data-pvc`), and track memory virtual mapping (LMDB memory map size) to prevent out-of-disk crashes during large document re-indexing jobs.
    * *Search Latency & QPS:* Track P95 and P99 search request latencies and queries per second (QPS) across the Employee Portal Command+K search palette.
    * *Index Document Counts & Updates:* Track document counts per index (`helpdesk`, `sop_docs`, `vhr_records`) to verify document synchronization consistency.

### 7.7 Implement Application Metrics & Tracing (OpenTelemetry) for OCI Applications
Implement the integration of end-to-end distributed tracing, Application Performance Monitoring (APM), and business metrics using OpenTelemetry (OTel) across OCI custom services (`customer-bff`, `employee-bff`, `customer-bll`, `internal-bll`, Domain APIs, `worker`, `pebble-proxy`) to establish complete request-path visibility and performance telemetry:

* **Why Seventh:** Instrumenting distributed traces and custom business metrics after service decomposition (`7.2`), domain completion (`7.4`), and infrastructure/database monitoring (`7.5`–`7.6`) provides full-stack correlation across HTTP/gRPC spans, SQL queries, and logs prior to chaos testing (`7.8`).
* **Zero-Code Auto-Instrumentation:**
  * **Kernel-Level eBPF Telemetry (Grafana Beyla):** Deploy Grafana Beyla as a zero-overhead alternative for Go binaries, automatically capturing inbound/outbound HTTP/gRPC route latencies, HTTP status codes, error rates (RED metrics), and outbound SQL (`pgxpool`) / REST calls without modifying application code or re-architecting binaries.
* **SDK-Based Instrumentation for Custom Business Telemetry:**
  * **Direct OpenTelemetry Client SDK Integration:** Integrate the OpenTelemetry Go SDK (`go.opentelemetry.io/otel`) directly into critical business APIs across BFF, BLL, and Domain services.
  * **Custom Business & Operational Metrics:** Instrument fine-grained counters, gauges, and histograms for domain-specific events:
    * Retail investor user onboarding, KYC status transitions, and Plaid bank account linking success/failure rates.
    * Trading checkout events, algorithmic order execution latency, and Smart Order Router bracket order fills.
    * Feline behavioral event throughput (zoomies, perch occupancy, napping intervals) and computer vision frame classification inference confidence distributions.
    * Feline Care Schedule updates and Emergency Medical Trading Hold toggle frequency.
    * Wearable on-call alert dispatch-to-acknowledgment (MTTA) duration histograms from `cmd/pebble-proxy`.
* **OTLP Pipeline, Collectors & Telemetry Storage Backend:**
  * **Standardized OTLP Export:** Push traces, spans, and metrics over gRPC (port `4317`) and HTTP (port `4318`) using the OpenTelemetry Protocol (OTLP).
  * **In-Cluster Telemetry Agent (Grafana Alloy):** Deploy an in-cluster Grafana Alloy agent in namespace `monitoring` to receive, batch, sample, and redact incoming spans and metrics.
  * **Storage & Correlation in Prometheus & Grafana Tempo:** Configure the collector to forward metrics to the in-cluster Prometheus instance (`monitoring/prometheus`) and distributed traces to Grafana Tempo, enabling seamless trace-to-metrics and trace-to-logs (Loki) correlation directly within Grafana dashboards.

### 7.8 Fault Tolerance, Chaos Engineering & Negative API Testing
Ensure that negative scenarios with external financial APIs and unexpected internal service degradation are gracefully handled with functioning alerting:

* **Why Eighth:** Chaos engineering and negative fault injection require a complete functional platform (`7.1`–`7.4`) and active infrastructure, database, and application observability/alerting (`7.5`–`7.7`) so every induced failure mode can be verified both for graceful service recovery and accurate alert firing.
* **External Error Injection:**
  * **Mock Bank API:** Add configurable error responses and latency injection toggled via management endpoints outside regular banking routes (`/sim/faults/bank`).
  * **Mock Broker API:** Add configurable rejected orders, order cancellations, and simulated execution halts toggled via management endpoints (`/sim/faults/broker`).
* **Chaos Engineering & Resilience Verification:**
  * Create automated test scripts to intentionally induce failure modes and verify cluster recovery and alerting:
    * Stop PostgreSQL for `oci-internal` (verify COTS graceful failure and alerting).
    * Stop PostgreSQL for `oci-core` (verify core app resilience and reconnect backoff).
    * Break network policies between BLL and Domain APIs (verify circuit breaking).
    * Break network policies between BFFs and BLLs (verify fallback responses).
    * Terminate individual Domain APIs (`workforce`, `facilities`, `core_invest`, `ops`) and verify pod self-healing.
    * Terminate BLL APIs (`customer-bll`, `internal-bll`) and verify BFF degradation.
    * Terminate BFF APIs (`customer-bff`, `employee-bff`).
    * Simulate external Bank and Broker API outages.
    * Inject deadletter messages into RabbitMQ queues and verify DLQ routing.
    * Inject malformed and schema-invalid messages into RabbitMQ queues to confirm safe parsing and poison-pill rejection.

### 7.9 Zero Trust Workload Identity & Dynamic Secret Management
Complete the zero-trust security architecture to eliminate trust assumptions between internal workloads, enforce network isolation, and eliminate long-lived static credentials:

* **Why Ninth:** With dynamic `cert-manager` mTLS (`pkg/mtls`) and Kubernetes NetworkPolicies already protecting the cluster, migrating from static Kubernetes Secrets to dynamic CSI-mounted Vault credentials and SPIFFE/SPIRE or OpenZiti workload identity meshes represents the final security hardening phase atop a stable, observable, and chaos-verified platform.
* **Dynamic Secret Management & CSI Secrets Driver:**
  * Migrate from static base64-encoded Kubernetes Secret manifests to HashiCorp Vault or cert-manager CSI secret drivers, providing short-lived, dynamically generated database credentials and API tokens.
* **Workload Identity Service Mesh:**
  * Explore SPIRE (SPIFFE Runtime Environment) or OpenZiti (https://github.com/openziti/ziti) to establish cryptographically attested workload identities for zero-trust service mesh connectivity without depending exclusively on network perimeter security.

### 7.10 Dynamic Historical Portfolio Valuation & Daily Net Asset Value (NAV) Snapshot Engine
Eliminate static and hardcoded historical performance points in `customer-bff` and `customer-bll` by introducing an automated daily Net Asset Value (NAV) snapshot engine and time-series portfolio valuation service:

* **Why Tenth (Phase 2 Domain Completion):** Resolves open technical debt identified during PR review on PR #137 where historical performance points (`Date`, `Valuation`) in `internal/app/customerbff/portfolio_handlers.go` and `internal/app/customerbll/brokerage.go` currently use hardcoded percentage multipliers (`0.78`, `0.85`, `0.92`) against the current total balance due to the absence of a historical time-series ledger.
* **The Problem:**
  Retail investors expect authentic portfolio growth charts over 1M, 3M, 6M, 1Y, and YTD intervals. Currently, because neither the database nor the broker integration persists periodic portfolio valuation snapshots, the BFF and BLL synthesize static valuation curves using fixed historical dates (`2026-01-01` through `2026-04-01`) scaled by arbitrary coefficients. This masks real investment gains/losses and prevents accurate performance analytics.
* **Architecture & Objectives:**
  * **Daily NAV Snapshot Persistence (`core_invest.portfolio_snapshots`):**
    * Introduce a partitioned time-series table in PostgreSQL (`core_invest.portfolio_snapshots`) with columns `snapshot_id (UUID)`, `portfolio_id (UUID)`, `snapshot_date (DATE)`, `nav_usd (DECIMAL)`, `cash_usd (DECIMAL)`, `equity_usd (DECIMAL)`, `daily_return_pct (DECIMAL)`, and `holdings_breakdown (JSONB)`.
    * Enforce a unique constraint on `(portfolio_id, snapshot_date)` to guarantee idempotency.
  * **Automated End-of-Day Valuation Batch Runner:**
    * Implement a scheduled cron worker (`cmd/worker` or batch CronJob) executing at market close (e.g., 21:00 UTC / 17:00 EST) that iterates active customer portfolios, computes current market values via real-time market data quotes, and persists immutable daily valuation records via `core-invest-api`.
  * **Historical Portfolio Valuation RPC in BLL & BFF:**
    * Add `GetPortfolioHistory(portfolio_id, start_date, end_date, interval)` to `core_invest.proto` and `customer_bll.proto`.
    * In `customer-bll`, query `core_invest.portfolio_snapshots` for real historical valuation points, calculating true cumulative returns, drawdowns, and benchmark comparisons against feline executive activity indices.
    * In `customer-bff`, serve dynamic historical valuation series directly to the Vue 3 customer portal charting components, completely removing hardcoded fallback curves.


