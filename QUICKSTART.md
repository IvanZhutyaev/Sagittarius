# Быстрый старт - Стрелец

## 🚀 Запуск за 5 минут

### Шаг 1: Установка зависимостей

```bash
# Убедитесь, что Docker запущен
docker --version

# Установите Go (если еще не установлен)
# https://golang.org/dl/
```

### Шаг 2: Клонирование и настройка

```bash
# Клонируйте репозиторий
git clone <your-repo-url>
cd Sagittarius

# Запустите setup скрипт
chmod +x scripts/setup.sh
./scripts/setup.sh
```

### Шаг 3: Проверка работы

```bash
# Проверьте статус всех сервисов
docker-compose ps

# Должны быть запущены:
# - postgres
# - redis
# - kafka
# - budget-service
# - bidder-service
# - auction-engine
# - notification-service
# - api-gateway
# - prometheus
# - grafana
# - jaeger
```

### Шаг 4: Тестовая ставка

```bash
# Отправьте тестовую ставку
curl -X POST http://localhost:8080/api/v1/bids \
  -H "Content-Type: application/json" \
  -H "X-User-ID: test-user-1" \
  -H "X-Idempotency-Key: test-key-1" \
  -d '{
    "auction_id": "test-auction-1",
    "amount": 1500
  }'
```

Ожидаемый ответ:
```json
{
  "bid_id": "uuid-here",
  "reservation_id": "uuid-here",
  "success": true
}
```

### Шаг 5: Мониторинг

Откройте в браузере:
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Jaeger**: http://localhost:16686

## 🔧 Решение проблем

### Сервисы не запускаются

```bash
# Проверьте логи
docker-compose logs <service-name>

# Перезапустите сервис
docker-compose restart <service-name>
```

### Kafka не работает

```bash
# Проверьте Kafka
docker-compose exec kafka kafka-topics --list --bootstrap-server localhost:9092

# Создайте топики вручную
./scripts/create-kafka-topics.sh
```

### База данных не подключается

```bash
# Проверьте PostgreSQL
docker-compose exec postgres psql -U user -d sagittarius -c "SELECT 1;"

# Пересоздайте базу
docker-compose down -v
docker-compose up -d postgres
```

## 📚 Дополнительная информация

См. [README.md](README.md) для полной документации.

