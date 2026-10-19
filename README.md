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

### 7.7 Enterprise Multi-Tier Application Architecture (Implemented)
Orange Cat Investments implements all architectural patterns across an enterprise-grade multi-tier system. The platform intentionally demonstrates an enterprise-grade microservices architecture designed to scale well beyond a single small business, featuring strict tier isolation: Frontends $\rightarrow$ Backend-for-Frontend (BFF) $\rightarrow$ Business Logic Layer (BLL) $\rightarrow$ Domain APIs (gRPC over isolated SQL databases) $\rightarrow$ Async Workers & Batch Schedulers.

* **Custom Application Types & Monorepo Mapping:**
  * **Internal Web Portal (`web/employee-portal` + `cmd/employee-bff`):**
    * *Architecture:* Vue 3 Single Page Application (SPA) backed by Go Employee BFF (`:8081`).
    * *COTS Application Directory / Launcher:* Base launchpad page with 1-click status cards linking directly to internal COTS platforms (Nextcloud, Forgejo, ERPNext/Frappe HR, Snipe-IT, and Grafana).
    * *Intranet & Workplace Policies:* Internal HR knowledge base with WebDAV document reader linking directly to policy documents, standard operating procedures (SOPs), and clinical protocols in Nextcloud departmental group folders.
    * *Engineering Architecture & Handbook:* Internal engineering documentation viewer with developer expectations, GitOps workflow, and direct markdown rendering of Architecture Decision Records (ADRs `0001`, `0002`, `0003`).
    * *Built-Out Internal Functionality:*
      * *Veterinary Care Schedules & Medical Holds:* Feline executive care editor (dietary protocols, feeding times, special medical directives, perch zones) and one-click Emergency Medical Trading Hold toggle halting algorithmic trading triggers.
      * *Electronic Veterinary Health Records (VHR):* Clinical examination records, dental scoring (1–5 scale), biometric weight progression charts, vaccination histories, and prescription tracking.
      * *IT Helpdesk & Issue Tracker:* Bi-directional synchronization with Forgejo issue tracker (`oci-internal/helpdesk`), priority triage, discussion threads, and on-call escalation.
      * *IT Onboarding & Lifecycle Sagas:* Distributed sagas (`OnboardingSaga` & `OffboardingEngine`) automating user accounts across Forgejo (Git repos & SSH keys), Nextcloud (storage quotas & group folders), and Snipe-IT (hardware asset checkouts).
      * *Timesheets & Attendance Tracking:* Clock-in/clock-out tracking with activity categorization, meal deduction calculations, and automated provisioning to Frappe HR DocTypes / ERPNext cost centers.
      * *Compensation & Payroll:* Salary/hourly compensation records, catnip allowances, direct deposit bank routing, and immutable paystub history with tax withholding summaries.
      * *Executive Reports & Business Intelligence:* Monthly P&L dashboards, operational burn rates, cash reserves, total AUM trends, and automated monthly management report generator depositing PDF/CSV packages into Nextcloud WebDAV compliance vault.
      * *Leave Requests & Catnip Breaks:* Paid time off, medical leave, and feline sabbatical requests with manager approval state machines and calendar integration.
      * *Performance Review Cycles & Assessments:* Dual-evaluation cycles for human staff and feline executives (rating whisker symmetry, nap discipline, zoomie precision, and alpha contribution).
      * *Workplace Incidents & Hazard Logs:* Safety hazard reporting (e.g. knocked water glasses, wire chew incidents), status tracking, and mitigation logs.
      * *Dynamic Quantitative Backtesting Workbench:* Quantitative historical alpha backtesting engine with customizable feline observation multipliers, Sharpe ratio, and drawdown metrics.
    * *Architectural Invariants:*
      * This application **NEVER directly connects to Domain APIs or database schemas**.
      * Exclusively consumes the Internal Business Logic Layer (`cmd/internal-bll`) and COTS REST/OAuth2/WebDAV APIs.
      * Manages browser authentication sessions via opaque HTTP-only cookies (`__Host-employee-session`) and validates double-submit CSRF tokens.

  * **Customer Web Portal (`web/customer-portal` + `cmd/customer-bff`):**
    * *Architecture:* Vue 3 Single Page Application (SPA) backed by Go Customer BFF (`:8083`).
    * *Account Summary Landing Page:* Real-time portfolio valuation, cash balance, YTD return metrics, asset allocation breakdowns, and historical valuation growth charts.
    * *Orange Observation & Investment Insights:* Live low-latency (<500ms) WebRTC habitat video stream player with real-time AI vision overlays (activity classification & confidence scores), plus the live Zoomie Index score banner.
    * *Bank Account Linking:* Plaid client integration (`pkg/bank/plaid`) with mock bank verification for ACH account linking.
    * *Funds Transfers & Capital Deposits:* Automated Dollar-Cost Averaging (DCA) deposit scheduler (weekly, bi-weekly, monthly) and transfer status tracking.
    * *Investment Order Management:* Active trading terminal supporting 1-click "Zoomie Pounce" instant executions, market orders, bracket orders (stop-loss + take-profit), and trailing stop orders.
    * *Performance Summaries & Regulatory Statements:* Statements & Tax Forms hub rendering IRS Form 1099-B capital gains tax returns and monthly portfolio statements generated via pure Go PDF 1.4 engine (`pkg/reporting`).
    * *Built-Out Customer Functionality:*
      * *Live Market Ticker Stream:* Server-Sent Events (SSE) pushing real-time market price updates and percentage changes.
      * *Developer API Keys & Webhooks:* Self-service HMAC API key generation for customer quant trading bots and custom webhook endpoint subscriptions.
      * *Feline Welfare & ESG Transparency Dashboard:* Public-facing welfare metrics (overall welfare score, veterinary adherence, perch comfort rating, individual feline executive vitals).
      * *Customer Support Desk:* Ticket submission with category routing (Trading, Banking, Habitat, UI Bug, General), synchronized with a dedicated Forgejo repository (`oci-internal/customer-support`) and bi-directional message thread.
      * *Account Settings, Liquidation & GDPR Privacy:* Self-service GDPR "Right to be Forgotten" pipeline that anonymizes customer PII across databases and sessions, automatically issuing an immutable, cryptographically verifiable PDF GDPR Erasure Certificate.
    * *Architectural Invariants:*
      * This application **NEVER directly connects to Domain APIs, scheduled jobs, or async queue consumers**.
      * Exclusively consumes the Customer Business Logic Layer (`cmd/customer-bll`).
      * Manages customer browser authentication sessions via ZITADEL OIDC, issuing opaque HTTP-only cookies (`__Host-customer-session`) and enforcing CSRF protection.

  * **Static Public Website (`web/marketing`):**
    * *Architecture:* Fast, lightweight static website generated using Hugo and served via a hardened web container.
    * *Content & Scope:* Single-page public marketing website highlighting OCI's feline behavioral observation alpha model, investment tier comparisons (Observer vs. Momentum Alpha), live Zoomie Index snapshot badge, and clear calls-to-action (CTA) to register via the Customer Web Portal. Consumes design tokens and branding guidelines from `config/branding.yaml`.
    * *Implementation & Delivery:*
      1. Complete Hugo site structure in `web/marketing/` (layouts, static assets, content, design tokens).
      2. Automated build script (`scripts/build-marketing.sh`): Compiles HTML/CSS/JS into `public/`.
      3. Multi-stage Docker build packaging static assets into a lean `cmd/portal-server` container (`oci-marketing:latest`).
      4. Kubernetes deployment & Ingress in `k8s/oci-core/marketing.yaml` binding to `https://oci.local` and `https://www.oci.local`.

  * **Domain APIs (Workforce, Facilities, Core Invest, Ops):**
    * *Architecture:* Standalone, single-responsibility microservices running on top of isolated PostgreSQL database schemas. (Currently exposing HTTP/REST microservice endpoints aligned with the Protobuf contract models, with native gRPC wire transport planned as a future milestone).
    * *Primary Concern:* Data correctness, transaction boundaries, schema invariants, and performant querying.
    * *Architectural Invariants:*
      * **Domain APIs do not call each other.** Cross-domain database joins and direct cross-domain API calls are strictly forbidden.
      * The only authorized clients of Domain APIs are **Business Logic Layer (BLL) APIs** and authorized background worker/batch jobs.
      * Zero public ingress: Domain APIs are completely unexposed to external traffic and protected by Kubernetes NetworkPolicies.
    * *Implementation & Delivery:*
      1. *Protobuf Contract Definitions:* Strict Protobuf 3 specs defined in `proto/oci/`:
         * `proto/oci/workforce/v1/workforce.proto` (Employees, Care Schedules, Reviews, VHR, Incidents)
         * `proto/oci/facilities/v1/facilities.proto` (Hardware Assets, Maintenance Logs, Firmware Releases, OTA Jobs)
         * `proto/oci/core_invest/v1/core_invest.proto` (Portfolios, Holdings, Orders, Observations, Deposits)
         * `proto/oci/ops/v1/ops.proto` (IT Tickets, Alert Subscriptions, Audit Logs)
      2. *Code Generation:* Generated Go data structures and contract interfaces in `pkg/gen/go/` matching Protobuf schemas.
      3. *Microservice Separation:* 4 dedicated service binaries created in `cmd/`:
         * `cmd/workforce-api` (Port 50051, isolated connection to PostgreSQL `workforce` schema)
         * `cmd/facilities-api` (Port 50052, isolated connection to PostgreSQL `facilities` schema)
         * `cmd/core-invest-api` (Port 50053, isolated connection to PostgreSQL `core_invest` schema)
         * `cmd/ops-api` (Port 50054, isolated connection to PostgreSQL `ops` schema)
      4. *Security & Dynamic mTLS:* Inter-service mutual TLS enforced via `pkg/mtls` dynamic certificate reloaders (`crypto/tls` callbacks `GetCertificate` & `GetClientCertificate`) for zero static certs.
      5. *Kubernetes Manifests:* Individual K8s Deployments, Services, and `NetworkPolicy` manifests in `k8s/oci-core/domain-apis.yaml` restricting ingress exclusively to authorized BLL, batch, and worker pods.

  * **Business Logic Layer APIs (Customer BLL & Internal BLL):**
    * *Architecture:* Middle-tier microservices that sit strictly between the BFFs and the Domain APIs.
    * *Primary Concern:* Business rules, multi-domain orchestration, quantitative calculations, distributed sagas, third-party adapter execution (broker/bank), and event emission.
    * *Implementation & Delivery:*
      1. *Customer BLL (`cmd/customer-bll` :8085):*
         * Orchestrates customer-facing operations: validates trade orders, queries feline medical holds from `workforce-api` before executing trades on `core-invest-api`, executes orders against broker adapter, verifies bank accounts via Plaid adapter, and computes portfolio performance metrics.
         * Emits asynchronous events to RabbitMQ (`events.investment.order_placed.v1`, `events.banking.ach_transfer_requested.v1`).
         * Exposes internal REST endpoints consumed by `cmd/customer-bff`.
      2. *Internal BLL (`cmd/internal-bll` :8086):*
         * Orchestrates corporate operations: executes `OnboardingSaga` and `OffboardingEngine` across `workforce-api` and `ops-api`, validates veterinary care directives, calculates executive BI metrics, and triggers edge firmware updates via `facilities-api`.
         * Emits asynchronous events to RabbitMQ (`events.workforce.incident_created.v1`, `events.facilities.ota_triggered.v1`, `events.reporting.generate_monthly_requested.v1`).
         * Exposes internal REST endpoints for corporate operations.
      3. *Phased BFF Decoupling:* `cmd/customer-bff` routes customer order validation and trade placement to `cmd/customer-bll` when `CUSTOMER_BLL_URL` is set, and rejects direct order execution if the BLL fails. Complete decoupling of remaining read routes across `customer-bff` and `employee-bff` is scheduled for subsequent refactoring phases; legacy direct repository bindings remain in place as transition fallbacks.
      4. *Kubernetes Manifests:* Created `k8s/oci-core/bll-services.yaml` with deployments, services, and NetworkPolicies restricting ingress strictly to BFF pods.

  * **Async Queue Consumer Application (`cmd/worker`):**
    * *Architecture:* Event-driven worker subscribing to RabbitMQ message streams using the Go Watermill framework.
    * *Supported Handlers:*
      * Observation-triggered algorithmic portfolio allocation and automated trade execution.
      * Dynamic quantitative strategy backtest computation engine.
      * Edge device OTA firmware update pipeline (download, checksum verification, device flashing, status callback).
      * Executive report generation orchestrator depositing PDF/CSV assets into Nextcloud WebDAV.
      * **Async Bank Transfers & ACH Settlement:**
        * Event contract: `events.banking.ach_transfer_requested.v1` with transfer ID, customer ID, bank account, amount USD, and transfer type.
        * `HandleACHTransferRequested` in `cmd/worker`: Consumes transfer requests from RabbitMQ, calls mock bank API to initiate clearing, simulates asynchronous settlement window, calls `core-invest-api` to credit/debit portfolio cash balance, and publishes `events.banking.ach_settled.v1`.

  * **Scheduled / Batch Job Application:**
    * *Architecture:* Time-triggered cron and batch worker executing periodic business maintenance workflows.
    * *Supported Handlers:*
      * Automated PKI intermediate CA rotation ticker checking and rotating expired certificates.
      * **Recurring ACH Deposit Processor:**
        * Scheduled cron runner executing daily at 00:00 UTC (configured in `cmd/worker` and Kubernetes CronJob in `k8s/oci-core/batch-jobs.yaml`).
        * Queries due recurring deposit schedules from `core-invest-api` and dispatches `events.banking.ach_transfer_requested.v1`.
      * **Automated End-of-Month Statement Batch Generator:**
        * Scheduled cron runner executing on the 1st of every month at 01:00 UTC (in `cmd/worker` and Kubernetes CronJob in `k8s/oci-core/batch-jobs.yaml`).
        * Batches active customer portfolios, invokes `pkg/reporting` to generate IRS 1099-B tax forms and monthly portfolio PDF statements, deposits copies into Nextcloud WebDAV vault, and emits completion notifications.

  * **Pub/Sub Application (Incident Alerting Pipeline):**
    * *Architecture:* Topic-based event publication and real-time subscriber fanout across wearables and web frontends.
    * *Workplace Incident Alert Pipeline Implementation:*
      1. *Incident Event Publication:* When a safety incident is logged in `cmd/internal-bll`, emits `events.workforce.incident_created.v1` to RabbitMQ with incident ID, room zone, severity level (`urgent`, `critical`, `info`), and hazard description.
      2. *Real-Time Web Portal Banner:* Server-Sent Events (SSE) streaming hub in `cmd/employee-bff` (`GET /api/v1/employee/incidents/stream`) and ingestion broadcast endpoint (`POST /api/v1/employee/incidents/broadcast`), broadcasting emergency alert banners (`🚨 Workplace Incident Alert`) with 1-click Acknowledge/Dismiss across all active employee portal browser sessions.
      3. *Wearable Alert Push to Pebble Watch:* Dedicated listener in `cmd/pebble-proxy` (`POST /api/v1/ops/pebble/incidents/broadcast`) converting incidents into pending wearable alerts, serving `GET /ops/pebble/alerts` with `AppMessageDict` keys for vibrating smartwatch notifications, and accepting physical button ACK signals via `POST /ops/pebble/ack`.

