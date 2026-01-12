package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

var Tracer trace.Tracer = noop.NewTracerProvider().Tracer("")

func initTracer(tp trace.TracerProvider) {
	Tracer = tp.Tracer("")
}

func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

func SetError(span trace.Span, err error) {
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func GenerateError(span trace.Span, format string, a ...any) error {
	err := fmt.Errorf(format, a...)
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
	return err
}

func WrapError(span trace.Span, err error, format string, a ...any) error {
	msg := fmt.Sprintf(format, a...)
	wrapped := fmt.Errorf("%s: %w", msg, err)
	span.RecordError(err)
	span.SetStatus(codes.Error, msg)
	return wrapped
}
