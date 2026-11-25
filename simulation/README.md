# Orange Cat Investments (OCI) Platform Simulator

The **OCI Stack Simulator** is a Go-based driver application designed to fully exercise the Orange Cat Investments software and hardware infrastructure stack. By emulating the daily operational workflows of OCI's human staff and feline executives, the simulator drives end-to-end API calls, event triggers, domain sagas, hardware asset management, quantitative backtesting, and wearable alert proxy workflows.

---

## Architecture Overview

The simulator architecture mirrors OCI's Domain-Driven Design (DDD) model:

```text
simulation/
├── config.go            # Configuration flags (URLs, modes, iteration, delays)
├── persona.go           # Persona and Action interface definitions & Registry
├── client.go            # HTTP client abstraction driving running OCI stack services
├── actions.go           # Persona action suites for all 13 OCI personas
├── driver.go            # Orchestration engine for running simulation workflows
├── driver_test.go       # Tests verifying simulation execution
└── cmd/
    └── simulator/       # Runnable CLI entrypoint (`go run ./simulation/cmd/simulator`)
        └── main.go
```

---

## Employee Personas & Daily Simulation Action Mapping

The simulator uses the 13 official OCI personas defined in `personas/` to execute semi-real daily action lists:

### 1. Garfield (`emp-feline-garfield`) — Chief Observation Officer (COO)
* **`garfield_morning_zoomies`**: Emits real-time observation telemetry during 3 AM peak activity windows via optical IP camera streams (`/facilities/assets?asset_type=edge_camera`).
* **`garfield_perch_inspection`**: Inspects primary habitat infrastructure (`/facilities/assets?status=active`) for operational readiness.
* **`garfield_care_schedule_audit`**: Queries care schedules (`/api/v1/workforce/care-schedules/emp-feline-garfield`) to ensure punctual lasagna and snack disbursements.
* **`garfield_sleep_in_hallway`**: Naps in hallway B; 1/3 chance of tripping a human employee, placing Garfield on vet medical hold (`POST /api/v1/workforce/care-schedules/emp-feline-garfield/medical-hold`) and logging a workplace injury ticket (`POST /ops/tickets`).

### 2. Barneby (`emp-feline-barneby`) — Senior Alpha Perch Analyst
* **`barneby_sunbeam_tracking`**: Queries habitat optical camera streams (`/facilities/assets?asset_type=edge_camera`) to track sunbeam positioning.
* **`barneby_care_schedule_review`**: Reviews secondary perch health and care assessment parameters (`/api/v1/workforce/care-schedules/emp-feline-barneby`).
* **`barneby_slap_water_glass`**: Slaps a water glass off a desk, creating a facilities maintenance ticket (`POST /facilities/assets/{id}/maintenance`), querying Pebble watch alerts (`GET /ops/pebble/alerts`), submitting a Pebble watch ACK (`POST /ops/pebble/ack`), and logging completed maintenance work via mobile field app sync (`POST /facilities/sync/maintenance-logs`).

### 3. Alice Vance (`emp-human-alice`) — Head of Human & Feline Resources
* **`alice_workforce_directory`**: Queries combined human and feline workforce employee directory (`/api/v1/workforce/employees`).
* **`alice_trigger_onboarding_saga`**: Simulates Frappe HR status changes (`/api/v1/webhooks/frappe-hr`) triggering the Go `OnboardingSaga` across workforce, Forgejo, and internal service planes.
* **`alice_review_leave_requests`**: Audits pending staff leave requests and feline catnip break compliance (`/api/v1/workforce/leave-requests`).
* **`alice_approve_timesheet`**: Reviews pending workforce timesheets and submits manager approvals (`PUT /api/v1/workforce/timesheets/{id}/status`), synchronizing DocType Timesheets to Frappe HR.
* **`alice_process_payroll_entry`**: Submits monthly payroll run (`POST /api/v1/workforce/payroll/run`) for Frappe HR salary slip generation and compensation distribution.
* **`alice_offboard_employee`**: Triggers automated deprovisioning engine (`POST /api/v1/workforce/employees/{id}/offboard`), invalidating sessions, archiving storage, and generating Snipe-IT equipment recovery tickets.
* **`alice_assign_laptop_to_rick`**: Provisions a new laptop hardware asset assigned to new hire Rick during onboarding (`POST /facilities/assets`).
* **`alice_publish_onboarding_doc`**: Publishes the employee onboarding handbook in Nextcloud (`POST /nextcloud/api/v1/documents`) and notifies Rick over Nextcloud Chat (`POST /nextcloud/api/v1/chat/messages`).

