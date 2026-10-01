# Orange Cat Investments (OCI) Platform Simulator

The **OCI Stack Simulator** is a Go-based driver application designed to fully exercise the Orange Cat Investments software and hardware infrastructure stack. By emulating the daily operational workflows of OCI's human staff and feline executives, the simulator drives end-to-end API calls, event triggers, domain sagas, hardware asset management, quantitative backtesting, and wearable alert proxy workflows.

---

## Architecture Overview

The simulator architecture mirrors OCI's Domain-Driven Design (DDD) model:

```text
simulation/
├── config.go            # Configuration flags (URLs, modes, iteration, delays)
├── persona.go           # Persona and Action interface definitions & Registry
├── client.go            # HTTP client abstraction supporting live & mock HTTP servers
├── actions.go           # Persona action suites for all 8 OCI personas
├── driver.go            # Orchestration engine for running simulation workflows
├── driver_test.go       # Unit tests verifying mock simulation execution
└── cmd/
    └── simulator/       # Runnable CLI entrypoint (`go run ./simulation/cmd/simulator`)
        └── main.go
```

---

## Employee Personas & Daily Simulation Action Mapping

The simulator uses the 8 official OCI personas defined in `personas/` to execute semi-real daily action lists:

### 1. Garfield (`emp-feline-garfield`) — Chief Observation Officer (COO)
* **`garfield_morning_zoomies`**: Emits real-time observation telemetry during 3 AM peak activity windows.
* **`garfield_perch_inspection`**: Inspects primary habitat perches (`/facilities/assets?asset_type=observation_perch`) for operational readiness and cushion ergonomics.
* **`garfield_care_schedule_audit`**: Queries care schedules (`/api/v1/workforce/care-schedules/emp-feline-garfield`) to ensure punctual lasagna and snack disbursements.
* **`garfield_sleep_in_hallway`**: Naps in hallway B; 1/3 chance of tripping a human employee, placing Garfield on vet medical hold (`POST /api/v1/workforce/care-schedules/emp-feline-garfield/medical-hold`) and logging a workplace injury ticket (`POST /ops/tickets`).

### 2. Barneby (`emp-feline-barneby`) — Senior Alpha Perch Analyst
* **`barneby_sunbeam_tracking`**: Queries smart collar telemetry (`/facilities/assets?asset_type=smart_collar`) to track sunbeam positioning.
* **`barneby_care_schedule_review`**: Reviews secondary perch health and care assessment parameters (`/api/v1/workforce/care-schedules/emp-feline-barneby`).
* **`barneby_slap_water_glass`**: Slaps a water glass off a desk, creating a facilities maintenance ticket (`POST /facilities/assets/{id}/maintenance`), querying Pebble watch alerts (`GET /ops/pebble/alerts`), submitting a Pebble watch ACK (`POST /ops/pebble/ack`), and logging completed maintenance work via mobile field app sync (`POST /facilities/sync/maintenance-logs`).

### 3. Alice Vance (`emp-human-alice`) — Head of Human & Feline Resources
* **`alice_workforce_directory`**: Queries combined human and feline workforce employee directory (`/api/v1/workforce/employees`).
* **`alice_trigger_onboarding_saga`**: Simulates Frappe HR status changes (`/api/v1/webhooks/frappe-hr`) triggering the Go `OnboardingSaga` across workforce, ZITADEL, and Forgejo identity planes.
* **`alice_review_leave_requests`**: Audits pending staff leave requests and feline catnip break compliance (`/api/v1/workforce/leave-requests`).

### 4. Bob Builder (`emp-human-bob`) — Lead Facilities & Edge Telemetry Engineer
* **`bob_scan_edge_cameras`**: Scans optical edge camera rigs across habitat zones (`/facilities/assets?asset_type=edge_camera`).
* **`bob_register_camera_rig`**: Registers a new 4K edge camera asset (`POST /facilities/assets`).
* **`bob_report_maintenance_ticket`**: Reports hardware maintenance issues and generates service tickets (`POST /facilities/assets/{id}/maintenance`).

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

### 9. Arthur Pendelton (`cust-longterm-arthur`) — Long-Term Value Investor (Customer)
* **`arthur_review_performance_reports`**: Audits historical quantitative backtest performance and strategy returns (`GET /core-invest/backtesting/runs`).
* **`arthur_deposit_investment_capital`**: Executes steady capital deposit orders into the OCI Alpha Growth Fund (`POST /core-invest/brokerage/orders`).
* **`arthur_monitor_feline_stream`**: Passively checks optical camera streams (`GET /core-invest/streams`) to verify feline habitat comfort and wellbeing.

### 10. Chloe Spark (`cust-active-chloe`) — Momentum Alpha Trader (Customer)
* **`chloe_scan_activity_streams`**: Scans real-time camera streams (`GET /core-invest/streams`) for high-activity feline behavioral signals.
* **`chloe_momentum_buy_order`**: Executes aggressive momentum buy orders (`POST /core-invest/brokerage/orders`) on feline zoomies activity spikes.
* **`chloe_take_profit_sell_order`**: Executes tactical take-profit sell orders (`POST /core-invest/brokerage/orders`) as cat activity returns to baseline.
* **`chloe_audit_execution_log`**: Audits real-time trade order execution logs and settlement status (`GET /core-invest/brokerage/orders`).

---

## Execution Modes

The simulator supports four execution modes:

1. **`mock` (Default)**: Spins up embedded Go in-memory HTTP handlers (`httptest.Server`). Runs complete persona daily workflows without requiring any pre-spun local services or database dependencies. Ideal for offline development and local testing.
2. **`live`**: Connects directly to deployed or running OCI stack services (API Server on port `8080` and Employee BFF on port `8081` by default).
3. **`step`**: Runs a single step-by-step pass through each registered persona's actions and prints action results.
4. **`continuous`**: Loops persona daily action lists continuously with configured action delays (`-delay`), simulating real-time continuous business operations until interrupted (`Ctrl+C`).

---

## How to Run the Simulator

### 1. List All Registered Personas
```bash
go run ./simulation/cmd/simulator -list-personas
```

### 2. Run Standalone Mock Simulation Drive (Default)
```bash
go run ./simulation/cmd/simulator -mode mock
```

### 3. Target a Specific Persona (e.g., Garfield)
```bash
go run ./simulation/cmd/simulator -persona emp-feline-garfield
```

### 4. Drive Live Local / K3s OCI Stack
Spin up the OCI backend services (e.g. `go run ./cmd/server` and `go run ./cmd/employee-bff`) and run:
```bash
go run ./simulation/cmd/simulator \
  -mode live \
  -server-url http://localhost:8080 \
  -bff-url http://localhost:8081 \
  -iterations 3 \
  -delay 200
```

### 5. CLI Flag Reference
| Flag | Default | Description |
|---|---|---|
| `-mode` | `mock` | Execution mode: `mock`, `live`, `step`, or `continuous` |
| `-persona` | `all` | Target persona ID (`emp-feline-garfield`, `emp-human-bob`, etc.) or `all` |
| `-server-url` | `http://localhost:8080` | Target URL for OCI Domain API Server |
| `-bff-url` | `http://localhost:8081` | Target URL for OCI Employee BFF |
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
