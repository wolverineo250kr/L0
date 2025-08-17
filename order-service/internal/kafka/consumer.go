package kafka

import (
    "context"
    "encoding/json"
    "log"
    "order-service/internal/cache"
    "order-service/internal/db"
    "order-service/models"

    "github.com/segmentio/kafka-go"
)

type Consumer struct {
    reader   *kafka.Reader
    db       *db.PostgresDB
    cache    *cache.Cache
    producer *Producer
}

func NewConsumer(brokers []string, topic, groupID string, db *db.PostgresDB, cache *cache.Cache, producer *Producer) *Consumer {
    r := kafka.NewReader(kafka.ReaderConfig{
        Brokers:        brokers,
        Topic:          topic,
        GroupID:        groupID,
        CommitInterval: 0,
    })
    return &Consumer{
        reader:   r,
        db:       db,
        cache:    cache,
        producer: producer,
    }
}

func (c *Consumer) Run(ctx context.Context) {
    for {
        m, err := c.reader.FetchMessage(ctx)
        if err != nil {
            log.Println("Ошибка выборки кафка:", err)
            continue
        }

        var order models.Order
        if err := json.Unmarshal(m.Value, &order); err != nil {
            log.Println("ошибка при преобразовании json:", err)
            _ = c.producer.SendToDLQ(m.Value)
            continue
        }

        if err := c.db.SaveOrder(&order); err != nil {
            log.Println("Ошибка сохранения БД:", err)
            _ = c.producer.SendToDLQ(m.Value)
            continue
        }

        c.cache.Set(order.OrderUID, &order)

        c.commit(ctx, m)
    }
}

func (c *Consumer) commit(ctx context.Context, m kafka.Message) {
    if err := c.reader.CommitMessages(ctx, m); err != nil {
        log.Println("ошибка коммита:", err)
    }
}

func (c *Consumer) sendToDLQ(msg []byte) {
    if err := c.producer.SendToDLQ(msg); err != nil {
        log.Println("Не удалось отправить в DLQ:", err)
    }
}
