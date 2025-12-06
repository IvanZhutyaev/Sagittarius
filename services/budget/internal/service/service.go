package service

import (
	"context"
	"fmt"
	"time"

	"github.com/sagittarius/auction/internal/shared/kafka"
	"github.com/sagittarius/auction/internal/shared/metrics"
	"github.com/sagittarius/auction/internal/shared/tracing"
	"github.com/sagittarius/auction/proto/events"
	"github.com/sagittarius/auction/services/budget/internal/repository"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	repo     *repository.Repository
	producer *kafka.Producer
	tracer   trace.Tracer
}

func NewService(repo *repository.Repository, producer *kafka.Producer) *Service {
	ctx := context.Background()
	tp := otel.GetTracerProvider()
	tracer := tp.Tracer("budget-service")
	return &Service{
		repo:     repo,
		producer: producer,
		tracer:   tracer,
	}
}

func (s *Service) ReserveFunds(ctx context.Context, userID string, amount int64, auctionID, bidID string) (string, error) {
	ctx, span := s.tracer.Start(ctx, "ReserveFunds")
	defer span.End()

	span.SetAttributes(
		attribute.String("user_id", userID),
		attribute.Int64("amount", amount),
		attribute.String("auction_id", auctionID),
		attribute.String("bid_id", bidID),
	)

	reservationID, err := s.repo.ReserveFunds(ctx, userID, amount, auctionID, bidID)
	if err != nil {
		metrics.BudgetOperations.WithLabelValues("reserve", "error").Inc()
		span.RecordError(err)
		return "", err
	}

	// Publish event
	event := &events.FundsReserved{
		ReservationId: reservationID,
		UserId:        userID,
		AuctionId:     auctionID,
		BidId:         bidID,
		Amount:        amount,
		ReservedAt:    timestamppb.Now(),
	}

	if err := s.producer.Publish(ctx, reservationID, event); err != nil {
		// Log error but don't fail the operation
		span.RecordError(err)
	}

	metrics.BudgetOperations.WithLabelValues("reserve", "success").Inc()
	return reservationID, nil
}

func (s *Service) ChargeFunds(ctx context.Context, reservationID string) (int64, string, error) {
	ctx, span := s.tracer.Start(ctx, "ChargeFunds")
	defer span.End()

	span.SetAttributes(attribute.String("reservation_id", reservationID))

	amount, userID, err := s.repo.ChargeFunds(ctx, reservationID)
	if err != nil {
		metrics.BudgetOperations.WithLabelValues("charge", "error").Inc()
		span.RecordError(err)
		return 0, "", err
	}

	// Get auction ID from reservation
	reservation, err := s.repo.GetReservation(ctx, reservationID)
	if err != nil {
		span.RecordError(err)
		return 0, "", err
	}

	// Publish event
	event := &events.FundsCharged{
		ReservationId: reservationID,
		UserId:        userID,
		AuctionId:     reservation.AuctionID,
		Amount:        amount,
		ChargedAt:     timestamppb.Now(),
	}

	if err := s.producer.Publish(ctx, reservationID, event); err != nil {
		span.RecordError(err)
	}

	metrics.BudgetOperations.WithLabelValues("charge", "success").Inc()
	return amount, userID, nil
}

func (s *Service) ReleaseFunds(ctx context.Context, reservationID string) error {
	ctx, span := s.tracer.Start(ctx, "ReleaseFunds")
	defer span.End()

	span.SetAttributes(attribute.String("reservation_id", reservationID))

	reservation, err := s.repo.GetReservation(ctx, reservationID)
	if err != nil {
		metrics.BudgetOperations.WithLabelValues("release", "error").Inc()
		span.RecordError(err)
		return err
	}

	err = s.repo.ReleaseFunds(ctx, reservationID)
	if err != nil {
		metrics.BudgetOperations.WithLabelValues("release", "error").Inc()
		span.RecordError(err)
		return err
	}

	// Publish event
	event := &events.FundsReleased{
		ReservationId: reservationID,
		UserId:        reservation.UserID,
		AuctionId:     reservation.AuctionID,
		Amount:        reservation.Amount,
		ReleasedAt:    timestamppb.Now(),
	}

	if err := s.producer.Publish(ctx, reservationID, event); err != nil {
		span.RecordError(err)
	}

	metrics.BudgetOperations.WithLabelValues("release", "success").Inc()
	return nil
}

func (s *Service) GetBalance(ctx context.Context, userID string) (int64, error) {
	ctx, span := s.tracer.Start(ctx, "GetBalance")
	defer span.End()

	span.SetAttributes(attribute.String("user_id", userID))

	balance, err := s.repo.GetBalance(ctx, userID)
	if err != nil {
		span.RecordError(err)
		return 0, err
	}

	return balance.Balance, nil
}