### 4. Bob Builder (`emp-human-bob`) — Lead Facilities & Edge Telemetry Engineer
* **`bob_scan_edge_cameras`**: Scans optical edge camera rigs across habitat zones (`/facilities/assets?asset_type=edge_camera`).
* **`bob_register_camera_rig`**: Registers a new 4K edge camera asset (`POST /facilities/assets`).
* **`bob_verify_pebble_wearables`**: Verifies assigned Pebble watch hardware assets for on-call alerting (`GET /facilities/assets?asset_type=pebble_watch`).
* **`bob_log_daily_timesheet`**: Submits daily 8-hour shift timesheet for habitat facilities maintenance (`POST /api/v1/workforce/timesheets`) with billable cost center allocation.
* **`bob_view_paystubs`**: Queries historical salary slips and itemized paystubs (`GET /api/v1/workforce/payroll/slips?employee_id=emp-human-bob`) for compensation review.

### 5. Carol Danvers (`emp-human-carol`) — Chief Investment Officer (CIO)
* **`carol_query_brokerage_orders`**: Audits execution logs of automated brokerage trade orders (`/core-invest/brokerage/orders`).
* **`carol_execute_automated_trade`**: Executes quantitative equity buy/sell orders triggered by feline activity multipliers (`POST /core-invest/brokerage/orders`).

### 6. David Quant (`emp-human-david`) — Feline Behavioral Data Scientist
* **`david_register_stream`**: Registers optical camera streams (`/core-invest/streams`) for computer vision model inference.
* **`david_initiate_backtest`**: Runs quantitative backtesting models (`/core-invest/backtesting/runs`) comparing feline activity event logs against asset performance metrics.

### 7. Dr. Elena Rostova (`emp-human-elena`) — Chief Veterinary Officer & Habitat Care Specialist
* **`elena_update_garfield_care_schedule`**: Dynamically updates feline dietary plans and feeding times (`PUT /api/v1/workforce/care-schedules/emp-feline-garfield`).
* **`elena_toggle_medical_hold`**: Verifies emergency medical trading holds (`POST /api/v1/workforce/care-schedules/emp-feline-garfield/medical-hold`) to pause automated trading when rest is required.
* **`elena_review_health_assessments`**: Audits scheduled feline health assessments (`GET /api/v1/workforce/review-cycles?review_type=feline_health_assessment`).
* **`elena_conduct_wellness_audit`**: Submits clinical wellness audits via Employee BFF.

### 8. Frank Operations (`emp-human-frank`) — Platform Security & K8s Infrastructure Lead
* **`frank_monitor_it_tickets`**: Monitors IT helpdesk tickets (`/ops/tickets`) originating from Forgejo issues or automated cluster alerts.
* **`frank_query_pebble_alerts`**: Queries wearable watch alert payloads (`/ops/pebble/alerts`) formatted for Pebble AppMessage protocol.
* **`frank_create_it_ticket`**: Submits infrastructure support tickets (`POST /ops/tickets`) for intermediate CA key rotations and dynamic SPIFFE ID mTLS reloads.
* **`frank_publish_security_doc`**: Writes zero-trust security architecture document in Nextcloud (`POST /nextcloud/api/v1/documents`) and alerts developers via Nextcloud Chat (`POST /nextcloud/api/v1/chat/messages`).
* **`frank_audit_gateway_routes`**: Audits Gateway API `HTTPRoute` responses (`dev.employee-bff.oci.local`, `dev.customer-bff.oci.local`, `dev.api.oci.local`).
* **`frank_waf_security_audit`**: Sends OWASP CRS SQLi/XSS/LFI attack probes to BFF endpoints and asserts `HTTP 403 Forbidden` (`WAF_RULE_VIOLATION`).
* **`frank_verify_audit_chain`**: Queries `GET /api/v1/ops/audit-logs/verify` and asserts cryptographic hash-chain integrity (`valid == true`).

### 9. Arthur Pendelton (`cust-longterm-arthur`) — Long-Term Value Investor (Customer)
* **`arthur_review_performance_reports`**: Audits historical quantitative backtest performance and strategy returns.
* **`arthur_deposit_investment_capital`**: Executes steady capital deposit orders into the OCI Alpha Growth Fund (`POST /api/v1/customer/trading/orders` with `Idempotency-Key`).
* **`arthur_monitor_feline_stream`**: Checks optical camera streams (`GET /api/v1/customer/streams`) and `coturn` ICE server discovery (`GET /api/v1/customer/webrtc/ice-servers`).
* **`arthur_schedule_recurring_ach_deposit`**, **`arthur_audit_linked_bank_accounts`**, **`arthur_verify_portfolio_holdings`**, **`arthur_link_new_bank_account`**, **`arthur_submit_customer_support_ticket`**, **`arthur_self_register_onboarding`**, **`arthur_generate_monthly_investment_statement`**, **`arthur_generate_tax_form_1099b`**: Exercise recurring ACH deposits, bank account linking, portfolio history/holdings, customer support, self-registration, monthly PDF statements, and Form 1099-B generation via `/api/v1/customer/*`.

