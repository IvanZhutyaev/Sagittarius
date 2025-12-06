# Changelog

## [1.0.0] - Production Ready Release

### Added
- ✅ Health checks (liveness/readiness) для всех сервисов
- ✅ Graceful shutdown с таймаутами
- ✅ Connection pooling для PostgreSQL и Redis
- ✅ Кэширование балансов в Redis
- ✅ Dead Letter Queue для Kafka
- ✅ CORS middleware с whitelist
- ✅ Rate limiting по IP и пользователю
- ✅ Unit тесты для критичных компонентов
- ✅ Kubernetes манифесты для всех сервисов
- ✅ Horizontal Pod Autoscaler для Bidder Service
- ✅ Prometheus алерты
- ✅ Grafana дашборды
- ✅ Load testing скрипты (k6)

### Changed
- Улучшен graceful shutdown во всех сервисах
- Добавлены HTTP timeouts
- Оптимизированы connection pools

### Security
- CORS с whitelist для production
- Rate limiting по IP
- JWT аутентификация
- Input validation

### Infrastructure
- Kubernetes манифесты готовы к деплою
- Health probes настроены
- Resource limits определены

### Monitoring
- Prometheus алерты настроены
- Grafana дашборды созданы
- OpenTelemetry трассировка работает

## [0.1.0] - Initial Release

### Added
- Все основные микросервисы
- Docker Compose конфигурация
- Базовая инфраструктура
- Мониторинг (Prometheus, Grafana, Jaeger)

