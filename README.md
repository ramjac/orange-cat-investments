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
* **Feline Acoustic & Vocalization Analysis:** Schema `core_invest.acoustic_events` and classification endpoint (`/core-invest/audio/classify`) distinguishing contentment purrs, playful chirps, and hunger meows.

### 5.2 Customer Experience & Retail Investor Surfaces (`web/customer-portal` & `cmd/customer-bff`)
* **Customer Web Portal & Dedicated Go BFF:** Vue 3 Single Page Application (`web/customer-portal` on `:5174`) and dedicated Go BFF server (`cmd/customer-bff` on `:8083`) utilizing ZITADEL OIDC authentication, Valkey server-side sessions, and double-submit CSRF protection.
* **Low-Latency In-Browser WebRTC Habitat Video Player:** Embedded WebRTC stream player component with real-time AI vision detection overlays for Garfield and Barneby.
* **Active Trading Terminal & Real-Time Price Ticker:** High-velocity execution terminal in `web/customer-portal` and live price ticker endpoint (`/api/v1/customer/ticker`).
* **Live "Zoomie Index" & Behavioral Momentum Alerts:** Real-time Zoomie Index scoring endpoint (`/api/v1/customer/zoomie-index`) and 1-click tactical pounce trade execution.
* **Automated Recurring Deposits & Capital Allocations:** Schema `core_invest.recurring_deposits`, Go BFF endpoints (`/api/v1/customer/deposits`), and Vue deposit management UI.
* **Customer Developer API Keys & Webhook Subscriptions:** Developer API key creation/revocation (`/api/v1/customer/api-keys`) and event webhook subscription management (`/api/v1/customer/webhooks`).
* **Feline Welfare & ESG Transparency Dashboard:** Transparency endpoint (`/api/v1/customer/transparency/welfare`) and ESG dashboard displaying veterinary adherence, daily nap hours, and perch comfort ratings backed by PostgreSQL queries.

### 5.3 Facilities & Mobile Field Maintenance (`facilities` & `mobile/flutter_app`)
* **Flutter Mobile Offline Database Sync:** Full local SQLite persistence (`sqflite`), QR scanner camera view, and offline queue sync engine (`SyncEngine`) posting batched logs to `/facilities/sync/maintenance-logs`.
* **Hardware Asset Management:** Keyset-paginated hardware inventory (`/facilities/assets`) supporting high-throughput edge camera fleets and IoT device tracking.
* **Edge Device Firmware OTA Pipeline:** Firmware release catalog (`/facilities/firmware`) and OTA job scheduling (`/facilities/firmware/ota-jobs`), executed by the background worker Watermill router (`ota_pipeline_handler`).
* **Smart Feeder Telemetry & Automated Snack Disbursement:** Schema `facilities.feeder_telemetry` and REST endpoints (`/facilities/feeders/telemetry`, `/facilities/feeders/disburse`).
* **Smart Collar Biometrics & Accelerometer Streaming:** Schema `facilities.collar_telemetry` and telemetry ingestion endpoint (`/facilities/collars/telemetry`).
* **Electronic Veterinary Health Records (VHR) & Clinical Charting:** Schema `workforce.vhr_records`, Go service layer, and Employee Portal VHR charting tab.
* **Perch Cushion Comfort & Thermal Sunbeam Sensors:** Schema `facilities.perch_telemetry` and telemetry ingestion endpoint (`/facilities/perches/telemetry`).
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

#### Step 3: Initialize Database Schemas & Seed Data
Execute the unified schema initialization script `scripts/init.sql`. This creates the schemas (`workforce`, `facilities`, `core_invest`, `ops`), table constraints, and initial employee records (Garfield, Barneby, Dr. Elena, etc.):

```bash
# Apply schema and initial records using psql:
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

The repository includes a comprehensive simulation driver (`simulation/cmd/simulator`) that exercises all 12 OCI personas (Garfield, Barneby, Alice, Bob, Carol, David, Dr. Elena, Frank, Arthur, Chloe, Rick, Elise) through their full daily operational routines:

```bash
# 1. Run in Mock Mode (offline, in-memory):
go run ./simulation/cmd/simulator -mode mock

# 2. Run in Live Mode (against running API Server & PostgreSQL):
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