### 10. Chloe Spark (`cust-active-chloe`) — Momentum Alpha Trader (Customer)
* **`chloe_scan_activity_streams`**: Scans real-time camera streams (`GET /api/v1/customer/streams`) and Zoomie Index (`GET /api/v1/customer/zoomie-index`).
* **`chloe_momentum_buy_order`**: Executes aggressive momentum buy orders (`POST /api/v1/customer/trading/orders` with `Idempotency-Key`).
* **`chloe_idempotent_order_replay`**: Submits an order twice with the same UUID `Idempotency-Key` and verifies `X-Idempotent-Replayed: true` with identical response body.
* **`chloe_take_profit_sell_order`**: Executes tactical take-profit sell orders (`POST /api/v1/customer/trading/orders`).
* **`chloe_audit_execution_log`**, **`chloe_query_live_market_quotes`**, **`chloe_margin_deposit_nsf_check`**, **`chloe_close_account_and_liquidate`**, **`chloe_request_gdpr_data_erasure`**: Verify trade execution logs, market quotes, NSF R01 validation, account liquidation, and GDPR Article 17 erasure + Valkey session purge.

### 11. Rick Newhire (`emp-human-rick`) — Junior Operations Associate
* **`rick_onboarding_checklist`**: Verifies workforce directory status during new hire onboarding (`GET /api/v1/workforce/employees`).
* **`rick_read_nextcloud_docs`**: Reads shared onboarding guides and security standard documentation in Nextcloud (`GET /nextcloud/api/v1/documents`).
* **`rick_send_nextcloud_chat_ack`**: Sends Nextcloud Chat message acknowledging receipt of assigned laptop and onboarding documents (`POST /nextcloud/api/v1/chat/messages`).
* **`rick_check_onboarding_status`**: Queries employee onboarding status (`GET /api/v1/workforce/employees?status=onboarding`) to track provisioning progress.
* **`rick_trigger_self_onboarding_webhook`**: Simulates Frappe HR onboarding saga webhooks (`POST /api/v1/webhooks/frappe-hr`) to exercise new hire lifecycle workflows.
* **`rick_unauthorized_medical_hold_denied`**: Negative OPA authorization check asserting a non-veterinary role receives `HTTP 403 Forbidden` when attempting to toggle a feline medical hold.

### 12. Elise Dev (`emp-human-elise`) — Software Developer
* **`elise_claim_kanban_card`**: Claims and assigns Card #42 on the engineering backlog kanban board to herself (`PATCH /api/v1/repos/oci/monorepo/issues/42`).
* **`elise_create_feature_branch`**: Creates feature branch `feature/42-gitops-promotion-pipeline` from `main` via direct Forgejo API (`POST /api/v1/repos/oci/monorepo/branches`).
* **`elise_push_code_commit`**: Pushes git commit to feature branch with multi-stage Dockerfile and promotion pipeline manifests (`POST /api/v1/repos/oci/monorepo/contents/Dockerfile`).
* **`elise_open_pull_request`**: Opens Pull Request #42 in Forgejo linking Card #42 for automated CI validation and peer review (`POST /api/v1/repos/oci/monorepo/pulls`).
* **`elise_write_nextcloud_doc`**: Publishes Employee Portal Vue 3 & BFF architecture specification document in Nextcloud (`POST /nextcloud/api/v1/documents`).
* **`elise_send_nextcloud_chat`**: Messages Rick and engineering team via Nextcloud Chat with technical document links (`POST /nextcloud/api/v1/chat/messages`).
* **`elise_monitor_it_tickets`**: Monitors IT helpdesk ticket queue for developer tooling issues (`GET /ops/tickets`).
* **`elise_create_code_review_ticket`**: Submits Forgejo code review & CI/CD workflow tickets (`POST /ops/tickets`) for software development and automated pipeline builds.
* **`elise_query_developer_tickets`**: Audits active developer IT tickets and build issues (`GET /ops/tickets`).
* **`elise_submit_habitat_helpdesk_ticket`**: Submits an urgent priority habitat IT helpdesk ticket (`POST /api/v1/ops/tickets`).

