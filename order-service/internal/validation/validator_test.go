package validation

import (
	"testing"
	"time"

	"order-service/models"
)

func TestValidateOrder(t *testing.T) {
	tests := []struct {
		name    string
		order   *models.Order
		wantErr bool
	}{
		{"valid order", createValidOrder(), false},
		{"empty order_uid", withOrder(func(o *models.Order) { o.OrderUID = "" }), true},
		{"invalid email", withOrder(func(o *models.Order) { o.Delivery.Email = "invalid" }), true},
		{"future date", withOrder(func(o *models.Order) { o.DateCreated = time.Now().Add(48 * time.Hour) }), true},
		{"empty track_number", withOrder(func(o *models.Order) { o.TrackNumber = "" }), true},
		{"invalid phone", withOrder(func(o *models.Order) { o.Delivery.Phone = "12345" }), true},
		{"negative amount", withOrder(func(o *models.Order) { o.Payment.Amount = -10 }), true},
		{"no items", withOrder(func(o *models.Order) { o.Items = []models.Item{} }), true},
		{"invalid item price", withOrder(func(o *models.Order) { o.Items[0].Price = 0 }), true},
		{"old date", withOrder(func(o *models.Order) { o.DateCreated = time.Now().Add(-11 * 365 * 24 * time.Hour) }), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateOrder(tc.order)
			if (err != nil) != tc.wantErr {
				t.Errorf("got err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateOrderForAPI(t *testing.T) {
	tests := []struct {
		name    string
		order   *models.Order
		wantErr bool
	}{
		{"reserved uid 'test'", withOrder(func(o *models.Order) { o.OrderUID = "test" }), true},
		{"reserved uid 'demo'", withOrder(func(o *models.Order) { o.OrderUID = "demo" }), true},
		{"valid order", createValidOrder(), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateOrderForAPI(tc.order)
			if (err != nil) != tc.wantErr {
				t.Errorf("got err=%v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func BenchmarkValidateOrder(b *testing.B) {
	o := createValidOrder()
	for i := 0; i < b.N; i++ {
		_ = ValidateOrder(o)
	}
}

func BenchmarkValidateOrderInvalid(b *testing.B) {
	o := withOrder(func(o *models.Order) { o.OrderUID = "" })
	for i := 0; i < b.N; i++ {
		_ = ValidateOrder(o)
	}
}

func createValidOrder() *models.Order {
	return &models.Order{
		OrderUID:    "b563feb7b2b84b6test123",
		TrackNumber: "WBILMTESTTRACK123",
		Entry:       "WBIL",
		Delivery: models.Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: models.Payment{
			Transaction:  "real-transaction-12345",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817,
			PaymentDt:    time.Now().Unix(),
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
			CustomFee:    0,
		},
		Items: []models.Item{{
			ChrtID:      9934930,
			TrackNumber: "WBILMTESTTRACK123",
			Price:       453,
			Rid:         "ab4219087a764ae0btest",
			Name:        "Mascaras",
			Sale:        30,
			Size:        "0",
			TotalPrice:  317,
			NmID:        2389212,
			Brand:       "Vivienne Sabo",
			Status:      202,
		}},
		Locale:          "en",
		CustomerID:      "test",
		DeliveryService: "meest",
		Shardkey:        "9",
		SmID:            99,
		DateCreated:     time.Now().Add(-24 * time.Hour),
		OofShard:        "1",
	}
}

func withOrder(modify func(*models.Order)) *models.Order {
	o := createValidOrder()
	modify(o)
	return o
}
