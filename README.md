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

## 5. Веб-интерфейс
- Открыть `http://localhost:8081/`
- Ввести `order_uid` и нажать **Найти**
- Появится информация о заказе

## 6. Тестирование Kafka
- Отправлять JSON заказов в топик `orders`
- Сервис автоматически сохранит заказ в БД и кэш
- Некорректные сообщения отправляются в `orders_dlq`

## 7. Kafka Producer
 
### Описание
Producer отвечает за отправку сообщений о заказах в Kafka.
### Основные функции:
- Отправка новых заказов в топик `orders`
- Отправка некорректных сообщений в DLQ (`orders_dlq`)
- Генерация уникальных ключей сообщений

Запуск продюсера `docker start kafka-producer`

## 8. Просмотр очереди Kafka Ui
Для удобства просомотра сообщений подключен веб интерфейс Kafka Ui
`http://localhost:8080/`

## 9. Подключение Prometheus к Grafana
Адрес Grafana ```http://localhost:3000/```
1. В боковом меню выберите **Configuration → Data Sources**.
2. Нажмите **Add data source**.
3. Выберите **Prometheus**.

Prometheus server URL: указать  ```http://host.docker.internal:9090 ```

## 10. jaeger
Адрес http://localhost:16686/
 
## 11. Тестирование проекта (Windows 10)

### Юнит-тесты
```
cd <ПУТЬ_ДО_ПРОЕКТА>\L0\order-service
go test -v ./internal/... -short
Интеграционные тесты
powershell
Copy code
# Поднятие тестовых контейнеров
docker-compose -f ..\docker-compose.test.yml up -d
Start-Sleep -Seconds 10
```
# Запуск интеграционных тестов
```
go test -v ./internal/db -tags=integration
go test -v ./internal/kafka -tags=integration
go test -v ./internal/middleware -tags=integration
go test -v ./internal/tracing -tags=integration
```
# Остановка контейнеров
```
docker-compose -f ..\docker-compose.test.yml down
```
# Все тесты подряд
 ```
go test -v ./internal/... -short
docker-compose -f ..\docker-compose.test.yml up -d
Start-Sleep -Seconds 10
go test -v ./internal/db -tags=integration
go test -v ./internal/kafka -tags=integration
go test -v ./internal/middleware -tags=integration
go test -v ./internal/tracing -tags=integration
docker-compose -f ..\docker-compose.test.yml down
```
