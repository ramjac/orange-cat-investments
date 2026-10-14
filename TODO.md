# OCI Platform Technical Debt & Hardcoded Data Remediation

This document tracks instances of pre-existing hardcoded customer, employee, and domain entity data located outside of PR #77 that need to be refactored to comply with the OCI data architecture and domain boundary standards defined in [AGENTS.md](file:///home/ramjac/orange-cat-investments/AGENTS.md).

---

## 1. Customer BFF (`cmd/customer-bff/main.go`)

- [ ] **In-Memory Dev Session Fixtures (`NewInMemorySessionStore`)**:
  - **Location:** `cmd/customer-bff/main.go:55-68`
  - **Issue:** Statically registers `sess-arthur-token` (`cust-longterm-arthur`, "Arthur Pendelton") and `sess-chloe-token` (`cust-active-chloe`, "Chloe Spark") directly in memory upon startup.
  - **Remediation:** Remove hardcoded demo tokens from default startup. For local developer testing, guard session initialization behind an explicit configuration flag (`ENABLE_DEV_MOCK_AUTH=true`) or load temporary session fixtures from an external seed config.
- [ ] **Dev Login Endpoint (`handleDevLogin`)**:
  - **Location:** `cmd/customer-bff/main.go:319-335`
  - **Issue:** Switches on `req.Persona == "chloe"` or `cust-active-chloe` vs defaulting to `cust-longterm-arthur` / "Arthur Pendelton".
  - **Remediation:** Replace dev login with OIDC authentication via ZITADEL (`/api/v1/auth/callback`), or query `workforce`/`core_invest` database entities dynamically rather than relying on persona string literals.
- [ ] **Fallback Portfolio Responses (Non-Broker Mode in `handleGetPortfolio`)**:
  - **Location:** `cmd/customer-bff/main.go:477-535`
  - **Issue:** When the external Alpaca Broker API is not configured (`b.alpaca == nil`), static holdings, valuation histories, and account names are returned for Chloe or Arthur.
  - **Remediation:** Query `core_invest.portfolios` and `core_invest.portfolio_holdings` database tables as the fallback system of record rather than returning static JSON payload templates.
- [ ] **API Keys & Webhooks Fallback (`handleGetAPIKeys` & `handleRegisterWebhook`)**:
  - **Location:** `cmd/customer-bff/main.go:855-875`, `915-940`
  - **Issue:** `handleGetAPIKeys` returns a hardcoded bot name `"Chloe Momentum Bot"`, and `handleRegisterWebhook` uses default target URL `https://quant-bot.chloespark.io/api/v1/oci-callback`.
  - **Remediation:** Persist customer API keys and registered webhooks in PostgreSQL tables (`core_invest.api_keys`, `core_invest.customer_webhooks`) with ownership bound to `session.CustomerID`.
- [ ] **Order Creation Fallback (`handleCreateOrder`)**:
  - **Location:** `cmd/customer-bff/main.go:580-610`
  - **Issue:** Falls back to default portfolio IDs when `req.PortfolioID` is not supplied and external broker is inactive.
  - **Remediation:** Require explicit `portfolio_id` parameter or query the database for the active portfolio associated with the authenticated customer session.

---

## 2. Operations Service (`internal/service/ops/service.go`)

- [ ] **Default On-Call Engineer Fallback (`ProcessPebbleAck`)**:
  - **Location:** `internal/service/ops/service.go:135`
  - **Issue:** Falls back to `ackBy = "emp-human-frank"` when the acknowledging user is not provided.
  - **Remediation:** Require an authenticated employee session token or resolve the on-call user identity dynamically from the Pebble watch paired device ID registered in `ops.pebble_pairings`.

---

## 3. In-Memory Mock Repositories (`internal/repository/...`)

- [ ] **Workforce Repository In-Memory Fallback**:
  - **Location:** `internal/repository/workforce/repository.go:175-410`
  - **Issue:** When the PostgreSQL connection pool is `nil`, the repository falls back to an internal map seeded with hardcoded employees (Garfield, Barneby, Alice, Elena, Rick, Elise, Frank, Bob) and their care schedules.
  - **Remediation:** Separate in-memory mock implementations into dedicated test-only files (`internal/repository/workforce/mock.go`) and ensure production server binaries require a healthy database connection instead of falling back to in-memory mocks.
- [ ] **Operations In-Memory Repository Fallback**:
  - **Location:** `internal/repository/ops/repository.go:67, 111, 211, 247`
  - **Issue:** Hardcodes `"emp-human-frank"` as the author for mock IT tickets in fallback mode.
  - **Remediation:** Move fallback data into explicit unit test fixtures.
- [ ] **Facilities Mock Repository**:
  - **Location:** `internal/repository/facilities/mock.go:31-33, 89-91`
  - **Issue:** Seeds `"emp-human-frank"`, `"emp-human-bob"`, and `"emp-human-rick"` as reporters and assignees.
  - **Remediation:** Restrict mock repository usage strictly to unit/integration testing packages.

---

## 4. Policy Reminder

Per [AGENTS.md](file:///home/ramjac/orange-cat-investments/AGENTS.md), hardcoded customer and employee data is strictly forbidden in production code deployed to `oci-core`. Hardcoding is permitted only in:
1. Database seed scripts (`scripts/init.sql`, `scripts/0*-*.sql`)
2. Test suites (`*_test.go`)
3. Simulation suite (`simulation/...`)
4. Isolated mock external services (`cmd/mock-broker/...`, `cmd/mock-bank/...`)
