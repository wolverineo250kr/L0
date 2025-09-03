package interfaces

import "order-service/models"

// Database интерфейс для работы с базой данных
type Database interface {
	SaveOrder(order *models.Order) error
	GetOrder(orderUID string) (*models.Order, error)
	GetRecentOrders(limit int) (map[string]*models.Order, error)
	Close() error
}

// Cache интерфейс для работы с кэшем
type Cache interface {
	Set(orderUID string, order *models.Order)
	Get(orderUID string) (*models.Order, bool)
	BulkSet(orders map[string]*models.Order)
}
