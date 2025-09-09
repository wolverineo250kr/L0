package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/segmentio/kafka-go"
)

type Order struct {
	OrderUID          string    `json:"order_uid" fake:"{uuid}"`
	TrackNumber       string    `json:"track_number" fake:"{lexify:WBILMTEST??????}"`
	Entry             string    `json:"entry" fake:"{randomstring:[WBIL,WBIL2,WBIL3]}"`
	Delivery          Delivery  `json:"delivery"`
	Payment           Payment   `json:"payment"`
	Items             []Item    `json:"items" fakesize:"1,3"`
	Locale            string    `json:"locale" fake:"{randomstring:[en,ru]}"`
	InternalSignature string    `json:"internal_signature" fake:"{sentence:2}"`
	CustomerID        string    `json:"customer_id" fake:"{uuid}"`
	DeliveryService   string    `json:"delivery_service" fake:"{randomstring:[meest,novaposhta,ukrposhta]}"`
	Shardkey          string    `json:"shardkey" fake:"{number:1,9}"`
	SmID              int       `json:"sm_id" fake:"{number:1,100}"`
	DateCreated       time.Time `json:"date_created"`
	OofShard          string    `json:"oof_shard" fake:"{number:1,9}"`
}

type Delivery struct {
	Name    string `json:"name" fake:"{name}"`
	Phone   string `json:"phone" fake:"+7##########"`
	Zip     string `json:"zip" fake:"{zip}"`
	City    string `json:"city" fake:"{city}"`
	Address string `json:"address" fake:"{street}"`
	Region  string `json:"region" fake:"{state}"`
	Email   string `json:"email" fake:"{email}"`
}

type Payment struct {
	Transaction  string `json:"transaction" fake:"{uuid}"`
	RequestID    string `json:"request_id" fake:"{uuid}"`
	Currency     string `json:"currency" fake:"{randomstring:[USD,UAH,EUR]}"`
	Provider     string `json:"provider" fake:"{randomstring:[wbpay,paypal,stripe]}"`
	Amount       int    `json:"amount" fake:"{number:100,10000}"`
	PaymentDt    int64  `json:"payment_dt" fake:"{number:1609459200,1640995200}"` // Timestamp между 2021-2022
	Bank         string `json:"bank" fake:"{randomstring:[alpha,privat,monobank]}"`
	DeliveryCost int    `json:"delivery_cost" fake:"{number:50,500}"`
	GoodsTotal   int    `json:"goods_total" fake:"{number:100,5000}"`
	CustomFee    int    `json:"custom_fee" fake:"{number:0,100}"`
}

type Item struct {
	ChrtID      int    `json:"chrt_id" fake:"{number:1000000,9999999}"`
	TrackNumber string `json:"track_number" fake:"{lexify:WBILMTEST??????}"`
	Price       int    `json:"price" fake:"{number:100,5000}"`
	Rid         string `json:"rid" fake:"{uuid}"`
	Name        string `json:"name" fake:"{productname}"`
	Sale        int    `json:"sale" fake:"{number:0,50}"`
	Size        string `json:"size" fake:"{randomstring:[S,M,L,XL]}"`
	TotalPrice  int    `json:"total_price"`
	NmID        int    `json:"nm_id" fake:"{number:1000000,9999999}"`
	Brand       string `json:"brand" fake:"{company}"`
	Status      int    `json:"status" fake:"{number:100,400}"`
}

func (i *Item) CalculateTotalPrice() {
	i.TotalPrice = i.Price * (100 - i.Sale) / 100
	// Гарантируем, что цена не будет нулевой или отрицательной
	if i.TotalPrice <= 0 {
		i.TotalPrice = i.Price / 2 // Минимум половина цены
	}
}

func main() {
	brokerAddress := os.Getenv("KAFKA_BROKER")
	if brokerAddress == "" {
		log.Fatal("переменная окружения KAFKA_BROKER не установлена")
	}

	writer := &kafka.Writer{
		Addr:     kafka.TCP(brokerAddress),
		Topic:    "orders",
		Balancer: &kafka.LeastBytes{},
	}
	defer func() {
		if err := writer.Close(); err != nil {
			log.Printf("Ошибка закрытия writer: %v", err)
		}
	}()

	dlqWriter := &kafka.Writer{
		Addr:     kafka.TCP(brokerAddress),
		Topic:    "orders_dlq",
		Balancer: &kafka.LeastBytes{},
	}
	defer func() {
		if err := dlqWriter.Close(); err != nil {
			log.Printf("Ошибка закрытия DLQ writer: %v", err)
		}
	}()

	//  генератор случайных данных
	gofakeit.Seed(time.Now().UnixNano())

	log.Println("Запускаем продюсер...")

	// 5 валидных сообщений
	for i := 0; i < 5; i++ {
		var order Order
		err := gofakeit.Struct(&order)
		if err != nil {
			log.Printf("Ошибка генерации заказа: %v", err)
			continue
		}

		// фиксируем дату в диапазоне [сейчас-10 лет ... сейчас]
		now := time.Now()
		tenYearsAgo := now.AddDate(-10, 0, 0)
		order.DateCreated = gofakeit.DateRange(tenYearsAgo, now)

		for j := range order.Items {
			order.Items[j].CalculateTotalPrice()
		}

		order.Payment.GoodsTotal = 0
		for _, item := range order.Items {
			order.Payment.GoodsTotal += item.TotalPrice
		}

		order.Payment.Amount = order.Payment.GoodsTotal + order.Payment.DeliveryCost + order.Payment.CustomFee

		orderJSON, err := json.Marshal(order)
		if err != nil {
			log.Printf("Ошибка маршалинга JSON: %v", err)
			continue
		}

		err = writer.WriteMessages(context.Background(),
			kafka.Message{
				Key:   []byte(order.OrderUID),
				Value: orderJSON,
			},
		)
		if err != nil {
			log.Printf("Ошибка отправки сообщения: %v", err)
		} else {
			log.Printf("Валидное сообщение отправлено: %s", order.OrderUID)
		}

		time.Sleep(1 * time.Second)
	}

	// невалидное сообщение в DLQ
	invalidOrder := `{"invalid": json}`
	err := dlqWriter.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte("err_" + gofakeit.UUID()),
			Value: []byte(invalidOrder),
			Headers: []kafka.Header{
				{
					Key:   "error_reason",
					Value: []byte("invalid_json"),
				},
			},
		},
	)
	if err != nil {
		log.Printf("Ошибка отправки в DLQ: %v", err)
	} else {
		log.Println("Невалидное сообщение отправлено в DLQ")
	}

	log.Println("Продюсер завершил работу")
}
