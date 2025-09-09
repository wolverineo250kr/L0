package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"order-service/internal/interfaces"
	"order-service/internal/validation"
	"order-service/models"
	"time"

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
}

func NewConsumer(brokers []string, topic, groupID, dlqTopic string, db interfaces.Database, cache interfaces.Cache) *Consumer {
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
	var order models.Order
	if err := json.Unmarshal(m.Value, &order); err != nil {
		return fmt.Errorf("ошибка при преобразовании JSON: %w", err)
	}

	// валидация заказа
	if err := validation.ValidateOrder(&order); err != nil {
		return fmt.Errorf("невалидные данные заказа: %w", err)
	}

	if err := c.db.SaveOrder(&order); err != nil {
		return fmt.Errorf("ошибка сохранения в БД: %w", err)
	}

	c.cache.Set(order.OrderUID, &order)
	log.Printf("Заказ %s успешно обработан и сохранен", order.OrderUID)
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
