package cache

import (
	"order-service/internal/interfaces"
	"order-service/internal/metrics"
	"order-service/models"
	"sync"
	"time"
)

var _ interfaces.Cache = (*Cache)(nil)

type Cache struct {
	mu      sync.RWMutex
	orders  map[string]cacheItem
	ttl     time.Duration
	maxSize int
}

type cacheItem struct {
	order     *models.Order
	createdAt time.Time
}

func New(ttl time.Duration, maxSize int) *Cache {
	c := &Cache{
		orders:  make(map[string]cacheItem),
		ttl:     ttl,
		maxSize: maxSize,
	}
	go c.cleanup()
	return c
}

func (c *Cache) BulkSet(orders map[string]*models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for uid, order := range orders {
		if len(c.orders) >= c.maxSize {
			c.evictOldest()
		}

		c.orders[uid] = cacheItem{order: order, createdAt: time.Now()}
	}
}

func (c *Cache) Get(orderUID string) (*models.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, ok := c.orders[orderUID]
	if ok && time.Since(item.createdAt) <= c.ttl {
		metrics.CacheOperations.WithLabelValues("get", "hit").Inc()
		return item.order, true
	}

	metrics.CacheOperations.WithLabelValues("get", "miss").Inc()
	return nil, false
}

func (c *Cache) Set(orderUID string, order *models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.orders) >= c.maxSize {
		c.evictOldest()
	}

	c.orders[orderUID] = cacheItem{order: order, createdAt: time.Now()}

	metrics.CacheOperations.WithLabelValues("set", "success").Inc()
}

func (c *Cache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, item := range c.orders {
		if oldestTime.IsZero() || item.createdAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = item.createdAt
		}
	}

	if oldestKey != "" {
		delete(c.orders, oldestKey)
	}
}

func (c *Cache) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		for uid, item := range c.orders {
			if time.Since(item.createdAt) > c.ttl {
				delete(c.orders, uid)
			}
		}
		c.mu.Unlock()
	}
}
