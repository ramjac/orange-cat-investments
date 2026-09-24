# ADR 0001: System Architecture Design for Orange Cat Investments (OCI) Platform

* **Status:** Accepted
* **Date:** 2025-02-23
* **Author:** Principal Enterprise Architect
* **Context:** Architectural specification for the Orange Cat Investments (OCI) self-hosted small-to-medium business platform.

---

## 1. Context and Problem Statement

Orange Cat Investments (OCI) requires a production-grade, self-hosted application platform operating on Kubernetes (K3s). The platform spans external revenue-generating operations (ingesting feline behavioral data, executing investment strategies, and serving investors) and internal business operations (managing physical habitational hardware, tracking feline and human workforce members, and automating IT/maintenance workflows).

To ensure high maintainability, strict security boundaries, clean domain isolation, and modular scalability, the architecture must define:
1. Domain boundary decomposition and database storage mapping.
2. An asynchronous, event-driven architecture connecting custom Go services (BFF, gRPC BLL services, and background workers).
3. A unified dual-IdP authentication and authorization security framework separating external customer access (via ZITADEL) and internal employee/employer operational access (via Forgejo OAuth2/OIDC).

---

## 2. Architectural Principles & Integration Boundaries

### 2.1 Domain-Driven Design (DDD) & Service Layers
The application architecture adheres strictly to Domain-Driven Design principles:
* **Domain APIs & Persistence:** Each bounded context is encapsulated by a dedicated Domain API in front of isolated database schemas. Direct cross-domain database queries or cross-Domain API calls are prohibited.
* **Business Logic Layer (BLL):** Microservices and async workers orchestrate multi-domain workflows by calling underlying Domain APIs over gRPC mTLS.
* **Backend-For-Frontend (BFF):** Dedicated Go REST APIs serving web and mobile frontends (`cmd/customer-bff` and `cmd/employee-bff`). The BFFs manage user sessions, CSRF validation, authorization, and response shaping. The BFFs call BLL services over gRPC mTLS.

### 2.2 Commercial Off-The-Shelf (COTS) / Open Source (OSS) vs. Custom Code Boundary
1. **System of Record for Generic Operations:**
   * **ERPNext / Frappe HR:** Single source of truth for double-entry financial accounting, legal payroll, billing, and core HR records.
   * **Homebox:** Office physical inventory for non-networked equipment.
   * **ZITADEL:** Identity provider (OIDC/SSO) for external customer authentication.
   * **Forgejo:** Git repositories, issue tracking, CI/CD pipelines (Actions), and OAuth2/OIDC identity provider for internal employees/employers/staff.
2. **Custom Go Domain Responsibilities:**
   * Feline behavioral stream ingestion (Orange Observer) and automated algorithmic portfolio trading (Core Investment Engine).
   * Edge telemetry and hardware asset tracking for custom habitational infrastructure (Facilities).
   * Specialty workflows combining human and feline staff data (Workforce domain extensions).
3. **Integration Pattern:** Custom Go microservices integrate with OSS tools strictly via REST APIs, gRPC, or asynchronous webhooks. Database tables are never shared across COTS and custom boundaries.

---

## 3. Domain Boundary Decomposition & PostgreSQL Schema Architecture

The platform's relational persistence is hosted on PostgreSQL 16 using schema isolation to enforce domain boundaries within a shared instance.

```text
+-------------------------------------------------------------------------+
|                          PostgreSQL 16 Database                         |
+------------------------------------+------------------------------------+
| Schema: workforce                  | Schema: facilities                 |
| - employees (human & feline)       | - hardware_assets                  |
| - onboarding_checklists            | - observation_perches              |
| - care_schedules                   | - maintenance_tickets              |
| - review_cycles                    | - asset_telemetry_logs             |
+------------------------------------+------------------------------------+
| Schema: core_invest                | Schema: ops                        |
| - observation_events               | - it_tickets (Forgejo ingests)     |
| - investment_portfolios            | - alert_subscriptions              |
| - allocation_strategies            | - audit_logs                       |
| - portfolio_snapshots              |                                    |
+------------------------------------+------------------------------------+
```

### 3.1 Schema Mapping
1. **`workforce` Schema:**
   * Entities: Human employees, feline staff members, onboarding checklists, feline care/dietary schedules, performance review cycles.
   * Key Invariants: Employees are typed as `human` or `feline`. Feline records require mandatory dietary/habitat care directives.
