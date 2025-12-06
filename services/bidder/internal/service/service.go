package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sagittarius/auction/internal/shared/circuitbreaker"
	"github.com/sagittarius/auction/internal/shared/idempotency"
	"github.com/sagittarius/auction/internal/shared/kafka"
	"github.com/sagittarius/auction/internal/shared/metrics"
	"github.com/sagittarius/auction/internal/shared/retry"
	"github.com/sagittarius/auction/proto/budget"
	"github.com/sagittarius/auction/proto/events"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	budgetClient    budget.BudgetServiceClient
	producer        *kafka.Producer
	idempotencyStore idempotency.Store
	circuitBreaker  *circuitbreaker.CircuitBreaker
	tracer          trace.Tracer
}

func NewService(
	budgetClient budget.BudgetServiceClient,
	producer *kafka.Producer,
	idempotencyStore idempotency.Store,
) *Service {
	tp := otel.GetTracerProvider()
	tracer := tp.Tracer("bidder-service")
	return &Service{
		budgetClient:     budgetClient,
		producer:         producer,
		idempotencyStore: idempotencyStore,
		circuitBreaker:   circuitbreaker.New("budget-service", 3, 30*time.Second),
		tracer:           tracer,
	}
}

type SubmitBidRequest struct {
	AuctionID      string
	UserID         string
	Amount         int64
	IdempotencyKey string
}

type SubmitBidResponse struct {
	BidID          string
	ReservationID  string
	Success        bool
	ErrorMessage   string
}

func (s *Service) SubmitBid(ctx context.Context, req SubmitBidRequest) (*SubmitBidResponse, error) {
	ctx, span := s.tracer.Start(ctx, "SubmitBid")
	defer span.End()

	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime).Seconds()
		metrics.BidProcessingDuration.WithLabelValues(req.AuctionID).Observe(duration)
	}()

	span.SetAttributes(
		attribute.String("auction_id", req.AuctionID),
		attribute.String("user_id", req.UserID),
		attribute.Int64("amount", req.Amount),
		attribute.String("idempotency_key", req.IdempotencyKey),
	)

	// Validate bid
	if err := s.validateBid(req); err != nil {
		metrics.BidSubmitted.WithLabelValues(req.AuctionID, "validation_error").Inc()
		return &SubmitBidResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, nil
	}

	// Check idempotency
	idempotencyKey := idempotency.GenerateKey("bid", req.IdempotencyKey, req.AuctionID, req.UserID)
	cached, err := s.idempotencyStore.Get(ctx, idempotencyKey)
	if err == nil && cached != nil {
		// Return cached response
		var cachedResponse SubmitBidResponse
		if err := json.Unmarshal(cached, &cachedResponse); err == nil {
			span.SetAttributes(attribute.Bool("idempotent", true))
			return &cachedResponse, nil
		}
	}

	bidID := uuid.New().String()

	// Reserve funds with circuit breaker
	var reservationID string
	var reserveErr error

	_, err = s.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		resp, err := s.budgetClient.ReserveFunds(ctx, &budget.ReserveFundsRequest{
			UserId:    req.UserID,
			Amount:    req.Amount,
			AuctionId: req.AuctionID,
			BidId:     bidID,
		})

		if err != nil {
			return nil, err
		}

		if !resp.Success {
			return nil, fmt.Errorf(resp.ErrorMessage)
		}

		reservationID = resp.ReservationId
		return nil, nil
	})

	if err != nil {
		metrics.BidSubmitted.WithLabelValues(req.AuctionID, "budget_error").Inc()
		span.RecordError(err)
		return &SubmitBidResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to reserve funds: %v", err),
		}, nil
	}

	// Publish bid submitted event
	event := &events.BidSubmitted{
		BidId:          bidID,
		AuctionId:      req.AuctionID,
		UserId:         req.UserID,
		Amount:         req.Amount,
		SubmittedAt:    timestamppb.Now(),
		IdempotencyKey: req.IdempotencyKey,
	}

	if err := s.producer.Publish(ctx, bidID, event); err != nil {
		// If Kafka fails, we should release funds
		s.releaseFundsAsync(ctx, reservationID, req.UserID)
		span.RecordError(err)
		return &SubmitBidResponse{
			Success:      false,
			ErrorMessage: "failed to submit bid",
		}, nil
	}

	response := &SubmitBidResponse{
		BidID:         bidID,
		ReservationID: reservationID,
		Success:       true,
	}

	// Cache response for idempotency
	responseData, _ := json.Marshal(response)
	s.idempotencyStore.Set(ctx, idempotencyKey, responseData, 1*time.Hour)

	metrics.BidSubmitted.WithLabelValues(req.AuctionID, "success").Inc()
	return response, nil
}

func (s *Service) validateBid(req SubmitBidRequest) error {
	if req.AuctionID == "" {
		return fmt.Errorf("auction_id is required")
	}
	if req.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if req.Amount < 100 {
		return fmt.Errorf("amount must be at least 100")
	}
	return nil
}

func (s *Service) releaseFundsAsync(ctx context.Context, reservationID, userID string) {
	go func() {
		_ = retry.WithExponentialBackoff(ctx, 3, 1*time.Second, func() error {
			_, err := s.budgetClient.ReleaseFunds(ctx, &budget.ReleaseFundsRequest{
				ReservationId: reservationID,
				UserId:        userID,
			})
			return err
		})
	}()
}

