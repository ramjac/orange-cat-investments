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

### 7.1 Technical Debt & Hardcoded Data Remediation (P0) (Completed)
Addressed pre-existing architectural debt and hardcoded entity data documented in [TODO.md](file:///home/ramjac/orange-cat-investments/TODO.md) to enforce strict domain boundaries, schema isolation, and compliance with [AGENTS.md](file:///home/ramjac/orange-cat-investments/AGENTS.md):
* **Customer BFF Session & Authentication Refactoring (`cmd/customer-bff`):**
  * **Removed In-Memory Dev Session Fixtures:** In [`NewInMemorySessionStore`](file:///home/ramjac/orange-cat-investments/cmd/customer-bff/main.go#L54-L71), removed statically registered test session tokens (`sess-arthur-token`, `sess-chloe-token`) from default application startup. Mock fixtures are strictly gated behind an explicit environment flag (`ENABLE_DEV_MOCK_AUTH=true` or `SEED_DEV_SESSIONS=true`).
  * **Eliminated Dev Login Persona Switching:** Replaced the static persona switch (`req.Persona == "chloe"` vs. defaulting to Arthur) in [`handleLogin`](file:///home/ramjac/orange-cat-investments/cmd/customer-bff/main.go#L360-L400) with dynamic customer ID, name, and email validation and database lookup.
  * **Dynamic Portfolio Persistence Fallback:** In [`handleGetPortfolio`](file:///home/ramjac/orange-cat-investments/cmd/customer-bff/main.go#L477-L535), replaced static JSON response payloads with SQL queries against `core_invest.investment_portfolios` and `core_invest.portfolio_holdings` as authoritative fallback persistence.
  * **Database-Backed Developer API Keys & Webhook Registrations:** In [`handleGetAPIKeys`](file:///home/ramjac/orange-cat-investments/cmd/customer-bff/main.go#L855-L875) and [`handleRegisterWebhook`](file:///home/ramjac/orange-cat-investments/cmd/customer-bff/main.go#L915-L940), removed hardcoded bot names and static URLs. Customer API keys and webhooks are persisted in PostgreSQL tables (`core_invest.customer_api_keys`, `core_invest.webhook_subscriptions`) scoped to authenticated `session.CustomerID`.
  * **Dynamic Portfolio Resolution in Order Creation:** In [`handleCreateOrder`](file:///home/ramjac/orange-cat-investments/cmd/customer-bff/main.go#L580-L610), dynamically queries active portfolio associated with authenticated customer session via `ciSvc.GetPortfolio`.
* **Operations Service Dynamic Identity & Device Pairing (`internal/service/ops`):**
  * **Dynamic On-Call Acknowledgments:** In [`ProcessPebbleAck`](file:///home/ramjac/orange-cat-investments/internal/service/ops/service.go#L130-L145), eliminated hardcoded fallback to `"emp-human-frank"`. Resolves on-call user identity dynamically via request context (`employee_id`), `req.WatchID`, `req.DeviceID`, or environment configuration.
* **Repository Architecture & Mock Isolation (`internal/repository`):**
  * **Workforce In-Memory Repository Separation:** Decoupled in-memory fallback into dedicated `internal/repository/workforce/mock.go` with `NewMockRepository()`. Production server binaries (`cmd/server`, `cmd/employee-bff`) strictly require a healthy PostgreSQL pool in production mode.
  * **Operations & Facilities Mock Cleanups:** Confined fallback ticket authors and mock assignees into dedicated mock repositories (`internal/repository/ops/mock.go`, `internal/repository/facilities/mock.go`) used strictly for unit tests and local dev fallback.
* **Domain Boundary & Vendor Encapsulation Enforcement:**
  * Enforce strict domain-centric variable naming (`brokerClient` / `bankClient`) across consuming packages, encapsulating vendor SDK nomenclature solely in driver packages (`pkg/broker/...`, `pkg/bank/...`).

### 7.2 Trimming Scope & Refocusing on the Original Small Business Objective (Completed)
The primary purpose of this repository is to demonstrate an end-to-end, production-grade, self-hosted platform for a small-to-medium business (SMB). The planned scope has been refocused away from novelty hardware and over-engineered consumer IoT:
* **Camera-Only Observation Engine:**
  * Constrained feline behavioral telemetry strictly to optical IP camera streams (RTSP/WebRTC) analyzed by computer vision inference.
  * Completely removed redundant, non-essential IoT hardware models and telemetry schemas:
    * Smart collar biometrics & accelerometer streaming (`facilities.collar_telemetry`)
    * Perch comfort, weight, and thermal sunbeam sensors (`facilities.perch_telemetry`)
    * Automated snack disbursement and feeder load-cell telemetry (`facilities.feeder_telemetry`)
    * Feline acoustic and vocalization harmonic classification (`core_invest.acoustic_events`)
    * Habitat perch physical specification table (`facilities.observation_perches`)
  * Streamlined the facilities hardware asset catalog to focus strictly on real SMB infrastructure: IP cameras, habitat edge compute gateways, networking switches, pebble watches, smartphones, laptops, and staff workstations.
* **De-emphasizing Consumer-Side Speculative Features:**
  * Pivoted engineering investment away from speculative retail trading features (e.g., FIX 5.0 institutional gateways, deep reinforcement learning trading bots, high-frequency tick streams) toward robust internal business operations, employee productivity, and reliable self-hosting.

### 7.3 Actionable Small Business Operational Capabilities
Ensure all essential daily operational tasks of an SMB (except email which notably won't be selfhosted) can be executed natively across the integrated custom Go services and off-the-shelf business applications (COTS: ERPNext, Frappe HR, Nextcloud, Forgejo, Snipe-IT):

#### 7.3.1 Timesheets & Attendance Tracking (Completed)
* **Frappe HR Bi-Directional Integration:**
  * Integrate the Employee Portal (`web/employee-portal`) with Frappe HR's attendance and timesheet REST APIs.
  * Implement an intuitive time-tracking interface allowing employees (e.g., facilities technicians, veterinary staff, software engineers) to log daily work hours, task codes, meal breaks, and habitat observation shifts.
* **Timesheet Approval Workflow:**
  * Provide department leads (e.g., Alice Vance for HR/Admin, Bob Builder for Facilities) with manager dashboards to review, adjust, approve, or reject employee timesheets.
  * Emit Watermill domain events (`events.workforce.timesheet_approved.v1`) to automatically register verified billable hours in ERPNext cost centers.

#### 7.3.2 Payroll Processing & Employee Compensation (Completed)
* **Automated Payroll Runs:**
  * Implemented Frappe HR / ERPNext `Payroll Entry` and `Salary Slip` client interfaces (`pkg/cots/frappe`) and workforce service orchestration (`internal/service/workforce`).
  * Supported automated earnings, tax deductions (Federal tax, Social Security), and net pay distribution calculations via `POST /api/v1/workforce/payroll/run`.
* **Employee Self-Service Pay Stubs:**
  * Implemented secure BFF and domain API endpoints (`/api/v1/employee/payroll/slips` and `/api/v1/workforce/payroll/slips/{id}`) querying Frappe HR salary slips.
  * Added dedicated "💰 Compensation & Payroll" portal tab in the Employee Portal (`web/employee-portal`) with YTD summary KPI metrics, historical salary slips table, and itemized direct deposit pay stub modal breakdown.
  * Integrated simulation persona actions (`alice_process_payroll_entry`, `bob_view_paystubs`) in `simulation/actions.go` verifying end-to-end payroll workflows.

#### 7.3.3 End-to-End Automated Employee Lifecycle Sagas (Completed)
* **Comprehensive Onboarding Saga:**
  * Implemented dedicated COTS REST API drivers for **Forgejo** (`pkg/cots/forgejo`), **Nextcloud** (`pkg/cots/nextcloud`), and **Snipe-IT** (`pkg/cots/snipeit`).
  * Expanded [`OnboardingSaga`](file:///home/ramjac/orange-cat-investments/internal/saga/saga.go) and domain adapters (`internal/saga/adapters.go`) triggered by Frappe HR new hire webhooks (`/api/v1/webhooks/frappe-hr`) or direct API (`POST /api/v1/workforce/employees/onboard`):
    * Automated VCS developer account creation and SSH public key registration in **Forgejo**.
    * Provisioned storage workspaces, departmental quotas, and group folders (Engineering, Operations, Veterinary) in **Nextcloud**.
    * Issued hardware allocations (MacBook Pro, Ergonomic Workstation, Encrypted RFID Badge, Pebble watch) in **Snipe-IT**.
    * Initialized care schedules, feeding rosters, and primary caretaker assignment in `workforce.care_schedules`.
* **Complete Separation & Offboarding Engine:**
  * Operationalized [`OffboardingEngine`](file:///home/ramjac/orange-cat-investments/internal/saga/saga.go) triggered via webhook or lifecycle API (`POST /api/v1/workforce/employees/{id}/offboard`):
    * Invalidates active Valkey session tokens and revokes OAuth2 refresh tokens.
    * Revokes Forgejo developer account access.
    * Locks account and archives personal Nextcloud storage to compliance vault (`/compliance/vault/archives/`).
    * Generates Snipe-IT equipment check-in tickets for asset recovery.
    * Seamlessly transitions feline habitat care protocols and emergency medical holds to senior veterinary staff (Dr. Elena Rostova).
    * Logs departure status in Frappe HR / ERPNext and emits Watermill domain events (`events.workforce.offboarding_completed.v1`).
* **Employee Portal Integration & Simulation Coverage:**
  * Added "👥 IT Onboarding & Lifecycle Sagas" tab to `web/employee-portal` featuring workforce lifecycle directory, single-click "Onboard New Hire" modal, and automated deprovisioning dialog.
  * Added simulation action `alice_offboard_employee` in `simulation/actions.go` verifying automated deprovisioning in end-to-end simulations (55/55 actions passing).

#### 7.3.4 Internal Documents, Knowledge Base & Team Collaboration (Completed)
* **Nextcloud Departmental Hub:**
  * Configured Nextcloud group folders structured by business departments: HR & Policies (50 GB), Facilities & Maintenance SOPs (100 GB), Veterinary Protocols (50 GB), and Executive Reporting & Finance (25 GB) via [`pkg/cots/nextcloud`](file:///home/ramjac/orange-cat-investments/pkg/cots/nextcloud/client.go) client.
  * Established company-wide shared operations calendars for Shift Rotations, Habitat Maintenance Windows, Corporate Milestones, and Veterinary Health Audits.
  * Preserved strict COTS isolation principles: Nextcloud remains the sole system of record for group folders, documents, and calendars with generic domain-centric naming (`WorkspaceClient`, `workspaceURL`).
* **Intranet Document Embedding & Portal UI:**
  * Integrated WebDAV path links and REST APIs into [`web/employee-portal`](file:///home/ramjac/orange-cat-investments/web/employee-portal/src/App.vue) under the "📁 Knowledge Base & Intranet" tab.
  * Built an embedded markdown document reader allowing staff to inspect SOPs, handbooks, and policies directly without switching tabs.
  * Built interactive modals to publish intranet documents with department tags, WebDAV mounting paths, and schedule shared operations calendar events.
  * Full OpenAPI 3.1 spec documentation for `/workforce/hub/folders`, `/workforce/documents`, `/workforce/documents/{id}`, and `/workforce/calendar/events`.
* **Simulation Coverage:**
  * Added `bob_review_maintenance_calendar` action in [`simulation/actions.go`](file:///home/ramjac/orange-cat-investments/simulation/actions.go) verifying shared operations calendar retrieval, bringing full platform simulation suite to 56/56 actions passing.

#### 7.3.5 Performance Reviews & Health Assessment Cycles (Completed)
* **Structured Multi-Phase Review Cycles:**
  * Expanded `workforce.review_cycles` in PostgreSQL (`scripts/init.sql`, `k8s/oci-core/postgres-core-configmap.yaml`, and `db/query/workforce.sql`) with period tracking (`review_period`), lifecycle phase (`phase`: `self_appraisal` -> `peer_feedback` -> `manager_review` -> `completed`), employee self-appraisals with score, peer feedback with peer score, manager assessment with KPI ratings and overall score.
  * Implemented dedicated lifecycle endpoints with route aliases across API server and Employee BFF:
    * `POST /api/v1/workforce/review-cycles/{id}/self-appraisal`
    * `POST /api/v1/workforce/review-cycles/{id}/peer-feedback`
    * `POST /api/v1/workforce/review-cycles/{id}/manager-assessment`
    * `POST /api/v1/workforce/review-cycles/{id}/vet-audit`
  * Complete OpenAPI 3.1 specification schemas and operations in [`api/openapi.yaml`](file:///home/ramjac/orange-cat-investments/api/openapi.yaml).
* **Integrated Veterinary Health Assessments:**
  * Coupled corporate staff appraisals with periodic feline clinical wellness audits conducted by Dr. Elena Rostova (`emp-human-elena`), capturing clinical wellness score (`wellness_audit_score`), habitat optimization recommendations (`habitat_recommendations`), clinical auditor ID, and linked VHR record (`vhr_record_id`).
* **Employee Web Portal UI (`web/employee-portal`):**
  * Enhanced the "📋 Review Cycles & Assessments" tab with an interactive 4-step lifecycle stepper (`1. Self-Appraisal` -> `2. Peer Feedback` -> `3. Manager Review` -> `4. Completed`), review period tags, and phase filtering buttons.
  * Built a multi-tab review & audit modal providing tabbed interfaces for employee self-reflections, peer feedback, manager grading with KPI ratings, and veterinary clinical wellness audits.
* **Simulation Coverage:**
  * Added `elena_conduct_wellness_audit` in [`simulation/actions.go`](file:///home/ramjac/orange-cat-investments/simulation/actions.go) under Dr. Elena Rostova and added `TestDriverSinglePersonaElena` in [`simulation/driver_test.go`](file:///home/ramjac/orange-cat-investments/simulation/driver_test.go), bringing platform simulation coverage to 57/57 actions passing.

#### 7.3.6 Helpdesk & Issue Management (Internal & Customer) (Completed)
* **Employee IT Helpdesk (Forgejo Issue Bridge):**
  * Added IT support intake and management interface in [`web/employee-portal`](file:///home/ramjac/orange-cat-investments/web/employee-portal/src/App.vue) under the "🛠️ IT Helpdesk & Issue Tracker" tab, supporting hardware, software, and habitat issues with category, status, and severity filtering.
  * Synchronized tickets with Forgejo repository issues (`oci-internal/helpdesk`) via [`pkg/cots/forgejo`](file:///home/ramjac/orange-cat-investments/pkg/cots/forgejo/client.go) client, tracking status, comments, and severity.
  * Automatic urgent priority routing: Tickets flagged as urgent (or high/critical severity) automatically dispatch on-call alerts to Pebble watch wearers (`/ops/pebble/alerts`) and mobile push notifications via `ntfy` (`internal/service/ops`).
* **Customer Bug & Feedback Intake:**
  * Implemented authenticated customer support intake and conversation history in [`web/customer-portal`](file:///home/ramjac/orange-cat-investments/web/customer-portal/src/App.vue) under the "🎧 Support & Inquiries" tab.
  * Ingested tickets into isolated Forgejo customer-issues tracker (`oci-internal/customer-support`), scrubbing internal comments (`is_internal = false`) so diagnostic engineering notes are never leaked to retail investors.
  * Added session authentication and double-submit CSRF enforcement across `/api/v1/customer/support/tickets` in [`cmd/customer-bff`](file:///home/ramjac/orange-cat-investments/cmd/customer-bff/main.go).
* **Database & OpenAPI Governance:**
  * Expanded `ops.it_tickets` schema with `severity`, `category`, `source`, `customer_id`, `assignee_username`, `resolution_notes`, and `is_urgent` fields (`scripts/03-ops-core.sql`, `scripts/init.sql`).
  * Added `ops.ticket_comments` table with `is_internal` boolean flag for comment scrubbing.
  * Complete OpenAPI 3.1 specifications documented in [`api/openapi.yaml`](file:///home/ramjac/orange-cat-investments/api/openapi.yaml).
* **Simulation Coverage:**
  * Added `elise_submit_habitat_helpdesk_ticket` (urgent habitat ticket creation) and `arthur_submit_customer_support_ticket` (customer ticket submission and sanitized comment retrieval) in [`simulation/actions.go`](file:///home/ramjac/orange-cat-investments/simulation/actions.go), bringing platform simulation coverage to 59/59 actions passing.

#### 7.3.7 Internal Business Intelligence & Automated Reporting (Completed)
* **Automated Management Report Generator:**
  * Implemented scheduled reporting worker in `cmd/worker` (`startExecutiveReportingWorker` and Watermill event handler `executive_reporting_trigger_handler`) aggregating metrics across ERPNext / Frappe HR (P&L revenue, operational expenses, net profit, operational burn, cash reserves, runway months) and `workforce` (total headcount, human/feline breakdown, attendance rates, overtime hours, approved timesheets).
  * Built high-performance report generator engine in [`pkg/reporting`](file:///home/ramjac/orange-cat-investments/pkg/reporting) producing professional monthly executive management summary PDF documents and CSV spreadsheet exports.
  * Supported scheduled monthly generation on the 1st of each month as well as on-demand generation via `POST /api/v1/workforce/reports/monthly/generate`.
* **Compliance Archival in Nextcloud:**
  * Automatically deposits generated executive report artifacts into Nextcloud's restricted `Executive/Reports/YYYY-MM/` directory via WebDAV ([`pkg/cots/nextcloud`](file:///home/ramjac/orange-cat-investments/pkg/cots/nextcloud/client.go)), guaranteeing immutable historical preservation and regulatory audit compliance.
  * Emits Watermill domain events (`events.reporting.report_generated.v1`, `events.reporting.report_archived.v1`) on RabbitMQ / internal pubsub channels.
  * Backed by PostgreSQL schema `workforce.executive_reports` (`scripts/01-workforce-core.sql`, `scripts/init.sql`).
* **Employee Web Portal UI (`web/employee-portal`):**
  * Added dedicated "📊 Executive Reports & BI" tab with executive KPI scorecards (Net Profit, Total Revenue, Operational Burn, Cash Reserves & Runway, Workforce Headcount, Attendance Rate).
  * Implemented historical executive reports table displaying report status, period, P&L overview, and Nextcloud WebDAV vault paths.
  * Built interactive "Generate Monthly Report" modal and "Inspect Report Details" modal with itemized dual-column view for ERPNext financial ledgers and workforce metrics.
  * Single-click binary downloads for PDF executive reports and CSV data exports directly in-browser.
* **OpenAPI 3.1 & Simulation Coverage:**
  * Full OpenAPI 3.1 specification for `/workforce/reports/monthly`, `/workforce/reports/monthly/generate`, `/workforce/reports/monthly/{id}`, and `/workforce/reports/monthly/{id}/download` in [`api/openapi.yaml`](file:///home/ramjac/orange-cat-investments/api/openapi.yaml).
  * Added simulation action `alice_generate_executive_report` in [`simulation/actions.go`](file:///home/ramjac/orange-cat-investments/simulation/actions.go) under Alice Vance (`emp-human-alice`), bringing platform simulation coverage to 60/60 actions passing.

### 7.4 Customer Lifecycle, Account Management & GDPR Compliance *(Completed)*
Transitioned customer interactions from mock placeholders to an authentic, legally compliant lifecycle:
* **Customer Self-Registration & Onboarding:**
  * Implemented self-service signup via ZITADEL OIDC, enforcing NIST SP 800-63B password complexity standards (`ValidatePasswordComplexity`).
  * Dynamic bank account linking via Plaid / ACH mock interface (`external-sim/mock-bank`), generating an initial portfolio account in `core_invest.investment_portfolios` with zero starting balance and optional initial cash deposit.
* **Account Closure & Liquidation Workflow:**
  * Self-service "Close Account" workflow in [`cmd/customer-bff`](file:///home/ramjac/orange-cat-investments/cmd/customer-bff/main.go) and [`web/customer-portal`](file:///home/ramjac/orange-cat-investments/web/customer-portal/src/App.vue): automatically liquidates all active equity holdings via broker market sell orders, simulates settlement proceeds, transfers net cash balances back to the customer's linked external bank account via Plaid ACH disbursement, and marks the portfolio as `closed`.
* **GDPR "Right to be Forgotten" Data Deletion Pipeline (Article 17):**
  * Automated GDPR data erasure engine:
    * Scrubs all personally identifiable information (name, email, password hash, contact references) across PostgreSQL database tables (`core_invest.customers`, `core_invest.investment_portfolios`, and IT tickets).
    * Invalidates and permanently purges active sessions and refresh cookies from Valkey (`PurgeCustomerSessions`).
    * Anonymizes transaction and trade ledgers with SHA-256 hashes to retain audit integrity under SEC Rule 17a-4 while decoupling former customer identities.
    * Issues a tamper-evident cryptographic data deletion certificate (`core_invest.gdpr_erasure_certificates`) signed with HMAC-SHA256, accessible via dedicated authenticated and public receipt endpoints (`GET /api/v1/customer/privacy/gdpr-certificate/{id}` and `/download`).
* **Automated Investment Statements & Tax Documentation:**
  * Built a statement generation pipeline producing monthly portfolio summaries and Form 1099-B tax reports rendered directly to pure Go PDF 1.4 format without external binary dependencies ([`pkg/reporting/customer_statement_pdf.go`](file:///home/ramjac/orange-cat-investments/pkg/reporting/customer_statement_pdf.go)).
  * Stores statement records and SHA-256 content hashes in `core_invest.investment_statements`.
  * Delivered securely within the Customer Portal with session-authorized preview and download endpoints (`GET /api/v1/customer/statements/{id}/download`).
* **Customer Portal UI, OpenAPI 3.1 & Simulation Coverage:**
  * Added dedicated `📄 Statements & Tax Forms` and `🔒 Account & GDPR Privacy` navigation tabs in [`web/customer-portal/src/App.vue`](file:///home/ramjac/orange-cat-investments/web/customer-portal/src/App.vue), complete with live NIST password strength calculation and confirmation modals.
  * Full OpenAPI 3.1 specification for `/customer/auth/register`, `/customer/account/close`, `/customer/privacy/gdpr-erasure`, `/customer/privacy/gdpr-certificate/{id}`, `/customer/privacy/gdpr-certificate/{id}/download`, `/customer/statements`, `/customer/statements/generate`, and `/customer/statements/{id}/download` in [`api/openapi.yaml`](file:///home/ramjac/orange-cat-investments/api/openapi.yaml).
  * Added 5 simulation actions in [`simulation/actions.go`](file:///home/ramjac/orange-cat-investments/simulation/actions.go) covering self-registration, monthly statements, Form 1099-B tax filing, liquidation, and GDPR erasure, bringing platform simulation coverage to 65/65 actions passing (100% success rate).

### 7.5 Multi-Node SMB High Availability & Resilient Home Server Topology *(Completed)*
Evolved the single-node deployment into a resilient multi-node home/office lab topology designed for 2 to 3 physical computers (e.g., mini PCs or Intel NUCs):
* **3-Node K3s HA Cluster Blueprint & Ansible Automation:**
  * Documented architectural decisions and quorum math in [`docs/adr/0002-multi-node-ha-topology.md`](file:///home/ramjac/orange-cat-investments/docs/adr/0002-multi-node-ha-topology.md) and full operational procedures in [`docs/multi-node-ha-setup-guide.md`](file:///home/ramjac/orange-cat-investments/docs/multi-node-ha-setup-guide.md).
  * Automated 3-node K3s cluster provisioning with embedded etcd high-availability (`--cluster-init`) and ARP-based virtual IP failover using `kube-vip` across [`ansible/playbooks/k3s_ha_cluster.yml`](file:///home/ramjac/orange-cat-investments/ansible/playbooks/k3s_ha_cluster.yml), [`ansible/roles/k3s_ha/`](file:///home/ramjac/orange-cat-investments/ansible/roles/k3s_ha/), and [`ansible/inventory/ha-cluster.ini.example`](file:///home/ramjac/orange-cat-investments/ansible/inventory/ha-cluster.ini.example), tolerating single-node hardware failures without control plane downtime.
  * Updated workload deployment automation in [`ansible/roles/k8s_deploy/tasks/main.yml`](file:///home/ramjac/orange-cat-investments/ansible/roles/k8s_deploy/tasks/main.yml) and [`ansible/site-ha.yml`](file:///home/ramjac/orange-cat-investments/ansible/site-ha.yml) to deploy modern namespaces (`oci-core`, `oci-internal`, `external-sim`, `monitoring`, `logging`).
* **Distributed Replicated Storage (Longhorn):**
  * Implemented distributed block storage engine manifests and automated setup script in [`k8s/ha/storage/deploy-longhorn.sh`](file:///home/ramjac/orange-cat-investments/k8s/ha/storage/deploy-longhorn.sh) and [`k8s/ha/storage/longhorn-storageclasses.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/storage/longhorn-storageclasses.yaml).
  * Defined multi-node persistent volume claims in [`k8s/ha/storage/ha-pvc-overrides.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/storage/ha-pvc-overrides.yaml) with 3-node block replication across physical NVMe disks, enabling automatic cross-node pod rescheduling and volume reattachment for `postgres-core-data-pvc`, `postgres-internal-data-pvc`, and `nextcloud-data-pvc`.
  * Configured automated recurring snapshots (every 4 hours) and daily backups in [`k8s/ha/storage/longhorn-backup-recurring.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/storage/longhorn-backup-recurring.yaml), plus Longhorn UI LAN ingress in [`k8s/ha/storage/longhorn-ui-ingress.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/storage/longhorn-ui-ingress.yaml).
* **Local Network (LAN) Employee Portal Access & Intranet PKI:**
  * Provided high-availability virtual IP configurations utilizing `kube-vip` ([`k8s/ha/lan/kube-vip-service-daemonset.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/lan/kube-vip-service-daemonset.yaml)) and MetalLB L2 pools ([`k8s/ha/lan/metallb.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/lan/metallb.yaml)) for cluster VIP `192.168.1.250`.
  * Configured intranet split-horizon DNS mappings for Pi-hole ([`k8s/ha/lan/dns/pihole-custom.list`](file:///home/ramjac/orange-cat-investments/k8s/ha/lan/dns/pihole-custom.list)), dnsmasq ([`k8s/ha/lan/dns/dnsmasq-oci-lan.conf`](file:///home/ramjac/orange-cat-investments/k8s/ha/lan/dns/dnsmasq-oci-lan.conf)), and router firewalls ([`k8s/ha/lan/dns/dns-setup-guide.md`](file:///home/ramjac/orange-cat-investments/k8s/ha/lan/dns/dns-setup-guide.md)) resolving `*.oci.local` to the VIP.
  * Established internal PKI with root CA generator ([`k8s/ha/lan/tls/generate-lan-root-ca.sh`](file:///home/ramjac/orange-cat-investments/k8s/ha/lan/tls/generate-lan-root-ca.sh)), cert-manager `ClusterIssuer` ([`k8s/ha/lan/tls/intranet-cluster-issuer.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/lan/tls/intranet-cluster-issuer.yaml)), and HA intranet TLS ingress ([`k8s/ha/lan/tls/intranet-ingress-ha.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/lan/tls/intranet-ingress-ha.yaml)), accompanied by client trust store installation instructions ([`k8s/ha/lan/tls/client-ca-trust-installation.md`](file:///home/ramjac/orange-cat-investments/k8s/ha/lan/tls/client-ca-trust-installation.md)) to eliminate all browser security warnings.
* **Secure Public Ingress for Customer Portal:**
  * **Recommended Method (Cloudflare Tunnel):** Built a multi-replica HA `cloudflared` deployment with node anti-affinity in [`k8s/ha/public-ingress/cloudflare/cloudflare-tunnel.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/public-ingress/cloudflare/cloudflare-tunnel.yaml), secret template in [`k8s/ha/public-ingress/cloudflare/cloudflare-tunnel-secret.template.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/public-ingress/cloudflare/cloudflare-tunnel-secret.template.yaml), and setup guide in [`k8s/ha/public-ingress/cloudflare/setup-cloudflare-tunnel.md`](file:///home/ramjac/orange-cat-investments/k8s/ha/public-ingress/cloudflare/setup-cloudflare-tunnel.md) delivering zero-open-port inbound ingress with automatic DDoS protection, edge TLS, and CGNAT immunity.
  * **Alternative Method (NGINX + Let's Encrypt + DDNS):** Provided cert-manager ACME HTTP-01/DNS-01 `ClusterIssuer` in [`k8s/ha/public-ingress/nginx-letsencrypt/letsencrypt-issuer.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/public-ingress/nginx-letsencrypt/letsencrypt-issuer.yaml), hardened public NGINX Ingress with rate limiting in [`k8s/ha/public-ingress/nginx-letsencrypt/public-customer-ingress.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/public-ingress/nginx-letsencrypt/public-customer-ingress.yaml), automated DDNS synchronization CronJob in [`k8s/ha/public-ingress/nginx-letsencrypt/ddns-updater.yaml`](file:///home/ramjac/orange-cat-investments/k8s/ha/public-ingress/nginx-letsencrypt/ddns-updater.yaml), and router port-forwarding guide in [`k8s/ha/public-ingress/nginx-letsencrypt/setup-nginx-letsencrypt.md`](file:///home/ramjac/orange-cat-investments/k8s/ha/public-ingress/nginx-letsencrypt/setup-nginx-letsencrypt.md).

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
    4. Code merged into `main` automatically deploys to the isolated `oci-dev` environment in Kubernetes.
    5. Automated integration tests and platform simulation runs execute against `oci-dev`.
    6. Only upon 100% test pass are deployment manifests in `k8s/oci-core/` updated to roll out the verified release to production (detailed in Section 7.9).

### 7.7 Production Security: Real Identity, Fine-Grained RBAC & Zero Trust (Completed)
Replaced mock authentication mechanisms with production-grade security boundaries, session management, and fine-grained role-based access control:
* **Authentic Customer Security (ZITADEL OIDC):**
  * Implemented secure customer session management in [`cmd/customer-bff`](file:///home/ramjac/orange-cat-investments/cmd/customer-bff/main.go) with `__Host-customer-session` cookies (`HttpOnly`, `SameSite=Lax`), double-submit CSRF protection (`csrfProtect`), and strict tenant database query scoping (`WHERE customer_id = $1`).
* **Fine-Grained Role-Based Access Control (RBAC):**
  * Built [`cmd/employee-bff/rbac.go`](file:///home/ramjac/orange-cat-investments/cmd/employee-bff/rbac.go) and [`cmd/employee-bff/auth.go`](file:///home/ramjac/orange-cat-investments/cmd/employee-bff/auth.go) enforcing role-based permission checks across Employee BFF endpoints:
    * **Human Resources (`hr`):** Authorized for payroll slips, timesheet approvals, onboarding/offboarding, and review cycle configurations.
    * **Facilities & Edge Telemetry (`facilities`):** Authorized for camera assets, PTZ controls, firmware OTA release pipelines, and maintenance sync endpoints.
    * **Veterinary Staff (`veterinary`):** Exclusive authority to toggle Emergency Medical Trading Holds, prescribe care schedules, and sign health assessments.
    * **Investment Team (`investment`):** Authorized for portfolio analytics, backtesting workbench, and quantitative strategy parameters.
    * **Infrastructure Ops (`ops`):** Authorized for cluster metrics, IT helpdesk dispatch, and cert-manager CA rotation workflows.
    * **Administrator (`admin`):** Universal administrative oversight.
  * Enforced double-submit CSRF protection (`CSRFProtectMiddleware`) and returning HTTP 403 Forbidden on unauthorized cross-role requests and HTTP 401 Unauthorized for missing/expired sessions.
* **Zero Trust Security Architecture:**
  * Enforced dynamic mTLS certificate reloader in Go standard library `crypto/tls` (`internal/service/pki`).
  * Structured Kubernetes `NetworkPolicy` resources to prevent cross-namespace network bridging.

### 7.8 Isolated Development Environment (`oci-dev`) for Core Applications (Completed)
Provisioned a dedicated, isolated development namespace (`oci-dev`) in [`k8s/namespaces/namespaces.yaml`](file:///home/ramjac/orange-cat-investments/k8s/namespaces/namespaces.yaml) and deployment manifests in [`k8s/oci-dev/`](file:///home/ramjac/orange-cat-investments/k8s/oci-dev/) mirroring the `oci-core` custom application stack for realistic staging and automated integration testing:
* **Collocated Development Workloads:**
  * Configured development versions of all core applications under `k8s/oci-dev/`: `oci-server-dev`, `oci-customer-bff-dev`, `oci-employee-bff-dev`, `oci-pebble-proxy-dev`, `oci-worker-dev`, `oci-customer-portal-dev`, and `oci-employee-portal-dev` tagged `:dev-latest`.
  * Enabled development configuration toggles (`ENABLE_DEV_MOCK_AUTH=true`, `SEED_DEV_SESSIONS=true`) and debug logging.
* **Runtime Data & Service Isolation:**
  * Partitioned development state from production `oci-core`:
    * Dedicated Valkey instance (`valkey-dev` in `k8s/oci-dev/valkey.yaml`) for dev sessions.
    * Dedicated development database (`postgres-dev` StatefulSet in `k8s/oci-dev/postgres-dev.yaml`) for testing and synthetic seed personas.
    * Isolated RabbitMQ virtual host (`/oci-dev`) ensuring background event processing is partitioned.
* **Network & Ingress Routing:**
  * Deployed [`k8s/oci-dev/oci-ingress.yaml`](file:///home/ramjac/orange-cat-investments/k8s/oci-dev/oci-ingress.yaml) routing dedicated internal development subdomains (`dev.api.oci.local`, `dev.customer.oci.local`, `dev.employee.oci.local`, `dev.customer-bff.oci.local`, `dev.employee-bff.oci.local`, `dev.pebble-proxy.oci.local`).

### 7.9 Automated Continuous Deployment, Gated Promotion & Integration Testing Pipeline (Completed)
Established an automated continuous integration, development staging, and gated promotion pipeline in [`.forgejo/workflows/ci.yaml`](file:///home/ramjac/orange-cat-investments/.forgejo/workflows/ci.yaml) that gates production releases through the `oci-dev` environment:
* **Automated Merge Deployment to `oci-dev`:**
  * Merges into `main` trigger Forgejo Actions to build and tag container images with `:dev-latest` and unique commit SHA `sha-<commit>`.
  * Manifests in [`k8s/oci-dev/`](file:///home/ramjac/orange-cat-investments/k8s/oci-dev/) are applied automatically with rolling updates and readiness probe checks.
* **Automated End-to-End Integration Testing Gate:**
  * Executes the full 12-persona platform simulation driver against the live staging cluster (`go run ./simulation/cmd/simulator -mode live -url https://dev.api.oci.local`).
  * Runs contract and integration suites across customer and employee journeys.
* **Strict Production Promotion Gate (`oci-dev` $\rightarrow$ `oci-core`):**
  * **Pass Condition:** Only upon 100% test pass without error are images promoted to `:prod-stable` and manifests in `k8s/oci-core/` rolled out.
  * **Failure Condition:** Any failure immediately aborts promotion, preventing broken builds from reaching production `oci-core`.

### 7.10 Comprehensive API Versioning & Lifecycle Governance (Completed)
Implemented formal API versioning, negotiation, and contract governance across all public and internal interfaces to ensure forward and backward compatibility as the platform evolves:
* **Standardized URI & Header Versioning:**
  * Enforced uniform `/api/v1/` route prefixes across all domain APIs (`facilities`, `workforce`, `core_invest`, `ops`) and BFF entrypoints (`cmd/customer-bff`, `cmd/employee-bff`).
  * Implemented [`pkg/versioning`](file:///home/ramjac/orange-cat-investments/pkg/versioning/versioning.go) middleware supporting content negotiation via `Accept: application/vnd.oci.v1+json` and `X-API-Version: v1` headers, returning HTTP 406 Not Acceptable on unsupported version requests, and setting `X-API-Version: v1` with `Vary: Accept, X-API-Version`.
  * Integrated versioning middleware across `cmd/server`, `cmd/customer-bff`, `cmd/employee-bff`, and `cmd/pebble-proxy`.
* **OpenAPI 3.1 Contract Linting & Automated Drift Detection:**
  * Authoritative OpenAPI 3.1 single source of truth in [`api/openapi.yaml`](file:///home/ramjac/orange-cat-investments/api/openapi.yaml).
  * Implemented automated schema validator script [`scripts/validate-openapi.py`](file:///home/ramjac/orange-cat-investments/scripts/validate-openapi.py) verifying OpenAPI 3.1.0 specifications, server base URL version prefixes, response definitions, and absence of decommissioned hardware models.
* **Deprecation & Sunset Policies (RFC 8594):**
  * Built [`versioning.Deprecate`](file:///home/ramjac/orange-cat-investments/pkg/versioning/versioning.go#L104-L113) helper to emit standardized RFC 8594 response headers: `Deprecation: true`, `Sunset: <HTTP-date>`, and `Link: </api/v2/...>; rel="successor-version"`.
* **Edge & Embedded Client Compatibility:**
  * Maintained multi-version protocol compatibility for field and wearable clients:
    * Backward-compatible SQLite synchronization contracts for the Flutter mobile field app (`mobile/flutter_app`).
    * Protocol buffers and payload versioning for the Pebble watch gateway proxy (`cmd/pebble-proxy`).
    * Stable webhook and REST contracts for customer programmatic trading bots.


