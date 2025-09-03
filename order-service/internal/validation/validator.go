package validation

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"order-service/models"
)

//  проверяет валидность данных заказа
func ValidateOrder(order *models.Order) error {
	if order == nil {
		return errors.New("order nil")
	}

	// Проверка основных полей
	if err := validateOrderFields(order); err != nil {
		return err
	}

	// Проверка доставки
	if err := validateDelivery(&order.Delivery); err != nil {
		return err
	}

	// Проверка оплаты
	if err := validatePayment(&order.Payment); err != nil {
		return err
	}

	// Проверка товаров
	if err := validateItems(order.Items); err != nil {
		return err
	}

	// Проверка дат
	if err := validateDates(order); err != nil {
		return err
	}

	return nil
}

func validateOrderFields(order *models.Order) error {
	if order.OrderUID == "" {
		return errors.New("order_uid обязательно")
	}
	if len(order.OrderUID) < 5 || len(order.OrderUID) > 50 {
		return errors.New("order_uid must be between 5 and 50 characters")
	}

	if order.TrackNumber == "" {
		return errors.New("track_number обязательно")
	}
	if len(order.TrackNumber) < 5 || len(order.TrackNumber) > 30 {
		return errors.New("track_number must be between 5 and 30 characters")
	}

	if order.Entry == "" {
		return errors.New("entry обязательно")
	}

	if order.CustomerID == "" {
		return errors.New("customer_id обязательно")
	}

	if order.DeliveryService == "" {
		return errors.New("delivery_service обязательно")
	}

	return nil
}

func validateDelivery(delivery *models.Delivery) error {
	if delivery.Name == "" {
		return errors.New("delivery name обязательно")
	}
	if len(delivery.Name) > 100 {
		return errors.New("delivery name слишком длинный (максимум 100 символов)")
	}

	if delivery.Phone == "" {
		return errors.New("delivery phone обязательно")
	}
	if !isValidPhone(delivery.Phone) {
		return errors.New("неверный формат телефона (ожидается +1234567890)")
	}

	if delivery.Zip == "" {
		return errors.New("delivery zip обязательно")
	}

	if delivery.City == "" {
		return errors.New("delivery city обязательно")
	}

	if delivery.Address == "" {
		return errors.New("delivery address обязательно")
	}

	if delivery.Region == "" {
		return errors.New("delivery region обязательно")
	}

	if delivery.Email != "" && !isValidEmail(delivery.Email) {
		return errors.New("неверный формат электронной почты")
	}

	return nil
}

func validatePayment(payment *models.Payment) error {
	if payment.Transaction == "" {
		return errors.New("payment transaction обязательно")
	}

	if payment.Currency == "" {
		return errors.New("payment currency обязательно")
	}
	if len(payment.Currency) != 3 {
		return errors.New("код валюты должен состоять из 3 символов (например, USD)")
	}

	if payment.Provider == "" {
		return errors.New("payment provider обязательно")
	}

	if payment.Amount <= 0 {
		return errors.New("payment amount должен быть положительным")
	}

	if payment.PaymentDt <= 0 {
		return errors.New("payment date должен быть положительным")
	}

	if payment.Bank == "" {
		return errors.New("payment bank обязательно")
	}

	if payment.DeliveryCost < 0 {
		return errors.New("delivery cost не может быть отрицательным")
	}

	if payment.GoodsTotal <= 0 {
		return errors.New("общая сумма товаров должна быть положительной")
	}

	if payment.CustomFee < 0 {
		return errors.New("таможенная пошлина не может быть отрицательной")
	}

	return nil
}

func validateItems(items []models.Item) error {
	if len(items) == 0 {
		return errors.New("заказ должен содержать хотя бы один товар")
	}

	for i, item := range items {
		if item.ChrtID == 0 {
			return fmt.Errorf("item[%d]: chrt_id обязательно", i)
		}

		if item.Price <= 0 {
			return fmt.Errorf("item[%d]: price должен быть положительным", i)
		}

		if item.Name == "" {
			return fmt.Errorf("item[%d]: name обязательно", i)
		}

		if item.Sale < 0 || item.Sale > 100 {
			return fmt.Errorf("item[%d]: sale должна быть от 0 до 100", i)
		}

		if item.TotalPrice <= 0 {
			return fmt.Errorf("item[%d]: total_price больше нуля", i)
		}

		if item.NmID == 0 {
			return fmt.Errorf("item[%d]: nm_id обязательно", i)
		}

		if item.Brand == "" {
			return fmt.Errorf("item[%d]: brand обязательно", i)
		}

		if item.Status < 0 {
			return fmt.Errorf("item[%d]: статус не может быть отрицательным", i)
		}
	}

	return nil
}

func validateDates(order *models.Order) error {
	// Проверка даты создания заказа
	if order.DateCreated.IsZero() {
		return errors.New("date_created обязательно")
	}

	// Заказ не может быть из будущего
	if order.DateCreated.After(time.Now().Add(24 * time.Hour)) {
		return errors.New("date_created не может быть в будущем")
	}

	// Заказ не может быть старше 10 лет
	if order.DateCreated.Before(time.Now().Add(-10 * 365 * 24 * time.Hour)) {
		return errors.New("date_created не может быть старше 10 лет")
	}

	// Проверка даты платежа (если указана)
	if order.Payment.PaymentDt > 0 {
		paymentTime := time.Unix(order.Payment.PaymentDt, 0)
		if paymentTime.After(time.Now().Add(24 * time.Hour)) {
			return errors.New("payment_dt не может быть в будущем")
		}
	}

	return nil
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func isValidPhone(phone string) bool {
	// Должен начинаться с +
	if len(phone) == 0 || phone[0] != '+' {
		return false
	}

	// После + должны быть только цифры
	for _, c := range phone[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}

	// Общая длина: + и от 4 до 19 цифр
	totalLength := len(phone)
	if totalLength < 5 { // + и минимум 4 цифры
		return false
	}
	if totalLength > 20 { // + и максимум 19 цифр
		return false
	}

	return true
}

// ValidateOrderForAPI - валидация для API с дополнительными проверками
func ValidateOrderForAPI(order *models.Order) error {
	if err := ValidateOrder(order); err != nil {
		return err
	}

	// Дополнительные проверки специфичные для API
	if order.OrderUID == "test" || order.OrderUID == "demo" {
		return errors.New("зарезервированное значение order_uid")
	}

	return nil
}
