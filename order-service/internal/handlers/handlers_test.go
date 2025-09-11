// internal/handlers/handlers_test.go
package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"order-service/internal/mocks"
	"order-service/models"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
)

func createTestHandler(mockCache *mocks.MockCache, mockDB *mocks.MockDatabase) *Handler {
	tracer := noop.NewTracerProvider().Tracer("test")
	return NewHandler(mockCache, mockDB, tracer)
}

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

	handler := createTestHandler(mockCache, mockDB)

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
		OrderUID:    "test123",
		TrackNumber: "WBILMTESTTRACK",
		DateCreated: time.Now(),
	}

	mockCache.EXPECT().Get("test123").Return(nil, false)
	mockDB.EXPECT().GetOrder("test123").Return(expectedOrder, nil)
	mockCache.EXPECT().Set("test123", expectedOrder)

	handler := createTestHandler(mockCache, mockDB)

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

	handler := createTestHandler(mockCache, mockDB)

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
	mockDB.EXPECT().GetOrder("test123").Return(nil, errors.New("db connection failed"))

	handler := createTestHandler(mockCache, mockDB)

	req := httptest.NewRequest("GET", "/order/test123", nil)
	w := httptest.NewRecorder()

	handler.OrderHandler(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestOrderHandler_BadRequest(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockDatabase(ctrl)
	mockCache := mocks.NewMockCache(ctrl)

	handler := createTestHandler(mockCache, mockDB)

	req := httptest.NewRequest("GET", "/order", nil)
	w := httptest.NewRecorder()

	handler.OrderHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestWebInterfaceHandler(t *testing.T) {
	handler := &Handler{}

	cwd, _ := os.Getwd()
	defer os.Chdir(cwd)
	os.Chdir(filepath.Join(cwd, "../.."))

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	handler.WebInterfaceHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestMetricsHandler(t *testing.T) {
	handler := &Handler{}

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	handler.MetricsHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "go_")
}
