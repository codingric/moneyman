package tracing

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func AspectoTraceProvider() (*sdktrace.TracerProvider, error) {
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
			semconv.ServiceNameKey.String("backend"),
			semconv.DeploymentEnvironmentKey.String("production"),
		)),
	)
	return tp, nil
}
