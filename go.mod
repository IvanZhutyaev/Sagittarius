module github.com/sagittarius/auction

go 1.21

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/golang-jwt/jwt/v5 v5.2.0
	github.com/google/uuid v1.5.0
	github.com/gorilla/websocket v1.5.1
	github.com/jackc/pgx/v5 v5.5.1
	github.com/prometheus/client_golang v1.18.0
	github.com/redis/go-redis/v9 v9.3.0
	github.com/segmentio/kafka-go v0.4.47
	github.com/sony/gobreaker v1.0.0
	github.com/stretchr/testify v1.8.4
	go.opentelemetry.io/otel v1.22.0
	go.opentelemetry.io/otel/exporters/jaeger v1.17.0
	go.opentelemetry.io/otel/sdk v1.22.0
	go.opentelemetry.io/otel/trace v1.22.0
	google.golang.org/grpc v1.61.0
	google.golang.org/grpc/health/grpc_health_v1 v1.61.0
	google.golang.org/protobuf v1.32.0
	gopkg.in/yaml.v3 v3.0.1
)
