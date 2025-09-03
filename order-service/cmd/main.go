package main

import (
	"context"
	"log"
	"net/http"
	"order-service/internal/cache"
	"order-service/internal/db"
	"order-service/internal/handlers"
	"order-service/internal/interfaces"
	"order-service/internal/kafka"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	kafkaBrokers := []string{"kafka:9092"}
	if val := os.Getenv("KAFKA_BROKERS"); val != "" {
		kafkaBrokers = []string{val}
	}

	postgresDSN := os.Getenv("POSTGRES_DSN")
	if postgresDSN == "" {
		log.Fatal("POSTGRES_DSN is not set")
	}

	var dbConn interfaces.Database
	var cacheStore interfaces.Cache

	dbConn, err := db.NewPostgresDB(postgresDSN)
	if err != nil {
		log.Fatal("Не удалось подключиться к базе данных:", err)
	}
	defer dbConn.Close()

	// кэш с ttl 5 минут
	cacheStore = cache.New(5*time.Minute, 1000)

	// инициализация кэша из db
	log.Println("Восстановление кэша из базы данных...")
	orders, err := dbConn.GetRecentOrders(1000)
	if err != nil {
		log.Printf("Восстановление кэша не удалось: %v", err)
	} else {
		cacheStore.BulkSet(orders)
		log.Printf("Кэш инициализирован с %d заказами", len(orders))
	}

	// Kafka Consumer (читает заказы и сохраняет в БД + кэш)
	consumer := kafka.NewConsumer(
		kafkaBrokers,
		"orders",
		"order_service_group",
		"orders_dlq",
		dbConn,
		cacheStore,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go consumer.Run(ctx)

	// HTTP Handlers (только просмотр заказов)
	handler := handlers.NewHandler(cacheStore, dbConn)

	http.Handle("/static/",
		http.StripPrefix("/static/", http.FileServer(http.Dir("web"))))
	http.HandleFunc("/order/", handler.OrderHandler)
	http.HandleFunc("/", handler.WebInterfaceHandler)

	// HTTP сервер
	srv := &http.Server{
		Addr:         ":8081",
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	go func() {
		log.Println("Запуск сервера на :8081")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка сервера: %v", err)
		}
	}()

	// Корректное завершение по SIGINT/SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("отключение сервера...")

	ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelTimeout()
	if err := srv.Shutdown(ctxTimeout); err != nil {
		log.Fatalf("Сервер принудительно отключен: %v", err)
	}
}
