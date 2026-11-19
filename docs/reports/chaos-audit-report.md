# OCI Rigorous Chaos & Observability Audit Report (`thinker` Cluster)

**Audit Date:** 2026-11-19 (UTC `13:05`–`14:08`)  
**Deployed Revision:** `96333bf` (`:sha-96333bf` verified across `oci-dev`, `oci-core`, and `external-sim`)  
**Execution Host:** External developer workstation → `thinker` NGINX Ingress (`192.168.1.246`)  
**Harness:** [`scripts/run-chaos-audit.sh`](file:///home/ramjac/orange-cat-investments/scripts/run-chaos-audit.sh)  
**Alert Dwell:** `75s` per `oci-dev` & `external-sim` mock scenario; `100s` per `oci-internal` COTS scenario  
**Safety Verification:** Zero restarts or mutations in `oci-core`, `forgejo-runner`, or unsuffixed `external-sim` mocks. Post-run clean simulator pass: **73 / 73** actions succeeded (`0` failures).

---

## 1. Executive Summary

We executed **34 single-fault chaos scenarios** one by one against `oci-dev` (18 workloads), `oci-internal` (6 COTS/DB workloads), and `external-sim` `-dev` mocks (`mock-bank-dev` & `mock-broker-dev`, 10 scenarios including every `/sim/faults` knob and `-fault-scenarios`), running the full 13-persona simulator (73 actions) and targeted probes under each outage and auditing caller logs, Loki ingestion, Prometheus alerts/targets, Gatus health checks, RabbitMQ queues, and post-restore recovery.

### Headline Findings

| Observability / Resilience Pillar | Live Audit Verdict | Key Metric |
| :--- | :--- | :--- |
| **Centralized Logging (Loki)** | **CRITICAL GAP** | **0 / 34** scenarios had pod logs in Loki (`baseline_loki_labels` only contains `filename`, `job` from host `/var/log/*log`). |
| **Prometheus Alerting** | **CRITICAL GAP** | **0 / 34** scenarios triggered an outage, availability, HTTP 5xx, gRPC, DLQ, or upstream-fault alert (only incidental `PostgresLowCacheHitRatio[pending]` fired after DB restart). |
| **Gatus Uptime Monitoring** | **DEGRADED AT BASELINE** | **10 / 18** Gatus endpoints are permanently red even at clean baseline; the 3 working COTS probes (`forgejo`, `nextcloud`, `snipe-it`) did alert accurately on `B1`–`B3`/`B6`. |
| **Caller Pod Log Descriptiveness** | **MIXED (12 Pass / 9 Partial / 13 Gap)** | `customer-bll`, `oci-worker-dev`, Valkey session/rate-limit, and Watermill AMQP logged rich, descriptive errors. Both BFFs (`oci-customer-bff`, `oci-employee-bff`) logged **0 errors** when BLLs were down and leaked raw internal IP/port/DB strings in HTTP responses. |
| **Mock Fault Injection (`7.17`)** | **2 LIVE BUGS FOUND** | `/sim/faults` was blocked by API key auth (`401`) on both mocks, and `simulation/driver_fault_patch.go` ignored `OCI_INGRESS_IP`. Once authenticated, `outage`, `latency_ms: 6500`, `reject_transfers`, `reject_orders`, and `halted_symbols` worked accurately with clean `503`/`504`/`422` mappings. |
| **Async Worker & DLQ (`7.18`)** | **WORKING — EXPOSED LIVE BUG** | `oci-worker-dev` retried 3× with exponential backoff, emitted CWE-532-safe logs, and dead-lettered cleanly—exposing **4,200 pre-existing messages** in `events.banking.ach_transfer_requested.v1.dlq` caused by a silent `bank API error (400): bank account not found` + `recurring_deposits_frequency_check` (`SQLSTATE 23514`). |
| **Self-Healing Recovery** | **MOSTLY CLEAN (2 Carryover Gaps)** | All 34 targets returned `1/1 Ready` and final simulator passed **73/73**, but `core-invest-api`/`ops-api` gRPC backoff (`A11`→`A12`, `A12`→`A13`) and RabbitMQ publisher reconnect (`A16`→`A17`) exhibited transient carryover failures right after restore. |

---

## 2. Complete 34-Scenario Audit Matrix

*Legend — Log Grade: `PASS` = descriptive caller error naming dependency & cause; `PARTIAL` = generic/noisy/ingress-only error or vendor leak; `GAP` = zero caller error logs or outage silently masked. Baseline Gatus has 10 permanently red probes (see §3.3); the Gatus column below lists **newly** degraded probes only.*

### Group A: `oci-dev` Workloads (18 Scenarios)

| ID | Target (`oci-dev`) | Sim (Pass/Total) | Extra Probe | Caller Log Grade | Loki | New Prom Alerts / Targets | New Gatus | Recovery |
| :-: | :--- | :--- | :--- | :--- | :-: | :--- | :--- | :--- |
| A1 | `deploy/oci-customer-portal-dev` | 73/73 (0 fail) | `dev.customer.oci.local/` → **503** | `PARTIAL` (ingress 503 only) | 0 | None | None | READY |
| A2 | `deploy/oci-employee-portal-dev` | 73/73 (0 fail) | `dev.employee.oci.local/` → **503** | `PARTIAL` (ingress 503 only) | 0 | None | None | READY |
| A3 | `deploy/oci-marketing` | 73/73 (0 fail) | `dev.oci.local/` → **503** | `PARTIAL` (ingress 503 only) | 0 | None | None | READY |
| A4 | `deploy/oci-customer-bff-dev` | 55/73 (**18 fail**) | — | `PARTIAL` (NGINX upstream timeout/503 HTML) | 0 | None | None | READY |
| A5 | `deploy/oci-employee-bff-dev` | 26/73 (**47 fail**) | — | `PARTIAL` (NGINX 503 HTML) | 0 | None | None | READY |
| A6 | `deploy/oci-pebble-proxy-dev` | 73/73 (0 fail) | `dev.pebble-proxy.oci.local/healthz` → **503** | `GAP` (no `oci-worker-dev` pebble push error logged) | 0 | None | None | READY |
| A7 | `deploy/customer-bll` | 59/73 (**14 fail**) | — | `GAP` (**0** errors in `oci-customer-bff-dev`; raw gRPC IP:8085 leaked to client) | 0 | None | None | READY |
| A8 | `deploy/internal-bll` | 16/73 (**57 fail**) | — | `GAP` (**0** errors in `oci-employee-bff-dev`; raw gRPC dial error leaked) | 0 | None | None | READY |
| A9 | `deploy/workforce-api` | 27/73 (**46 fail**) | — | `PASS` (`customer-bll` logged `feline executive resolution failed` + gRPC `Unavailable`; `internal-bll` silent) | 0 | None | None | READY |
| A10 | `deploy/facilities-api` | 62/73 (**11 fail**) | — | `GAP` (`internal-bll` & `oci-employee-bff-dev` logged **0** `facilities-api` errors) | 0 | None | None | READY |
| A11 | `deploy/core-invest-api` | 54/73 (**19 fail**) | — | `PASS` (`customer-bll` logged `failed to persist order/recurring deposit to core-invest-api via gRPC`) | 0 | None | None | READY *(carryover into A12)* |
| A12 | `deploy/ops-api` | 52/73 (**21 fail**) | — | `GAP` (`internal-bll` logged `core-invest-api` carryover errors, **0** `ops-api` errors; raw IP:50054 leaked) | 0 | None | None | READY *(carryover into A13)* |
| A13 | `deploy/oci-worker-dev` | 63/73 (10 carryover fail) | RMQ: `ach_transfer_requested` **1 msg / 0 consumers**, `.dlq` **4,200 msgs** | `GAP` (sync BFFs unaffected; zero worker-absent alert) | 0 | None (`RabbitMQStagnantConsumers` & `DLQ` blind to `oci-dev`) | None | READY |
| A14 | `statefulset/postgres-dev` | 33/73 (**40 fail**) | — | `PARTIAL` (`customer-bll` descriptive; HTTP responses leak `user=oci_admin database=oci_dev`!) | 0 | None (subsequent `PostgresLowCacheHitRatio[pending]` after restart) | None | READY |
| A15 | `deploy/valkey` | 37/73 (**36 fail**) | — | `PASS` (55 descriptive `valkey error fetching customer session` / `rate limiter falling back to memory` logs) | 0 | Incidental `PostgresLowCacheHitRatio[pending]` only | None | READY |
| A16 | `deploy/rabbitmq` | 72/73 (**1 fail**) | — | `PASS` (63 descriptive Watermill `CONNECTION_FORCED` / `Cannot reconnect to AMQP` logs) | 0 | Incidental `PostgresLowCacheHitRatio[pending]` only | None | READY *(carryover into A17)* |
| A17 | `deploy/meilisearch` | 72/73 (1 RMQ carryover) | — | `GAP` (**0** search errors logged; `MeilisearchDown` did not fire because pod vanished from K8s SD) | 0 | Incidental `PostgresLowCacheHitRatio[pending]` only | None | READY |
| A18 | `deploy/zitadel` | 73/73 (0 fail) | `dev.auth.oci.local/debug/healthz` → **503** | `GAP` (sim uses dev login bypass; **0** `customer-bff` errors; Prom scrapes `oci-core` ZITADEL only) | 0 | Incidental `PostgresLowCacheHitRatio[pending]` only | None | READY |

### Group B: `oci-internal` COTS & Internal DB (6 Scenarios)

| ID | Target (`oci-internal`) | Sim (Pass/Total) | Caller Log Grade | Loki | New Prom Alerts / Down Targets | New Gatus Unhealthy | Recovery |
| :-: | :--- | :--- | :--- | :-: | :--- | :--- | :--- |
| B1 | `deploy/forgejo` | 67/73 (**6 fail** — Elise/Marcus) | `GAP` (sim got NGINX 503 HTML; **0** BFF/BLL logs) | 0 | Target `forgejo` **DOWN**, **0** Prom alerts | **`forgejo-code-review`** | READY |
| B2 | `deploy/nextcloud` | 73/73 (0 fail) | `GAP` (silently masked by `internal-bll` in-memory fallback — confirms §7.23) | 0 | Target `nextcloud` **DOWN**, **0** Prom alerts | **`nextcloud-workspace`** | READY |
| B3 | `deploy/snipe-it` | 73/73 (0 fail) | `GAP` (**0** errors; no live simulator calls reach Snipe-IT) | 0 | None | **`snipe-it-inventory`** | READY |
| B4 | `deploy/erpnext-frappe` | 73/73 (0 fail) | `GAP` (**0** errors; no live simulator calls fail) | 0 | None | None (already red at baseline with 404) | READY |
| B5 | `deploy/frappe-hr` | 73/73 (0 fail) | `GAP` (**0** errors; no live simulator calls fail) | 0 | None | None (no Gatus probe exists) | READY |
| B6 | `statefulset/postgres-internal` | 67/73 (**6 fail** — Forgejo DB errors) | `PASS` (83 descriptive Forgejo `Error SQL Query ... dial tcp ...:5432: connection refused` logs) | 0 | Targets `forgejo` & pod `:3000` **DOWN**, **0** Prom alerts | **`nextcloud-workspace`** | READY |

### Group C: `external-sim` Dev Mock Fault Injection (10 Scenarios)

*Note: Initial unauthenticated pass failed `C1`–`C3` & `C5`–`C8` with `HTTP 401` due to Gap 5; the table below reflects the authenticated live pass (`HTTP 200` injection) plus the unauthenticated `C10` gate.*

| ID | Target (`external-sim`) | Fault Mode | Sim (Pass/Total) | Caller Log Grade | Loki | New Prom Alerts | New Gatus | Recovery |
| :-: | :--- | :--- | :--- | :--- | :-: | :--- | :--- | :--- |
| C1 | `mock-bank-dev` | `{"outage":true}` | 72/73 (**1 fail**: `arthur_link_new_bank_account` 503) | `PASS` (`customer-bll` logged `bank clearing failed (503)`; worker retried 3×, logged `Unavailable`, routed to `.dlq`) | 0 | None | None | FAULT_RESET |
| C2 | `mock-bank-dev` | `{"latency_ms":6500}` | 73/73 (0 sync fail; async ACH timed out at **5.00s** ×3) | `PASS` (`customer-bll` enforced exact `5s` deadline; worker logged `DeadlineExceeded: bank request timed out`) | 0 | None | None | FAULT_RESET |
| C3 | `mock-bank-dev` | `{"reject_transfers":true}` | 73/73 (0 sync fail; async ACH rejected `412` ×3) | `PASS` (`customer-bll` logged `bank API error (412): transfer rejected`; worker mapped `FailedPrecondition` & DLQ'd) | 0 | None | None | FAULT_RESET |
| C4 | `mock-bank-dev` | `scale=0` | 72/73 (**1 fail**: NGINX 503 HTML on link token) | `PASS` (`customer-bll` dial refused ×3; worker CWE-532-safe DLQ log with `event_id` & `correlation_id`) | 0 | None | None | READY |
| C5 | `mock-broker-dev` | `{"outage":true}` | 68/73 (**5 fail**: orders/quotes returned clean `{"error":"broker service unavailable"}`) | `PASS` (`customer-bll` logged descriptive `broker order execution failed` / market data errors; minor `"alpaca api error"` string in internal BLL log, **zero** vendor leaks in HTTP responses) | 0 | None | None | FAULT_RESET |
| C6 | `mock-broker-dev` | `{"latency_ms":6500}` | 63/73 (**10 fail**: `{"error":"broker request timed out"}` + portfolio & Chloe auth timeout cascade!) | `PASS` (`5s` deadline enforced on order; **cascade gap**: multi-call portfolio/auth path timed out client) | 0 | None | None | FAULT_RESET |
| C7 | `mock-broker-dev` | `{"reject_orders":true}` | 70/73 (**3 fail**: all orders returned `422` `{"error":"broker rejected request: order rejected by injected fault"}`) | `PASS` (`customer-bll` logged `(412): order rejected`; BFF returned clean `422` JSON) | 0 | Incidental `PostgresLowCacheHitRatio[pending]` | None | FAULT_RESET |
| C8 | `mock-broker-dev` | `{"halted_symbols":["NVDA","AAPL","MSFT","TSLA","AMZN"]}` | 71/73 (**2 fail**: Chloe NVDA/AAPL `422` halted; Arthur SPY **passed**) | `PASS` (exact per-symbol rejection logged & returned as structured `422` JSON) | 0 | Incidental `PostgresLowCacheHitRatio[pending]` | None | FAULT_RESET |
| C9 | `mock-broker-dev` | `scale=0` | 68/73 (**5 fail**: clean `{"error":"broker service unavailable"}`) | `PASS` (`customer-bll` logged exact dial refused URLs; clean `503` JSON to clients) | 0 | Incidental `PostgresLowCacheHitRatio[pending]` | None | READY |
| C10 | `mock-broker-dev` + `mock-bank-dev` | `-fault-scenarios` gate | **FAILED** (DNS lookup error + `401` auth error) | `GAP` (`driver_fault_patch.go` ignored `OCI_INGRESS_IP` and `/sim/faults` requires API auth headers) | 0 | Incidental `PostgresLowCacheHitRatio[pending]` | None | SUITE_DONE |

---

## 3. Detailed Gap Analysis

### 3.1 Gap 1 — Zero Kubernetes Pod Logs in Loki (`k8s/logging/promtail.yaml`)
- **Evidence:** `GET /loki/api/v1/labels` returned strictly `{"status":"success","data":["filename","job"]}`. Every single scenario (`A1`–`C10`) returned `0` streams for `{namespace=~"oci-dev|oci-internal|external-sim"}`.
- **Root Cause:** [`k8s/logging/promtail.yaml`](file:///home/ramjac/orange-cat-investments/k8s/logging/promtail.yaml#L15-L22) only scrapes static host `/var/log/*log` with no `kubernetes_sd_configs`, no RBAC `ServiceAccount`/`ClusterRole`, and no `/var/log/pods` mount. Meanwhile [`k8s/monitoring/alloy.yaml`](file:///home/ramjac/orange-cat-investments/k8s/monitoring/alloy.yaml) only collects OTLP metrics and traces.

### 3.2 Gap 2 — Prometheus Alerting & Scrape Blind Spots (`k8s/monitoring/prometheus.yaml`)
- **Evidence:** Scaling down 18 `oci-dev` workloads, 6 `oci-internal` workloads, and 2 `-dev` mocks triggered **zero** outage/availability/error-rate alerts in Prometheus.
- **Root Causes:**
  1. **No workload-availability rules:** `alert_rules.yml` has `PodCrashLoopBackOff`, `PodPending`, `OOMKilled`, but **no** `DeploymentReplicasZero` / `DeploymentReplicasMismatch` / `StatefulSetReplicasZero` (via `kube_deployment_status_replicas_available == 0` or `kube_statefulset_status_replicas_ready == 0`).
  2. **No scrape-target-down rule:** When `forgejo` (`B1`) and `nextcloud` (`B2`) went down, Prometheus marked their scrape targets `DOWN` (`up == 0`), yet no generic `TargetDown` (`up == 0`) rule exists.
  3. **Vanishing-pod blind spot (`MeilisearchDown`):** `meilisearch` uses `kubernetes_sd_configs: role: pod` filtered to `Running` pods. When scaled to `0`, the target disappears completely instead of emitting `up == 0`, so `MeilisearchDown` never fires.
  4. **No HTTP 5xx / gRPC error-rate / upstream-fault alerts:** Even though every Go service exports OTel HTTP/gRPC metrics to Alloy → Prometheus, there are zero PromQL alerts for elevated HTTP `5xx` rates, gRPC `Unavailable`/`DeadlineExceeded` rates, or external bank/broker failures.
  5. **No `oci-dev` RabbitMQ/ZITADEL scraping:** `rabbitmq` and `zitadel` scrape jobs hardcode `oci-core` FQDNs only, leaving `oci-dev`'s **4,200-message DLQ** and `oci-worker-dev` stagnant consumers completely invisible to `RabbitMQDeadLetterQueueNotEmpty` and `RabbitMQStagnantConsumers`.

### 3.3 Gap 3 — 10 of 18 Baseline Gatus Probes Permanently Red (`k8s/monitoring/gatus.yaml`)
- **Evidence:** At clean baseline, 10 of 18 endpoints in [`k8s/monitoring/gatus.yaml`](file:///home/ramjac/orange-cat-investments/k8s/monitoring/gatus.yaml) were failing:
  - **6 timeouts** (`customer-bll-health`, `internal-bll-health`, `workforce-api-health`, `facilities-api-health`, `core-invest-api-health`, `ops-api-health`) — `monitoring` namespace cannot reach those ports in `oci-core` (NetworkPolicy or port/protocol mismatch).
  - **3 DNS failures** (`customer-portal-spa`, `employee-portal-spa`, `static-marketing-site`) — external `https://*.oci.local` hostnames do not resolve in cluster CoreDNS; they should probe internal ClusterIP service URLs.
  - **1 HTTP 404** (`erpnext-frappe-hr` on `/api/method/ping`).
  - **Missing coverage:** `frappe-hr` has no probe, and no `oci-dev` or `-dev` mock probes exist.

### 3.4 Gap 4 — Silent Caller Logs & Internal Error Leakage in BFFs/BLLs
- **Missing Server-Side Error Logs:**
  - `oci-customer-bff` logged **0 errors** across 14 failures when `customer-bll` was down (`A7`).
  - `oci-employee-bff` logged **0 errors** across 57 failures when `internal-bll` was down (`A8`).
  - `internal-bll` logged **0 errors** across 11 `facilities-api` failures (`A10`) and 21 `ops-api` failures (`A12`).
- **Sensitive Internal Details Leaked to HTTP Callers:**
  - `A7` (`customer-bll` down): `{"error":"connection error: desc = \"transport: Error while dialing: dial tcp 10.152.183.181:8085: connect: connection refused\""}`
  - `A14` (`postgres-dev` down): `{"code":"Internal Server Error","message":"rpc error: code = Unknown desc = failed to connect to \`user=oci_admin database=oci_dev\`: ..."}` — leaks DB username (`oci_admin`) and database name (`oci_dev`) directly in HTTP 500 JSON!
  - `A12` (`ops-api` down): `barneby_slap_water_glass` returned HTTP **400** (instead of 503) with raw `dial tcp 10.152.183.189:50054: connect: connection refused`.
- **Minor Vendor Name in `customer-bll` Server Logs:** `C5`/`C7`/`C8` cleanly sanitized `"alpaca"` from all HTTP responses, while `customer-bll` structured logs still recorded `error="alpaca api error (...)"` from `pkg/broker/alpaca.APIError.Error()`.

### 3.5 Gap 5 — Fault Injection Auth & Transport Bugs (`7.17`)
- **Bug A:** Both [`cmd/mock-bank/auth.go`](file:///home/ramjac/orange-cat-investments/cmd/mock-bank/auth.go#L14) and [`cmd/mock-broker/auth.go`](file:///home/ramjac/orange-cat-investments/cmd/mock-broker/auth.go#L14) only exempt `/healthz` (not `/sim/faults`) from API credential validation, causing unauthenticated `PUT`/`DELETE /sim/faults` calls to fail with `HTTP 401`.
- **Bug B:** [`simulation/driver_fault_patch.go`](file:///home/ramjac/orange-cat-investments/simulation/driver_fault_patch.go#L16) constructs a bare `&http.Client{Timeout: 30 * time.Second}` instead of sharing `d.client`'s `OCI_INGRESS_IP`-aware `http.Transport`, causing `-fault-scenarios` to fail DNS lookup from external workstations without `/etc/hosts` entries.

### 3.6 Gap 6 — Silent Background ACH Clearing Failures (4,200 DLQ Messages) & SQL Constraint Violation
- **Evidence:** Live worker logs across nearly every scenario (even baseline!) showed:
  1. `customer-bll`: `bank clearing failed: bank API error (400): bank account not found (code=)` → `oci-worker-dev` retried 3× and dead-lettered into `events.banking.ach_transfer_requested.v1.dlq` (**4,200 ready messages** in `oci-dev` RabbitMQ!).
  2. `customer-bll`: `WARN failed to persist recurring deposit schedule in core-invest-api via gRPC: ERROR: new row for relation "recurring_deposits" violates check constraint "recurring_deposits_frequency_check" (SQLSTATE 23514)`.
- Both failures were **completely invisible** to the simulator (`73/73` passed) because the HTTP endpoint returned `201` before async clearing and swallowed the recurring-deposit persistence error as a `WARN`.

### 3.7 Gap 7 — Post-Restore Reconnect Carryover & Broker Latency Cascade
- **gRPC Backoff Carryover:** After `core-invest-api` (`A11`) and `ops-api` (`A12`) were restored to `1/1 Ready`, `internal-bll`'s gRPC client stayed in exponential reconnect backoff (`TRANSIENT_FAILURE`), causing initial actions in the next scenario (`A12`/`A13`) to fail with `connection refused`.
- **RabbitMQ Publisher Carryover:** After `rabbitmq` (`A16`) was restored, `customer-bll`'s AMQP publisher failed one `arthur_schedule_recurring_ach_deposit` event in `A17` before reconnecting.
- **Latency Cascade (`C6`):** Under `6500ms` mock-broker latency, Arthur's order hit the bounded `5s` timeout (`504`), but multi-call portfolio/login paths cascaded into client-side timeouts (`10` failures including Chloe's login).

### 3.8 Gap 8 — Unwired / Silently Masked COTS Dependencies (`B2`–`B5`)
- Bringing down `nextcloud` (`B2`), `snipe-it` (`B3`), `erpnext-frappe` (`B4`), and `frappe-hr` (`B5`) produced **73/73** simulator passes and **0** `internal-bll`/`employee-bff` error logs:
  - `nextcloud` (`B2`) confirmed `README.md` §7.23 (`NEXTCLOUD_ADMIN_USER`/`PASS` unset → silent in-memory fallback).
  - `snipe-it`, `erpnext-frappe`, and `frappe-hr` currently have no live failing simulator paths or degrade silently without logging errors.

---

## 4. Realism & Code-Verified Remediation Blueprint (Phase 7: `7.25`–`7.29`)

Every discovered gap was traced directly to its underlying Kubernetes manifest or Go source lines to confirm realistic, low-risk remediation:

| Item | Gap Addressed | Feasibility | Exact Code & Manifest Mechanics Verified |
| :-: | :--- | :---: | :--- |
| **`7.25`** | Zero K8s pod logs in Loki | **High (Single manifest)** | [`k8s/logging/promtail.yaml`](file:///home/ramjac/orange-cat-investments/k8s/logging/promtail.yaml) already runs `grafana/promtail:2.9.4` as a `DaemonSet`. Adding a `ServiceAccount` + read-only `ClusterRole`/`ClusterRoleBinding`, mounting `/var/log/pods` and `/var/log/containers` read-only, and adding a `kubernetes-pods` scrape job with `pipeline_stages: [- cri: {}]` immediately ships all container logs with `namespace`, `pod`, `container`, and `app` labels. |
| **`7.26`** | Zero Prometheus availability, error-rate, upstream, or `oci-dev` DLQ alerts | **High (Manifests + 1 OTel counter)** | Live Prometheus inspection confirmed `kube_deployment_status_replicas_available`, `kube_statefulset_status_replicas_ready`, and `http_server_request_duration_seconds_count` are **already exported** in Prometheus. Enabling `rabbitmq_prometheus` (`:15692`, `ConfigMap/rabbitmq-config`) in [`k8s/oci-dev/rabbitmq.yaml`](file:///home/ramjac/orange-cat-investments/k8s/oci-dev/rabbitmq.yaml) (matching [`k8s/oci-core/rabbitmq.yaml`](file:///home/ramjac/orange-cat-investments/k8s/oci-core/rabbitmq.yaml)) + adding `oci_upstream_requests_total` in [`pkg/telemetry/metrics.go`](file:///home/ramjac/orange-cat-investments/pkg/telemetry/metrics.go) + [`internal/app/customerbll/upstream.go`](file:///home/ramjac/orange-cat-investments/internal/app/customerbll/upstream.go) makes every outage and mock fault alertable. |
| **`7.27`** | 10 / 18 Gatus probes broken at baseline | **High (Manifests + NetworkPolicy)** | [`k8s/oci-core/bll-services.yaml`](file:///home/ramjac/orange-cat-investments/k8s/oci-core/bll-services.yaml) and [`k8s/oci-core/domain-apis.yaml`](file:///home/ramjac/orange-cat-investments/k8s/oci-core/domain-apis.yaml) run **mTLS gRPC** on `:8085`, `:8086`, and `:50051`–`:50054` (using `tcpSocket` readiness probes; no HTTP `:8080/healthz` listener exists on Domain APIs) and block `monitoring` via `NetworkPolicy`. Switching those 6 Gatus checks to `tcp://...:8085|8086|50051..50054` (`[CONNECTED] == true`), allowing `monitoring/gatus` in the `oci-core` `NetworkPolicy` ingress rules, and switching the 3 SPA/marketing checks from external `https://*.oci.local` to in-cluster ClusterIP URLs fixes all 10 broken baseline probes. |
| **`7.28`** | Silent BFF/BLL caller logs, IP/DB leak in HTTP responses, & `15s` `GetPortfolio` timeout cascade | **High (Go handlers)** | 1. [`internal/app/customerbff/server.go`](file:///home/ramjac/orange-cat-investments/internal/app/customerbff/server.go#L103) (`writeGRPCError`) and [`portfolio_handlers.go`](file:///home/ramjac/orange-cat-investments/internal/app/customerbff/portfolio_handlers.go#L57) (`handleGetPortfolio`, which currently turns BLL `Unavailable` into `404 portfolio not found`) need structured `slog.Error` logging and client message sanitization so `dial tcp ...` and `user=oci_admin database=oci_dev` never leak to HTTP clients.<br>2. [`internal/app/customerbff/auth_handlers.go`](file:///home/ramjac/orange-cat-investments/internal/app/customerbff/auth_handlers.go#L58) (`handleLogin`) calls `GetPortfolio` on every login even when `req.Name != ""`, and [`internal/app/customerbll/brokerage.go`](file:///home/ramjac/orange-cat-investments/internal/app/customerbll/brokerage.go#L31-L56) chains **three sequential `5s` timeouts** (`ListAccounts` → `GetAccount` → `bPositions` = `15s`) when `mock-broker-dev` is slow (`C6`). Skipping `GetPortfolio` when `name != ""` and short-circuiting on the first broker timeout eliminates the cascade. |
| **`7.29`** | `/sim/faults` `401` + DNS failure, 4,200-msg ACH DLQ backlog, & `SQLSTATE 23514` | **High (Small targeted Go fixes)** | 1. Exempt `r.URL.Path == "/sim/faults"` in [`cmd/mock-bank/auth.go`](file:///home/ramjac/orange-cat-investments/cmd/mock-bank/auth.go#L14) and [`cmd/mock-broker/auth.go`](file:///home/ramjac/orange-cat-investments/cmd/mock-broker/auth.go#L14) and reuse `d.client`'s `OCI_INGRESS_IP` transport in [`simulation/driver_fault_patch.go`](file:///home/ramjac/orange-cat-investments/simulation/driver_fault_patch.go#L16).<br>2. [`internal/app/customerbff/portfolio_handlers.go`](file:///home/ramjac/orange-cat-investments/internal/app/customerbff/portfolio_handlers.go#L151-L156) (`handleCreateDeposit`) decodes `req.BankAccount` and `req.AccountID` and **drops both** when constructing `InitiateACHDepositRequest`; [`internal/app/customerbll/banking.go`](file:///home/ramjac/orange-cat-investments/internal/app/customerbll/banking.go#L46-L104) resolves `accountID` and **never uses it** (`achPayload["bank_account"]` gets `in.BankAccount`), and calls `CreateRecurringDeposit` even when `freq == "one_time"` (`chloe_margin_deposit_nsf_check`), violating `CHECK (frequency IN ('weekly','biweekly','monthly'))` ([`scripts/init.sql`](file:///home/ramjac/orange-cat-investments/scripts/init.sql#L472)). Forwarding `BankAccount`/`AccountID`, using the resolved `accountID` (`acc_arthur_chase_chk` / `acc_chloe_svcu_chk`), and only calling `CreateRecurringDeposit` for recurring frequencies fixes both bugs cleanly. |
