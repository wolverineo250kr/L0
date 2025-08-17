package handlers

import (
    "bytes"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "log"
    "net/http"
    "order-service/internal/cache"
    "order-service/internal/db"
    "order-service/internal/kafka"
    "order-service/models"
    "regexp"
    "strings"
)

type Handler struct {
    Cache    *cache.Cache
    DB       *db.PostgresDB
    Producer *kafka.Producer
}

func NewHandler(c *cache.Cache, db *db.PostgresDB, p *kafka.Producer) *Handler {
    return &Handler{
        Cache:    c,
        DB:       db,
        Producer: p,
    }
}

func (h *Handler) sendToDLQ(body io.ReadCloser) error {
    data, err := io.ReadAll(body)
    if err != nil {
        return fmt.Errorf("не удалось прочитать тело запроса: %v", err)
    }
    defer body.Close()

    err = h.Producer.SendToDLQ(data)
    if err != nil {
        return fmt.Errorf("не удалось отправить в DLQ: %v", err)
    }

    log.Printf("Отправлено в DLQ: %s", string(data))
    return nil
}

func (h *Handler) OrderHandler(w http.ResponseWriter, r *http.Request) {
    // URL: /order/{order_uid}
    path := r.URL.Path
    parts := strings.Split(path, "/")
    if len(parts) != 3 {
        http.Error(w, "Bad Request", http.StatusBadRequest)
        return
    }
    orderUID := parts[2]

    // Пробуем получить из кэша
    order, ok := h.Cache.Get(orderUID)
    if !ok {
        // Попытка загрузить из БД
        orderFromDB, err := h.DB.GetOrder(orderUID)
        if err != nil {
            http.Error(w, "Order not found", http.StatusNotFound)
            return
        }
        // Сохраняем в кэш на будущее
        h.Cache.Set(orderUID, orderFromDB)
        order = orderFromDB
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(order)
}

func (h *Handler) WebInterfaceHandler(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "./web/index.html")
}

func (h *Handler) AddOrderHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
        return
    }

    bodyBytes, err := io.ReadAll(r.Body)
    if err != nil {
        http.Error(w, "Не удалось прочитать тело запроса", http.StatusBadRequest)
        return
    }
    r.Body.Close()
    r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

    var order models.Order
    if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
        if err := h.sendToDLQ(io.NopCloser(bytes.NewReader(bodyBytes))); err != nil {
            log.Printf("Не удалось отправить в DLQ: %v", err)
        }
        http.Error(w, "Неверный формат JSON", http.StatusBadRequest)
        return
    }

    if err := validateOrder(&order); err != nil {
        if err := h.sendToDLQ(io.NopCloser(bytes.NewReader(bodyBytes))); err != nil {
            log.Printf("Не удалось отправить в DLQ: %v", err)
        }
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if err := h.sendToKafka(&order); err != nil {
        log.Printf("Не удалось отправить заказ в кафку: %v", err)
        http.Error(w, "Внутренняя серверная ошибка", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusAccepted)
    json.NewEncoder(w).Encode(map[string]string{
        "status":   "accepted",
        "message":  "Заказ в обработке",
        "order_id": order.OrderUID,
    })
}

func validateOrder(order *models.Order) error {
    if order.OrderUID == "" {
        return errors.New("идентификатор заказа (order_uid) обязателен")
    }
    if len(order.OrderUID) < 5 || len(order.OrderUID) > 50 {
        return errors.New("идентификатор заказа должен быть от 5 до 50 символов")
    }
    if order.TrackNumber == "" {
        return errors.New("трек-номер обязателен")
    }
    if len(order.TrackNumber) < 5 || len(order.TrackNumber) > 30 {
        return errors.New("трек-номер должен быть от 5 до 30 символов")
    }

    // Валидация Delivery
    if order.Delivery.Name == "" {
        return errors.New("имя получателя обязательно")
    }
    if len(order.Delivery.Name) > 100 {
        return errors.New("имя получателя слишком длинное (максимум 100 символов)")
    }

    if order.Delivery.Phone == "" {
        return errors.New("телефон получателя обязателен")
    }
    if !isValidPhone(order.Delivery.Phone) {
        return errors.New("неверный формат телефона (ожидается +1234567890)")
    }

    if order.Delivery.Email != "" && !isValidEmail(order.Delivery.Email) {
        return errors.New("неверный формат email")
    }

    // Валидация Payment
    if order.Payment.Transaction == "" {
        return errors.New("идентификатор транзакции обязателен")
    }
    if order.Payment.Currency == "" {
        return errors.New("валюта обязательна")
    }
    if len(order.Payment.Currency) != 3 {
        return errors.New("код валюты должен состоять из 3 символов (например USD)")
    }
    if order.Payment.Amount < 0 {
        return errors.New("сумма платежа не может быть отрицательной")
    }

    // Валидация Items
    if len(order.Items) == 0 {
        return errors.New("должен быть хотя бы один товар в заказе")
    }
    for i, item := range order.Items {
        if item.ChrtID == 0 {
            return fmt.Errorf("товар[%d]: Идентификатор товара (chrt_id) обязателен", i)
        }
        if item.Price <= 0 {
            return fmt.Errorf("товар[%d]: Цена должна быть положительной", i)
        }
        if item.Name == "" {
            return fmt.Errorf("товар[%d]: Название товара обязательно", i)
        }
        if item.Sale < 0 || item.Sale > 100 {
            return fmt.Errorf("товар[%d]: Скидка должна быть от 0 до 100 процентов", i)
        }
    }

    return nil
}

func isValidEmail(email string) bool {
    emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    return emailRegex.MatchString(email)
}

func isValidPhone(phone string) bool {
    if len(phone) < 5 || len(phone) > 20 {
        return false
    }
    if phone[0] != '+' {
        return false
    }
    for _, c := range phone[1:] {
        if c < '0' || c > '9' {
            return false
        }
    }
    return true
}

func (h *Handler) sendToKafka(order *models.Order) error {
    return h.Producer.SendOrder(order)
}