2. **`facilities` Schema:**
   * Entities: Hardware assets (edge cameras, observation perches, smart collars), habitat maintenance logs, location zones, device health metrics.
   * Key Invariants: Hardware assets maintain operational state (`active`, `maintenance_required`, `decommissioned`).
3. **`core_invest` Schema:**
   * Entities: Raw & processed cat observation events, investor account balances, portfolio strategy parameters, asset allocation ledger.
   * Key Invariants: Financial transactions maintain strict audit trails linked to observation event correlation IDs.
4. **`ops` Schema:**
   * Entities: Internal IT helpdesk tickets (ingested via Forgejo webhooks), alert notifications, system audit logs.

---

## 4. Event-Driven & Inter-Service Architecture

Communication between the Go REST BFFs, gRPC BLL services, and Go Worker service uses RabbitMQ as the message broker, orchestrated via the Watermill framework in Go.

```text
  +----------------------+  +------------------------+  +-------------------+
  | Customer Portal (Vue)|  | Employee Portal (Vue)  |  |   Hugo / Public   |
  |  (web/customer-portal)  |  (web/employee-portal) |  |     Website       |
  +----------+-----------+  +-----------+------------+  +---------+---------+
             |                          |                         | Static
             v REST                     v REST                    v
  +----------------------+  +------------------------+
  |  Go Customer BFF     |  |   Go Employee BFF      |
  | (ZITADEL Auth Session)|  |(Forgejo OAuth2 Session)|
  +----------+-----------+  +-----------+------------+
             |                          |
             +------------+-------------+
                          | gRPC over mTLS
                          v
  +------------------------------------------------------+
  |            Business Logic Layer (BLL)                |
  |   (Core Investment, Facilities, Workforce Services)  |
  +------------+-----------------------------+-----------+
               | Publish                     | Subscribe
               v                             v
  +------------------------------------------------------+
  |             RabbitMQ Event Broker (Watermill)        |
  +-------------------------+----------------------------+
                            ^
                            | Consume & Process
  +-------------------------+----------------------------+
  |            Go Asynchronous Worker Engine             |
  | - Cron Schedulers   - Event Stream Processors        |
  +------------------------------------------------------+
```

### 4.1 Standard Event Envelope
All asynchronous messages emitted to RabbitMQ wrap domain payloads in a standardized Go generic envelope:

```go
type EventEnvelope[T any] struct {
    EventID       string    `json:"event_id"`       // UUIDv7
    EventType     string    `json:"event_type"`     // e.g., "observation.cat_spotted.v1"
    OccurredAt    time.Time `json:"occurred_at"`    // UTC Timestamp
    CorrelationID string    `json:"correlation_id"` // Tracing ID passed from context
    Payload       T         `json:"payload"`        // Typed domain payload struct
}
```

### 4.2 Asynchronous Workflows
* **Cat Activity Ingestion:**
  1. `Orange Observer` detects feline activity and emits `events.observation.cat_spotted.v1`.
  2. `Go Worker` consumes the event via Watermill, executes algorithmic portfolio checks, and calls `core_invest` BLL gRPC service to execute dividend allocation.
* **Scheduled Operations:**
  1. `Go Worker` runs a midnight ticker scanning `facilities.hardware_assets` for devices past due for preventive maintenance.
  2. Worker publishes `events.facilities.maintenance_due.v1` to trigger notifications and mobile work orders.

---

## 5. Security, Dual Authentication & Authorization Framework

The platform maintains two distinct authentication domains to preserve strict identity boundary separation between external investors and internal corporate staff.

```text
+-------------------------------------------------------------------------------+
|                            Dual Identity Provider Model                       |
+---------------------------------------+---------------------------------------+
| 1. External Customer Authentication   | 2. Internal Employee Authentication   |
| - Identity Provider: ZITADEL (OIDC)   | - Identity Provider: Forgejo (OAuth2) |
| - Surface: Customer Portal            | - Surface: Employee Portal            |
|   (`web/customer-portal`)             |   (`web/employee-portal`)             |
| - BFF: Customer BFF (`cmd/customer-bff`)| - BFF: Employee BFF (`cmd/employee-bff`)|
+---------------------------------------+---------------------------------------+
```

### 5.1 Dual OIDC/OAuth Auth Flows & Valkey Sessions

