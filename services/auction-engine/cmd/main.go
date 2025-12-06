package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sagittarius/auction/internal/shared/kafka"
	"github.com/sagittarius/auction/internal/shared/tracing"
	"github.com/sagittarius/auction/proto/events"
	"github.com/sagittarius/auction/services/auction-engine/internal/engine"
	"github.com/sagittarius/auction/services/auction-engine/internal/models"
	"go.opentelemetry.io/otel/attribute"
)

func main() {
	kafkaBrokers := []string{getEnv("KAFKA_BROKERS", "localhost:9092")}
	jaegerURL := getEnv("JAEGER_URL", "http://localhost:14268/api/traces")

	// Initialize tracing
	tp, err := tracing.InitTracer("auction-engine", jaegerURL)
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

	// Initialize Kafka
	bidProducer := kafka.NewProducer(kafkaBrokers, "auction-events")
	defer bidProducer.Close()

	bidConsumer := kafka.NewConsumer(kafkaBrokers, "bid-submitted", "auction-engine")
	defer bidConsumer.Close()

	// Initialize engine
	auctionEngine := engine.NewEngine(bidProducer)

	// Start consuming bid submissions
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		log.Println("Auction Engine started, consuming bids...")
		if err := bidConsumer.Consume(ctx, func(ctx context.Context, key string, value []byte) error {
			var bidEvent events.BidSubmitted
			if err := json.Unmarshal(value, &bidEvent); err != nil {
				log.Printf("Failed to unmarshal BidSubmitted: %v", err)
				return err
			}

			return auctionEngine.ProcessBid(
				ctx,
				bidEvent.BidId,
				bidEvent.AuctionId,
				bidEvent.UserId,
				bidEvent.Amount,
			)
		}); err != nil {
			log.Printf("Error consuming messages: %v", err)
		}
	}()

	// Example: Start a test auction (in production, this would come from API)
	go func() {
		time.Sleep(2 * time.Second) // Wait for services to be ready
		testAuction := models.NewAuction(
			"test-auction-1",
			"item-1",
			"seller-1",
			1000,
			5*time.Minute,
			models.FirstPrice,
		)
		if err := auctionEngine.StartAuction(context.Background(), testAuction); err != nil {
			log.Printf("Failed to start test auction: %v", err)
		} else {
			log.Println("Test auction started: test-auction-1")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Auction Engine...")
	cancel()
	log.Println("Auction Engine stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