### 7.8 Monitoring and Alerting
Make better use of our logging and monitoring infra to surface issues
* **Monitors:**
  * Monitors for COTS apps
  * Healthcheck monitors for websites and APIs
  * Queue monitors for deadletter queues and also for excessive queue depth on the regular queues
  * Monitor for Error level logs
* **Alert notifications:**
  * Pub/sub alert to watches and internal web banner?

### 7.9 Fault Tolerance and General Error Handling
Ensuring that negative scenarios with external APIs are well handled and that internal failures are reasonably handled and have functioning alerts.
* **External Errors:**
  * Mock Bank API - Add error responses that can be turned on/off by a new endpoint outside the regular banking endpoints
  * Mock Broker API - Add error responses that can be turned on/off by a new endpoint outside the regular broker endpoints
* **Chaos Monkey:**
  * Make scripts of actions to intentionally break OCI in certain ways. Likely by intentional kubernetes misconfiguration. After each of these, observe the results in terms of logs and alerts firing before restoring functionality and moving to the next test. Note anywhere there is a lack of logging/alerting.
  	* Turn off Postgres for OCI internal
  	* Turn off Postgres for OCI core
  	* Break network access between BLLs and Domain APIs
  	* Break network access between BFFs and BLLs
  	* Turn off a Domain API (repeat this for each Domain API)
  	* Turn off a BLL API (repeat this for each BLL)
  	* Turn off a BFF API (repeat this for each BFF)
  	* Turn off the mock Bank API
  	* Turn off the mock Broker API
  	* Add a deadletter message to a queue
  	* Add a malformed/invalid message to a queue

