package telemetry

import (
	"context"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitProvider(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if otelEndpoint == "" {
		otelEndpoint = "alloy.monitoring.svc.cluster.local:4317"
	}

	traceExporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(otelEndpoint),
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)

	// We don't fail fast on exporter creation error because it might just be the collector is down
	var tracerProvider *sdktrace.TracerProvider
	if err == nil && traceExporter != nil {
		bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
		tracerProvider = sdktrace.NewTracerProvider(
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
			sdktrace.WithResource(res),
			sdktrace.WithSpanProcessor(bsp),
		)
		otel.SetTracerProvider(tracerProvider)
	}

	metricExporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(otelEndpoint),
		otlpmetricgrpc.WithInsecure(),
		otlpmetricgrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)

	var meterProvider *sdkmetric.MeterProvider
	if err == nil && metricExporter != nil {
		meterProvider = sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(15*time.Second))),
		)
		otel.SetMeterProvider(meterProvider)
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	_ = initMetrics(serviceName)

	shutdown := func(ctx context.Context) error {
		if tracerProvider != nil {
			_ = tracerProvider.Shutdown(ctx)
		}
		if meterProvider != nil {
			_ = meterProvider.Shutdown(ctx)
		}
		return nil
	}

	return shutdown, nil
}
