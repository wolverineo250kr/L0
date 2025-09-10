package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"order-service/internal/metrics"

	"order-service/internal/interfaces"
	"order-service/models"
	"time"

	_ "github.com/lib/pq"
)

var _ interfaces.Database = (*PostgresDB)(nil)

type PostgresDB struct {
	Conn *sql.DB
}

func NewPostgresDB(dsn string) (*PostgresDB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &PostgresDB{Conn: db}, nil
}

func (p *PostgresDB) Close() error {
	return p.Conn.Close()
}

func (p *PostgresDB) SaveOrder(order *models.Order) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.OrderProcessingTime.WithLabelValues("db", "save_order").Observe(duration)
	}()

	tx, err := p.Conn.Begin()
	if err != nil {
		metrics.DBOperations.WithLabelValues("save", "error").Inc()
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// orders
	_, err = tx.Exec(`
        INSERT INTO orders(order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard)
        VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
        ON CONFLICT (order_uid) DO UPDATE SET track_number=EXCLUDED.track_number, entry=EXCLUDED.entry`,
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature,
		order.CustomerID, order.DeliveryService, order.Shardkey, order.SmID, order.DateCreated, order.OofShard)
	if err != nil {
		metrics.DBOperations.WithLabelValues("save", "error").Inc()
		return err
	}

	// deliveries
	_, err = tx.Exec(`
        INSERT INTO deliveries(order_uid, name, phone, zip, city, address, region, email)
        VALUES($1,$2,$3,$4,$5,$6,$7,$8)
        ON CONFLICT (order_uid) DO UPDATE SET name=EXCLUDED.name`,
		order.OrderUID, order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip,
		order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email)
	if err != nil {
		metrics.DBOperations.WithLabelValues("save", "error").Inc()
		return err
	}

	// payments
	_, err = tx.Exec(`
        INSERT INTO payments(order_uid, transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee)
        VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
        ON CONFLICT (order_uid) DO UPDATE SET transaction=EXCLUDED.transaction`,
		order.OrderUID, order.Payment.Transaction, order.Payment.RequestID, order.Payment.Currency,
		order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank,
		order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee)
	if err != nil {
		metrics.DBOperations.WithLabelValues("save", "error").Inc()
		return err
	}

	_, err = tx.Exec(`DELETE FROM items WHERE order_uid = $1`, order.OrderUID)
	if err != nil {
		metrics.DBOperations.WithLabelValues("save", "error").Inc()
		return err
	}

	for _, item := range order.Items {
		_, err = tx.Exec(`
            INSERT INTO items(chrt_id, order_uid, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
            VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
            ON CONFLICT (chrt_id) DO UPDATE SET price=EXCLUDED.price`,
			item.ChrtID, order.OrderUID, item.TrackNumber, item.Price, item.Rid, item.Name,
			item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status)
		if err != nil {
			metrics.DBOperations.WithLabelValues("save", "error").Inc()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		metrics.DBOperations.WithLabelValues("save", "error").Inc()
		return err
	}
	metrics.DBOperations.WithLabelValues("save", "success").Inc()
	return nil
}

func (p *PostgresDB) GetOrder(orderUID string) (*models.Order, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		metrics.OrderProcessingTime.WithLabelValues("db", "get_order").Observe(duration)
	}()

	order := &models.Order{}

	row := p.Conn.QueryRow(`
        SELECT order_uid, track_number, entry, locale, internal_signature, customer_id, delivery_service, shardkey, sm_id, date_created, oof_shard
        FROM orders WHERE order_uid = $1`, orderUID)

	var dateCreated time.Time
	err := row.Scan(&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature,
		&order.CustomerID, &order.DeliveryService, &order.Shardkey, &order.SmID, &dateCreated, &order.OofShard)
	if errors.Is(err, sql.ErrNoRows) {
		metrics.DBOperations.WithLabelValues("get", "error").Inc()
		return nil, fmt.Errorf("заказ %s не найден: %w", orderUID, err)
	} else if err != nil {
		metrics.DBOperations.WithLabelValues("get", "error").Inc()
		return nil, err
	}
	order.DateCreated = dateCreated

	d := models.Delivery{}
	row = p.Conn.QueryRow(`
        SELECT name, phone, zip, city, address, region, email
        FROM deliveries WHERE order_uid = $1`, orderUID)
	err = row.Scan(&d.Name, &d.Phone, &d.Zip, &d.City, &d.Address, &d.Region, &d.Email)
	if errors.Is(err, sql.ErrNoRows) {
		metrics.DBOperations.WithLabelValues("get", "error").Inc()
		return nil, fmt.Errorf("доставка заказа %s не найдена: %w", orderUID, err)
	}
	order.Delivery = d

	pmt := models.Payment{}
	row = p.Conn.QueryRow(`
        SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
        FROM payments WHERE order_uid = $1`, orderUID)
	err = row.Scan(&pmt.Transaction, &pmt.RequestID, &pmt.Currency, &pmt.Provider, &pmt.Amount,
		&pmt.PaymentDt, &pmt.Bank, &pmt.DeliveryCost, &pmt.GoodsTotal, &pmt.CustomFee)
	if errors.Is(err, sql.ErrNoRows) {
		metrics.DBOperations.WithLabelValues("get", "error").Inc()
		return nil, fmt.Errorf("оплата заказа %s не найдена: %w", orderUID, err)
	}
	order.Payment = pmt

	rows, err := p.Conn.Query(`
        SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
        FROM items WHERE order_uid = $1`, orderUID)
	if err != nil {
		metrics.DBOperations.WithLabelValues("get", "error").Inc()
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); cerr != nil {
			log.Printf("Ошибка закрытия rows: %v", cerr)
		}
	}()

	items := []models.Item{}
	for rows.Next() {
		var item models.Item
		if err := rows.Scan(&item.ChrtID, &item.TrackNumber, &item.Price, &item.Rid, &item.Name, &item.Sale, &item.Size,
			&item.TotalPrice, &item.NmID, &item.Brand, &item.Status); err != nil {
			metrics.DBOperations.WithLabelValues("get", "error").Inc()
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		metrics.DBOperations.WithLabelValues("get", "error").Inc()
		return nil, fmt.Errorf("ошибка при переборе items заказа %s: %w", orderUID, err)
	}

	order.Items = items
	metrics.DBOperations.WithLabelValues("get", "success").Inc()
	return order, nil
}

func (p *PostgresDB) GetRecentOrders(limit int) (map[string]*models.Order, error) {
	rows, err := p.Conn.Query(`
        SELECT order_uid FROM orders 
        ORDER BY date_created DESC 
        LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Ошибка закрытия rows: %v", err)
		}
	}()

	orderMap := make(map[string]*models.Order)

	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			continue
		}

		order, err := p.GetOrder(uid)
		if err != nil {
			log.Printf("не удалось загрузить заказ %s: %v", uid, err)
			continue
		}

		orderMap[uid] = order
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при переборе заказов: %w", err)
	}
	return orderMap, nil
}
