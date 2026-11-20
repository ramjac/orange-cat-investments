package telemetry

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func TestBusinessMetricsRecordDataPoints(t *testing.T) {
	ctx := context.Background()
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	prev := otel.GetMeterProvider()
	otel.SetMeterProvider(mp)
	defer otel.SetMeterProvider(prev)

	require.NoError(t, initMetrics("test-service"))

	RecordOnboardingAttempt(ctx, true)
	RecordBankLinkingAttempt(ctx, false)
	RecordOrderExecutionLatency(ctx, 12.5)
	RecordBracketOrderFill(ctx)
	RecordFelineBehavioralEvent(ctx, "zoomies")
	RecordFelineInferenceConfidence(ctx, 0.94)
	RecordCareScheduleUpdate(ctx)
	RecordEmergencyHoldToggle(ctx)
	RecordPebbleMTTA(ctx, 4.2)
	RecordUpstreamRequest(ctx, "broker", "ok")
	RecordUpstreamRequest(ctx, "broker", "timeout")
	RecordUpstreamRequest(ctx, "bank", "unavailable")

	var rm metricdata.ResourceMetrics
	require.NoError(t, reader.Collect(ctx, &rm))

	recorded := make(map[string]int)
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			switch data := m.Data.(type) {
			case metricdata.Sum[int64]:
				recorded[m.Name] += len(data.DataPoints)
			case metricdata.Histogram[float64]:
				recorded[m.Name] += len(data.DataPoints)
			}
		}
	}

	expectedMetrics := []string{
		"oci.customer.onboarding.total",
		"oci.customer.bank_linking.total",
		"oci.trading.order_execution.latency",
		"oci.trading.bracket_order_fills.total",
		"oci.feline.behavioral_events.total",
		"oci.feline.inference.confidence",
		"oci.feline.care_schedule_updates.total",
		"oci.feline.emergency_hold_toggles.total",
		"oci.operational.pebble_mtta.duration",
		"oci.upstream.requests.total",
	}
	for _, name := range expectedMetrics {
		assert.Greater(t, recorded[name], 0, "expected metric %s to have at least one data point", name)
	}
}
