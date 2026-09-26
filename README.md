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
* **Backend-For-Frontend (BFF):** Web and mobile frontends connect strictly to dedicated Go BFF APIs (`cmd/customer-bff` and `cmd/employee-bff`). The BFF manages user authentication (OIDC/sessions), authorization, request aggregation, and API response shaping tailored to frontend requirements. The BFF can only invoke BLL services.
* **Inter-Service Protocols & Security:**
  * **Frontend $\rightarrow$ BFF:** HTTPS REST with secure, HTTP-only opaque session cookies and CSRF protection.
  * **BFF $\rightarrow$ BLL $\rightarrow$ Domain APIs:** gRPC over Mutual TLS (mTLS). Client whitelists and role-based interceptors restrict call permissions at each layer.

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
* **Frontend Applications:** Vue.js (Customer & Employee Portals), Hugo (Static Marketing Site).
* **Backend Services & Tooling:** Go (1.22+). Go with [cdk8s](https://cdk8s.io/) for Helm and Kubernetes manifest generation.
* **Configuration & Glue:** YAML for Kubernetes manifests and Forgejo Actions; Bash for setup scripts; HCL Terraform for cloud infrastructure fallbacks.
* **Repository Strategy:** Single Monorepo pattern.

---

## 4. Employee Personas & Daily Workflow Mapping

To exercise and simulate the entire software stack using AI agents, OCI defines 8 distinct employee personas across human staff and feline executives:

1. **Garfield (`emp-feline-garfield`)** — Chief Observation Officer (COO)
2. **Barneby (`emp-feline-barneby`)** — Senior Alpha Perch Analyst
3. **Alice Vance (`emp-human-alice`)** — Head of Human & Feline Resources
4. **Bob Builder (`emp-human-bob`)** — Lead Facilities & Edge Telemetry Engineer
5. **Carol Danvers (`emp-human-carol`)** — Chief Investment Officer (CIO)
6. **David Quant (`emp-human-david`)** — Feline Behavioral Data Scientist
7. **Dr. Elena Rostova (`emp-human-elena`)** — Chief Veterinary Officer & Habitat Specialist
8. **Frank Operations (`emp-human-frank`)** — Platform Security & K8s Infrastructure Lead

Detailed profile specifications are located in the `personas/` directory.

---

## 5. Future Work & Feature Backlog (Unsupported Employee Actions)

Based on comparing the daily actions of OCI employee personas against current software stack capabilities, the following features represent unsupported actions to be implemented in subsequent development phases:

### A. Core Investment Engine & Observation Telemetry
* **Real-time WebRTC/RTSP Computer Vision Stream Ingestion:** Ingestion pipeline for live optical camera feeds from habitat edge cameras to feed computer vision inference models.
* **Automated Brokerage Execution Integrations:** Connect Go trading workers to external market exchanges/brokerages via FIX or REST APIs for automated order placement based on feline observation triggers.
* **Dynamic Backtesting Workbench UI:** Interactive portal tool for David Quant to run historical backtests comparing feline activity event logs against equity asset performance.

### B. Facilities & Mobile Field Maintenance
* **Flutter Mobile Offline Database Sync:** Full local SQLite persistence and background sync engine for Bob Builder to sync QR-scanned maintenance logs when re-establishing connectivity.
* **Edge Device Firmware OTA Pipeline:** Automated over-the-air firmware update pipeline for edge cameras, feeder units, and smart collar IoT devices.

### C. Workforce Operations & Automated Sagas
* **Frappe HR & ZITADEL Integration Webhooks:** Webhook triggers automatically initiating the Go `OnboardingSaga` and `OffboardingEngine` when employee statuses change in Frappe HR.
* **Interactive Care Schedule UI:** Employee Portal Vue SPA interface for Dr. Elena Rostova to dynamically edit feline dietary plans, feeding times, and emergency medical trading holds.

### D. Operations & Wearable Alerts
* **Pebble Companion App Gateway Proxy:** Bluetooth/HTTP proxy service translating Go API alert events into Pebble C AppMessage payloads and returning ACK button interactions to `ops.it_tickets`.
* **Automated Root CA Key Rotation Worker:** Scheduled job automating intermediate CA renewal and certificate re-issuance through cert-manager without human intervention.
