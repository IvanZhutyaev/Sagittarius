# Стрелец (Sagittarius) - High-Load Real-Time Bidding Platform

Production-ready система онлайн-аукционов в реальном времени с экстремально низкой задержкой (< 100 мс), гарантированной консистентностью данных и горизонтальной масштабируемостью.

## 🏗️ Архитектура

Система построена на микросервисной архитектуре с использованием:
- **Event-driven подход** через Apache Kafka
- **CQRS** для разделения моделей записи и чтения
- **Saga-паттерн** для распределенных транзакций
- **Идемпотентность** всех операций
- **Горизонтальное масштабирование** каждого сервиса

## 📦 Сервисы

1. **API Gateway** - Единая точка входа, аутентификация, rate limiting
2. **Auction Engine** - Проведение аукционов (First-price, Second-price)
3. **Bidder Service** - Прием и валидация ставок
4. **Budget Service** - Управление балансами участников
5. **Notification Service** - WebSocket уведомления в реальном времени
6. **Analytics Aggregator** - Агрегация метрик в ClickHouse

## 🚀 Быстрый старт

### Требования

- Docker и Docker Compose
- Go 1.21+ (для локальной разработки)
- Python 3.11+ (для analytics service)

### Запуск всей системы

```bash
# Клонировать репозиторий
git clone <repository-url>
cd Sagittarius

# Сгенерировать protobuf код
chmod +x scripts/generate-proto.sh
./scripts/generate-proto.sh

# Запустить все сервисы
docker-compose up -d

# Проверить статус
docker-compose ps
```

### Проверка работы

```bash
# Health check API Gateway
curl http://localhost:8080/health

# Проверить метрики Prometheus
open http://localhost:9090

# Открыть Grafana
open http://localhost:3000
# Логин: admin, Пароль: admin

# Открыть Jaeger
open http://localhost:16686
```

## 📡 API Endpoints

### Submit Bid

```bash
curl -X POST http://localhost:8080/api/v1/bids \
  -H "Content-Type: application/json" \
  -H "X-User-ID: test-user-1" \
  -H "X-Idempotency-Key: unique-key-123" \
  -d '{
    "auction_id": "test-auction-1",
    "amount": 1500
  }'
```

### WebSocket Notifications

```javascript
const ws = new WebSocket('ws://localhost:8083/ws?user_id=test-user-1');
ws.onmessage = (event) => {
  console.log('Notification:', JSON.parse(event.data));
};
```

## 🧪 Тестирование

```bash
# Запустить unit тесты
go test ./... -v

# Запустить интеграционные тесты
docker-compose -f docker-compose.test.yml up --abort-on-container-exit
```

## 📊 Мониторинг

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000
- **Jaeger**: http://localhost:16686

### Ключевые метрики

- Задержка обработки ставки (P50, P95, P99)
- Количество активных аукционов
- Win rate
- Ошибки по типам
- Загрузка Kafka

## 🔧 Конфигурация

Все настройки через environment variables:

```bash
# Budget Service
DATABASE_URL=postgres://user:password@localhost:5432/sagittarius
KAFKA_BROKERS=localhost:9092

# Bidder Service
BUDGET_SERVICE_URL=localhost:50051
REDIS_URL=localhost:6379

# И т.д.
```

## 📝 Разработка

### Структура проекта

```
.
├── services/           # Микросервисы
│   ├── api-gateway/
│   ├── auction-engine/
│   ├── bidder/
│   ├── budget/
│   ├── notification/
│   └── analytics/
├── internal/shared/    # Общие библиотеки
├── proto/              # Protobuf схемы
├── migrations/         # Миграции БД
├── monitoring/         # Конфигурация мониторинга
└── docker-compose.yml  # Инфраструктура
```

### Добавление нового сервиса

1. Создать директорию в `services/`
2. Добавить Dockerfile
3. Обновить `docker-compose.yml`
4. Добавить метрики в Prometheus

## 🛡️ Безопасность

- JWT аутентификация
- Rate limiting
- Input validation
- SQL injection protection (prepared statements)
- Идемпотентность операций

## 📈 Производительность

- **Latency**: P95 < 100 мс
- **Throughput**: 10K+ операций/сек
- **Availability**: 99.95% uptime
- **Consistency**: Strong consistency для финансовых операций

## 📄 Лицензия

MIT License

## 🤝 Вклад

Пожалуйста, создавайте issues и pull requests!

## 📞 Контакты

Для вопросов и предложений создавайте issue в репозитории.
