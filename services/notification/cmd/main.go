package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/sagittarius/auction/internal/shared/kafka"
	"github.com/sagittarius/auction/internal/shared/tracing"
	"github.com/sagittarius/auction/services/notification/internal/service"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins in development
	},
}

func main() {
	kafkaBrokers := []string{getEnv("KAFKA_BROKERS", "localhost:9092")}
	httpPort := getEnv("HTTP_PORT", "8083")
	jaegerURL := getEnv("JAEGER_URL", "http://localhost:14268/api/traces")

	// Initialize tracing
	tp, err := tracing.InitTracer("notification-service", jaegerURL)
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

	// Initialize Kafka consumer
	consumer := kafka.NewConsumer(kafkaBrokers, "auction-events", "notification-service")

	// Initialize service
	svc := service.NewService(consumer)

	// Start consuming events
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go svc.StartConsuming(ctx)

	// Setup HTTP server
	router := gin.Default()

	// WebSocket endpoint
	router.GET("/ws", func(c *gin.Context) {
		userID := c.Query("user_id")
		if userID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
			return
		}

		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("Failed to upgrade connection: %v", err)
			return
		}
		defer conn.Close()

		svc.RegisterConnection(userID, conn)
		defer svc.UnregisterConnection(userID)

		// Keep connection alive
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Start server
	srv := &http.Server{
		Addr:    ":" + httpPort,
		Handler: router,
	}

	go func() {
		log.Printf("Notification Service listening on :%s", httpPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Notification Service...")
	cancel()
	log.Println("Notification Service stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

