// internal/kafka/consumer_integration_test.go
//go:build integration

package kafka

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"

	"order-service/internal/db"
	"order-service/internal/mocks"
	"order-service/models"

	"github.com/golang/mock/gomock"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel"
)

func createIntegrationTestOrder() models.Order {
	gofakeit.Seed(time.Now().UnixNano())

	order := models.Order{
		OrderUID:          gofakeit.UUID(),
		TrackNumber:       gofakeit.LetterN(12),
		Entry:             gofakeit.LetterN(4),
		Locale:            "ru",
		InternalSignature: gofakeit.Sentence(2),
		CustomerID:        gofakeit.UUID(),
		DeliveryService:   gofakeit.Company(),
		Shardkey:          gofakeit.LetterN(3),
		SmID:              gofakeit.Number(1, 100),
		DateCreated:       gofakeit.DateRange(time.Now().AddDate(-10, 0, 0), time.Now()),
		OofShard:          gofakeit.LetterN(3),
		Delivery: models.Delivery{
			Name:    gofakeit.Name(),
			Phone:   "+7" + gofakeit.DigitN(10),
			Zip:     gofakeit.Zip(),
			City:    gofakeit.City(),
			Address: gofakeit.Street(),
			Region:  gofakeit.State(),
			Email:   gofakeit.Email(),
		},
		Payment: models.Payment{
			Transaction:  gofakeit.UUID(),
			RequestID:    gofakeit.UUID(),
			Currency:     "RUB",
			Provider:     "wbpay",
			Amount:       0,
			PaymentDt:    time.Now().Unix(),
			Bank:         "zheltyi",
			DeliveryCost: gofakeit.Number(50, 500),
			GoodsTotal:   0,
			CustomFee:    gofakeit.Number(0, 100),
		},
	}

	// Генерируем 1–3 товара
	numItems := gofakeit.Number(1, 3)
	order.Items = make([]models.Item, numItems)
	for i := range order.Items {
		price := gofakeit.Number(100, 5000)
		sale := gofakeit.Number(0, 50)
		totalPrice := price * (100 - sale) / 100
		if totalPrice <= 0 {
			totalPrice = price / 2
		}

		order.Items[i] = models.Item{
			ChrtID:      int64(gofakeit.Number(1000000, 9999999)),
			TrackNumber: order.TrackNumber,
			Price:       price,
			Rid:         gofakeit.UUID(),
			Name:        gofakeit.ProductName(),
			Sale:        sale,
			Size:        gofakeit.RandomString([]string{"S", "M", "L", "XL"}),
			TotalPrice:  totalPrice,
			NmID:        int64(gofakeit.Number(1000000, 9999999)),
			Brand:       gofakeit.Company(),
			Status:      gofakeit.Number(100, 400),
		}
	}

	for _, item := range order.Items {
		order.Payment.GoodsTotal += item.TotalPrice
	}
	order.Payment.Amount = order.Payment.GoodsTotal + order.Payment.DeliveryCost + order.Payment.CustomFee

	return order
}

func TestKafkaConsumer_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	broker := os.Getenv("KAFKA_BROKERS")
	if broker == "" {
		t.Fatal("KAFKA_BROKERS environment variable is not set")
	}

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Fatal("POSTGRES_DSN environment variable is not set")
	}

	dbConn, err := db.NewPostgresDB(dsn)
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}
	defer dbConn.Close()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockCache := mocks.NewMockCache(ctrl)
	mockCache.EXPECT().Set(gomock.Any(), gomock.Any()).AnyTimes()
	mockCache.EXPECT().BulkSet(gomock.Any()).AnyTimes()

	consumer := NewConsumer(
		[]string{broker},
		"orders",
		"group1",
		"orders_dlq",
		dbConn,
		mockCache,
		otel.Tracer("test"),
	)

	order := createIntegrationTestOrder()
	msgBytes, _ := json.Marshal(order)
	msg := kafka.Message{Value: msgBytes}

	err = consumer.processMessage(context.Background(), msg)
	assert.NoError(t, err)

	savedOrder, err := dbConn.GetOrder(order.OrderUID)
	assert.NoError(t, err)
	assert.Equal(t, order.OrderUID, savedOrder.OrderUID)
	assert.Equal(t, order.Delivery.Name, savedOrder.Delivery.Name)
	assert.Equal(t, order.Items[0].Name, savedOrder.Items[0].Name)

	log.Println("интеграционный тест прошел успешно, заказ сохранен в постгре")
}
