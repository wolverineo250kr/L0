// internal/kafka/consumer.go
package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"order-service/internal/interfaces"
	"order-service/internal/metrics"
	"order-service/internal/validation"
	"order-service/models"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader      *kafka.Reader
	dlqWriter   *kafka.Writer
	db          interfaces.Database
	cache       interfaces.Cache
	maxRetries  int
	retryDelay  time.Duration
	backoffMode string // "fixed" или "exponential"
	tracer      trace.Tracer
}

func NewConsumer(brokers []string, topic, groupID, dlqTopic string, db interfaces.Database, cache interfaces.Cache, tracer trace.Tracer) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		CommitInterval: 0,
	})
	return &Consumer{
		reader: r,
		db:     db,
		cache:  cache,
		dlqWriter: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    dlqTopic,
			Balancer: &kafka.LeastBytes{},
		},
		maxRetries:  3,
		retryDelay:  2 * time.Second,
		backoffMode: "exponential", // можно "fixed"
		tracer:      tracer,
	}
}

func (c *Consumer) Run(ctx context.Context) {
	for {
		m, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				log.Println("консюмер остановился по контексту")
				return
			}
			log.Println("Ошибка выборки Kafka:", err)
			continue
		}

		if err := c.processWithRetry(ctx, m); err != nil {
			log.Printf("Ошибка после всех ретраев: %v", err)
			c.sendToDLQ(ctx, m.Value)
		} else {
			c.commit(ctx, m)
		}
	}
}

// обёртка с ретраями
func (c *Consumer) processWithRetry(ctx context.Context, m kafka.Message) error {
	var err error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		err = c.processMessage(ctx, m)
		if err == nil {
			return nil
		}

		// если последняя попытка — выходим
		if attempt == c.maxRetries {
			break
		}

		// ждем перед повтором
		delay := c.retryDelay
		if c.backoffMode == "exponential" {
			delay = time.Duration(float64(c.retryDelay) * math.Pow(2, float64(attempt)))
		}
		log.Printf("ошибка обработки (попытка %d/%d): %v, жду %v перед повтором", attempt+1, c.maxRetries, err, delay)

		select {
		case <-time.After(delay):
			// продолжаем ретрай
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return err
}

func (c *Consumer) processMessage(ctx context.Context, m kafka.Message) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.OrderProcessingTime.WithLabelValues("kafka", "process_message").Observe(duration)
	}()

	ctx, span := c.tracer.Start(ctx, "kafka.process_message")
	defer span.End()

	var order models.Order
	if err := json.Unmarshal(m.Value, &order); err != nil {
		errMsg := "ошибка при преобразовании JSON"
		err := fmt.Errorf(errMsg+": %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, errMsg)
		metrics.OrdersProcessed.WithLabelValues("kafka", "error").Inc()
		return err
	}

	span.SetAttributes(attribute.String("order.uid", order.OrderUID))

	if err := validation.ValidateOrder(&order); err != nil {
		errMsg := "невалидные данные заказа"
		err := fmt.Errorf(errMsg+": %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, errMsg)
		metrics.OrdersProcessed.WithLabelValues("kafka", "error").Inc()
		return err
	}

	if err := c.db.SaveOrder(&order); err != nil {
		errMsg := "ошибка сохранения в БД"
		err := fmt.Errorf(errMsg+": %w", err)
		span.RecordError(err)
		span.SetStatus(codes.Error, errMsg)
		metrics.OrdersProcessed.WithLabelValues("kafka", "error").Inc()
		return err
	}

	c.cache.Set(order.OrderUID, &order)
	msgSucc := "Заказ " + order.OrderUID + " успешно обработан и сохранен"
	log.Println(msgSucc)
	span.SetStatus(codes.Ok, msgSucc)
	metrics.OrdersProcessed.WithLabelValues("kafka", "success").Inc()
	return nil
}

func (c *Consumer) commit(ctx context.Context, m kafka.Message) {
	if err := c.reader.CommitMessages(ctx, m); err != nil {
		log.Println("Ошибка коммита:", err)
	}
}

func (c *Consumer) sendToDLQ(ctx context.Context, msg []byte) {
	err := c.dlqWriter.WriteMessages(ctx, kafka.Message{
		Value: msg,
	})
	if err != nil {
		switch {
		case errors.Is(err, context.Canceled):
			log.Println("контекст отменён, выйдем")
		case errors.Is(err, context.DeadlineExceeded):
			log.Println("таймаут при записи в DLQ")
		default:
			log.Println("не удалось отправить в DLQ:", err)
		}
	}
}

func (c *Consumer) Close() {
	if err := c.reader.Close(); err != nil {
		log.Println("Ошибка закрытия reader:", err)
	}
	if err := c.dlqWriter.Close(); err != nil {
		log.Println("Ошибка закрытия DLQ writer:", err)
	}
}
