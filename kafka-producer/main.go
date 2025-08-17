package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/segmentio/kafka-go"
)

const (
    topic          = "orders"
    dlqTopic       = "orders_dlq"
    brokerAddress  = "kafka:9092"
    sampleOrder    = `{
  "order_uid": "b563feb7b2b84b6test",
  "track_number": "WBILMTESTTRACK",
  "entry": "WBIL",
  "delivery": {
    "name": "Test Testov",
    "phone": "+9720000000",
    "zip": "2639809",
    "city": "Kiryat Mozkin",
    "address": "Ploshad Mira 15",
    "region": "Kraiot",
    "email": "test@gmail.com"
  },
  "payment": {
    "transaction": "b563feb7b2b84b6test",
    "request_id": "",
    "currency": "USD",
    "provider": "wbpay",
    "amount": 1817,
    "payment_dt": 1637907727,
    "bank": "alpha",
    "delivery_cost": 1500,
    "goods_total": 317,
    "custom_fee": 0
  },
  "items": [
    {
      "chrt_id": 9934930,
      "track_number": "WBILMTESTTRACK",
      "price": 453,
      "rid": "ab4219087a764ae0btest",
      "name": "Mascaras",
      "sale": 30,
      "size": "0",
      "total_price": 317,
      "nm_id": 2389212,
      "brand": "Vivienne Sabo",
      "status": 202
    }
  ],
  "locale": "en",
  "internal_signature": "",
  "customer_id": "test",
  "delivery_service": "meest",
  "shardkey": "9",
  "sm_id": 99,
  "date_created": "2021-11-26T06:22:19Z",
  "oof_shard": "1"
}`
    invalidOrder   = `{"invalid": json}`
)

func main() {
    brokerAddress := os.Getenv("KAFKA_BROKER")
    if brokerAddress == "" {
        log.Fatal("переменная окружения кафка брокер не установлена")
    }

    writer := kafka.NewWriter(kafka.WriterConfig{
        Brokers: []string{brokerAddress},
        Topic:   topic,
    })
    defer writer.Close()

    dlqWriter := kafka.NewWriter(kafka.WriterConfig{
        Brokers: []string{brokerAddress},
        Topic:   dlqTopic,
    })
    defer dlqWriter.Close()

    log.Println("запускаем продюсер...")

    // Отправляем валидное сообщение
    err := sendMessage(writer, "b563feb7b2b84b6test", []byte(sampleOrder))
    if err != nil {
        log.Printf("ошибка отправки валидного сообщения: %v", err)
    } else {
        log.Println("валидное сообщение успешно отправлено")
    }

    // Отправляем невалидное сообщение в dql
    err = sendToDLQ(dlqWriter, "err_12345", []byte(invalidOrder))
    if err != nil {
        log.Printf("ошибка отправки в dql: %v", err)
    } else {
        log.Println("невалидное сообщение отправлено в dql")
    }


    	//ticker := time.NewTicker(5 * time.Second)
    	//defer ticker.Stop()
        //
    	//for range ticker.C {
    	//	if time.Now().Second()%2 == 0 {
    	//		err := sendMessage(writer, generateID(), []byte(sampleOrder))
    	//		if err != nil {
    	//			log.Printf("Ошибка отправки: %v", err)
    	//			_ = sendToDLQ(dlqWriter, "err_"+generateID(), []byte(sampleOrder))
    	//		}
    	//	} else {
    	//		_ = sendToDLQ(dlqWriter, "err_"+generateID(), []byte(invalidOrder))
    	//	}
    	//}

}

func sendMessage(writer *kafka.Writer, key string, value []byte) error {
    return writer.WriteMessages(context.Background(),
        kafka.Message{
            Key:   []byte(key),
            Value: value,
        },
    )
}

func sendToDLQ(writer *kafka.Writer, key string, value []byte) error {
    return writer.WriteMessages(context.Background(),
        kafka.Message{
            Key:   []byte(key),
            Value: value,
            Headers: []kafka.Header{
                {
                    Key:   "error_reason",
                    Value: []byte("invalid_json"),
                },
            },
        },
    )
}

func generateID() string {
    return fmt.Sprintf("id_%d", time.Now().UnixNano())
}