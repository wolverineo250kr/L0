package validation

import (
	"fmt"
	"time"

	"order-service/models"

	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	validate.RegisterValidation("phone", validatePhone)
	validate.RegisterValidation("future_date", validateNotFutureDate)
	validate.RegisterValidation("not_ancient", validateNotAncientDate)
}

func ValidateOrder(order *models.Order) error {
	if order == nil {
		return fmt.Errorf("order is nil")
	}

	if err := validate.Struct(order); err != nil {
		return formatValidationError(err)
	}

	if err := validateDates(order); err != nil {
		return err
	}

	if err := validateItemsAdditional(order.Items); err != nil {
		return err
	}

	return nil
}

func ValidateOrderForAPI(order *models.Order) error {
	if err := ValidateOrder(order); err != nil {
		return err
	}

	if order.OrderUID == "test" || order.OrderUID == "demo" {
		return fmt.Errorf("зарезервированное значение order_uid")
	}

	return nil
}

func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()

	if len(phone) == 0 || phone[0] != '+' {
		return false
	}

	for _, c := range phone[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}

	totalLength := len(phone)
	return totalLength >= 5 && totalLength <= 20
}

func validateNotFutureDate(fl validator.FieldLevel) bool {
	date, ok := fl.Field().Interface().(time.Time)
	if !ok {
		return false
	}
	return !date.After(time.Now().Add(24 * time.Hour))
}

func validateNotAncientDate(fl validator.FieldLevel) bool {
	date, ok := fl.Field().Interface().(time.Time)
	if !ok {
		return false
	}
	return !date.Before(time.Now().Add(-10 * 365 * 24 * time.Hour))
}

func validateItemsAdditional(items []models.Item) error {
	for i, item := range items {
		if item.TotalPrice != item.Price*(100-item.Sale)/100 {
			return fmt.Errorf("item[%d]: total_price не соответствует price и sale", i)
		}
	}
	return nil
}

func validateDates(order *models.Order) error {
	if order.DateCreated.After(time.Now().Add(24 * time.Hour)) {
		return fmt.Errorf("date_created не может быть в будущем")
	}

	if order.DateCreated.Before(time.Now().Add(-10 * 365 * 24 * time.Hour)) {
		return fmt.Errorf("date_created не может быть старше 10 лет")
	}

	paymentTime := time.Unix(order.Payment.PaymentDt, 0)
	if paymentTime.After(time.Now().Add(24 * time.Hour)) {
		return fmt.Errorf("payment_dt не может быть в будущем")
	}

	return nil
}

// свои текста ошибок
func formatValidationError(err error) error {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		errorMessages := make([]string, 0, len(validationErrors))

		for _, e := range validationErrors {
			var message string

			switch e.Tag() {
			case "required":
				message = fmt.Sprintf("%s: поле обязательно для заполнения", e.Field())
			case "min":
				message = fmt.Sprintf("%s: минимальная длина - %s символов", e.Field(), e.Param())
			case "max":
				message = fmt.Sprintf("%s: максимальная длина - %s символов", e.Field(), e.Param())
			case "len":
				message = fmt.Sprintf("%s: должна быть длина %s символов", e.Field(), e.Param())
			case "email":
				message = fmt.Sprintf("%s: неверный формат электронной почты", e.Field())
			case "gt":
				message = fmt.Sprintf("%s: должно быть больше %s", e.Field(), e.Param())
			case "gte":
				message = fmt.Sprintf("%s: должно быть больше или равно %s", e.Field(), e.Param())
			case "phone":
				message = fmt.Sprintf("%s: неверный формат телефона (ожидается +1234567890)", e.Field())
			case "uppercase":
				message = fmt.Sprintf("%s: должно быть в верхнем регистре", e.Field())
			case "lte":
				message = fmt.Sprintf("%s: должно быть не более %s", e.Field(), e.Param())
			default:
				message = fmt.Sprintf("%s: нарушено правило '%s'", e.Field(), e.Tag())
			}

			errorMessages = append(errorMessages, message)
		}

		fullMessage := "Ошибки валидации:\n"
		for i, msg := range errorMessages {
			fullMessage += fmt.Sprintf("%d. %s\n", i+1, msg)
		}

		return fmt.Errorf(fullMessage)
	}
	return err
}
