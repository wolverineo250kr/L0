// internal/tracing/tracing_integration_test.go
//go:build integration

package tracing_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"order-service/internal/tracing"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.28.0"
)

func TestJaegerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Пропуск интеграционного теста в коротком режиме")
	}

	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "jaegertracing/all-in-one:1.40",
		ExposedPorts: []string{"14268/tcp", "16686/tcp"},
		WaitingFor:   wait.ForListeningPort("14268/tcp"),
	}
	jaegerC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	defer jaegerC.Terminate(ctx)

	port, err := jaegerC.MappedPort(ctx, "14268")
	require.NoError(t, err)

	collectorURL := fmt.Sprintf("http://localhost:%s/api/traces", port.Port())

	tp, err := initTracerWithURL("integration-test-service", collectorURL)
	require.NoError(t, err)
	defer tp.Shutdown(ctx)

	tracer := tracing.GetTracer("integration-component")

	// Создаём спан
	ctx, span := tracer.Start(ctx, "test-span")
	time.Sleep(10 * time.Millisecond)
}

func initTracerWithURL(serviceName, collectorURL string) (*tracesdk.TracerProvider, error) {
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(
		jaeger.WithEndpoint(collectorURL),
	))
	if err != nil {
		return nil, fmt.Errorf("не удалось создать экспортера Jaeger: %v", err)
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exp),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
		)),
	)

	otel.SetTracerProvider(tp)
	return tp, nil
}
