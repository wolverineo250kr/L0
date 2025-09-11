package kafka

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"go.opentelemetry.io/otel"

	"order-service/internal/mocks"
	"order-service/models"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/golang/mock/gomock"
	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
)

func createTestOrder() *models.Order {
	gofakeit.Seed(time.Now().UnixNano())

	order := &models.Order{
		OrderUID:          gofakeit.UUID(),
		TrackNumber:       gofakeit.LetterN(12),
		Entry:             "WBIL",
		Locale:            "ru",
		InternalSignature: gofakeit.Sentence(2),
		CustomerID:        gofakeit.UUID(),
		DeliveryService:   "meest",
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
			Bank:         "alpha",
			DeliveryCost: gofakeit.Number(50, 500),
			GoodsTotal:   0,
			CustomFee:    gofakeit.Number(0, 100),
		},
	}

	// Генерируем 1-3 товаров
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

	order.Payment.GoodsTotal = 0
	for _, item := range order.Items {
		order.Payment.GoodsTotal += item.TotalPrice
	}
	order.Payment.Amount = order.Payment.GoodsTotal + order.Payment.DeliveryCost + order.Payment.CustomFee

	return order
}

func TestConsumer_ProcessMessage_ValidMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDatabase(ctrl)
	mockCache := mocks.NewMockCache(ctrl)

	consumer := NewConsumer(
		[]string{"localhost:9092"},
		"test",
		"group",
		"dlq",
		mockDB,
		mockCache,
		otel.Tracer("test"),
	)

	order := createTestOrder()
	messageBytes, _ := json.Marshal(order)
	msg := kafka.Message{Value: messageBytes}

	mockDB.EXPECT().SaveOrder(gomock.Any()).Return(nil)
	mockCache.EXPECT().Set(order.OrderUID, gomock.Any())

	err := consumer.processMessage(context.Background(), msg)
	assert.NoError(t, err)
}
