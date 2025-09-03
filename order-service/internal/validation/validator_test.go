package validation

import (
	"testing"
	"time"

	"order-service/models"
)

func TestValidateOrder(t *testing.T) {
	testCases := []struct {
		name    string
		order   *models.Order
		wantErr bool
	}{
		{
			name:    "valid order",
			order:   createValidOrder(),
			wantErr: false,
		},
		{
			name:    "empty order_uid",
			order:   createOrderWithEmptyUID(),
			wantErr: true,
		},
		{
			name:    "invalid email",
			order:   createOrderWithInvalidEmail(),
			wantErr: true,
		},
		{
			name:    "future date",
			order:   createOrderWithFutureDate(),
			wantErr: true,
		},
		{
			name:    "empty track_number",
			order:   createOrderWithEmptyTrackNumber(),
			wantErr: true,
		},
		{
			name:    "invalid phone",
			order:   createOrderWithInvalidPhone(),
			wantErr: true,
		},
		{
			name:    "negative amount",
			order:   createOrderWithNegativeAmount(),
			wantErr: true,
		},
		{
			name:    "no items",
			order:   createOrderWithNoItems(),
			wantErr: true,
		},
		{
			name:    "invalid item price",
			order:   createOrderWithInvalidItemPrice(),
			wantErr: true,
		},
		{
			name:    "test transaction id",
			order:   createOrderWithTestTransactionID(),
			wantErr: true,
		},
		{
			name:    "old date",
			order:   createOrderWithOldDate(),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateOrder(tc.order)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateOrder() error = %v, wantErr %v", err, tc.wantErr)
			}

			// Для случаев с ошибкой проверяем, что ошибка не nil
			if tc.wantErr && err == nil {
				t.Error("Expected error but got nil")
			}

			// Для валидных случаев проверяем, что ошибки нет
			if !tc.wantErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
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
			PaymentDt:    1637907727,
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
			CustomFee:    0,
		},
		Items: []models.Item{
			{
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
			},
		},
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "test",
		DeliveryService:   "meest",
		Shardkey:          "9",
		SmID:              99,
		DateCreated:       time.Now().Add(-24 * time.Hour), // Вчера
		OofShard:          "1",
	}
}

func createOrderWithEmptyUID() *models.Order {
	order := createValidOrder()
	order.OrderUID = ""
	return order
}

func createOrderWithInvalidEmail() *models.Order {
	order := createValidOrder()
	order.Delivery.Email = "invalid-email"
	return order
}

func createOrderWithFutureDate() *models.Order {
	order := createValidOrder()
	order.DateCreated = time.Now().Add(48 * time.Hour) // Послезавтра
	return order
}

func createOrderWithEmptyTrackNumber() *models.Order {
	order := createValidOrder()
	order.TrackNumber = ""
	return order
}

func createOrderWithInvalidPhone() *models.Order {
	order := createValidOrder()
	order.Delivery.Phone = "invalid-phone" // Не начинается с +
	return order
}

func createOrderWithNegativeAmount() *models.Order {
	order := createValidOrder()
	order.Payment.Amount = -100 // Отрицательная сумма
	return order
}

func createOrderWithNoItems() *models.Order {
	order := createValidOrder()
	order.Items = []models.Item{} // Пустой список товаров
	return order
}

func createOrderWithInvalidItemPrice() *models.Order {
	order := createValidOrder()
	order.Items[0].Price = 0 // Цена товара = 0
	return order
}

func createOrderWithTestTransactionID() *models.Order {
	order := createValidOrder()
	order.Payment.Transaction = "b563feb7b2b84b6test" // Запрещенный test ID
	return order
}

func createOrderWithOldDate() *models.Order {
	order := createValidOrder()
	order.DateCreated = time.Now().Add(-11 * 365 * 24 * time.Hour) // 11 лет назад
	return order
}

// Дополнительные тесты для ValidateOrderForAPI
func TestValidateOrderForAPI(t *testing.T) {
	testCases := []struct {
		name    string
		order   *models.Order
		wantErr bool
	}{
		{
			name:    "reserved order_uid 'test'",
			order:   createOrderWithReservedUID("test"),
			wantErr: true,
		},
		{
			name:    "reserved order_uid 'demo'",
			order:   createOrderWithReservedUID("demo"),
			wantErr: true,
		},
		{
			name:    "valid order for API",
			order:   createValidOrder(),
			wantErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateOrderForAPI(tc.order)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateOrderForAPI() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func createOrderWithReservedUID(uid string) *models.Order {
	order := createValidOrder()
	order.OrderUID = uid
	return order
}

// Тесты для отдельных функций валидации
func TestIsValidEmail(t *testing.T) {
	testCases := []struct {
		email    string
		expected bool
	}{
		{"test@example.com", true},
		{"invalid-email", false},
		{"test@domain", false},
		{"@domain.com", false},
		{"test@.com", false},
	}

	for _, tc := range testCases {
		t.Run(tc.email, func(t *testing.T) {
			result := isValidEmail(tc.email)
			if result != tc.expected {
				t.Errorf("isValidEmail(%s) = %v, expected %v", tc.email, result, tc.expected)
			}
		})
	}
}

func TestIsValidPhone(t *testing.T) {
	testCases := []struct {
		phone    string
		expected bool
	}{
		{"+9720000000", true},
		{"+1234567890", true},
		{"invalid-phone", false},
		{"9720000000", false},    // нет +
		{"+972-000-0000", false}, // содержит дефисы
		{"+", false},
		{"+123", true},                    // короткий но валидный
		{"+12345678901234567890", true},   // длинный но валидный
		{"+123456789012345678901", false}, // слишком длинный
	}

	for _, tc := range testCases {
		t.Run(tc.phone, func(t *testing.T) {
			result := isValidPhone(tc.phone)
			if result != tc.expected {
				t.Errorf("isValidPhone(%s) = %v, expected %v", tc.phone, result, tc.expected)
			}
		})
	}
}

// Benchmark тесты для проверки производительности
func BenchmarkValidateOrder(b *testing.B) {
	validOrder := createValidOrder()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateOrder(validOrder)
	}
}

func BenchmarkValidateOrderInvalid(b *testing.B) {
	invalidOrder := createOrderWithEmptyUID()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidateOrder(invalidOrder)
	}
}
