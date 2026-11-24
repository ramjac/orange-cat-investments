# Orange Cat Investments (OCI) — Architecture & Operations Documentation Hub

Welcome to the central technical, security, and operational reference for **Orange Cat Investments (OCI)**. This index organizes the repository's documentation so new engineers, SREs, and security or financial auditors can quickly understand how data flows, how systems are isolated and hardened, and how the platform is operated day to day.

---

## Navigation by Reader Role

### 1. New Employee / Software Engineer Onboarding
Start here to understand the platform's 5-tier microservices architecture, domain boundaries, data flows, and local development/CI workflows:
1. [Project Overview & Quickstart](../README.md) — Business domains, implemented capabilities, local 5-tier topology, and home-server setup.
2. [Developer & Agent Conventions](../AGENTS.md) — Mandatory coding standards, domain boundaries, `gopls` MCP usage, and verification gates.
3. [System Architecture Reference](architecture/system-architecture.md) — Complete 5-tier service catalog, 4-layer isolation enforcement, Kubernetes namespace topology, and white-label modularization.
4. [Data Flow & Storage Lifecycle](architecture/data-flow-and-storage.md) — End-to-end sequence diagrams for trading, banking, WebRTC telemetry, sagas, firmware OTA, and GDPR erasure, plus full schema/cache/queue inventories.
5. [Platform Simulator Guide](../simulation/README.md) — 13-persona end-to-end simulation driver, action catalog, and CLI flags.
6. [Corporate & Customer Personas](../personas/) — Specifications for all 13 human, feline, and retail investor personas.
7. [API & Contract Specifications](../api/openapi.yaml) & [Protobuf Definitions](../proto/oci/) — OpenAPI 3.1 REST specification and gRPC service contracts.

---

### 2. Security, Risk & Financial Auditor
Start here for defense-in-depth controls, cryptographic guarantees, audit log verification, and automated compliance gates:
1. [Security, Compliance & Audit Reference](architecture/security-and-compliance.md) — Defense-in-depth control matrix, Dual-IdP auth, OPA/Rego Policy-as-Code, SPIFFE mTLS, OpenBao dynamic secrets, Kyverno admission control, hash-chain audit log verification, SOC 2 / ISO 27001 / GDPR / SEC-FINRA alignment, and auditor verification commands.
2. [Data Flow & Storage Lifecycle](architecture/data-flow-and-storage.md) — Exact transactional boundaries, `ops.audit_logs` immutability, GDPR Article 17 erasure flow, and data retention controls.
3. [CI Contract Verification Script](../scripts/verify-ci-contracts.sh) — 31 automated static, manifest, SAST, and policy contract assertions executed in CI.
4. [Architecture Decision Records (ADRs)](adr/) — Formal decision history across all platform phases (see table below).
5. [Chaos & Observability Audit Report](reports/chaos-audit-report.md) — 34-scenario single-fault chaos audit and subsequent Phase 7 remediation record.

---

### 3. SRE, Platform Security & Operations (`emp-human-frank`)
Start here for observability pipelines, cluster deployment, HA topology, and incident runbooks:
1. [Observability & Day-2 Operations](operations/observability-and-day2-operations.md) — OpenTelemetry + Alloy + Prometheus/Tempo, Promtail + Loki CRI pod logging, Gatus 19-probe monitoring, Forgejo CI/CD promotion pipeline, and Day-2 SOPs.
2. [Multi-Node HA Setup & Operations Guide](multi-node-ha-setup-guide.md) — 3-node MicroK8s on Ubuntu Server, `kube-vip`, Longhorn 3-replica storage, and HA failover drills.
3. **Operational Runbooks:**
   * [Control Plane & Host Certificate Rotation](runbooks/control-plane-cert-rotation.md) — MicroK8s annual cert rotation, `oci-ca-bundle` distribution, and emergency recovery.
   * [COTS Patching & Lifecycle Management](runbooks/cots-patching-lifecycle.md) — Upgrade and rollback SOPs for Forgejo, Nextcloud, Snipe-IT, ERPNext/Frappe HR, and ZITADEL.
   * [Stateful Store Alerts](runbooks/stateful-store-alerts.md) — Triage and remediation for Valkey, PostgreSQL, and Meilisearch Prometheus alerts.
   * [Cluster Load Testing & Concurrency Verification](runbooks/cluster-load-testing.md) — External load generation, sysctl tuning, Prometheus TSDB sizing, `oci-dev` DB retention, and runner eviction recovery.
   * [Chaos Engineering](runbooks/chaos-engineering.md) — Sandboxed `oci-dev` fault injection and tier partition drills.
   * [Kubernetes Monitoring Blind Spots](runbooks/k8s-monitoring-blind-spots.md) — K8s API event shipping, Dead Man's Switch (`Watchdog`), and backup freshness metrics.
4. **Networking, TLS & Edge Setup Guides:**
   * [LAN Split-Horizon DNS Guide](../k8s/ha/lan/dns/dns-setup-guide.md)
   * [Employee Device CA Trust Store Guide](../k8s/ha/lan/tls/client-ca-trust-installation.md)
   * [Cloudflare Tunnel Public Ingress Guide](../k8s/ha/public-ingress/cloudflare/setup-cloudflare-tunnel.md)
   * [Direct Public Gateway / Let's Encrypt Guide](../k8s/ha/public-ingress/nginx-letsencrypt/setup-nginx-letsencrypt.md)
5. **Database Migrations:**
   * [Database Migrations Protocol](../scripts/migrations/README.md) — Three-copy SQL sync rule and `scripts/migrations/NNN-*.sql` procedures.

---

## Architecture Decision Records (ADRs)

| ADR | Title | Scope / Phase | Status |
| :--- | :--- | :--- | :--- |
| [ADR 0001](adr/0001-system-architecture-design.md) | System Architecture Design for OCI Platform | DDD domains, Dual IdP, 5-tier foundation | Accepted (Evolved by ADRs 0004–0007) |
| [ADR 0002](adr/0002-multi-node-ha-topology.md) | Multi-Node HA & Resilient SMB Cluster Topology | 3-node MicroK8s, `kube-vip`, Longhorn, Cloudflare Tunnel | Accepted |
| [ADR 0003](adr/0003-gitops-containerization-promotion-pipeline.md) | Self-Hosted GitOps, Containerization & Promotion | Forgejo Actions, `oci-dev` simulation gate, `oci-core` promotion | Accepted (Evolved by ADR 0005) |
| [ADR 0004](adr/0004-zero-trust-workload-identity-and-dynamic-secrets.md) | Zero-Trust Workload Identity & Dynamic Secrets | `cert-manager-csi-driver-spiffe`, SPIFFE SANs, OpenBao, ESO | Accepted |
| [ADR 0005](adr/0005-software-supply-chain-security-and-admission-control.md) | Software Supply Chain Security & Admission Control | Semgrep SAST, Distroless, Harbor/Trivy, Cosign/SBOM, Kyverno | Accepted |
| [ADR 0006](adr/0006-zero-trust-networking-gateway-api-and-edge-waf.md) | Zero-Trust Networking, Gateway API & Edge WAF | Default-deny `NetworkPolicy`, Envoy Gateway API, Coraza WAF, `coturn` | Accepted |
| [ADR 0007](adr/0007-transactional-outbox-idempotency-audit-chain-and-opa-authz.md) | Transactional Outbox, Idempotency, Audit Chain & OPA Authz | SQL Outbox, Valkey `Idempotency-Key`, `ops.audit_logs`, OPA/Rego | Accepted |
