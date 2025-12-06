package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sagittarius/auction/internal/shared/kafka"
	"github.com/sagittarius/auction/internal/shared/metrics"
	"github.com/sagittarius/auction/proto/events"
	"github.com/sagittarius/auction/services/auction-engine/internal/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Engine struct {
	auctions map[string]*models.Auction
	bids     map[string][]*models.Bid // auction_id -> bids
	mu       sync.RWMutex
	producer *kafka.Producer
	tracer   trace.Tracer
}

func NewEngine(producer *kafka.Producer) *Engine {
	tp := otel.GetTracerProvider()
	tracer := tp.Tracer("auction-engine")
	return &Engine{
		auctions: make(map[string]*models.Auction),
		bids:     make(map[string][]*models.Bid),
		producer: producer,
		tracer:   tracer,
	}
}

func (e *Engine) StartAuction(ctx context.Context, auction *models.Auction) error {
	ctx, span := e.tracer.Start(ctx, "StartAuction")
	defer span.End()

	span.SetAttributes(
		attribute.String("auction_id", auction.ID),
		attribute.String("auction_type", string(auction.Type)),
	)

	e.mu.Lock()
	e.auctions[auction.ID] = auction
	e.bids[auction.ID] = make([]*models.Bid, 0)
	e.mu.Unlock()

	auction.Start()

	// Publish event
	event := &events.AuctionStarted{
		AuctionId:      auction.ID,
		ItemId:         auction.ItemID,
		SellerId:       auction.SellerID,
		StartingPrice:  auction.StartingPrice,
		DurationSeconds: int64(auction.EndsAt.Sub(auction.StartedAt).Seconds()),
		AuctionType:    string(auction.Type),
		StartedAt:      timestamppb.New(auction.StartedAt),
	}

	if err := e.producer.Publish(ctx, auction.ID, event); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to publish AuctionStarted event: %w", err)
	}

	metrics.ActiveAuctions.Inc()

	// Start timer for auction end
	go e.scheduleAuctionEnd(ctx, auction.ID, auction.EndsAt)

	return nil
}

func (e *Engine) ProcessBid(ctx context.Context, bidID, auctionID, userID string, amount int64) error {
	ctx, span := e.tracer.Start(ctx, "ProcessBid")
	defer span.End()

	span.SetAttributes(
		attribute.String("bid_id", bidID),
		attribute.String("auction_id", auctionID),
		attribute.String("user_id", userID),
		attribute.Int64("amount", amount),
	)

	e.mu.RLock()
	auction, exists := e.auctions[auctionID]
	e.mu.RUnlock()

	if !exists {
		event := &events.BidRejected{
			BidId:      bidID,
			AuctionId:  auctionID,
			UserId:     userID,
			Reason:     "auction not found",
			RejectedAt: timestamppb.Now(),
		}
		e.producer.Publish(ctx, bidID, event)
		metrics.BidSubmitted.WithLabelValues(auctionID, "rejected").Inc()
		return fmt.Errorf("auction not found")
	}

	if !auction.IsActive() {
		event := &events.BidRejected{
			BidId:      bidID,
			AuctionId:  auctionID,
			UserId:     userID,
			Reason:     "auction is not active",
			RejectedAt: timestamppb.Now(),
		}
		e.producer.Publish(ctx, bidID, event)
		metrics.BidSubmitted.WithLabelValues(auctionID, "rejected").Inc()
		return fmt.Errorf("auction is not active")
	}

	bid := &models.Bid{
		ID:        bidID,
		AuctionID: auctionID,
		UserID:    userID,
		Amount:    amount,
		Timestamp: time.Now(),
	}

	if !auction.ProcessBid(bid) {
		event := &events.BidRejected{
			BidId:      bidID,
			AuctionId:  auctionID,
			UserId:     userID,
			Reason:     "bid amount too low",
			RejectedAt: timestamppb.Now(),
		}
		e.producer.Publish(ctx, bidID, event)
		metrics.BidSubmitted.WithLabelValues(auctionID, "rejected").Inc()
		return fmt.Errorf("bid amount too low")
	}

	// Store bid
	e.mu.Lock()
	e.bids[auctionID] = append(e.bids[auctionID], bid)
	e.mu.Unlock()

	// Publish events
	acceptedEvent := &events.BidAccepted{
		BidId:      bidID,
		AuctionId:  auctionID,
		UserId:     userID,
		Amount:     amount,
		AcceptedAt: timestamppb.Now(),
	}
	if err := e.producer.Publish(ctx, bidID, acceptedEvent); err != nil {
		span.RecordError(err)
	}

	newLeaderEvent := &events.NewLeader{
		AuctionId: auctionID,
		BidId:     bidID,
		UserId:    userID,
		Amount:    amount,
		Timestamp: timestamppb.Now(),
	}
	if err := e.producer.Publish(ctx, auctionID, newLeaderEvent); err != nil {
		span.RecordError(err)
	}

	metrics.BidSubmitted.WithLabelValues(auctionID, "accepted").Inc()
	return nil
}

