package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"order-service/internal/mocks"
	"order-service/models"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderHandler_OrderFoundInCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDatabase(ctrl)
	mockCache := mocks.NewMockCache(ctrl)

	expectedOrder := &models.Order{
		OrderUID:    "test123",
		TrackNumber: "WBILMTESTTRACK",
	}

	mockCache.EXPECT().Get("test123").Return(expectedOrder, true)

	handler := NewHandler(mockCache, mockDB)

	req := httptest.NewRequest("GET", "/order/test123", nil)
	w := httptest.NewRecorder()

	handler.OrderHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response models.Order
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "test123", response.OrderUID)
}

func TestOrderHandler_OrderNotFoundInCacheButFoundInDB(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDatabase(ctrl)
	mockCache := mocks.NewMockCache(ctrl)

	expectedOrder := &models.Order{
		OrderUID:        "test123",
		TrackNumber:     "TEST123",
		Entry:           "TEST",
		Locale:          "en",
		CustomerID:      "test",
		DeliveryService: "test",
		DateCreated:     time.Now(),
		Delivery: models.Delivery{
			Name:    "Test",
			Phone:   "+1234567890",
			Zip:     "123",
			City:    "Test",
			Address: "Test",
			Region:  "Test",
		},
		Payment: models.Payment{
			Transaction:  "test123",
			Currency:     "USD",
			Provider:     "test",
			Amount:       100,
			PaymentDt:    1637907727,
			Bank:         "test",
			DeliveryCost: 10,
			GoodsTotal:   100,
		},
		Items: []models.Item{{
			ChrtID:      1,
			TrackNumber: "TEST123",
			Price:       100,
			Rid:         "test",
			Name:        "Test",
			TotalPrice:  100,
			NmID:        1,
			Brand:       "Test",
			Status:      1,
		}},
	}

	mockCache.EXPECT().Get("test123").Return(nil, false)
	mockDB.EXPECT().GetOrder("test123").Return(expectedOrder, nil)
	mockCache.EXPECT().Set("test123", expectedOrder)

	handler := NewHandler(mockCache, mockDB)

	req := httptest.NewRequest("GET", "/order/test123", nil)
	w := httptest.NewRecorder()

	handler.OrderHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	assert.NotEqual(t, "null", strings.TrimSpace(w.Body.String()))
}

func TestOrderHandler_OrderNotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDatabase(ctrl)
	mockCache := mocks.NewMockCache(ctrl)

	mockCache.EXPECT().Get("notfound").Return(nil, false)
	mockDB.EXPECT().GetOrder("notfound").Return(nil, sql.ErrNoRows)

	handler := NewHandler(mockCache, mockDB)

	req := httptest.NewRequest("GET", "/order/notfound", nil)
	w := httptest.NewRecorder()

	handler.OrderHandler(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestOrderHandler_DatabaseError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDatabase(ctrl)
	mockCache := mocks.NewMockCache(ctrl)

	mockCache.EXPECT().Get("test123").Return(nil, false)
	mockDB.EXPECT().GetOrder("test123").Return(nil, errors.New("database connection failed"))

	handler := NewHandler(mockCache, mockDB)

	req := httptest.NewRequest("GET", "/order/test123", nil)
	w := httptest.NewRecorder()

	handler.OrderHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAddOrderHandler_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDatabase(ctrl)
	mockCache := mocks.NewMockCache(ctrl)

	order := &models.Order{
		OrderUID:        "test123",
		TrackNumber:     "WBILMTESTTRACK",
		Entry:           "WBIL",
		Locale:          "en",
		CustomerID:      "test",
		DeliveryService: "meest",
		DateCreated:     time.Now(),
		Delivery: models.Delivery{
			Name:    "Test User",
			Phone:   "+1234567890",
			Zip:     "123456",
			City:    "Test City",
			Address: "Test Address",
			Region:  "Test Region",
			Email:   "test@example.com",
		},
		Payment: models.Payment{
			Transaction:  "trans123",
			RequestID:    "",
			Currency:     "USD",
			Provider:     "test",
			Amount:       100,
			PaymentDt:    1637907727,
			Bank:         "test",
			DeliveryCost: 10,
			GoodsTotal:   100,
			CustomFee:    0,
		},
		Items: []models.Item{{
			ChrtID:      9934930,
			TrackNumber: "WBILMTESTTRACK",
			Price:       100,
			Rid:         "ab4219087a764ae0btest",
			Name:        "Test Item",
			Sale:        0,
			Size:        "0",
			TotalPrice:  100,
			NmID:        2389212,
			Brand:       "Test Brand",
			Status:      202,
		}},
	}

	orderJSON, _ := json.Marshal(order)

	mockDB.EXPECT().SaveOrder(gomock.Any()).Return(nil)
	mockCache.EXPECT().Set("test123", gomock.Any())

	handler := NewHandler(mockCache, mockDB)

	req := httptest.NewRequest("POST", "/api/order/add", strings.NewReader(string(orderJSON)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.AddOrderHandler(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var response map[string]string
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "created", response["status"])
}
