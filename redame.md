# Описание
Демонстрационный сервис для обработки и отображения данных о заказах.  
Сервис получает заказы из Kafka, сохраняет их в PostgreSQL, кэширует в памяти и предоставляет HTTP API для добавления закахов и веб-интерфейс для поиска заказов по `order_uid`.

**Ключевые особенности:**
- Автоматическое создание БД и таблиц при первом запуске
- Полная Docker-ориентированная архитектура

# Архитектура
- Kafka → Order Service → PostgreSQL
- В Order Service:
    - In-memory cache для ускоренного доступа
    - HTTP API для выдачи данных
- Web-интерфейс обращается к HTTP API

# Особенности
- Подписка на Kafka топик `orders`
- Сохранение заказов в базу (3НФ)
- Кэширование последних заказов в памяти
- Восстановление кэша из БД при старте сервиса
- Валидация сообщений и отправка некорректных в DLQ
- HTTP API для поиска заказа по `order_uid`
- HTTP API для создания заказа
- Простая веб-страница для ввода `order_uid` и получения информации

## Использование

## 1. Настройка PostgreSQL
- База `orders_db` создастся автоматически при первой инициализации проекта
- Таблицы будут загружены из миграций в первый раз автоматически

## 2. Настройка Kafka
- Создать топики:
    - `orders` — для заказов (если не создан при старте order-service)<br>
      `docker exec -it kafka kafka-topics --create --topic orders --partitions 1 --replication-factor 1 --bootstrap-server localhost:9092`<br><br>
    - `orders_dlq` — для некорректных сообщений<br>
      `docker exec -it kafka kafka-topics --create --topic orders_dlq --partitions 1 --replication-factor 1 --bootstrap-server localhost:9092`

## 3. Конфигурация сервиса
- Данные на подключение к БД и Kafka через `.env`:<br>
  DB_HOST=localhost<br>
  DB_PORT=5432<br>
  DB_USER=order_user<br>
  DB_PASSWORD=order_pass<br>
  DB_NAME=orders_db
  <br><br>
KAFKA_BROKERS=localhost:9092<br>
KAFKA_TOPIC=orders<br>
KAFKA_DLQ_TOPIC=orders_dlq<br>

## 4. Запуск сервиса
- Собрать и запустить сервис:<br>
  ### 1) cd в папку проекта<br>
  ### 2) запустить команду `docker-compose up -d`
  ### 3) подождать запуска всех контейнеров (кроме kafka-producer. Запускается вручную.)

## 5. Доступ к API
- Получение заказа по `order_uid`:<br>
  `GET /order/{order_uid}`
- Ответ в JSON формате с данными заказа
  <br><br>
- Создание заказа:<br>
  `POST /api/order/add`  
  <br>
  **Тело запроса (JSON):**
```json
{
  "order_uid": "TESTO",
  "track_number": "egrrhgrh99",
  "entry": "WBIL",
  "delivery": {
    "name": "Test Testov",
    "phone": "+9720000000",
    "zip": "2639809",
    "city": "Kiryat Mozkin",
    "address": "Ploshad Mira 15",
    "region": "Kraiot",
    "email": "test@gmail.com"
  },
  "payment": {
    "transaction": "nopstgr99",
    "request_id": "",
    "currency": "USD",
    "provider": "wbpay",
    "amount": 1817,
    "payment_dt": 1637907727,
    "bank": "alpha",
    "delivery_cost": 1500,
    "goods_total": 317,
    "custom_fee": 0
  },
  "items": [
    {
      "chrt_id": 9934930,
      "track_number": "egrrhgrh99",
      "price": 453,
      "rid": "nopstgr77",
      "name": "Mascaras",
      "sale": 30,
      "size": "0",
      "total_price": 317,
      "nm_id": 2389212,
      "brand": "Vivienne Sabo",
      "status": 202
    },
    {
      "chrt_id": 9934931,
      "track_number": "egrrhgrh99",
      "price": 453,
      "rid": "nopstgr78",
      "name": "Mascaras",
      "sale": 30,
      "size": "0",
      "total_price": 317,
      "nm_id": 2389213,
      "brand": "Vivienne Sabo",
      "status": 202
    }
  ],
  "locale": "en",
  "internal_signature": "",
  "customer_id": "test",
  "delivery_service": "meest",
  "shardkey": "9",
  "sm_id": 99,
  "date_created": "2025-11-26T06:22:19Z",
  "oof_shard": "1"
}
```
- Ответ в JSON формате со статусом запрроса
```json
{
    "message": "Заказ в обработке",
    "order_id": "TESTO",
    "status": "accepted"
}
```

## 6. Веб-интерфейс
- Открыть `http://localhost:8081/`
- Ввести `order_uid` и нажать **Найти**
- Появится информация о заказе

## 7. Тестирование Kafka
- Отправлять JSON заказов в топик `orders`
- Сервис автоматически сохранит заказ в БД и кэш
- Некорректные сообщения отправляются в `orders_dlq`

## 8. Kafka Producer
 
### Описание
Producer отвечает за отправку сообщений о заказах в Kafka.
### Основные функции:
- Отправка новых заказов в топик `orders`
- Отправка некорректных сообщений в DLQ (`orders_dlq`)
- Генерация уникальных ключей сообщений

Запуск продюсера `docker start kafka-producer`

## 9. Просмотр очереди Kafka Ui
Для удобства просомотра сообщений подключен веб интерфейс Kafka Ui
`http://localhost:8080/`