# OCI Synthetic Simulation Driver

The OCI platform relies on a headless, autonomous simulation driver to emulate real-world business activities across the company's four primary domain APIs (`workforce`, `facilities`, `core_invest`, `ops`) and the integrated COTS ecosystem (Forgejo, Nextcloud, Snipe-IT).

Instead of writing brittle end-to-end UI tests, Orange Cat Investments validates system correctness, database locking, and eventual consistency using concurrent programmatic personas executing complex saga workflows in a live environment.

## The 13 Business Personas

The simulator operates by assuming the identities of 13 registered corporate personas (both human and feline). Each persona possesses a role-specific daily action list, randomized execution paths, and specific behavioral intent.

### 1. Garfield (`emp-feline-garfield`) — Chief Executive Officer (CEO)
* **`garfield_sleep_in_hallway`**: Simulates the feline CEO sleeping in critical human walkways, generating high-priority `habitat_hazard` infrastructure tickets (`POST /ops/tickets`).
* **`garfield_request_premium_tuna`**: Automatically denies cheap cat food and requests premium `Bluefin` tuna via API (`PUT /api/v1/workforce/care-schedules/emp-feline-garfield`).
* **`garfield_review_performance`**: Conducts a random performance review of human subordinates (`GET /api/v1/workforce/review-cycles`), grading them strictly based on treat delivery latency.

### 2. Barneby (`emp-feline-barneby`) — Chief Financial Officer (CFO)
* **`barneby_slap_water_glass`**: Synthetically models the feline CFO knocking a water glass onto the main operations server, triggering severe cascading system alerts (`POST /ops/tickets`, `GET /ops/pebble/alerts`, `POST /ops/pebble/ack`, `POST /facilities/sync/maintenance-logs`).
* **`barneby_audit_brokerage_trades`**: Audits the mock stock broker trade ledger to ensure OCI Alpha Growth fund liquidity targets are met (`GET /core-invest/brokerage/orders`).
* **`barneby_sleep_on_keyboard`**: Halts all active CI/CD pipeline deployments by taking a nap on the production terminal (`POST /ops/tickets`).

### 3. Alice Human (`emp-human-alice`) — Human Resources Director
* **`alice_approve_timesheets`**: Queries pending contractor timesheets (`GET /api/v1/workforce/timesheets?status=submitted`) and issues bulk approvals (`PUT /api/v1/workforce/timesheets`).
* **`alice_generate_executive_report`**: Compiles the monthly TPS (Treat Performance Standard) report for the feline executive board (`GET /api/v1/workforce/reports/tps`).
* **`alice_onboard_new_hire`**: Triggers the HR onboarding saga (`POST /api/v1/workforce/sagas/onboarding`) to provision Snipe-IT laptops, Nextcloud storage, and Forgejo developer accounts for new employees.
* **`alice_write_handbook_doc`**: Authors the quarterly Employee Handbook update in Nextcloud (`POST /nextcloud/api/v1/documents`).
* **`alice_send_announcement_chat`**: Broadcasts company-wide announcements via Nextcloud Chat (`POST /nextcloud/api/v1/chat/messages`).

### 4. Bob Facility (`emp-human-bob`) — Facilities Edge Engineer
* **`bob_review_maintenance_calendar`**: Checks the shared Nextcloud operations calendar for upcoming habitat maintenance windows.
* **`bob_query_edge_cameras`**: Queries the registry of physical edge camera rigs across habitat zones (`/facilities/assets?asset_type=edge_camera`).
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
* **`elena_review_health_assessments`**: Audits scheduled feline health assessments and records clinical observation metrics (`GET /api/v1/workforce/review-cycles?review_type=feline_health_assessment`).

### 8. Frank Operations (`emp-human-frank`) — Platform Security & K8s Infrastructure Lead
* **`frank_monitor_it_tickets`**: Monitors IT helpdesk tickets (`/ops/tickets`) originating from Forgejo issues or automated cluster alerts.
* **`frank_query_pebble_alerts`**: Queries wearable watch alert payloads (`/ops/pebble/alerts`) formatted for Pebble AppMessage protocol.
* **`frank_create_it_ticket`**: Submits infrastructure support tickets (`POST /ops/tickets`) for intermediate CA key rotations and dynamic SPIFFE ID mTLS reloads.
* **`frank_publish_security_doc`**: Writes zero-trust security architecture document in Nextcloud (`POST /nextcloud/api/v1/documents`) and alerts developers via Nextcloud Chat (`POST /nextcloud/api/v1/chat/messages`).

### 9. Arthur Pendelton (`cust-longterm-arthur`) — Long-Term Value Investor (Customer)
* **`arthur_review_performance_reports`**: Audits historical quantitative backtest performance and strategy returns (`GET /core-invest/backtesting/runs`).
* **`arthur_deposit_investment_capital`**: Executes steady capital deposit orders into the OCI Alpha Growth Fund (`POST /core-invest/brokerage/orders`).
* **`arthur_monitor_feline_stream`**: Passively checks optical camera streams (`GET /core-invest/streams`) to verify feline habitat comfort and wellbeing.

### 10. Chloe Spark (`cust-active-chloe`) — Momentum Alpha Trader (Customer)
* **`chloe_scan_activity_streams`**: Scans real-time camera streams (`GET /core-invest/streams`) for high-activity feline behavioral signals.
* **`chloe_momentum_buy_order`**: Executes aggressive momentum buy orders (`POST /core-invest/brokerage/orders`) on feline zoomies activity spikes.
* **`chloe_take_profit_sell_order`**: Executes tactical take-profit sell orders (`POST /core-invest/brokerage/orders`) as cat activity returns to baseline.
* **`chloe_audit_execution_log`**: Audits real-time trade order execution logs and settlement status (`GET /core-invest/brokerage/orders`).

### 11. Rick Newhire (`emp-human-rick`) — Junior Operations Associate
* **`rick_onboarding_checklist`**: Verifies workforce directory status during new hire onboarding (`GET /api/v1/workforce/employees`).
* **`rick_read_nextcloud_docs`**: Reads shared onboarding guides and security standard documentation in Nextcloud (`GET /nextcloud/api/v1/documents`).
* **`rick_send_nextcloud_chat_ack`**: Sends Nextcloud Chat message acknowledging receipt of assigned laptop and onboarding documents (`POST /nextcloud/api/v1/chat/messages`).
* **`rick_check_onboarding_status`**: Queries employee onboarding status (`GET /api/v1/workforce/employees?status=onboarding`) to track provisioning progress.
* **`rick_trigger_self_onboarding_webhook`**: Simulates Frappe HR onboarding saga webhooks (`POST /api/v1/webhooks/frappe-hr`) to exercise new hire lifecycle workflows.

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
| `-bff-url` | `""` | Target URL for OCI Employee BFF |
| `-cust-bff-url`| `""` | Target URL for OCI Customer BFF |
| `-forgejo-url` | `""` | Target URL for Forgejo Git repository host |
| `-broker-url`  | `""` | Target URL for Mock Broker API |
| `-bank-url`    | `""` | Target URL for Mock Bank API |
| `-iterations` | `1` | Number of daily action loops to perform |
| `-delay` | `100` | Delay between persona action executions in milliseconds |
| `-verbose` | `true` | Enable detailed logging of request/response outputs |
| `-list-personas`| `false` | Print registered personas and exit |

---

## Verification & Testing

To run the simulator unit test suite:

```bash
go test -v ./simulation/...
```
