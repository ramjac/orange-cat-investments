package telemetry

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter metric.Meter

	// Customer Metrics
	InvestorOnboardingCounter metric.Int64Counter
	BankLinkingCounter        metric.Int64Counter

	// Trading Metrics
	OrderExecutionLatency metric.Float64Histogram
	BracketOrderFills     metric.Int64Counter

	// Feline Metrics
	FelineBehavioralEvents   metric.Int64Counter
	CareScheduleUpdates      metric.Int64Counter
	EmergencyHoldToggles     metric.Int64Counter

	// Operational Metrics
	PebbleMTTADuration metric.Float64Histogram
)

func initMetrics(serviceName string) error {
	meter = otel.GetMeterProvider().Meter(serviceName)

	var err error
	InvestorOnboardingCounter, err = meter.Int64Counter("oci.customer.onboarding.total",
		metric.WithDescription("Total number of investor onboarding attempts"),
	)
	if err != nil {
		return err
	}

	BankLinkingCounter, err = meter.Int64Counter("oci.customer.bank_linking.total",
		metric.WithDescription("Total number of bank account linking attempts"),
	)
	if err != nil {
		return err
	}

	OrderExecutionLatency, err = meter.Float64Histogram("oci.trading.order_execution.latency",
		metric.WithDescription("Latency of algorithmic order execution in milliseconds"),
		metric.WithUnit("ms"),
	)
	if err != nil {
		return err
	}

	BracketOrderFills, err = meter.Int64Counter("oci.trading.bracket_order_fills.total",
		metric.WithDescription("Total number of bracket order fills by the Smart Order Router"),
	)
	if err != nil {
		return err
	}

	FelineBehavioralEvents, err = meter.Int64Counter("oci.feline.behavioral_events.total",
		metric.WithDescription("Total throughput of feline behavioral events (e.g. zoomies, perching, napping)"),
	)
	if err != nil {
		return err
	}

	CareScheduleUpdates, err = meter.Int64Counter("oci.feline.care_schedule_updates.total",
		metric.WithDescription("Frequency of feline care schedule updates"),
	)
	if err != nil {
		return err
	}

	EmergencyHoldToggles, err = meter.Int64Counter("oci.feline.emergency_hold_toggles.total",
		metric.WithDescription("Frequency of emergency medical trading hold toggles"),
	)
	if err != nil {
		return err
	}

	PebbleMTTADuration, err = meter.Float64Histogram("oci.operational.pebble_mtta.duration",
		metric.WithDescription("Mean Time To Acknowledge (MTTA) duration for wearable on-call alerts"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return err
	}

	return nil
}

func RecordOnboardingAttempt(ctx context.Context, success bool) {
	if InvestorOnboardingCounter != nil {
		InvestorOnboardingCounter.Add(ctx, 1, metric.WithAttributes(attribute.Bool("success", success)))
	}
}

func RecordBankLinkingAttempt(ctx context.Context, success bool) {
	if BankLinkingCounter != nil {
		BankLinkingCounter.Add(ctx, 1, metric.WithAttributes(attribute.Bool("success", success)))
	}
}

func RecordOrderExecutionLatency(ctx context.Context, latencyMs float64) {
	if OrderExecutionLatency != nil {
		OrderExecutionLatency.Record(ctx, latencyMs)
	}
}

func RecordBracketOrderFill(ctx context.Context) {
	if BracketOrderFills != nil {
		BracketOrderFills.Add(ctx, 1)
	}
}

func RecordFelineBehavioralEvent(ctx context.Context, eventType string) {
	if FelineBehavioralEvents != nil {
		FelineBehavioralEvents.Add(ctx, 1, metric.WithAttributes(attribute.String("event_type", eventType)))
	}
}

func RecordCareScheduleUpdate(ctx context.Context) {
	if CareScheduleUpdates != nil {
		CareScheduleUpdates.Add(ctx, 1)
	}
}

func RecordEmergencyHoldToggle(ctx context.Context) {
	if EmergencyHoldToggles != nil {
		EmergencyHoldToggles.Add(ctx, 1)
	}
}

func RecordPebbleMTTA(ctx context.Context, durationSec float64) {
	if PebbleMTTADuration != nil {
		PebbleMTTADuration.Record(ctx, durationSec)
	}
}