### 13. Marcus Lead (`emp-human-marcus`) — Principal Systems Architect & Senior Developer
* **`senior_dev_author_adr`**: Authors and publishes Architecture Decision Records (ADR 0003) in Nextcloud (`POST /nextcloud/api/v1/documents`).
* **`senior_dev_review_pull_request`**: Performs architectural peer review and approves Elise Dev's Pull Request #42 via direct Forgejo API (`POST /api/v1/repos/oci/monorepo/pulls/42/reviews`).
* **`senior_dev_merge_pull_request`**: Merges approved Pull Request #42 into `main` branch via direct Forgejo API (`POST /api/v1/repos/oci/monorepo/pulls/42/merge`), closing Card #42 and triggering promotion pipeline.
* **`senior_dev_audit_ci_pipeline`**: Audits active developer tickets and CI/CD workflow pipeline status across platform deployments (`GET /ops/tickets`).

---

## Execution Modes

The simulator supports three execution modes:

1. **`live` (Default)**: Connects directly to deployed or running OCI stack services (e.g. Employee BFF on port `8081` and Customer BFF on port `8083`). Acts as a true end-to-end integration test catching runtime defects.
2. **`step`**: Runs a single step-by-step pass through each registered persona's actions and prints action results.
3. **`continuous`**: Loops persona daily action lists continuously with configured action delays (`-delay`), simulating real-time continuous business operations until interrupted (`Ctrl+C`).

---

## How to Run the Simulator

### 1. List All Registered Personas
```bash
go run ./simulation/cmd/simulator -list-personas
```

### 2. Run Simulation Drive Against Running Stack (Default)
```bash
go run ./simulation/cmd/simulator
```

### 3. Target a Specific Persona (e.g., Garfield or Rick)
```bash
go run ./simulation/cmd/simulator -persona emp-human-rick
```

### 4. Drive Live Local / MicroK8s OCI Stack
Target staging or local cluster directly:
```bash
# Target Kubernetes oci-dev staging environment with explicit URLs
go run ./simulation/cmd/simulator -mode live -bff-url https://dev.employee-bff.oci.local -cust-bff-url https://dev.customer-bff.oci.local

# Target local services with explicit service flags
go run ./simulation/cmd/simulator \
  -mode live \
  -bff-url http://localhost:8081 \
  -cust-bff-url http://localhost:8083 \
  -iterations 3 \
  -delay 200
```

### 5. CLI Flag Reference
| Flag | Default | Description |
|---|---|---|
| `-mode` | `live` | Execution mode: `live`, `step`, or `continuous` |
| `-persona` | `all` | Target persona ID (`emp-feline-garfield`, `emp-human-marcus`, etc.) or `all` |
| `-bff-url` | `""` (→ `http://localhost:8081`) | Target URL for OCI Employee BFF |
| `-cust-bff-url`| `""` (→ `http://localhost:8083`) | Target URL for OCI Customer BFF |
| `-forgejo-url` | `""` (→ `http://localhost:3000`) | Target URL for Forgejo Git repository host |
| `-broker-url`  | `""` (→ `http://localhost:8089`) | Target URL for Mock Broker API |
| `-bank-url`    | `""` (→ `http://localhost:8088`) | Target URL for Mock Bank API |
| `-meili-url`   | `""` (→ `MEILISEARCH_URL` / `http://localhost:7700`) | Meilisearch base URL used when `-seed-search` is set |
| `-seed-search` | `false` | Seed Meilisearch development fixtures before running simulation |
| `-tokens-file` | `""` | Path to saved tokens and session state JSON file |
| `-forgejo-token` | `""` | General Forgejo REST API token override |
| `-forgejo-elise-token` | `""` | Forgejo REST API token override for Elise Dev |
| `-forgejo-marcus-token` | `""` | Forgejo REST API token override for Marcus Lead |
| `-iterations` | `1` | Number of daily action loops to perform |
| `-delay` | `100` | Delay between persona action executions in milliseconds |
| `-verbose` | `true` | Enable detailed logging of request/response outputs |
| `-list-personas`| `false` | Print registered personas and exit |
| `-fault-scenarios` | `false` | Inject faults into `-dev` mocks (`/sim/faults`), assert BFF `503`/`504`/`422` handling, and reset |
| `-validate-flags` | `false` | Validate CLI flags and exit immediately (used by CI contract checks) |

### 6. Environment Variables
| Variable | Purpose |
|---|---|
| `OCI_INGRESS_IP` | Routes `*.oci.local` requests directly to the target cluster ingress IP without editing `/etc/hosts` |
| `FORGEJO_ELISE_TOKEN` | Personal Access Token (`--scopes all`) for `emp-human-elise` |
| `FORGEJO_MARCUS_TOKEN` | Personal Access Token (`--scopes all`) for `emp-human-marcus` |
| `MEILISEARCH_URL` / `MEILI_MASTER_KEY` | Meilisearch coordinates when running with `-seed-search` |

---

## Verification & Testing

To run the simulator unit test suite:

```bash
go test -v ./simulation/...
```
