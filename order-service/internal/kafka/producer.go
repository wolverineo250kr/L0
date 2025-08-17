package kafka

import (
    "context"
    "crypto/sha1"
    "encoding/hex"
    "encoding/json"
    "github.com/segmentio/kafka-go"
    "order-service/models"
    "time"
)

type Producer struct {
    writer *kafka.Writer
}

func NewProducer(brokers []string) *Producer {
    return &Producer{
        writer: &kafka.Writer{
            Addr:         kafka.TCP(brokers...),
            Balancer:     &kafka.LeastBytes{},
            RequiredAcks: kafka.RequireAll,
            Async:        false,
            WriteTimeout: 10 * time.Second,
        },
    }
}

func (p *Producer) SendOrder(order *models.Order) error {
    orderData, err := json.Marshal(order)
    if err != nil {
        return err
    }

    return p.writer.WriteMessages(context.Background(),
        kafka.Message{
            Topic: "orders",
            Key:   []byte(order.OrderUID),
            Value: orderData,
        },
    )
}

func (p *Producer) SendToDLQ(msg []byte) error {
    hash := sha1.Sum(msg)
    originalKey := hex.EncodeToString(hash[:])

    return p.writer.WriteMessages(context.Background(),
        kafka.Message{
            Topic: "orders_dlq",
            Key:   []byte("err_" + originalKey),
            Value: msg,
        },
    )
}

func (p *Producer) Close() error {
    return p.writer.Close()
}
