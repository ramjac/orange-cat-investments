# AGENTS.md - Developer & Agent Instructions for Orange Cat Investments (OCI)

Welcome! This repository hosts the software application platform for **Orange Cat Investments (OCI)**. This document serves as instructions and conventions for autonomous software agents (and human engineers) modifying or expanding this codebase.

---

## 1. High-Level Architecture Overview

OCI is organized as a single **Monorepo** following **Domain-Driven Design (DDD)** and microservices architecture principles.

### Key Operational Planes & Tech Stack:
* **Orchestration & Infrastructure:** K3s Kubernetes, Ansible, Cert-Manager, Trust-Manager, Valkey, RabbitMQ.
* **Database Layer:** PostgreSQL 16 using schema isolation (`workforce`, `facilities`, `core_invest`, `ops`).
* **Custom Backend Services:**
  * **Go REST BFF (`cmd/bff`):** Connects web/mobile frontends using `net/http` + `chi`. Uses server-side Valkey session management and double-submit CSRF protection.
  * **Go gRPC BLL Microservices:** Internal domain services communicating over mTLS (`pkg/mtls`).
  * **Go Background Worker (`cmd/worker`):** Event subscriber (Watermill / RabbitMQ) and ticker/cron runner.
* **Client Applications:**
  * **Vue 3 SPA:** Authenticated Customer & Employee Portal (`web/portal`).
  * **Hugo:** Static public marketing site (`web/marketing`).
  * **Flutter:** Field maintenance mobile app (`mobile/flutter_app`).
  * **Pebble C SDK:** On-call watch app (`embedded/pebble`).

---

## 2. Directory Structure Conventions

```text
.
├── api/                   # OpenAPI 3.1 YAML specifications (`openapi.yaml`)
├── docs/                  # Architecture Decision Records (`adr/`) and specifications
├── proto/                 # Protobuf definitions (`oci/core/`, `oci/ops/`)
├── pkg/                   # Shared Go packages (`gen/go`, `mtls`, `envelope`, etc.)
├── scripts/               # SQL scripts (`init.sql`) and environment bootstrapping
├── cmd/                   # Go application entrypoints (`bff`, `worker`, `domain-*`)
└── internal/              # Core domain logic, handlers, repositories, services
```

---

## 3. Mandatory Coding Standards & Principles

### Domain Boundary Rules
1. **Schema Isolation:** Custom Go services strictly access their designated database schema. Direct cross-schema table queries or cross-Domain API calls are prohibited.
2. **COTS vs. Custom Isolation:** ERPNext/Frappe HR remain systems of record for generic accounting and payroll. Custom Go code is reserved for OCI core domain algorithms and hardware asset telemetry. Go microservices interface with COTS via REST/webhooks, never by sharing database tables.

### Security Invariants
1. **Zero Long-Lived / Static Certs:** All inter-service gRPC calls require mTLS via `cert-manager`. Certificates must be reloaded dynamically in Go standard library `crypto/tls` (`GetCertificate` / `GetClientCertificate`). Never use `subPath` secret mounts.
2. **No JWTs in Web SPA:** Web browser clients authenticate via OIDC (ZITADEL) handled at the Go BFF layer. Access tokens are stored exclusively in Valkey. Browsers receive only an opaque `__Host-session` cookie (`HttpOnly`, `Secure`, `SameSite=Lax`).
3. **Double-Submit CSRF:** All state-changing HTTP requests (`POST`, `PUT`, `DELETE`) to the BFF must include the `X-CSRF-Token` header matching the CSRF cookie.

### Event Protocol Standard
All domain messages emitted across RabbitMQ must be wrapped in the standard Go `EventEnvelope[T]` with UUIDv7 `EventID`, UTC timestamp `OccurredAt`, and OTel context `CorrelationID`.

---

## 4. Verification & Testing Procedures

Before marking any task as complete or submitting code changes, agents must verify their work using appropriate validation methods:

1. **SQL Validation:**
   Ensure `scripts/init.sql` parses correctly and enforces PostgreSQL constraints:
   ```bash
   python3 -c "import psql" # or validate syntax via python/psql CLI if available
   ```
2. **OpenAPI Specification Validation:**
   Verify `api/openapi.yaml` for YAML correctness and OpenAPI 3.1 compliance:
   ```bash
   python3 -c "import yaml; yaml.safe_load(open('api/openapi.yaml'))"
   ```
3. **Go Code Quality (when Go files exist):**
   ```bash
   go test ./...
   go vet ./...
   ```
4. **General Rule:** Always read modified files after editing to verify formatting, content accuracy, and absence of syntax errors.