### 7.1 Technical Debt & Hardcoded Data Remediation (P0)
Address pre-existing architectural debt and hardcoded entity data documented in [TODO.md](file:///home/ramjac/orange-cat-investments/TODO.md) to enforce strict domain boundaries, schema isolation, and compliance with [AGENTS.md](file:///home/ramjac/orange-cat-investments/AGENTS.md):
* **Customer BFF Session & Authentication Refactoring (`cmd/customer-bff`):**
  * **Remove In-Memory Dev Session Fixtures:** In [`NewInMemorySessionStore`](file:///home/ramjac/orange-cat-investments/cmd/customer-bff/main.go#L54-L71), remove statically registered test session tokens (`sess-arthur-token`, `sess-chloe-token`) from default application startup. Guard local mock fixtures behind an explicit environment flag (`ENABLE_DEV_MOCK_AUTH=true`) or load them exclusively from external seed configuration files during test runs.
  * **Eliminate Dev Login Persona Switching:** Replace the static persona switch (`req.Persona == "chloe"` vs. defaulting to Arthur) in [`handleLogin`](file:///home/ramjac/orange-cat-investments/cmd/customer-bff/main.go#L380-L404) with production ZITADEL OIDC authentication callbacks (`/api/v1/auth/callback`), or dynamic database-backed account lookups against `core_invest.portfolios` and customer tables.
  * **Dynamic Portfolio Persistence Fallback:** In [`handleGetPortfolio`](file:///home/ramjac/orange-cat-investments/cmd/customer-bff/main.go#L477-L535), replace static JSON response payloads (used when external broker integration is nil) with SQL queries against `core_invest.portfolios` and `core_invest.portfolio_holdings` as the authoritative fallback system of record.
  * **Database-Backed Developer API Keys & Webhook Registrations:** In [`handleGetAPIKeys`](file:///home/ramjac/orange-cat-investments/cmd/customer-bff/main.go#L855-L875) and [`handleRegisterWebhook`](file:///home/ramjac/orange-cat-investments/cmd/customer-bff/main.go#L915-L940), remove hardcoded bot names (`"Chloe Momentum Bot"`) and static URLs (`https://quant-bot.chloespark.io/...`). Persist customer API keys and webhooks in dedicated PostgreSQL tables (`core_invest.api_keys`, `core_invest.customer_webhooks`) scoped to the authenticated `session.CustomerID`.
  * **Dynamic Portfolio Resolution in Order Creation:** In [`handleCreateOrder`](file:///home/ramjac/orange-cat-investments/cmd/customer-bff/main.go#L580-L610), require an explicit `portfolio_id` parameter or resolve the active portfolio dynamically from the customer's authenticated session, eliminating hardcoded fallback IDs.
* **Operations Service Dynamic Identity & Device Pairing (`internal/service/ops`):**
  * **Dynamic On-Call Acknowledgments:** In [`ProcessPebbleAck`](file:///home/ramjac/orange-cat-investments/internal/service/ops/service.go#L130-L145), remove the hardcoded fallback to `"emp-human-frank"`. Require an authenticated employee session token or resolve the on-call user identity dynamically from the Pebble watch paired device ID registered in `ops.pebble_pairings`.
* **Repository Architecture & Mock Isolation (`internal/repository`):**
  * **Workforce In-Memory Repository Separation:** In [`repository.go`](file:///home/ramjac/orange-cat-investments/internal/repository/workforce/repository.go#L175-L410), decouple the in-memory fallback map (pre-seeded with Garfield, Barneby, Alice, etc.) into a dedicated test-only package (`internal/repository/workforce/mock.go`). Ensure production server binaries (`cmd/server`) fail fast when a healthy PostgreSQL connection is unavailable instead of silently falling back to mock state.
  * **Operations & Facilities Mock Cleanups:** Move hardcoded fallback ticket authors (`"emp-human-frank"`) and mock assignees (`"emp-human-bob"`, `"emp-human-rick"`) from default repository initialization paths into explicit unit test fixtures.
* **Domain Boundary & Vendor Encapsulation Enforcement:**
  * Enforce strict domain-centric variable naming across consuming packages (e.g. `brokerClient` / `bankClient` instead of vendor-specific product names), restricting vendor SDK nomenclature solely to driver packages (`pkg/broker/...`, `pkg/bank/...`).

### 7.2 Trimming Scope & Refocusing on the Original Small Business Objective
The primary purpose of this repository is to demonstrate an end-to-end, production-grade, self-hosted platform for a small-to-medium business (SMB). To ensure clarity, maintainability, and operational realism, the planned scope is refocused away from novelty hardware and over-engineered consumer IoT:
* **Camera-Only Observation Engine:**
  * Constrain feline behavioral telemetry strictly to optical IP camera streams (RTSP/WebRTC) analyzed by computer vision inference.
  * Deprecate and remove redundant, non-essential IoT hardware models and telemetry schemas:
    * Smart collar biometrics & accelerometer streaming (`facilities.collar_telemetry`)
    * Perch comfort, weight, and thermal sunbeam sensors (`facilities.perch_telemetry`)
    * Automated snack disbursement and feeder load-cell telemetry (`facilities.feeder_telemetry`)
    * Feline acoustic and vocalization harmonic classification (`core_invest.acoustic_events`)
  * Streamline the facilities hardware asset catalog to focus strictly on real SMB infrastructure: IP cameras, habitat edge compute gateways, networking switches, pebble watches, smartphones, and staff workstations.
* **De-emphasizing Consumer-Side Speculative Features:**
  * Pivot engineering investment away from speculative retail trading features (e.g., FIX 5.0 institutional gateways, deep reinforcement learning trading bots, high-frequency tick streams) toward robust internal business operations, employee productivity, and reliable self-hosting.

### 7.3 Actionable Small Business Operational Capabilities
Ensure all essential daily operational tasks of an SMB (except email which notably won't be selfhosted) can be executed natively across the integrated custom Go services and off-the-shelf business applications (COTS: ERPNext, Frappe HR, Nextcloud, Forgejo, Snipe-IT):

#### 7.3.1 Timesheets & Attendance Tracking
* **Frappe HR Bi-Directional Integration:**
  * Integrate the Employee Portal (`web/employee-portal`) with Frappe HR's attendance and timesheet REST APIs.
  * Implement an intuitive time-tracking interface allowing employees (e.g., facilities technicians, veterinary staff, software engineers) to log daily work hours, task codes, meal breaks, and habitat observation shifts.
* **Timesheet Approval Workflow:**
  * Provide department leads (e.g., Alice Vance for HR/Admin, Bob Builder for Facilities) with manager dashboards to review, adjust, approve, or reject employee timesheets.
  * Emit Watermill domain events (`events.workforce.timesheet_approved.v1`) to automatically register verified billable hours in ERPNext cost centers.

#### 7.3.2 Payroll Processing & Employee Compensation
* **Automated Payroll Runs:**
  * Configure ERPNext / Frappe HR Payroll Entry processing driven by approved timesheets and salary structures.
  * Automate earnings calculations, tax withholdings, social contributions, and direct deposit ledger entries.
* **Employee Self-Service Pay Stubs:**
  * Implement secure BFF endpoints (`/api/v1/employee/payroll/slips`) querying Frappe HR salary slips for authenticated employees.
  * Provide a dedicated "Compensation & Payroll" tab in the Employee Portal where staff can view historical salary slips, year-to-date tax deductions, and download official PDF pay stubs.

#### 7.3.3 End-to-End Automated Employee Lifecycle Sagas
* **Comprehensive Onboarding Saga:**
  * Expand [`OnboardingSaga`](file:///home/ramjac/orange-cat-investments/internal/saga/saga.go#L58-L159) triggered by Frappe HR new hire webhooks (`/api/v1/webhooks/frappe-hr`) to fully provision internal IT assets:
    * Create user account and SSH/GPG keys in **Forgejo** for Git/development access.
    * Provision user workspace, quota, and default group folders (e.g., HR, Engineering) in **Nextcloud**.
    * Issue initial hardware assignments (workstation, laptop, security badge) in **Snipe-IT**.
    * Initialize care schedule and shift roster in `workforce.care_schedules`.
* **Complete Separation & Offboarding Engine:**
  * Operationalize [`OffboardingEngine`](file:///home/ramjac/orange-cat-investments/internal/saga/saga.go#L161-L232) to execute an automated multi-step deprovisioning workflow upon employee resignation or termination:
    * Instantly invalidate active Valkey session tokens and revoke ZITADEL/Forgejo OAuth2 refresh tokens.
    * Lock and archive personal Nextcloud storage to a secure compliance vault.
    * Generate Snipe-IT equipment check-in tickets for asset recovery.
    * If departing an executive feline care role, immediately transition affected habitat protocols to an active caretaker and log departure status in Frappe HR.

#### 7.3.4 Internal Documents, Knowledge Base & Team Collaboration
* **Nextcloud Departmental Hub:**
  * Configure Nextcloud group folders structured by business departments: HR & Policies, Facilities & Maintenance SOPs, Veterinary Protocols, and Executive Reporting.
  * Establish company-wide shared calendars for shift rotations, habitat maintenance windows, and corporate milestones.
* **Intranet Document Embedding:**
  * Integrate Nextcloud WebDAV APIs into `web/employee-portal` to allow employees to view and search company policies, safety protocols, and standard operating procedures directly within the portal without switching tabs.

#### 7.3.5 Performance Reviews & Health Assessment Cycles
* **Structured Multi-Phase Review Cycles:**
  * Build out `workforce.review_cycles` in the database and Employee Portal UI for managing periodic (quarterly/annual) performance evaluations.
  * Support employee self-appraisals, peer feedback, manager assessments, and objective scoring against organizational KPIs.
* **Integrated Veterinary Health Assessments:**
  * Couple staff reviews with periodic feline clinical wellness audits conducted by Dr. Elena Rostova, storing health scoring and habitat optimization recommendations alongside staff performance records.

#### 7.3.6 Helpdesk & Issue Management (Internal & Customer)
* **Employee IT Helpdesk (Forgejo Issue Bridge):**
  * Implement an IT support intake form in the Employee Portal allowing staff to submit hardware, software, or habitat issues.
  * Synchronize tickets with Forgejo repository issues (`oci-internal/helpdesk`), automatically tracking ticket status, comments, and severity.
  * Route urgent priority tickets (e.g., server failure, habitat climate anomaly) directly to the on-call engineer's Pebble watch or mobile push notification service (`ntfy`).
* **Customer Bug & Feedback Intake:**
  * Provide an authenticated customer support form in `web/customer-portal` (`/api/v1/customer/support/tickets`) allowing retail investors to report UI bugs or transaction inquiries.
  * Ingest tickets into an isolated Forgejo customer-issues tracker, scrubbing internal comments before exposing sanitized updates back to the customer.

#### 7.3.7 Internal Business Intelligence & Automated Reporting
* **Automated Management Report Generator:**
  * Implement a scheduled reporting worker in `cmd/worker` that aggregates metrics across ERPNext (P&L, operational burn, cash reserves) and `workforce` (headcount, attendance rates, overtime hours).
  * Automatically generate executive summary reports (PDF and CSV) on the 1st of each month.
* **Compliance Archival in Nextcloud:**
  * Automatically deposit generated executive reports into Nextcloud's restricted `Executive/Reports/YYYY-MM/` directory via WebDAV, ensuring historical preservation and audit compliance.

### 7.4 Customer Lifecycle, Account Management & GDPR Compliance
Transition customer interactions from mock endpoints to an authentic, legally compliant lifecycle:
* **Customer Self-Registration & Onboarding:**
  * Implement complete self-service signup via ZITADEL OIDC, enforcing email verification and password complexity.
  * Dynamic bank account linking via Plaid / ACH mock interface (`external-sim/mock-bank`), generating an initial portfolio account in `core_invest.portfolios` with zero starting balance.
* **Account Closure & Liquidation Workflow:**
  * Provide a self-service "Close Account" workflow in `web/customer-portal`: automatically liquidate all open equity holdings, wait for settlement, and transfer net cash balances back to the customer's linked external bank account.
* **GDPR "Right to be Forgotten" Data Deletion Pipeline:**
  * Implement an automated GDPR data erasure engine:
    * Scrub all personally identifiable information (name, email, phone, physical address, IP logs) from PostgreSQL tables.
    * Invalidate and purge all active session records in Valkey.
    * Anonymize transaction ledgers to retain mathematical integrity for accounting and regulatory compliance while decoupling customer identities.
    * Generate a cryptographic data deletion certificate for the user confirming compliance with GDPR Article 17.
* **Automated Investment Statements & Tax Documentation:**
  * Build a statement generation pipeline producing monthly portfolio summary PDFs and annual Form 1099-B tax reports.
  * Deliver documents securely within the Customer Portal with encryption-at-rest and strict session-authorized download endpoints.

### 7.5 Multi-Node SMB High Availability & Resilient Home Server Topology
Evolve the single-node setup into a resilient multi-node home/office lab topology designed for 2 to 3 physical computers (e.g., mini PCs or Intel NUCs):
* **3-Node K3s HA Cluster Blueprint:**
  * Document and automate cluster setup using K3s with embedded etcd high-availability across 3 nodes to tolerate single-node failure without control plane downtime.
  * Implement distributed replicated storage using Longhorn or Rook-Ceph for persistent storage classes (`postgres-core-data-pvc`, `postgres-internal-data-pvc`, `nextcloud-data-pvc`), enabling automatic failover of database pods across nodes.
* **Local Network (LAN) Employee Portal Access:**
  * Configure intranet DNS (via Pi-hole, dnsmasq, or router split-horizon DNS) resolving `employee.oci.local` to the cluster's internal virtual IP (VIP) managed by kube-vip or MetalLB.
  * Issue intranet TLS certificates using internal Step-CA or cert-manager Local CA, installing the root certificate on employee devices to eliminate browser security warnings.
* **Secure Public Ingress for Customer Portal:**
  * Document and provide deployment configurations for exposing customer-facing surfaces (`web/customer-portal` and `cmd/customer-bff`) to the internet:
    * **Recommended Method:** Cloudflare Tunnel (`cloudflared` daemon in K8s) providing zero-open-port inbound ingress with automatic DDoS protection and SSL termination.
    * **Alternative Method:** NGINX Ingress Controller with Let's Encrypt automated TLS via cert-manager and dynamic DNS (DDNS) for direct router port forwarding (80/443).

### 7.6 Self-Hosted GitOps & CI/CD Pipelines via Forgejo
Simulate a complete real-world software engineering lifecycle within the self-hosted environment:
* **Forgejo Actions CI/CD Implementation:**
  * Deploy Forgejo Actions runner pods in `oci-internal` to execute continuous integration pipelines directly on the cluster.
  * Define workflow pipelines (`.forgejo/workflows/ci.yaml`) to run automated tests (`go test -race ./...`, `go vet`, frontend unit tests) on every push and pull request.
* **Automated GitOps Promotion Workflow:**
  * Add a Senior developer persona who will be responsible for authoring ADRs and reviewing/approving pull requests by Elise Dev
  * Model the full software developer lifecycle using persona Elise Dev (`emp-human-elise`):
    1. Elise creates a feature branch and pushes changes to a Forgejo repository.
    2. Opens a Pull Request in Forgejo; automated CI workflow validates the build.
    3. Peer review and PR merge trigger an automated build pushing container images to the local Forgejo container registry.
    4. GitOps agent updates Kubernetes manifests in `k8s/oci-core/` and performs rolling pod updates without manual intervention.

### 7.7 Production Security: Real Identity, Fine-Grained RBAC & Zero Trust
Replace all mock authentication mechanisms with production-grade security boundaries:
* **Authentic Customer Security (ZITADEL OIDC):**
  * Require valid customer credentials (email/password with optional MFA/TOTP) through ZITADEL; eliminate developer mock tokens entirely.
  * Enforce strict tenant boundaries in the database repository and BFF layers: all SQL queries must enforce customer ID equality checks (`WHERE customer_id = $1`) derived from authenticated session tokens to prevent horizontal privilege escalation.
* **Fine-Grained Role-Based Access Control (RBAC):**
  * Enforce role-based permission checks across Employee BFF endpoints based on verified Forgejo OAuth2 claims:
    * **Human Resources (`emp-human-alice`):** Access to payroll slips, timesheet approvals, employee onboarding/offboarding, and review cycle configurations. Restricted from habitat hardware controls.
    * **Facilities & Edge Telemetry (`emp-human-bob`):** Access to camera assets, PTZ controls, firmware OTA release pipelines, and maintenance sync endpoints. Restricted from HR and financial ledgers.
    * **Veterinary Staff (`emp-human-elena`):** Exclusive authority to toggle Emergency Medical Trading Holds, prescribe care schedules, and sign clinical health assessments.
    * **Investment Team (`emp-human-carol`, `emp-human-david`):** Access to portfolio analytics, backtesting workbench, and quantitative strategy parameters.
    * **Infrastructure Ops (`emp-human-frank`):** Access to cluster metrics, IT helpdesk dispatch, and cert-manager CA rotation workflows.
  * Block unauthorized cross-role API requests with HTTP 403 Forbidden.
* **Zero Trust Security Architecture:**
  * Enforce mutual TLS (mTLS) with dynamically rotating short-lived certificates across all inter-service communications (BFF $\rightarrow$ Server $\rightarrow$ Postgres/Valkey).
  * Apply Kubernetes `NetworkPolicy` resources across `oci-core` and `oci-internal` to prevent cross-namespace network bridging except through explicitly allowed ingress/egress CIDR rules and ports.
  * Eliminate all static secrets by integrating HashiCorp Vault or cert-manager CSI secret drivers.
  	* Explore OpenZiti (https://github.com/openziti/ziti) or Spire (https://github.com/spiffe/spire) for a more complete security set up

