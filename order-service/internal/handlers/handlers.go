package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"order-service/internal/interfaces"
	"order-service/internal/validation"
	"order-service/models"
	"strings"
)

type Handler struct {
	Cache interfaces.Cache
	DB    interfaces.Database
}

func NewHandler(c interfaces.Cache, db interfaces.Database) *Handler {
	return &Handler{
		Cache: c,
		DB:    db,
	}
}

func (h *Handler) OrderHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) != 3 {
		http.Error(w, "Плохой запрос", http.StatusBadRequest)
		return
	}
	orderUID := parts[2]
	log.Printf("Поиск заказа: %s", orderUID)

	order, ok := h.Cache.Get(orderUID)
	if ok {
		log.Printf("Заказ %s найден в кешеe", orderUID)
	} else {
		log.Printf("Заказ %s отсувтует в кеше, идем в бд", orderUID)
		dbOrder, err := h.DB.GetOrder(orderUID)
		if err != nil {
			log.Printf("ошибка бд для %s: %v", orderUID, err)
			if errors.Is(err, sql.ErrNoRows) {
				http.Error(w, "заказ не найден", http.StatusNotFound)
			} else {
				log.Printf("ошибка бд: %v", err)
				http.Error(w, "внутреняя ошибка сервера", http.StatusInternalServerError)
			}
			return
		}
		log.Printf("бд вернул заказ: %+v", dbOrder)
		order = dbOrder
		h.Cache.Set(orderUID, dbOrder)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(order); err != nil {
		log.Printf("ошибка распарсивания JSON: %v", err)
	}
}

func (h *Handler) AddOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод запрещен", http.StatusMethodNotAllowed)
		return
	}

	var order models.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		http.Error(w, "Плохой JSON", http.StatusBadRequest)
		return
	}

	if err := validation.ValidateOrderForAPI(&order); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.DB.SaveOrder(&order); err != nil {
		log.Printf("Failed to save order: %v", err)
		http.Error(w, "внутреняя ошибка сервера", http.StatusInternalServerError)
		return
	}

	h.Cache.Set(order.OrderUID, &order)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":   "created",
		"message":  "Заказа успешно создан",
		"order_id": order.OrderUID,
	}); err != nil {
		log.Printf("ошибка распарсивания JSON: %v", err)
	}
}

func (h *Handler) WebInterfaceHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./web/index.html")
}
