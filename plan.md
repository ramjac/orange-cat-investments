1. **Fix Beyla Image Tag**:
   - Update `k8s/monitoring/beyla.yaml` to use `grafana/beyla:1.4.2` instead of `v1.4.0`.

2. **Fix `InitMetrics` duplication**:
   - Remove `_ = telemetry.InitMetrics("")` from all `main.go` files and integrate `InitMetrics` logic into `InitProvider` to simplify the interface.

3. **Clean up and Fix `pkg/telemetry/metrics.go`**:
   - Remove internal monologue comments and unused variables.
   - Use OpenTelemetry attributes correctly using `go.opentelemetry.io/otel/attribute`. For example, `RecordOnboardingAttempt(ctx, success)` should add `attribute.Bool("success", success)`.
   - Ensure the metric recording helper functions properly use the provided parameters.

4. **Inject Metrics Calls into Codebase**:
   - `RecordOnboardingAttempt`: Inject into `internal/saga/saga.go` inside the OnboardingSaga.
   - `RecordBankLinkingAttempt`: Inject into `internal/service/core_invest/lifecycle_gdpr.go` (if bank linking exists) or `internal/app/customerbff/portfolio_handlers.go`.
   - `RecordOrderExecutionLatency`: Inject into mock broker execution or `internal/app/customerbll/orders.go`.
   - `RecordBracketOrderFill`: Inject into `cmd/mock-broker/engine/engine.go` or BLL.
   - `RecordFelineBehavioralEvent`: Inject into `internal/handler/mock_bll_client_test.go` or wherever feline events are dispatched. Let's find the proper handlers for Feline Events (maybe `internal/app/worker` or `cmd/worker/main.go`). Let's search.
   - `RecordCareScheduleUpdate`: Inject into `internal/handler/care_schedule_handlers.go` (or `internal/service/workforce/employees_care.go`).
   - `RecordEmergencyHoldToggle`: Inject into `internal/service/core_invest/orders_deposits.go` or wherever emergency hold toggling is defined. Let's search.
   - `RecordPebbleMTTA`: Inject into `cmd/pebble-proxy/alerts_handlers.go`.

5. **Address Tempo Retention and Beyla RBAC**:
   - Update `tempo.yaml` retention to a sensible value like `168h` (7 days) for blocks.
   - Update `beyla.yaml` RBAC to include `nodes/proxy` access.
   - Add logging/error handling to `telemetry.go` for OTLP exporter creation.

6. **Submit changes**.