#### A. External Customer Auth Flow (ZITADEL)
* **Identity Provider:** ZITADEL
* **Target Audience:** Public investors and retail customers.
* **Surface:** `web/customer-portal` (`customer.oci.local`)
* **Session Strategy:**
  1. Login redirects user to ZITADEL OIDC `/authorize`.
  2. Authorization code exchanged by `cmd/customer-bff` for ZITADEL ID and Access tokens.
  3. Tokens stored in Valkey server-side session.
  4. Browser receives an opaque `__Host-customer-session` cookie (`HttpOnly`, `Secure`, `SameSite=Lax`).

#### B. Internal Employee / Employer Auth Flow (Forgejo)
* **Identity Provider:** Forgejo
* **Target Audience:** Corporate staff, managers, facilities engineers, caretakers.
* **Surface:** `web/employee-portal` (`employee.oci.local` / `ops.oci.local`)
* **Session Strategy:**
  1. Login redirects user to Forgejo OAuth2 `/login/oauth/authorize`.
  2. Authorization code exchanged by `cmd/employee-bff` for Forgejo tokens and profile.
  3. User role mapping checked against Forgejo team memberships (e.g. `facilities-techs`, `hr-managers`).
  4. Tokens stored in Valkey server-side session.
  5. Browser receives an opaque `__Host-employee-session` cookie (`HttpOnly`, `Secure`, `SameSite=Lax`).

### 5.2 CSRF Mitigation
Double-submit cookie pattern combined with `X-CSRF-Token` header for state-changing requests (`POST`, `PUT`, `DELETE`) across both BFFs.

### 5.3 Internal Inter-Service Identity & Auth (SPIFFE + mTLS)
All internal gRPC communication (`BFF` $\rightarrow$ `BLL` $\rightarrow$ `Domain APIs`) uses Mutual TLS (mTLS) managed by `cert-manager` and `trust-manager`.

* **Workload Identity:** Every service pod runs under a dedicated Kubernetes ServiceAccount. `cert-manager` issues X.509 certs with a SPIFFE ID in the URI SAN:
  `spiffe://oci.local/ns/{namespace}/sa/{serviceaccount}`
* **Certificate Lifecycle:**
  * Lifetime: 7 days, renewed automatically every 3 days.
  * In-Process Rotation: Go services use standard library `crypto/tls` callbacks (`GetCertificate` and `GetClientCertificate`) to reload certificate files from directory mounts dynamically without pod restarts. SubPath mounts are strictly forbidden.
* **gRPC Interceptor Authorization:** gRPC servers use a unary/stream interceptor that parses the client certificate's SPIFFE ID URI SAN and checks it against an explicit role caller matrix.

---

## 6. Client Surface Matrix

| Surface | Target Audience | Tech Stack | Ingress Host | Auth Mechanism / IdP |
| :--- | :--- | :--- | :--- | :--- |
| **Marketing Site** | Public / Investors | Hugo (Static) | `oci.local` | N/A |
| **Customer Portal** | External Investors | Vue 3 SPA (`web/customer-portal`) | `customer.oci.local` | ZITADEL OIDC + Valkey Session Cookie |
| **Employee Portal** | Internal Staff & Employers | Vue 3 SPA (`web/employee-portal`) | `employee.oci.local` | Forgejo OAuth2 + Valkey Session Cookie |
| **Mobile App** | Field Technicians | Flutter | API Ingress | Forgejo Auth / OAuth2 Token |
| **Watch App** | On-Call Engineers | Pebble C SDK | API Ingress | Companion App Proxy Token |
| **CLI Tool** | Operational Staff | Go Cobra | Direct gRPC | Personal Access Token |

---

## 7. Operational & Security Checklist

* **Dual Identity Isolation:** Customer identities (ZITADEL) and Employee identities (Forgejo) are strictly separated across distinct BFF and SPA surfaces.
* **Zero Static Certificates:** All workload certificates are ephemeral (7 days) and dynamically reloaded.
* **Schema Isolation:** PostgreSQL logic enforced via isolated schemas (`workforce`, `facilities`, `core_invest`, `ops`).
* **No JWTs in SPA Storage:** All OIDC tokens kept server-side in Valkey.
* **Observability:** OpenTelemetry context propagated across gRPC metadata and Watermill event headers (`CorrelationID`).
