package tracing

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.38.0"
)

const OTLPEndpointKey = "OTEL_EXPORTER_OTLP_ENDPOINT"

func initTracerProvider(ctx context.Context, res *resource.Resource) (*trace.TracerProvider, error) {
	if os.Getenv(OTLPEndpointKey) == "" {
		return nil, nil
	}

	traceExporter, err := otlptracegrpc.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize OTLP trace exporter: %w", err)
	}

	return trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithBatcher(traceExporter),
	), nil
}

func SetupOpenTelemetry(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	var shutdownFuncs []func(context.Context) error
	var err error

	shutdown := func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
		resource.WithFromEnv(),
		resource.WithTelemetrySDK(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize resource: %w", err)
	}

	traceProvider, err := initTracerProvider(ctx, res)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer provider: %w", err)
	}
	if traceProvider != nil {
		initTracer(traceProvider)
		shutdownFuncs = append(shutdownFuncs, traceProvider.Shutdown)
	}

	propagator := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(propagator)

	return shutdown, nil
}
