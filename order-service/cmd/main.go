package main

import (
    "context"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "order-service/internal/cache"
    "order-service/internal/db"
    "order-service/internal/handlers"
    "order-service/internal/kafka"
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
    if val := os.Getenv("POSTGRES_DSN"); val != "" {
        postgresDSN = val
    }

    dbConn, err := db.NewPostgresDB(postgresDSN)
    if err != nil {
        log.Fatal("Не удалось подключиться к базе данных:", err)
    }
    defer dbConn.Close()

    cacheStore := cache.New()

    log.Println("Восстановление кэша из базы данных...")
    orders, err := dbConn.GetRecentOrders(1000)
    if err != nil {
        log.Printf("Восстановление кэша не удалося: %v", err)
    } else {
        cacheStore.BulkSet(orders)
        log.Printf("Кэш инициализирован с %d заказами", len(orders))
    }

    kafkaProducer := kafka.NewProducer(kafkaBrokers)
    defer kafkaProducer.Close()

    consumer := kafka.NewConsumer(kafkaBrokers, "orders", "order_service_group", dbConn, cacheStore,kafkaProducer)


    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    go consumer.Run(ctx)

    handler := handlers.NewHandler(cacheStore, dbConn, kafkaProducer)

    http.Handle("/static/",
        http.StripPrefix("/static/", http.FileServer(http.Dir("web"))))
    http.HandleFunc("/api/order/add", handler.AddOrderHandler)
    http.HandleFunc("/order/", handler.OrderHandler)
    http.HandleFunc("/", handler.WebInterfaceHandler)

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

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    log.Println("Выключение сервера...")

    ctxTimeout, cancelTimeout := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancelTimeout()
    if err := srv.Shutdown(ctxTimeout); err != nil {
        log.Fatalf("Сервер принудительно отключен: %v", err)
    }
}
