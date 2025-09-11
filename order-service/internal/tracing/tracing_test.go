package tracing_test

import (
	"context"
	"order-service/internal/tracing"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestInitTracer(t *testing.T) {
	tp, err := tracing.InitTracer("test-service")
	if err != nil {
		t.Fatalf("InitTracer вернул ошибку: %v", err)
	}
	if tp != nil {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				t.Errorf("ошибка при shutdown: %v", err)
			}
		}()
	}

	tracer := tracing.GetTracer("test-component")
	if tracer == nil {
		t.Error("ожидали не nil tracer, получили nil")
	}
}

func TestGetTracer(t *testing.T) {
	tr := tracing.GetTracer("test-component")
	if tr == nil {
		t.Fatal("GetTracer вернул nil")
	}

	// проверим, что это реально trace.Tracer
	_, ok := tr.(trace.Tracer)
	if !ok {
		t.Fatal("GetTracer вернул не trace.Tracer")
	}
}
