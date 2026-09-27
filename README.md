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

1. **System of Record for Generic Operations:** ERPNext and Frappe HR are the single source of truth for double-entry bookkeeping, tax accounting, legal payroll processing, standard customer billing, and core HR workflows. Homebox is the source of record for non-networked office physical inventory.
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
* **Office Asset Tracking:** [Homebox](https://homebox.software/en/) for non-networked office inventory.
* **Databases:** PostgreSQL 16 (Relational DB with isolated schemas `workforce`, `facilities`, `core_invest`, `ops`).
* **Caching & Session Storage:** Valkey (Redis-compatible).
* **Event Streaming & Queuing:** RabbitMQ with Watermill pub/sub router engine.
* **Search Engine:** Meilisearch.
* **Metrics & Monitoring:** Prometheus + Grafana.
* **Log Aggregation:** Loki + Promtail.
* **Frontend Applications:** Vue 3 (Employee Portal), Hugo (Static Marketing Site).
* **Mobile Applications:** Flutter SDK (Field Maintenance App with SQLite offline sync).
* **Wearables & Embedded:** Pebble Alloy Embedded JS Companion App (`embedded/pebble`) and Go Gateway Proxy (`cmd/pebble-proxy`).
* **Backend Services & Tooling:** Go 1.22+.

---

## 4. Employee & Customer Personas (Workflow Simulation)

To exercise and simulate the entire software stack using autonomous drivers and AI agents, OCI defines 10 distinct personas across corporate staff, feline executives, and retail investors:

### Corporate Staff & Feline Executives
1. **Garfield (`emp-feline-garfield`)** — Chief Observation Officer (COO)
2. **Barneby (`emp-feline-barneby`)** — Senior Alpha Perch Analyst
3. **Alice Vance (`emp-human-alice`)** — Head of Human & Feline Resources
4. **Bob Builder (`emp-human-bob`)** — Lead Facilities & Edge Telemetry Engineer
5. **Carol Danvers (`emp-human-carol`)** — Chief Investment Officer (CIO)
6. **David Quant (`emp-human-david`)** — Feline Behavioral Data Scientist
7. **Dr. Elena Rostova (`emp-human-elena`)** — Chief Veterinary Officer & Habitat Specialist
8. **Frank Operations (`emp-human-frank`)** — Platform Security & K8s Infrastructure Lead

### Retail Investor Customers
9. **Arthur Pendelton (`cust-longterm-arthur`)** — Long-Term Value Investor (passive regular fund deposits, quarterly performance reviews)
10. **Chloe Spark (`cust-active-chloe`)** — Short-Term Momentum Alpha Trader (active tactical buy/sell orders on feline zoomies activity spikes)

Detailed profile specifications are located in the `personas/` directory.

---

## 5. Implemented Capabilities & Completed Milestones

All core functional domains, distributed sagas, client surfaces, and background pipelines have been implemented and verified:

### A. Core Investment Engine & Observation Telemetry (`core_invest`)
* **Real-time WebRTC/RTSP Stream Ingestion:** Registered via `/core-invest/streams` and `/api/v1/core-invest/streams`.
* **Automated Brokerage Execution Integrations:** Automated order placement via `/core-invest/brokerage/orders`.
* **Dynamic Backtesting Workbench:** Running historical backtests via `/core-invest/backtesting/runs`.

### B. Facilities & Mobile Field Maintenance (`facilities` & `mobile/flutter_app`)
* **Flutter Mobile Offline Database Sync:** Full local SQLite persistence (`sqflite`), QR scanner camera view, and offline queue sync engine (`SyncEngine`) posting batched logs to `/facilities/sync/maintenance-logs`.
* **Hardware Asset Management:** Keyset-paginated hardware inventory (`/facilities/assets`) supporting high-throughput edge camera fleets.
* **Edge Device Firmware OTA Pipeline:** Firmware release catalog (`/facilities/firmware`) and OTA job scheduling (`/facilities/firmware/ota-jobs`), executed by the background worker Watermill router (`ota_pipeline_handler`).

### C. Workforce Operations & Automated Sagas (`workforce` & `web/employee-portal`)
* **Frappe HR Webhook Ingestion:** Webhook endpoint `/api/v1/webhooks/frappe-hr` triggering distributed `OnboardingSaga` and `OffboardingEngine` flows.
* **Interactive Care Schedule UI:** Vue 3 Employee Portal interface for Dr. Elena Rostova to dynamically edit feline dietary protocols, feeding times, and toggle Emergency Medical Trading Holds (`/api/v1/workforce/care-schedules/{felineId}/medical-hold`).

### D. Operations & Wearable Alerts (`ops` & `cmd/pebble-proxy`)
* **Pebble Companion App Gateway Proxy:** Dedicated proxy server (`cmd/pebble-proxy`) translating active IT tickets into Pebble watch AppMessage dictionary payloads (`/ops/pebble/alerts`) and receiving ACK button clicks (`/ops/pebble/ack`).
* **Automated Root/Intermediate CA Key Rotation:** Background ticker worker (`cmd/worker` + `internal/service/pki`) continuously evaluating certificate expiration against a 30-day threshold and automating certificate re-issuance via `cert-manager`.

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
       │ Background Worker       │ (Event subscriber daemon)         │
       │ Vue 3 Employee Portal   │ http://localhost:5173             │
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

#### Step 5: Launch the Employee Web Portal

In a new terminal, launch the Vue 3 single-page application:
```bash
cd web/employee-portal
npm install
npm run dev
```
Open your browser to **`http://localhost:5173`**. You will be greeted by the Employee Portal where you can:
* Switch between feline executives (Garfield, Barneby).
* View and edit dietary protocols and feeding time slots.
* Toggle the **Emergency Medical Trading Hold** button (pauses algorithmic trading triggers).

#### Step 6: Launch the Flutter Mobile App (Field Maintenance)

If you have Flutter installed, you can launch the mobile maintenance app on desktop Linux/macOS, Android emulator, or connected phone:
```bash
cd mobile/flutter_app
flutter pub get
flutter run -d linux   # or -d macos / -d chrome / -d <device-id>
```
Technicians can scan simulated QR codes (`QR-CAM-ORANGE-01`), record offline maintenance actions, and trigger the offline synchronization engine.

---

### 6.4 Kubernetes Cluster Deployment (MicroK8s, K3s, Minikube, kind)

If you have an existing Kubernetes cluster or local node (such as MicroK8s, Minikube, kind, or an existing K3s installation) configured in your local `~/.kube/config`:

```bash
chmod +x scripts/deploy-k8s.sh
./scripts/deploy-k8s.sh
```

This single command:
* Checks `kubectl` connectivity to your cluster.
* Applies all namespaces (`oci-core`, `postgres`, `apps`, `monitoring`, `logging`).
* Deploys PostgreSQL 16 with persistent volume claims, table constraints, and initial seed records.
* Deploys Valkey, RabbitMQ, Forgejo, ZITADEL, Nextcloud, Homebox, ERPNext, and Frappe HR.
* Deploys Prometheus, Grafana, Loki, and Promtail monitoring stacks.
* Waits for core database and queuing workloads to become healthy.

To connect your local workstation to the cluster services for testing:
```bash
# Forward PostgreSQL, Valkey, and RabbitMQ
kubectl port-forward -n postgres svc/postgres-service 5432:5432 &
kubectl port-forward -n apps svc/valkey-service 6379:6379 &
kubectl port-forward -n apps svc/rabbitmq-service 5672:5672 15672:15672 &
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

The repository includes a comprehensive simulation driver (`simulation/cmd/simulator`) that exercises all 10 OCI personas (Garfield, Barneby, Alice, Bob, Carol, David, Dr. Elena, Frank, Arthur, Chloe) through their full daily operational routines:

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
* Safely deletes all OCI domain applications, databases, logging, monitoring, and namespaces (`apps`, `logging`, `monitoring`, `oci-core`, `postgres`).
* Deletes and releases OCI persistent volume claims (`postgres-data-pvc`).
* **Preserves System Infrastructure:** Keeps all cluster system services, storage classes, ingress controllers (e.g. MicroK8s NGINX ingress), container registries, and `kubectl` control plane connectivity intact.

To also stop and remove standalone Docker containers from Section 6.3:
```bash
docker stop oci-postgres oci-valkey oci-rabbitmq && docker rm oci-postgres oci-valkey oci-rabbitmq
```


---

## 7. Future Work & Persona Feature Roadmap

A comprehensive audit of the OCI platform against the operational needs of all 10 corporate, feline, and customer personas reveals several high-value capabilities planned for future development. While the foundational schemas, backend services, simulation engines, and developer APIs are operational, the following capabilities represent the feature roadmap derived directly from persona workflows:

### 7.1 Customer Experience & Retail Investor Surfaces
* **Customer Web Portal (`web/customer-portal`) & Customer BFF (`cmd/customer-bff`):**
  * *Target Personas:* **Arthur Pendelton** (`cust-longterm-arthur`), **Chloe Spark** (`cust-active-chloe`)
  * *Current State:* Architecture design and BFF specification exist, but the Vue 3 customer SPA and `cmd/customer-bff` service have not yet been instantiated.
  * *Future Scope:* Build the Vue 3 Customer Web Portal with ZITADEL OIDC customer authentication, session management, portfolio valuation charts, historical asset growth, and real-time trade logs.
* **Low-Latency In-Browser WebRTC Habitat Video Player:**
  * *Target Personas:* **Arthur Pendelton**, **Chloe Spark**
  * *Current State:* Stream endpoints (`/core-invest/streams`) catalog RTSP and WebRTC stream URLs, but there is no embedded browser player or active WebRTC media bridge.
  * *Future Scope:* Deploy a lightweight streaming media bridge (e.g., MediaMTX or go2rtc) and embed a sub-500ms WebRTC video player in the Customer Web Portal so investors can watch Garfield and Barneby lounging in their perches in real time.
* **Active Trading Terminal & Real-Time WebSocket Ticker:**
  * *Target Persona:* **Chloe Spark**
  * *Current State:* Brokerage trade orders are submitted via REST API (`POST /core-invest/brokerage/orders`).
  * *Future Scope:* Build a high-velocity execution trading interface with TradingView/Canvas charts, real-time trade fill ticker streamed over WebSockets, and rapid 1-click execution.
* **Live "Zoomie Index" & Behavioral Momentum Push Alerts:**
  * *Target Persona:* **Chloe Spark**
  * *Current State:* Cat observation events are recorded in `core_invest.observation_events`, but there is no real-time telemetry streaming channel to retail clients.
  * *Future Scope:* Implement a real-time Server-Sent Events (SSE) / WebSocket behavioral momentum indicator that scores feline movement velocity and alert states, dispatching instant push notifications when 3 AM zoomies or sudden energy spikes trigger tactical trading opportunities.
* **Automated Recurring Deposits & Bank ACH Integrations:**
  * *Target Persona:* **Arthur Pendelton**
  * *Current State:* Trades are recorded against static cash balances in `core_invest.investment_portfolios`.
  * *Future Scope:* Integrate with banking rails (Stripe ACH, Plaid) and create an automated recurring deposit scheduler allowing passive long-term investors to set up monthly automated dollar-cost averaging into OCI investment funds.
* **Customer Developer API Keys & Webhook Subscriptions:**
  * *Target Persona:* **Chloe Spark**
  * *Current State:* REST endpoints exist for direct API calls, but lack self-service customer API key management.
  * *Future Scope:* Enable self-service generation of scoped HMAC/Bearer API keys in the Customer Portal, enabling retail quant traders to connect custom trading algorithms and receive real-time webhook callbacks on cat observation events.
* **Feline Welfare & ESG Transparency Dashboard:**
  * *Target Persona:* **Arthur Pendelton**
  * *Current State:* Care schedules are managed internally via `workforce.care_schedules` in the Employee Portal.
  * *Future Scope:* Provide a public-facing transparency dashboard showcasing veterinary adherence, nutritional compliance, daily nap hours, and perch comfort ratings to assure retail investors of ethical feline care standards.

### 7.2 Feline Welfare, Biometrics & Habitat IoT
* **Smart Feeder Telemetry & Automated Snack Disbursement:**
  * *Target Personas:* **Garfield** (`emp-feline-garfield`), **Dr. Elena Rostova** (`emp-human-elena`)
  * *Current State:* Database schema supports `feeder` in `facilities.hardware_assets`, but no feeder control protocol is implemented.
  * *Future Scope:* Implement driver integrations for IoT smart feeders, recording actual food consumption weights in real time against `workforce.care_schedules` and allowing automated snack disbursements upon successful observation milestones.
* **Smart Collar Biometrics & Accelerometer Streaming:**
  * *Target Personas:* **Barneby** (`emp-feline-barneby`), **David Quant** (`emp-human-david`), **Dr. Elena Rostova**
  * *Current State:* Hardware inventory registers `smart_collar` serial numbers, but no continuous sensor telemetry stream exists.
  * *Future Scope:* Build a BLE / MQTT telemetry ingestion pipeline for feline smart collars, capturing 3-axis accelerometer data (pounce G-force, jump heights), heart rate, and circadian sleep cycle metrics.
* **Electronic Veterinary Health Records (VHR) & Clinical Charting:**
  * *Target Persona:* **Dr. Elena Rostova**
  * *Current State:* Basic care schedules and medical holds are managed in `workforce.care_schedules`.
  * *Future Scope:* Extend the `workforce` schema with veterinary charting tables for clinical checkups, vaccination logs, prescription schedules, dental health scoring, and historical weight progression graphs.
* **Perch Cushion Comfort & Thermal Sunbeam Sensors:**
  * *Target Personas:* **Garfield**, **Bob Builder** (`emp-human-bob`)
  * *Current State:* Perch assets are recorded in `facilities.observation_perches` with height, max weight, and cushion type metadata.
  * *Future Scope:* Integrate IoT load cells, pressure mats, and temperature sensors on observation perches to monitor cushion wear, perch occupancy duration, and optimal sunbeam thermal tracking.
* **Feline Acoustic / Vocalization Analysis:**
  * *Target Personas:* **Garfield**, **David Quant**
  * *Current State:* Only optical camera streams are ingested.
  * *Future Scope:* Audio stream ingestion and ML acoustic classification to distinguish between contented purring, playful chirping, and urgent demands for feeding.

### 7.3 Quantitative AI, Computer Vision & Brokerage Execution
* **Embedded Real-Time Computer Vision Inference Engine:**
  * *Target Personas:* **David Quant**, **Carol Danvers** (`emp-human-carol`)
  * *Current State:* Stream URLs and metadata are stored in `core_invest.camera_streams`, but observation events are emitted artificially or via the simulation driver.
  * *Future Scope:* Implement an active video frame inference pipeline (e.g., YOLOv8 / MediaPipe / OpenVINO cat-pose detector) processing RTSP streams in real time to automatically publish `events.observation.cat_spotted.v1` events to RabbitMQ.
* **ML Model Registry & Confidence Drift Tracking:**
  * *Target Persona:* **David Quant**
  * *Current State:* Confidence scores are recorded as floats in `core_invest.observation_events`.
  * *Future Scope:* Build a versioned model artifact registry and evaluation harness monitoring classification accuracy and confidence score drift across seasonal lighting variations.
* **Production Brokerage API Gateways (FIX / REST):**
  * *Target Personas:* **Carol Danvers**, **Chloe Spark**
  * *Current State:* Simulated order execution writes directly to `core_invest.brokerage_orders`.
  * *Future Scope:* Replace the simulated order execution stub with production FIX protocol and REST integrations (e.g., Interactive Brokers Client Portal API, Alpaca Markets API) featuring Smart Order Routing (SOR) and liquidity aggregation.
* **Algorithmic Order Types & Pre-Trade Risk Engine:**
  * *Target Personas:* **Carol Danvers**, **Chloe Spark**
  * *Current State:* Brokerage orders support basic `buy` and `sell` actions.
  * *Future Scope:* Support advanced conditional order types (Bracket orders, Trailing Stop-Loss, OCO, IOC, GTC) and an automated pre-trade risk management engine enforcing maximum drawdown limits, position concentration caps, and slippage guardrails.
* **Automated Investor Statements & Tax Document Generation:**
  * *Target Personas:* **Carol Danvers**, **Arthur Pendelton**
  * *Current State:* Orders and snapshots are stored in PostgreSQL.
  * *Future Scope:* Background worker job generating quarterly PDF performance tear sheets, Sharpe/Sortino ratio analytics, and annual Form 1099-B tax documentation.

### 7.4 Workforce Management & Workplace Operations
* **Performance Reviews & Assessment Cycles Management:**
  * *Target Personas:* **Alice Vance** (`emp-human-alice`), **Barneby**, **Dr. Elena Rostova**
  * *Current State:* Table `workforce.review_cycles` exists in the database schema, but has no API handlers or UI views.
  * *Future Scope:* Complete the REST endpoints and Employee Portal UI for `workforce.review_cycles`, enabling managers to schedule, grade, and record joint performance reviews for human staff and monthly feline health assessments.
* **Leave Requests & Catnip Break Workflow UI:**
  * *Target Persona:* **Alice Vance**
  * *Current State:* Table `workforce.leave_requests` exists in the database schema, but lacks API endpoints and frontend forms.
  * *Future Scope:* Complete API handlers and portal workflows for `workforce.leave_requests`, supporting leave submission, manager approval chains, and feline catnip break compliance tracking.
* **Cross-Species Workplace Incident & Conflict Logging:**
  * *Target Persona:* **Alice Vance**
  * *Current State:* Workplace issues are tracked ad-hoc in Forgejo.
  * *Future Scope:* Incident reporting mechanism to record and resolve habitat disputes (e.g., accidental perch disruption, desk coffee spillage, unauthorized treat confiscation).

### 7.5 Facilities Engineering & Platform Infrastructure
* **Remote PTZ & Camera Lens Calibration:**
  * *Target Persona:* **Bob Builder**
  * *Current State:* Hardware assets store metadata and support mobile maintenance logs.
  * *Future Scope:* Remote Pan-Tilt-Zoom (PTZ), digital zoom, and optical focus calibration APIs to adjust edge camera angles without entering the habitat and disturbing feline observation staff.
* **Habitat Environmental Telemetry (HVAC, Lux, Decibels):**
  * *Target Personas:* **Bob Builder**, **Dr. Elena Rostova**
  * *Current State:* Environmental conditions are not recorded in the database.
  * *Future Scope:* Ingestion and alerting pipeline for habitat ambient temperature, relative humidity, light intensity (lux), and noise levels (dB) to ensure optimal animal welfare.
* **Active Push Notification Services (`ntfy` / WebPush):**
  * *Target Persona:* **Frank Operations** (`emp-human-frank`)
  * *Current State:* Pebble watch companion proxy polls `/ops/pebble/alerts`.
  * *Future Scope:* Implement push notification broker integration (`ntfy` / WebPush) so infrastructure alerts and IT tickets immediately notify on-call engineers rather than depending exclusively on client polling.
* **Automated Database Backup & Disaster Recovery (Velero / Barman):**
  * *Target Persona:* **Frank Operations**
  * *Current State:* Database PVCs are provisioned, but automated snapshot schedules are not configured.
  * *Future Scope:* Production backup manifests utilizing Barman/pgBackRest for point-in-time PostgreSQL recovery and Velero for Kubernetes persistent volume snapshots.

