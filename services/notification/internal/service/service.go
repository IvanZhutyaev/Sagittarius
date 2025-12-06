package service

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sagittarius/auction/internal/shared/kafka"
	"github.com/sagittarius/auction/proto/events"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Service struct {
	connections map[string]*websocket.Conn
	mu          sync.RWMutex
	consumer    *kafka.Consumer
	tracer      trace.Tracer
}

func NewService(consumer *kafka.Consumer) *Service {
	tp := otel.GetTracerProvider()
	tracer := tp.Tracer("notification-service")
	return &Service{
		connections: make(map[string]*websocket.Conn),
		consumer:    consumer,
		tracer:      tracer,
	}
}

func (s *Service) RegisterConnection(userID string, conn *websocket.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connections[userID] = conn
}

func (s *Service) UnregisterConnection(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.connections, userID)
}

func (s *Service) SendNotification(userID string, message interface{}) error {
	s.mu.RLock()
	conn, exists := s.connections[userID]
	s.mu.RUnlock()

	if !exists {
		return nil // User not connected
	}

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

func (s *Service) StartConsuming(ctx context.Context) {
	// Consume auction events
	go s.consumeTopic(ctx, "auction-events", s.handleAuctionEvent)
	go s.consumeTopic(ctx, "bid-submitted", s.handleBidEvent)
}

func (s *Service) consumeTopic(ctx context.Context, topic string, handler func(context.Context, string, []byte) error) {
	consumer := kafka.NewConsumer([]string{"localhost:9092"}, topic, "notification-service")
	defer consumer.Close()

	if err := consumer.Consume(ctx, handler); err != nil {
		log.Printf("Error consuming %s: %v", topic, err)
	}
}

func (s *Service) handleAuctionEvent(ctx context.Context, key string, value []byte) error {
	ctx, span := s.tracer.Start(ctx, "handleAuctionEvent")
	defer span.End()

	var event events.AuctionFinished
	if err := json.Unmarshal(value, &event); err != nil {
		span.RecordError(err)
		return err
	}

	span.SetAttributes(
		attribute.String("auction_id", event.AuctionId),
		attribute.String("winner_id", event.WinnerId),
	)

	// Notify winner
	if event.WinnerId != "" {
		notification := map[string]interface{}{
			"type":       "auction_won",
			"auction_id": event.AuctionId,
			"price":      event.FinalPrice,
		}
		if err := s.SendNotification(event.WinnerId, notification); err != nil {
			log.Printf("Failed to send notification to winner %s: %v", event.WinnerId, err)
		}
	}

	return nil
}

func (s *Service) handleBidEvent(ctx context.Context, key string, value []byte) error {
	ctx, span := s.tracer.Start(ctx, "handleBidEvent")
	defer span.End()

	var event events.BidAccepted
	if err := json.Unmarshal(value, &event); err != nil {
		span.RecordError(err)
		return err
	}

	span.SetAttributes(
		attribute.String("bid_id", event.BidId),
		attribute.String("user_id", event.UserId),
	)

	notification := map[string]interface{}{
		"type":       "bid_accepted",
		"bid_id":     event.BidId,
		"auction_id": event.AuctionId,
		"amount":     event.Amount,
	}

	return s.SendNotification(event.UserId, notification)
}

