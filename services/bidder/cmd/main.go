package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"github.com/sagittarius/auction/internal/shared/idempotency"
	"github.com/sagittarius/auction/internal/shared/kafka"
	"github.com/sagittarius/auction/internal/shared/tracing"
	"github.com/sagittarius/auction/proto/budget"
	"github.com/sagittarius/auction/services/bidder/internal/handler"
	"github.com/sagittarius/auction/services/bidder/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	kafkaBrokers := []string{getEnv("KAFKA_BROKERS", "localhost:9092")}
	budgetServiceURL := getEnv("BUDGET_SERVICE_URL", "localhost:50051")
	redisURL := getEnv("REDIS_URL", "localhost:6379")
	httpPort := getEnv("HTTP_PORT", "8080")
	jaegerURL := getEnv("JAEGER_URL", "http://localhost:14268/api/traces")

	// Initialize tracing
	tp, err := tracing.InitTracer("bidder-service", jaegerURL)
	if err != nil {
		log.Printf("Failed to initialize tracer: %v", err)
	}
	defer func() {
		if tp != nil {
			if err := tp.Shutdown(context.Background()); err != nil {
				log.Printf("Error shutting down tracer: %v", err)
			}
		}
	}()

	// Initialize Redis for idempotency with connection pooling
	redisClient := redis.NewClient(&redis.Options{
		Addr:         redisURL,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
	})
	defer redisClient.Close()

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	idempotencyStore := idempotency.NewRedisStore(redisClient)

	// Initialize Kafka producer
	producer := kafka.NewProducer(kafkaBrokers, "bid-submitted")
	defer producer.Close()

	// Initialize gRPC client for Budget Service
	conn, err := grpc.Dial(budgetServiceURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Budget Service: %v", err)
	}
	defer conn.Close()

	budgetClient := budget.NewBudgetServiceClient(conn)

	// Initialize service and handler
	svc := service.NewService(budgetClient, producer, idempotencyStore)
	h := handler.NewHandler(svc)

	// Setup HTTP server
	router := gin.Default()

	// Middleware
	router.Use(gin.Recovery())

	// Health handler
	healthHandler := handler.NewHealthHandler(redisClient, conn)

	// Routes
	api := router.Group("/api/v1")
	{
		api.POST("/bids", h.SubmitBid)
		api.GET("/health", h.HealthCheck)
	}

	// Kubernetes probes
	router.GET("/health/live", healthHandler.Liveness)
	router.GET("/health/ready", healthHandler.Readiness)

	// Metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Start server with timeouts
	srv := &http.Server{
		Addr:         ":" + httpPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Bidder Service HTTP server listening on :%s", httpPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Bidder Service...")
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Bidder Service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

