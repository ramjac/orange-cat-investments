# OCI Platform Technical Debt & Hardcoded Data Remediation

This document tracks instances of pre-existing hardcoded customer, employee, and domain entity data located outside of PR #77 that need to be refactored to comply with the OCI data architecture and domain boundary standards defined in [AGENTS.md](file:///home/ramjac/orange-cat-investments/AGENTS.md).

---

## 1. Customer BFF (`cmd/customer-bff/main.go`)

- [x] **In-Memory Dev Session Fixtures (`NewInMemorySessionStore`)**:
  - **Location:** `cmd/customer-bff/main.go:55-68`
  - **Issue:** Statically registers `sess-arthur-token` (`cust-longterm-arthur`, "Arthur Pendelton") and `sess-chloe-token` (`cust-active-chloe`, "Chloe Spark") directly in memory upon startup.
  - **Remediation:** (Completed) Removed hardcoded demo tokens from default startup. Mock sessions are gated strictly behind `ENABLE_DEV_MOCK_AUTH=true` or `SEED_DEV_SESSIONS=true`.
- [x] **Dev Login Endpoint (`handleDevLogin`)**:
  - **Location:** `cmd/customer-bff/main.go:319-335`
  - **Issue:** Switches on `req.Persona == "chloe"` or `cust-active-chloe` vs defaulting to `cust-longterm-arthur` / "Arthur Pendelton".
  - **Remediation:** (Completed) Replaced static switch with dynamic customer ID, name, and email resolution with strict validation and database lookup.
- [x] **Fallback Portfolio Responses (Non-Broker Mode in `handleGetPortfolio`)**:
  - **Location:** `cmd/customer-bff/main.go:477-535`
  - **Issue:** When the external Alpaca Broker API is not configured (`b.alpaca == nil`), static holdings, valuation histories, and account names are returned for Chloe or Arthur.
  - **Remediation:** (Completed) Queries `core_invest.investment_portfolios` via `ciSvc.GetPortfolio` as authoritative fallback persistence instead of returning static JSON payload templates.
- [x] **API Keys & Webhooks Fallback (`handleGetAPIKeys` & `handleRegisterWebhook`)**:
  - **Location:** `cmd/customer-bff/main.go:855-875`, `915-940`
  - **Issue:** `handleGetAPIKeys` returns a hardcoded bot name `"Chloe Momentum Bot"`, and `handleRegisterWebhook` uses default target URL `https://quant-bot.chloespark.io/api/v1/oci-callback`.
  - **Remediation:** (Completed) Persists customer API keys and webhooks in PostgreSQL tables `core_invest.customer_api_keys` and `core_invest.webhook_subscriptions` scoped to `session.CustomerID`.
- [x] **Order Creation Fallback (`handleCreateOrder`)**:
  - **Location:** `cmd/customer-bff/main.go:580-610`
  - **Issue:** Falls back to default portfolio IDs when `req.PortfolioID` is not supplied and external broker is inactive.
  - **Remediation:** (Completed) Dynamically queries active portfolio associated with authenticated customer session via `ciSvc.GetPortfolio`.

---

## 2. Operations Service (`internal/service/ops/service.go`)

- [x] **Default On-Call Engineer Fallback (`ProcessPebbleAck`)**:
  - **Location:** `internal/service/ops/service.go:135`
  - **Issue:** Falls back to `ackBy = "emp-human-frank"` when the acknowledging user is not provided.
  - **Remediation:** (Completed) Eliminates hardcoded fallback to "emp-human-frank". Dynamically resolves on-call identity via request context (`employee_id`), `req.WatchID`, `req.DeviceID`, or environment configuration.

---

## 3. In-Memory Mock Repositories (`internal/repository/...`)

- [x] **Workforce Repository In-Memory Fallback**:
  - **Location:** `internal/repository/workforce/repository.go:175-410`
  - **Issue:** When the PostgreSQL connection pool is `nil`, the repository falls back to an internal map seeded with hardcoded employees (Garfield, Barneby, Alice, Elena, Rick, Elise, Frank, Bob) and their care schedules.
  - **Remediation:** (Completed) Decoupled in-memory fallback into dedicated `internal/repository/workforce/mock.go` with `NewMockRepository()`. Cleaned `pgxRepository` to strictly use PostgreSQL `pgxpool.Pool`. Production servers fail fast if DB connection fails in production mode.
- [x] **Operations In-Memory Repository Fallback**:
  - **Location:** `internal/repository/ops/repository.go:67, 111, 211, 247`
  - **Issue:** Hardcodes `"emp-human-frank"` as the author for mock IT tickets in fallback mode.
  - **Remediation:** (Completed) Decoupled fallback into `internal/repository/ops/mock.go` with `NewMockRepository()`. `pgxRepository` in `repository.go` executes queries strictly against `ops.it_tickets`.
- [x] **Facilities Mock Repository**:
  - **Location:** `internal/repository/facilities/mock.go:31-33, 89-91`
  - **Issue:** Seeds `"emp-human-frank"`, `"emp-human-bob"`, and `"emp-human-rick"` as reporters and assignees.
  - **Remediation:** (Completed) Confined mock repository to unit/integration testing and local dev fallback; production servers fail fast without a valid database connection.

---

## 4. Core Investment Service (`internal/service/core_invest/service.go`)

- [x] **Simulated Trade Execution Architecture**:
  - **Location:** `internal/service/core_invest/service.go:137`
  - **Context:** Trade orders are routed through the `BrokerClient` interface (Faux Alpaca / Mock Broker API) in simulated environments.
  - **Rationale:** Direct integration with real-world FIX/REST institutional brokerage protocols involves external dependencies, credentials, and proprietary session state not present in local/test execution contexts. The `BrokerClient` abstraction isolates market simulation from domain logic. Verified compliant with [AGENTS.md](file:///home/ramjac/orange-cat-investments/AGENTS.md).

---

## 5. Policy Reminder

Per [AGENTS.md](file:///home/ramjac/orange-cat-investments/AGENTS.md), hardcoded customer and employee data is strictly forbidden in production code deployed to `oci-core`. Hardcoding is permitted only in:
1. Database seed scripts (`scripts/init.sql`, `scripts/0*-*.sql`)
2. Test suites (`*_test.go`)
3. Simulation suite (`simulation/...`)
4. Isolated mock external services (`cmd/mock-broker/...`, `cmd/mock-bank/...`)
