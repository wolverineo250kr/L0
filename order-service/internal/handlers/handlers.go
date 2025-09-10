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

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Handler struct {
	Cache  interfaces.Cache
	DB     interfaces.Database
	Tracer trace.Tracer
}

func NewHandler(c interfaces.Cache, db interfaces.Database, tracer trace.Tracer) *Handler {
	return &Handler{
		Cache:  c,
		DB:     db,
		Tracer: tracer,
	}
}

func (h *Handler) OrderHandler(w http.ResponseWriter, r *http.Request) {
	_, span := h.Tracer.Start(r.Context(), "http.get_order")
	defer span.End()

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 3 {
		errMsg := "Плохой запрос"
		http.Error(w, errMsg, http.StatusBadRequest)
		span.SetStatus(codes.Error, errMsg)
		return
	}
	orderUID := parts[2]
	span.SetAttributes(attribute.String("order.uid", orderUID))
	log.Printf("Поиск заказа: %s", orderUID)

	order, ok := h.Cache.Get(orderUID)
	if !ok {
		dbOrder, err := h.DB.GetOrder(orderUID)
		if err != nil {
			span.RecordError(err)
			if errors.Is(err, sql.ErrNoRows) {
				errMsg := "заказ не найден"
				http.Error(w, errMsg, http.StatusNotFound)
				span.SetStatus(codes.Error, errMsg)
			} else {
				errMsg := "внутренняя ошибка сервера DB error"
				http.Error(w, errMsg, http.StatusInternalServerError)
				span.SetStatus(codes.Error, errMsg)
			}
			return
		}
		order = dbOrder
		h.Cache.Set(orderUID, dbOrder)
	} else {
		log.Printf("Заказ %s найден в кэше", orderUID)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(order); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "ошибка распарсивания JSON")
		return
	}
	span.SetStatus(codes.Ok, "заказ получен")
}

func (h *Handler) AddOrderHandler(w http.ResponseWriter, r *http.Request) {
	_, span := h.Tracer.Start(r.Context(), "http.add_order")
	defer span.End()

	if r.Method != http.MethodPost {
		errMsg := "Метод запрещен"
		http.Error(w, errMsg, http.StatusMethodNotAllowed)
		span.SetStatus(codes.Error, errMsg)
		return
	}

	var order models.Order
	if err := json.NewDecoder(r.Body).Decode(&order); err != nil {
		errMsg := "Плохой JSON"
		http.Error(w, errMsg, http.StatusBadRequest)
		span.RecordError(err)
		span.SetStatus(codes.Error, errMsg)
		return
	}

	if err := validation.ValidateOrderForAPI(&order); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		span.RecordError(err)
		span.SetStatus(codes.Error, "валидация не пройдена")
		return
	}

	if err := h.DB.SaveOrder(&order); err != nil {
		errMsg := "внутренняя ошибка сервера"
		http.Error(w, errMsg, http.StatusInternalServerError)
		span.RecordError(err)
		span.SetStatus(codes.Error, errMsg)
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

	span.SetAttributes(attribute.String("order.uid", order.OrderUID))
	span.SetStatus(codes.Ok, "заказ создан")
}

func (h *Handler) WebInterfaceHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./web/index.html")
}

var promHandler = promhttp.Handler()

func (h *Handler) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	promHandler.ServeHTTP(w, r)
}