func (e *Engine) scheduleAuctionEnd(ctx context.Context, auctionID string, endsAt time.Time) {
	duration := time.Until(endsAt)
	if duration <= 0 {
		e.finishAuction(ctx, auctionID)
		return
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-timer.C:
		e.finishAuction(ctx, auctionID)
	case <-ctx.Done():
		return
	}
}

func (e *Engine) finishAuction(ctx context.Context, auctionID string) {
	ctx, span := e.tracer.Start(ctx, "FinishAuction")
	defer span.End()

	e.mu.Lock()
	auction, exists := e.auctions[auctionID]
	if !exists {
		e.mu.Unlock()
		return
	}

	if auction.GetStatus() == models.AuctionStatusFinished {
		e.mu.Unlock()
		return
	}

	bids := e.bids[auctionID]
	e.mu.Unlock()

	var winnerID, winningBidID string
	var finalPrice int64

	if len(bids) == 0 {
		// No bids, auction ends without winner
		auction.Finish("", "", auction.StartingPrice)
	} else {
		// Determine winner based on auction type
		if auction.Type == models.FirstPrice {
			// First price: winner pays their bid
			highestBid := bids[0]
			for _, bid := range bids[1:] {
				if bid.Amount > highestBid.Amount {
					highestBid = bid
				}
			}
			winnerID = highestBid.UserID
			winningBidID = highestBid.ID
			finalPrice = highestBid.Amount
		} else {
			// Second price (Vickrey): winner pays second highest bid
			if len(bids) == 1 {
				winnerID = bids[0].UserID
				winningBidID = bids[0].ID
				finalPrice = auction.StartingPrice
			} else {
				// Find highest and second highest
				highest := bids[0]
				secondHighest := bids[1]
				if secondHighest.Amount > highest.Amount {
					highest, secondHighest = secondHighest, highest
				}

				for _, bid := range bids[2:] {
					if bid.Amount > highest.Amount {
						secondHighest = highest
						highest = bid
					} else if bid.Amount > secondHighest.Amount {
						secondHighest = bid
					}
				}

				winnerID = highest.UserID
				winningBidID = highest.ID
				finalPrice = secondHighest.Amount
			}
		}

		auction.Finish(winnerID, winningBidID, finalPrice)
	}

	// Publish event
	event := &events.AuctionFinished{
		AuctionId:    auctionID,
		WinnerId:     winnerID,
		WinningBidId: winningBidID,
		FinalPrice:   finalPrice,
		FinishedAt:   timestamppb.Now(),
	}

	if err := e.producer.Publish(ctx, auctionID, event); err != nil {
		span.RecordError(err)
	}

	metrics.ActiveAuctions.Dec()
}

func (e *Engine) GetAuction(auctionID string) (*models.Auction, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	auction, exists := e.auctions[auctionID]
	return auction, exists
}

