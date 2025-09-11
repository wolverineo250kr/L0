// internal/cache/cache_test.go
package cache

import (
	"testing"
	"time"

	"order-service/models"

	"github.com/stretchr/testify/assert"
)

func TestCache_SetAndGet(t *testing.T) {
	cache := New(5*time.Minute, 100)
	order := &models.Order{OrderUID: "test123", TrackNumber: "WBILMTEST"}

	cache.Set("test123", order)
	result, found := cache.Get("test123")

	assert.True(t, found)
	assert.Equal(t, order.OrderUID, result.OrderUID)
}

func TestCache_Get_NotFound(t *testing.T) {
	cache := New(5*time.Minute, 100)

	result, found := cache.Get("nonexistent")

	assert.False(t, found)
	assert.Nil(t, result)
}

func TestCache_Expiration(t *testing.T) {
	cache := New(100*time.Millisecond, 100)
	order := &models.Order{OrderUID: "test123"}

	cache.Set("test123", order)
	time.Sleep(150 * time.Millisecond)

	result, found := cache.Get("test123")

	assert.False(t, found)
	assert.Nil(t, result)
}

func TestCache_Eviction(t *testing.T) {
	cache := New(5*time.Minute, 2)

	order1 := &models.Order{OrderUID: "test1"}
	order2 := &models.Order{OrderUID: "test2"}
	order3 := &models.Order{OrderUID: "test3"}

	cache.Set("test1", order1)
	time.Sleep(10 * time.Millisecond)
	cache.Set("test2", order2)
	time.Sleep(10 * time.Millisecond)
	cache.Set("test3", order3)

	_, found1 := cache.Get("test1")
	_, found2 := cache.Get("test2")
	_, found3 := cache.Get("test3")

	assert.False(t, found1, "test1 должен быть вытеснен")
	assert.True(t, found2, "test2 должен остаться")
	assert.True(t, found3, "test3 должен остаться")
}
