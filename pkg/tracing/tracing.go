package tracing

import (
	"context"
	"os"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

var ServiceName string

func AspectoTraceProvider(servicename string) (*sdktrace.TracerProvider, error) {
	exp, err := otlptracegrpc.New(context.Background(),
		otlptracegrpc.WithEndpoint("collector.aspecto.io:4317"),
		otlptracegrpc.WithHeaders(map[string]string{
			"Authorization": os.Getenv("ASPECTO_KEY"),
		}))
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(servicename),
			semconv.DeploymentEnvironmentKey.String("production"),
		)),
	)
	return tp, nil
}

func InitTraceProvider(servicename string) (func(), error) {
	ServiceName = servicename
	tp, e := AspectoTraceProvider(servicename)
	if e != nil {
		return nil, e
	}
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return func() { tp.Shutdown(context.Background()) }, nil
}

func NewSpan(name string, ctx context.Context) (context.Context, trace.Span) {
	tracer := otel.Tracer(ServiceName)
	return tracer.Start(ctx, name)
}