### 7.11 Zero Trust Security Architecture: Dynamic mTLS, Network Policies & Secret Management
Complete the zero-trust security architecture to eliminate trust assumptions between internal workloads, enforce network isolation, and eliminate long-lived static credentials:
* **Dynamic Inter-Service Mutual TLS (mTLS):**
  * Implement dynamic certificate reloaders using Go standard library `crypto/tls` callbacks (`GetCertificate` and `GetClientCertificate`) in inter-service gRPC/HTTP clients and servers across `cmd/server`, `cmd/customer-bff`, `cmd/employee-bff`, and `cmd/worker`.
  * Ensure rotated certificates provisioned by `cert-manager` from the internal CA are loaded directly from directory mounts without pod restarts (avoiding `subPath` volume mounts).
* **Micro-Segmentation via Kubernetes Network Policies:**
  * Author and apply `NetworkPolicy` manifests across all Kubernetes namespaces (`oci-core`, `oci-internal`, `oci-dev`, `external-sim`, `monitoring`, `logging`).
  * Enforce strict default-deny ingress and egress rules, explicitly whitelisting only authorized traffic flows (e.g., BFFs $\rightarrow$ Server on port 8080, Server $\rightarrow$ Postgres on port 5432, Server $\rightarrow$ Valkey on port 6379, Worker $\rightarrow$ RabbitMQ on port 5672).
  * Prevent unauthorized cross-namespace network bridging between internal enterprise tools (`oci-internal`) and customer-facing core systems (`oci-core`).
* **Zero Static Secrets & Workload Identity:**
  * Migrate from static base64-encoded Kubernetes Secret manifests to HashiCorp Vault or cert-manager CSI secret drivers, providing short-lived, dynamically generated database credentials and API tokens.
  * Explore SPIRE (SPIFFE Runtime Environment) or OpenZiti (https://github.com/openziti/ziti) to establish cryptographically attested workload identities for zero-trust service mesh connectivity without depending on network perimeter security.

