package tracing

import (
	"context"
	"os"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

var ServiceName string
var TracingProvider *sdktrace.TracerProvider

func InitTraceProvider(servicename string) (shutdown func(), err error) {
	ServiceName = servicename

	switch {
	case os.Getenv("OTEL_GRPC_ENDPOINT") != "":
		log.Info().Msg("New GRPC TraceProvider")
		//"collector.aspecto.io:4317"
		exp, err := otlptracegrpc.New(
			context.Background(),
			otlptracegrpc.WithEndpoint(os.Getenv("OTEL_GRPC_ENDPOINT")),
			otlptracegrpc.WithHeaders(map[string]string{
				"Authorization": os.Getenv("OTEL_AUTH_KEY"),
			}),
		)
		if err != nil {
			return nil, err
		}
		TracingProvider = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp),
			sdktrace.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(servicename),
				semconv.DeploymentEnvironmentKey.String("production"),
			)),
		)
	case os.Getenv("OTEL_JAEGER_ENDPOINT") != "":
		exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(os.Getenv("OTEL_JAEGER_ENDPOINT"))))
		if err != nil {
			return nil, err
		}
		TracingProvider = sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(exp),
			sdktrace.WithResource(resource.NewWithAttributes(
				semconv.SchemaURL,
				semconv.ServiceNameKey.String(servicename),
				semconv.DeploymentEnvironmentKey.String("production"),
			)),
		)

		log.Info().Msg("New Jaeger TraceProvider")
	}

	otel.SetTracerProvider(TracingProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	shutdown = func() { TracingProvider.Shutdown(context.Background()) }
	return
}

func NewSpan(name string, ctx context.Context) (context.Context, trace.Span) {
	tracer := otel.Tracer(ServiceName)
	return tracer.Start(ctx, name)
}
