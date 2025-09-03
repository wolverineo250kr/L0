package kafka

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"order-service/internal/interfaces"
	"order-service/internal/validation"
	"order-service/models"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader    *kafka.Reader
	dlqWriter *kafka.Writer
	db        interfaces.Database
	cache     interfaces.Cache
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

		// Обрабатываем сообщение
		if err := c.processMessage(ctx, m); err != nil {
			log.Printf("Ошибка обработки сообщения: %v", err)
			c.sendToDLQ(ctx, m.Value)
		} else {
			c.commit(ctx, m)
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, m kafka.Message) error {
	var order models.Order
	if err := json.Unmarshal(m.Value, &order); err != nil {
		return fmt.Errorf("ошибка при преобразовании JSON: %w", err)
	}

	// добавляем проверку
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
