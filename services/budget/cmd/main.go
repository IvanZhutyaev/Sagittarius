package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sagittarius/auction/internal/shared/kafka"
	"github.com/sagittarius/auction/internal/shared/tracing"
	"github.com/sagittarius/auction/proto/budget"
	"github.com/sagittarius/auction/services/budget/internal/handler"
	"github.com/sagittarius/auction/services/budget/internal/repository"
	"github.com/sagittarius/auction/services/budget/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Configuration
	dbURL := getEnv("DATABASE_URL", "postgres://user:password@localhost:5432/sagittarius?sslmode=disable")
	kafkaBrokers := []string{getEnv("KAFKA_BROKERS", "localhost:9092")}
	grpcPort := getEnv("GRPC_PORT", "50051")
	jaegerURL := getEnv("JAEGER_URL", "http://localhost:14268/api/traces")

	// Initialize tracing
	tp, err := tracing.InitTracer("budget-service", jaegerURL)
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

	// Initialize database
	db, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// Initialize Kafka producer
	producer := kafka.NewProducer(kafkaBrokers, "budget-events")
	defer producer.Close()

	// Initialize repository and service
	repo := repository.NewRepository(db)
	svc := service.NewService(repo, producer)

	// Initialize gRPC server
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	budget.RegisterBudgetServiceServer(s, handler.NewHandler(svc))

	// Health check
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(s, healthServer)
	healthServer.SetServingStatus("budget.BudgetService", grpc_health_v1.HealthCheckResponse_SERVING)

	// Reflection for development
	reflection.Register(s)

	// Graceful shutdown
	go func() {
		log.Printf("Budget Service gRPC server listening on :%s", grpcPort)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Budget Service...")
	s.GracefulStop()
	log.Println("Budget Service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

