# ADR 0007: Transactional SQL Outbox, Valkey Idempotency, Hash-Chain Audit Log & OPA Policy-as-Code

* **Status:** Accepted
* **Date:** 2026-11-24
* **Author:** Marcus Lead (`emp-human-marcus`), Principal Systems Architect & Frank Operations (`emp-human-frank`), Platform Security & K8s Infrastructure Lead
* **Context:** Architectural specification for Roadmap Phase 10 (`7.40`–`7.42`), guaranteeing atomic domain-event delivery without violating `NetworkPolicy` isolation, safe client retries for financial mutations, cryptographic audit log immutability, and declarative policy-as-code authorization.

---

## 1. Context and Problem Statement

1. **Dual-Write Risk Between PostgreSQL and RabbitMQ:** Publishing domain events directly from BLLs after Domain API gRPC calls risked silent event loss if RabbitMQ or the caller failed immediately after the database commit. Meanwhile, allowing `oci-worker` to poll PostgreSQL directly would violate Phase 9 `NetworkPolicy` isolation (`domain-apis-network-policy`).
2. **Duplicate Financial Mutations on Network Retries:** Retrying `POST /api/v1/customer/trading/orders` or `POST /api/v1/customer/deposits` after client timeouts risked executing duplicate trades or ACH transfers.
3. **Mutable Audit Records:** Standard relational rows in `ops.audit_logs` lacked database-level immutability triggers or cryptographic chaining to prove records had not been altered or deleted.
4. **Hardcoded Authorization Conditionals:** High-risk authorization checks (feline medical holds, employee offboarding, anti-self-approval leave workflows, pre-trade risk halts) were scattered across Go handler conditionals rather than unified, auditable policy rules.

---

## 2. Decision Outcome

1. **Schema-Scoped Transactional SQL Outbox & gRPC-Bridged Relay (`7.40`):**
   * Added `<schema>.outbox_events` tables (`workforce`, `facilities`, `core_invest`, `ops`) via [`003-transactional-outbox.sql`](../../scripts/migrations/003-transactional-outbox.sql).
   * Domain APIs persist `envelope.EventEnvelope[T]` rows inside the exact same `pgx.Tx` transaction as domain mutations ([`pkg/outbox`](../../pkg/outbox/outbox.go)).
   * `oci-worker` runs a gRPC-bridged [`Outbox Forwarder`](../../internal/app/worker/outbox_relay.go) polling `customer-bll` and `internal-bll` every `500ms` (`FOR UPDATE SKIP LOCKED`, `30s` lease), preserving strict `NetworkPolicy` separation between workers and PostgreSQL.
2. **Valkey `Idempotency-Key` Middleware (`7.40`):**
   * Implemented [`pkg/cache/valkey/idempotency.go`](../../pkg/cache/valkey/idempotency.go) requiring a UUID `Idempotency-Key` header on `POST /api/v1/customer/trading/orders` and `POST /api/v1/customer/deposits`.
   * Uses atomic `SET NX EX 86400` locks (`idempotency:{customer_id}:{uuid}`): returns `409 Conflict` for concurrent in-flight keys, replays cached status/body with `X-Idempotent-Replayed: true` on safe retries, and releases locks on `>= 500` errors.
3. **Tamper-Evident Cryptographic Hash-Chain Audit Log (`7.41`):**
   * Hardened `ops.audit_logs` ([`004-tamper-evident-audit-log.sql`](../../scripts/migrations/004-tamper-evident-audit-log.sql)) with `BEFORE UPDATE OR DELETE` and `BEFORE TRUNCATE` triggers raising `SQLSTATE 42501`, plus a `BEFORE INSERT` trigger serialized under `pg_advisory_xact_lock(741001)` computing a chained SHA-256 `record_hash` from `previous_hash` (starting at `'GENESIS'`).
   * Exposed `AppendAuditLog`, `VerifyAuditChain`, and `ListAuditLogs` across `ops-api` and `internal-bll`, surfaced at `GET /api/v1/ops/audit-logs/verify` on `employee-bff`, and ensured GDPR erasures log only SHA-256 `customer_id_hash` values.
4. **Embedded OPA / Rego Policy-as-Code Authorization (`7.42`):**
   * Created [`pkg/authz/policy`](../../pkg/authz/policy/engine.go) embedding `github.com/open-policy-agent/opa/v1/rego` with compiled Rego v1 policies (`workforce.rego`, `trading.rego`).
   * Propagated authenticated actor metadata (`x-oci-actor-id`, `x-oci-actor-role`, `x-oci-actor-dept`) from BFFs to BLLs ([`pkg/grpcutil/actor.go`](../../pkg/grpcutil/actor.go)).
   * Enforced fail-closed evaluation in `internal-bll` and `customer-bll`, logging every denial (`authz.policy.denied`) to `ops.audit_logs`.

---

## 3. Consequences

* **Verifiable Financial & Governance Integrity:** Auditors can cryptographically verify the unbroken chain of sensitive actions via `/api/v1/ops/audit-logs/verify`, while clients gain safe idempotent retries and atomic event delivery without relaxing zero-trust network boundaries.
