1. **Add `propagation.TextMapCarrier` and `Inject/Extract` methods in `pkg/envelope/`**:
   - Create `tracing.go` in `pkg/envelope/` that implements `TextMapCarrier` for `message.Metadata`.
   - Implement `InjectTraceContext` and `ExtractTraceContext` functions that use OpenTelemetry's global propagator.
2. **Implement Watermill tracing middleware**:
   - Create `tracing_middleware.go` in `internal/app/worker/`.
   - The middleware should extract the context, create a child span for the consumer, and if the span has a trace ID, set it into the event's payload as `correlation_id` (by unmarshalling, updating, and marshalling back, to ensure we don't log the payload).
3. **Register middleware in router**:
   - Update `internal/app/worker/router.go` to add `router.AddMiddleware(TraceContextMiddleware)`.
4. **Wrap Publisher in `pkg/envelope/pubsub.go`**:
   - Make sure context is injected when publishing. Wait, the problem says "Inject on publish and extract in a Watermill middleware registered in `internal/app/worker/router.go`".
   - We need to create a wrapper for publisher `PublisherWithTracing` inside `pkg/envelope/` and modify `pubsub.go` to use it.
5. **Update Grafana datasources**:
   - Update `k8s/monitoring/grafana.yaml` to include `tracesToLogsV2` and `tracesToMetrics` configuration for the Tempo datasource.
6. **Add Unit Tests**:
   - Update/create `tracing_test.go` in `pkg/envelope/` that tests publisher/consumer span correlation using `tracetest.NewInMemoryExporter()`.
7. **Complete pre-commit steps to ensure proper testing, verification, review, and reflection are done**.
8. **Mark task as complete in `README.md`**.
