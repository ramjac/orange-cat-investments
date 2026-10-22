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

* **Orchestration & Hosting:** Kubernetes (K3s). Adheres strictly to [12-Factor App](https://12factor.net/) principles.
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
* **`oci-core`:** Collocates all custom OCI applications (`oci-server`, `oci-customer-bff`, `oci-employee-bff`, `oci-pebble-proxy`, `oci-worker`, `oci-customer-portal`, `oci-employee-portal`) together with external customer identity (**ZITADEL**), core operational infrastructure (`valkey`, `rabbitmq`, `meilisearch`), and dedicated proprietary database **`postgres-core`** (`workforce`, `facilities`, `core_invest`, `ops`).
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
* **Turnkey Small Business Suite Fast-Path:** Standalone COTS-first SMB stack via `docker-compose.smb.yaml` and `./scripts/deploy-k8s.sh --profile small-business`.
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
  * Uses cert-manager's `trust-manager` addon and [`k8s/ha/lan/tls/oci-ca-bundle.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/lan/tls/oci-ca-bundle.yaml) to continuously sync `secret/oci-lan-root-ca-key-pair` in namespace `cert-manager` to `ConfigMap/oci-ca-bundle` across all application namespaces (`oci-core`, `oci-internal`, `monitoring`).
* **Zero-Downtime Dynamic CA Pool Hot-Swapping (`pkg/mtls`):**
  * [`pkg/mtls.DynamicCAPoolReloader`](file:///home/ramjac/orange-cat-investments/pkg/mtls/mtls.go) watches file modification timestamps on mounted `ca.crt` bundles and hot-swaps active `ClientCAs` and `RootCAs` pools dynamically under RWMutex lock via `tls.Config.GetConfigForClient` and `tls.Config.VerifyConnection`, accepting new CA roots without connection drops or pod restarts.
* **Decoupled mTLS Probing Architecture:**
  * Plaintext HTTP `/healthz` and `/livez` listener on port `:8080` via `mtls.StartHealthListener`, keeping business RPCs on ports `50051`–`50054` strictly enforced with mutual TLS (`ClientAuth: RequireAndVerifyClientCert`), resolving Kubelet probe handshake errors.
* **Active Kubernetes Cert-Manager CA Reconciler in `cmd/worker`:**
  * [`internal/service/pki/ca_rotation.go`](file:///home/ramjac/orange-cat-investments/internal/service/pki/ca_rotation.go) interfaces directly with Kubernetes API using standard library crypto and in-cluster credentials to inspect expiration metrics, trigger automated reissuances 30 days prior to expiry, verify `trust-manager` bundle synchronization, and emit `ops.ca_rotated.v1` events.
* **COTS Persistent Volume Architecture & Idempotency (Nextcloud):**
  * Mandated PVCs (`nextcloud-data-pvc`) mounted directly to `/var/www/html` to persist configuration and data. Enforced unprivileged database users (`nextcloud_user` without `CREATEROLE` or superuser privileges) across COTS manifests to prevent role explosion and table ownership corruption across pod restarts.
* **Identity & Session Store Resiliency (ZITADEL & Valkey):**
  * Dedicated PersistentVolumeClaim on Valkey (`valkey-data-pvc` on `/data`) with `valkey-server --appendonly yes --dir /data` enabled. ZITADEL database connection limits and dedicated `startupProbe`/`livenessProbe` configured for DNS/CNI startup tolerance.
* **Control Plane Certificate Expiration Runbooks & Alerts:**
  * Documented annual control plane cert rotation SOPs in [`docs/runbooks/control-plane-cert-rotation.md`](file:///home/ramjac/orange-cat-investments/docs/runbooks/control-plane-cert-rotation.md) and Prometheus Alertmanager rules in [`k8s/monitoring/prometheus.yaml`](file:///home/ramjac/orange-cat-investments/k8s/monitoring/prometheus.yaml) (`CertManagerCertExpiringSoon` and `BlackboxSSLCertExpiringSoon`).

### 5.9 Distributed Caching, Valkey Session Management & Rate Limiting (BFF Layer)
Orange Cat Investments leverages Valkey 7.2 in namespace `oci-core` (`valkey-service.oci-core.svc.cluster.local:6379`) backed by persistent storage (`valkey-data-pvc`) and AOF persistence:

* **Distributed Valkey Session Store for BFF Multi-Pod Scaling:**
  * Implemented in [`pkg/session/valkey`](file:///home/ramjac/orange-cat-investments/pkg/session/valkey/store.go) supporting both Valkey distributed storage and thread-safe in-memory fallback for local dev and offline tests.
  * Customer sessions (`session:customer:{token}`) and Employee sessions (`session:employee:{token}`) with 24-hour TTL.
  * Multi-device session revocation via Redis Sets (`user:sessions:customer:{id}`, `user:sessions:employee:{id}`) enabling atomic bulk purges on logout, account closure, or GDPR erasure.
* **Cache-Aside & Read-Through Query Caching:**
  * Implemented in [`pkg/cache/valkey`](file:///home/ramjac/orange-cat-investments/pkg/cache/valkey/cache.go) with raw-byte caching (`GetRaw` / `SetRaw`) for zero-allocation conditional HTTP responses.
  * Customer Portal: Portfolio Valuation (`cache:portfolio:{customer_id}`, 30s TTL), Quotes (`cache:quotes:{symbol}`, 5s TTL), Habitat Telemetry (5s TTL), ESG Welfare Metrics (15m TTL).
  * Employee Portal: Feline Care Schedules (`cache:care_schedule:{feline_id}`, 2m TTL), Nextcloud metadata (`cache:cots:nextcloud:folders`, 60s TTL).
* **Event-Driven & Eager Cache Invalidation:**
  * [`InvalidationListener`](file:///home/ramjac/orange-cat-investments/pkg/cache/valkey/invalidation.go) subscribing via Watermill / RabbitMQ to `trade_executed`, `order_placed`, `care_schedule_updated`, and `incident_created` events.
  * Synchronous eager invalidation across mutation paths (orders, deposits, care schedule updates, medical hold toggles, GDPR erasures).
* **HTTP Conditional Validation (`ETag` & `304 Not Modified`):**
  * Payloads are hashed with SHA-256 (`ComputeETag` / `HandleConditionalETag`). Matching `If-None-Match` requests return `304 Not Modified` with zero response body.
* **Distributed Sliding-Window Rate Limiting:**
  * [`RateLimiter`](file:///home/ramjac/orange-cat-investments/pkg/cache/valkey/ratelimit.go) using Valkey sorted sets (`ZREMRANGEBYSCORE`, `ZCARD`, `ZADD`) with in-memory fallback and standard `X-RateLimit-*` and `Retry-After` headers.
  * Applied to login (5 req/15min), orders (10 req/s), deposits (5 req/15min), and Pebble Watch acknowledgments (10 req/min).

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
| **K3s / Ansible** *(Optional)* | v1.28+ | Full Kubernetes cluster deployment |

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
go run ./cmd/server
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

### 6.4 Kubernetes Cluster Deployment (MicroK8s, K3s, Minikube, kind)

If you have an existing Kubernetes cluster or local node (such as MicroK8s, Minikube, kind, or an existing K3s installation) configured in your local `~/.kube/config`:

> [!NOTE]
> **Cluster Prerequisites (Cert-Manager & Intranet PKI):**
> Orange Cat Investments enforces dynamic mutual TLS (mTLS) for inter-service communication via `cert-manager`. Before running `deploy-k8s.sh`, ensure `cert-manager` is active:
> * **MicroK8s:** Run `microk8s enable cert-manager`
> * **K3s / Standard K8s:** Install cert-manager via Helm or the official manifest:
>   `kubectl apply -f https://github.com/cert-manager/cert-manager/releases/latest/download/cert-manager.yaml`
> * **Intranet Root CA:** Ensure the intranet Root CA secret (`oci-lan-root-ca-key-pair`) and ClusterIssuer (`k8s/ha/lan/tls/intranet-cluster-issuer.yaml`) are provisioned for `oci-lan-issuer`.

```bash
chmod +x scripts/deploy-k8s.sh
./scripts/deploy-k8s.sh
```

This single command:
* Checks `kubectl` connectivity to your cluster.
* Applies all namespaces (`oci-core`, `oci-internal`, `external-sim`, `monitoring`, `logging`).
* Deploys custom OCI applications, customer Zitadel IdP, core infrastructure, and dedicated `postgres-core` database in `oci-core`.
* Deploys off-the-shelf business applications (Forgejo, Nextcloud, Snipe-IT, ERPNext, Frappe HR) and dedicated `postgres-internal` database in `oci-internal`.
* Deploys external simulation mocks (Mock Broker, Mock Bank) in `external-sim`.
* Deploys Prometheus, Grafana, Loki, and Promtail monitoring stacks.
* Waits for core database and queuing workloads to become healthy.

To connect your local workstation to the cluster services for testing:
```bash
# Forward PostgreSQL Core, Valkey, and RabbitMQ
kubectl port-forward -n oci-core svc/postgres-core-service 5432:5432 &
kubectl port-forward -n oci-core svc/valkey-service 6379:6379 &
kubectl port-forward -n oci-core svc/rabbitmq-service 5672:5672 15672:15672 &
```

> **Clean Reset:** To tear down all OCI workloads and reset the cluster back to a clean state at any time, run `./scripts/cleanup-k8s.sh` (see [Section 6.8](#68-cluster-cleanup--environment-reset)).


---

### 6.5 Bare-Metal K3s Cluster Provisioning via Ansible

For users provisioning a fresh, unconfigured bare-metal server (e.g. mini PC, Intel NUC, or dedicated home lab node) from scratch:

1. **Configure Ansible Inventory:**
   Edit `ansible/inventory/hosts.ini` with your home server's IP address and SSH user:
   ```ini
   [k3s_servers]
   192.168.1.100 ansible_user=ubuntu ansible_ssh_private_key_file=~/.ssh/id_ed25519
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
> The simulation engine and mock personas are strictly restricted to the **`oci-dev`** namespace (or local mock environments). The production environment (`oci-core`) is safeguarded: any attempt to target production domains (`api.oci.local`) or production cluster services (`*.oci-core.*`) in live mode is automatically rejected by built-in configuration guards.

```bash
# 1. Run in Mock Mode (offline, embedded in-memory):
go run ./simulation/cmd/simulator -mode mock

# 2. Run in Live Mode against Kubernetes oci-dev Staging (smart URL routing):
go run ./simulation/cmd/simulator -mode live -url https://dev.api.oci.local

# 3. Run in Live Mode against local test services:
go run ./simulation/cmd/simulator \
  -mode live \
  -server-url http://localhost:8080 \
  -bff-url http://localhost:8081
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

The following roadmap outlines planned technical debt remediation, scope streamlining, and architectural enhancements to evolve Orange Cat Investments into an authentic, production-grade self-hosted small-to-medium business (SMB) platform:

### 7.8 Monitoring, Observability & COTS Telemetry
Make better use of our logging and monitoring infrastructure to surface issues and provide comprehensive cluster and application visibility:
* **COTS & Infrastructure Monitoring:**
  * **Forgejo Native Prometheus Telemetry:** Ensure Forgejo's native Prometheus metrics endpoint (`/metrics` or `https://forgejo.oci.local/metrics`) is monitored by Prometheus: track HTTP request latency, active Git operations, SSH sessions, background queue length, database connection pool statistics, repository count, and runner job queue wait times.
  * **Nextcloud (Sidecar Exporter Pattern):** Nextcloud does not expose a native Prometheus endpoint by default; instead, its built-in serverinfo app exposes system telemetry as JSON/XML at `/ocs/v2.php/apps/serverinfo/api/v1/info?format=json`.
    * *Setup:* Generate a dedicated Nextcloud monitoring token (`occ config:app:set serverinfo token --value your-token`) and deploy `nextcloud-exporter` (such as `xperimental/nextcloud-exporter`) either as a sidecar container in your Nextcloud pod or as a tiny standalone Deployment.
    * *What to Track:* Active users, storage free space, database size, PHP-FPM active/max processes (critical when running the FPM image behind NGINX), and pending background cron jobs.
  * **Snipe-IT Prometheus Monitoring Gap Analysis & Plan:**
    * *Current Status Assessment:* Snipe-IT (PHP/Laravel) does not expose a native Prometheus `/metrics` endpoint out of the box. Upstream provides standard REST endpoints (`/api/v1/...`) and a basic health check, but no Prometheus-formatted metric exposition.
    * *Remediation Plan:* Deploy a dedicated exporter sidecar container (e.g. `snipeit-exporter` or a custom lightweight Go/Python daemon) configured with a scoped read-only Snipe-IT API bearer token.
    * *What to Track:* Hardware asset inventory breakdown by status label (Ready to Deploy, Deployed, Broken/Out for Repair, Archived), checkout-to-employee ratios, pending hardware audit overdue alerts, software license seat utilization and expiration dates, PHP-FPM active/idle worker pools, and API latency.
  * **Frappe HR Prometheus Monitoring Gap Analysis & Plan:**
    * *Current Status Assessment:* Frappe HR / Frappe Framework runs on a Python WSGI/Gunicorn architecture with asynchronous background workers backed by Redis Queue (RQ). Out of the box, standard HTTP ports do not expose an unauthenticated Prometheus `/metrics` endpoint.
    * *Remediation Plan:* Deploy `rq-exporter` against the internal Redis queue instance used by Frappe to track background job backlog (short, default, long queues), failed job counts, and queue wait latency. Install the `frappe-prometheus` extension app or deploy a lightweight exporter sidecar utilizing Frappe REST API tokens.
    * *What to Track:* Pending leave requests, unapproved feline catnip break requests, pending payroll run approvals, attendance synchronization queue lag, and active employee headcount.
  * **ERPNext Prometheus Monitoring Gap Analysis & Plan:**
    * *Current Status Assessment:* ERPNext shares the underlying Frappe Framework architecture with Frappe HR and similarly lacks a native `/metrics` endpoint on its web interface.
    * *Remediation Plan:* Deploy an ERPNext telemetry sidecar or scheduled probe querying ERPNext's REST API and diagnostic CLI (`bench doctor`).
    * *What to Track:* Background job execution latency, ledger submission queue depth, pending invoice/billing document backlogs, automated financial statement clearing errors, database connection pool utilization against `postgres-internal`, and email/webhook delivery queue depths.
* **Probes & Healthchecks:**
  * Automated synthetic healthcheck monitors for internal websites, BFF proxies, and Domain APIs.
* **Queue & Message Streaming Monitors:**
  * Queue monitors for RabbitMQ deadletter queues (DLQ) and alert thresholds for excessive queue depth or stagnant consumers on regular queues.
* **Log Aggregation & Anomaly Alerts:**
  * Log-based alert rules in Loki/Promtail for sudden spikes in `ERROR` and `FATAL` log events.
* **Alert Notifications & Escalation:**
  * Real-time pub/sub alert dispatching fanning out critical notifications to Pebble smartwatches, employee portal web banners, and push notification channels.

### 7.9 Fault Tolerance, Chaos Engineering & Negative API Testing
Ensure that negative scenarios with external financial APIs and unexpected internal service degradation are gracefully handled with functioning alerting:
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

### 7.11 Zero Trust Workload Identity & Dynamic Secret Management
Complete the zero-trust security architecture to eliminate trust assumptions between internal workloads, enforce network isolation, and eliminate long-lived static credentials:
* **Dynamic Secret Management & CSI Secrets Driver:**
  * Migrate from static base64-encoded Kubernetes Secret manifests to HashiCorp Vault or cert-manager CSI secret drivers, providing short-lived, dynamically generated database credentials and API tokens.
* **Workload Identity Service Mesh:**
  * Explore SPIRE (SPIFFE Runtime Environment) or OpenZiti (https://github.com/openziti/ziti) to establish cryptographically attested workload identities for zero-trust service mesh connectivity without depending exclusively on network perimeter security.

### 7.12 COTS Application Lifecycle Management & Upstream Security Patching Runbooks
Document comprehensive, step-by-step Standard Operating Procedures (SOPs) and operational runbooks for IT systems administrators (`emp-human-frank`, Platform Security & K8s Infrastructure Lead) to safely apply upstream security patches, minor updates, and major version upgrades across all Commercial-Off-The-Shelf (COTS) business platforms in `oci-internal` and identity systems in `oci-core`:
* **Standardized IT Maintenance & Pre-Flight Procedure:**
  * Automated pre-upgrade database snapshots via Kubernetes backup cronjobs (`postgres-internal-backup-cronjob` and `postgres-core-backup-cronjob`).
  * Staging verification gate in `oci-dev` prior to applying manifests to production (`oci-internal` and `oci-core`).
  * Maintenance windows and Ingress maintenance banners.
* **Application-Specific Upgrade Runbooks:**
  * **Forgejo (`k8s/oci-internal/forgejo.yaml`):** Image bumps, database migrations against `postgres-internal`, Forgejo Actions Runner updates, and OAuth2 provider validation.
  * **Nextcloud (`k8s/oci-internal/nextcloud.yaml`):** Sequential major version upgrades (28 $\rightarrow$ 29 $\rightarrow$ 30), post-upgrade `occ db:add-missing-indices`, and app compatibility audits.
  * **Snipe-IT (`k8s/oci-internal/snipe-it.yaml`):** Laravel database migrations (`php artisan migrate --force`), preserving encryption keys (`APP_KEY`), and asset QR code verification.
  * **ERPNext & Frappe HR (`k8s/oci-internal/erpnext-frappe.yaml`, `frappe-hr.yaml`):** Multi-app DocType synchronization via Frappe bench migrations, cache clearing, and timesheet/payroll ledger verification.
  * **ZITADEL Customer Identity Provider (`k8s/oci-core/zitadel.yaml`):** Executing ZITADEL setup jobs (`zitadel init` / `zitadel setup`) for schema migrations and zero-downtime rolling updates.
* **Automated Post-Upgrade Smoke Testing & Rollback Runbooks:**
  * Automated validation via simulation driver (`go run ./simulation/cmd/simulator -mode live -persona emp-human-frank`).
  * Single-command rollback SOPs restoring persistent volume snapshots and rolling back deployment revisions (`kubectl rollout undo deployment/<cots-app> -n oci-internal`).

### 7.15 Fast Typo-Tolerant Full-Text Search via Meilisearch (Employee & Customer Portals)
Integrate the in-cluster **Meilisearch** instance (`meilisearch-service.oci-core.svc.cluster.local:7700`) into the customer and employee web portals to deliver instantaneous, typo-tolerant full-text search as a working platform showcase:
* **Employee Portal (Unified Enterprise & Clinical Search):**
  * Implement a global "Command+K" search palette in `web/employee-portal` allowing corporate staff, caretakers, and engineers to instantly search:
    * IT Helpdesk tickets and resolutions (`oci-internal/helpdesk`).
    * Knowledge Base documents, workplace policies, standard operating procedures (SOPs), and clinical protocols stored in Nextcloud.
    * Feline Electronic Veterinary Health Records (VHR) and biometric notes.
    * Facilities hardware asset inventory and firmware releases.
* **Customer Portal (Investment Catalog & Research Search):**
  * Instant search bar in `web/customer-portal` for:
    * Equity market tickers, corporate fund descriptions, and feline behavioral alpha strategies.
    * ESG feline welfare scorecards and habitat telemetry archives.
    * Historical tax forms and investment performance statements.
* **Backend Ingestion & Index Synchronization Pipeline:**
  * Create lightweight change-data-capture (CDC) or event-driven indexers in Go (`cmd/worker` or dedicated service) subscribing to RabbitMQ domain events (`ticket_created`, `vhr_updated`, `asset_registered`, `trade_executed`) to continuously sync searchable document indices in Meilisearch.
  * Implement BFF proxy search endpoints (`/api/v1/employee/search`, `/api/v1/customer/search`) with role-based document filtering and tenant isolation.

### 7.16 Pure gRPC Wire Transport & Complete BFF Read-Route Decoupling to BLL
Finalize the microservices transition from transitional hybrid REST fallbacks to pure, decoupled gRPC wire transport:
* **Complete BFF Decoupling from Direct Database Access:**
  * Migrate all remaining direct database repository lookups (`pgxpool`) in `cmd/customer-bff` and `cmd/employee-bff` (e.g. portfolio queries, care schedules, asset lists) to route exclusively through the Customer BLL (`cmd/customer-bll`) and Internal BLL (`cmd/internal-bll`).
  * Remove PostgreSQL connection pool dependencies entirely from both BFF binaries, ensuring BFFs function purely as presentation and session management proxies.
* **Native gRPC Wire Transport over Mutual TLS:**
  * Transition inter-service communication between BLL services and Domain APIs (`workforce-api`, `facilities-api`, `core-invest-api`, `ops-api`) from REST/HTTP to pure gRPC over mTLS using the compiled Protobuf services in `pkg/gen/go/`.
  * Leverage gRPC streaming where appropriate (e.g., real-time habitat telemetry and live portfolio valuation recalculations).

### 7.17 Automated Health Dashboard & Multi-Tenant Uptime Monitoring via Gatus
Deploy and configure **[Gatus](https://github.com/TwiN/gatus)**, an automated service health dashboard and status page, to provide real-time uptime monitoring and transparent system status across all OCI custom services and commercial COTS applications:
* **Target Services & Health Probing Endpoints:**
  * **OCI Customer & Public Planes:** Static Marketing Site (`https://oci.local`), Customer Web Portal (`https://customer.oci.local`), Customer BFF (`http://oci-customer-bff-service.oci-core.svc.cluster.local:8083/healthz`), Customer BLL (`http://customer-bll-service.oci-core.svc.cluster.local:8085/healthz`), and external simulation services (Mock Bank `:8088/healthz`, Mock Broker `:8089/healthz`).
  * **OCI Employee & Internal Planes:** Employee Web Portal (`https://employee.oci.local`), Employee BFF (`http://oci-employee-bff-service.oci-core.svc.cluster.local:8081/healthz`), Internal BLL (`http://internal-bll-service.oci-core.svc.cluster.local:8086/healthz`), and Domain APIs (`:8080/healthz`).
  * **COTS Enterprise Platforms:** Forgejo (`http://forgejo-service.oci-internal.svc.cluster.local:3000/api/v1/version`), Nextcloud (`http://nextcloud-service.oci-internal.svc.cluster.local:80/status.php`), Snipe-IT (`http://snipe-it-service.oci-internal.svc.cluster.local:80/healthz`), ERPNext/Frappe HR (`http://erpnext-service.oci-internal.svc.cluster.local:8000/api/method/ping`), and ZITADEL (`http://zitadel-service.oci-core.svc.cluster.local:8080/debug/healthz`).
* **Dashboard Capabilities & Architecture:**
  * Deploy Gatus as a lightweight, single-pod Deployment (`twinproduction/gatus:latest`) in namespace `monitoring` or `oci-core` backed by a declarative ConfigMap (`gatus-config`).
  * Expose a unified, mobile-responsive status page at `https://status.oci.local` featuring uptime percentages (7-day / 30-day), hourly latency heatmaps, TLS certificate expiration tracking, and incident history.
* **Alerting Integrations:**
  * Configure Gatus alert endpoints to trigger automatic incident events into RabbitMQ or webhooks targeting `cmd/pebble-proxy` and the employee portal real-time SSE stream whenever a monitored endpoint fails consecutive health checks.


