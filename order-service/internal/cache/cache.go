package cache

import (
    "order-service/models"
    "sync"
)

type Cache struct {
    mu     sync.RWMutex
    orders map[string]*models.Order
}

func New() *Cache {
    return &Cache{
        orders: make(map[string]*models.Order),
    }
}

func (c *Cache) BulkSet(orders map[string]*models.Order) {
    c.mu.Lock()
    defer c.mu.Unlock()

    for uid, order := range orders {
        c.orders[uid] = order
    }
}

func (c *Cache) Get(orderUID string) (*models.Order, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    order, ok := c.orders[orderUID]
    return order, ok
}

func (c *Cache) Set(orderUID string, order *models.Order) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.orders[orderUID] = order
}