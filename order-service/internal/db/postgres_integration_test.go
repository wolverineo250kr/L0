// internal/db/postgres_integration_test.go
//go:build integration

package db

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"testing"
	"time"

	"order-service/models"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupTestDB(t *testing.T) (*PostgresDB, func()) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "test_db",
			"POSTGRES_USER":     "test_user",
			"POSTGRES_PASSWORD": "test_pass",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections"),
	}

	postgresContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)

	port, err := postgresContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	dsn := fmt.Sprintf("postgres://test_user:test_pass@localhost:%s/test_db?sslmode=disable", port.Port())

	time.Sleep(3 * time.Second)

	db, err := NewPostgresDB(dsn)
	require.NoError(t, err)

	err = db.Conn.Ping()
	require.NoError(t, err, "Failed to ping database")

	err = createTestTables(db.Conn)
	require.NoError(t, err)

	return db, func() {
		db.Close()
		postgresContainer.Terminate(ctx)
	}
}

func createTestTables(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS orders (
			order_uid TEXT PRIMARY KEY,
			track_number TEXT NOT NULL,
			entry TEXT NOT NULL,
			locale TEXT,
			internal_signature TEXT,
			customer_id TEXT,
			delivery_service TEXT,
			shardkey TEXT,
			sm_id BIGINT,
			date_created TIMESTAMPTZ NOT NULL,
			oof_shard TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS deliveries (
			id SERIAL PRIMARY KEY,
			order_uid TEXT NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
			name TEXT,
			phone TEXT,
			zip TEXT,
			city TEXT,
			address TEXT,
			region TEXT,
			email TEXT,
			CONSTRAINT deliveries_order_uid_key UNIQUE(order_uid)
		)`,
		`CREATE TABLE IF NOT EXISTS payments (
			id SERIAL PRIMARY KEY,
			order_uid TEXT NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
			transaction TEXT,
			request_id TEXT,
			currency TEXT,
			provider TEXT,
			amount BIGINT,
			payment_dt BIGINT,
			bank TEXT,
			delivery_cost BIGINT,
			goods_total BIGINT,
			custom_fee BIGINT,
			CONSTRAINT payments_order_uid_key UNIQUE(order_uid)
		)`,
		`CREATE TABLE IF NOT EXISTS items (
			id SERIAL PRIMARY KEY,
			order_uid TEXT NOT NULL REFERENCES orders(order_uid) ON DELETE CASCADE,
			chrt_id BIGINT,
			track_number TEXT,
			price BIGINT,
			rid TEXT,
			name TEXT,
			sale BIGINT,
			size TEXT,
			total_price BIGINT,
			nm_id BIGINT,
			brand TEXT,
			status BIGINT,
			CONSTRAINT items_chrt_id_key UNIQUE(chrt_id)
		)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(q); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

func createTestOrder() *models.Order {
	gofakeit.Seed(time.Now().UnixNano())

	var order models.Order
	err := gofakeit.Struct(&order)
	if err != nil {
		panic(err)
	}

	now := time.Now()
	twoYearsAgo := now.AddDate(-2, 0, 0)
	order.DateCreated = gofakeit.DateRange(twoYearsAgo, now)

	numItems := gofakeit.Number(1, 3)
	order.Items = make([]models.Item, numItems)

	for i := range order.Items {
		var item models.Item
		gofakeit.Struct(&item)

		item.TotalPrice = item.Price * (100 - item.Sale) / 100

		order.Items[i] = item
	}

	order.Payment.GoodsTotal = 0
	for _, item := range order.Items {
		order.Payment.GoodsTotal += item.TotalPrice
	}
	order.Payment.Amount = order.Payment.GoodsTotal + order.Payment.DeliveryCost + order.Payment.CustomFee

	return &order
}

func TestPostgresDB_SaveAndGetOrder_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	order := createTestOrder()
	order.OrderUID = "test-integration-" + gofakeit.UUID()

	err := db.SaveOrder(order)
	assert.NoError(t, err)

	retrievedOrder, err := db.GetOrder(order.OrderUID)
	assert.NoError(t, err)

	assert.Equal(t, order.OrderUID, retrievedOrder.OrderUID)
	assert.Equal(t, order.TrackNumber, retrievedOrder.TrackNumber)
	assert.Equal(t, order.Entry, retrievedOrder.Entry)
	assert.Equal(t, order.Locale, retrievedOrder.Locale)
	assert.Equal(t, order.CustomerID, retrievedOrder.CustomerID)
	assert.Equal(t, order.DeliveryService, retrievedOrder.DeliveryService)
	assert.Equal(t, order.Shardkey, retrievedOrder.Shardkey)
	assert.Equal(t, order.SmID, retrievedOrder.SmID)
	assert.WithinDuration(t, order.DateCreated, retrievedOrder.DateCreated, time.Second)

	assert.Equal(t, order.Delivery.Name, retrievedOrder.Delivery.Name)
	assert.Equal(t, order.Delivery.Phone, retrievedOrder.Delivery.Phone)
	assert.Equal(t, order.Delivery.Zip, retrievedOrder.Delivery.Zip)
	assert.Equal(t, order.Delivery.City, retrievedOrder.Delivery.City)
	assert.Equal(t, order.Delivery.Address, retrievedOrder.Delivery.Address)
	assert.Equal(t, order.Delivery.Region, retrievedOrder.Delivery.Region)
	assert.Equal(t, order.Delivery.Email, retrievedOrder.Delivery.Email)

	assert.Equal(t, order.Payment.Transaction, retrievedOrder.Payment.Transaction)
	assert.Equal(t, order.Payment.Currency, retrievedOrder.Payment.Currency)
	assert.Equal(t, order.Payment.Provider, retrievedOrder.Payment.Provider)
	assert.Equal(t, order.Payment.Amount, retrievedOrder.Payment.Amount)
	assert.Equal(t, order.Payment.PaymentDt, retrievedOrder.Payment.PaymentDt)
	assert.Equal(t, order.Payment.Bank, retrievedOrder.Payment.Bank)
	assert.Equal(t, order.Payment.DeliveryCost, retrievedOrder.Payment.DeliveryCost)
	assert.Equal(t, order.Payment.GoodsTotal, retrievedOrder.Payment.GoodsTotal)
	assert.Equal(t, order.Payment.CustomFee, retrievedOrder.Payment.CustomFee)

	assert.Len(t, retrievedOrder.Items, len(order.Items))
	for i, item := range order.Items {
		retrievedItem := retrievedOrder.Items[i]
		assert.Equal(t, item.ChrtID, retrievedItem.ChrtID)
		assert.Equal(t, item.TrackNumber, retrievedItem.TrackNumber)
		assert.Equal(t, item.Price, retrievedItem.Price)
		assert.Equal(t, item.Rid, retrievedItem.Rid)
		assert.Equal(t, item.Name, retrievedItem.Name)
		assert.Equal(t, item.Sale, retrievedItem.Sale)
		assert.Equal(t, item.Size, retrievedItem.Size)
		assert.Equal(t, item.TotalPrice, retrievedItem.TotalPrice)
		assert.Equal(t, item.NmID, retrievedItem.NmID)
		assert.Equal(t, item.Brand, retrievedItem.Brand)
		assert.Equal(t, item.Status, retrievedItem.Status)
	}
}

func TestPostgresDB_GetRecentOrders_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	for i := 0; i < 5; i++ {
		order := createTestOrder()
		order.OrderUID = fmt.Sprintf("test-%d-", i) + gofakeit.UUID()
		order.DateCreated = time.Now().Add(-time.Duration(i) * time.Hour)
		err := db.SaveOrder(order)
		assert.NoError(t, err)
	}

	ordersMap, err := db.GetRecentOrders(3)
	assert.NoError(t, err)
	assert.Len(t, ordersMap, 3)

	orders := make([]*models.Order, 0, len(ordersMap))
	for _, o := range ordersMap {
		orders = append(orders, o)
	}

	sort.Slice(orders, func(i, j int) bool {
		return orders[i].DateCreated.After(orders[j].DateCreated)
	})

	for i := 1; i < len(orders); i++ {
		assert.True(t,
			orders[i].DateCreated.Before(orders[i-1].DateCreated) ||
				orders[i].DateCreated.Equal(orders[i-1].DateCreated),
			"Заказы должны быть упорядочены от новых к старым. Order %s: %v, previous: %v",
			orders[i].OrderUID, orders[i].DateCreated, orders[i-1].DateCreated,
		)
	}
}

func TestPostgresDB_GetOrder_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	order, err := db.GetOrder("nonexistent-order-123")
	assert.Error(t, err)
	assert.Nil(t, order)
	assert.Contains(t, err.Error(), "не найден")
}

func TestPostgresDB_SaveOrder_Duplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	db, cleanup := setupTestDB(t)
	defer cleanup()

	order := createTestOrder()
	order.OrderUID = "duplicate-test-123"

	err := db.SaveOrder(order)
	assert.NoError(t, err)

	err = db.SaveOrder(order)
	assert.NoError(t, err)

	retrievedOrder, err := db.GetOrder(order.OrderUID)
	assert.NoError(t, err)
	assert.Equal(t, order.OrderUID, retrievedOrder.OrderUID)
}
