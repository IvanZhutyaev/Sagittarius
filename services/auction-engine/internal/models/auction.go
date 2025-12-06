package models

import (
	"sync"
	"time"
)

type AuctionType string

const (
	FirstPrice  AuctionType = "first_price"
	SecondPrice AuctionType = "second_price"
)

type AuctionStatus string

const (
	AuctionStatusPending   AuctionStatus = "pending"
	AuctionStatusActive    AuctionStatus = "active"
	AuctionStatusFinished  AuctionStatus = "finished"
	AuctionStatusCancelled AuctionStatus = "cancelled"
)

type Auction struct {
	ID            string
	ItemID        string
	SellerID      string
	StartingPrice int64
	CurrentPrice  int64
	Type          AuctionType
	Status        AuctionStatus
	StartedAt     time.Time
	EndsAt        time.Time
	WinnerID      string
	WinningBidID  string
	FinalPrice    int64
	mu            sync.RWMutex
}

type Bid struct {
	ID        string
	AuctionID string
	UserID    string
	Amount    int64
	Timestamp time.Time
}

func NewAuction(id, itemID, sellerID string, startingPrice int64, duration time.Duration, auctionType AuctionType) *Auction {
	now := time.Now()
	return &Auction{
		ID:            id,
		ItemID:        itemID,
		SellerID:      sellerID,
		StartingPrice: startingPrice,
		CurrentPrice:  startingPrice,
		Type:          auctionType,
		Status:        AuctionStatusPending,
		StartedAt:     now,
		EndsAt:        now.Add(duration),
	}
}

func (a *Auction) Start() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Status = AuctionStatusActive
	a.StartedAt = time.Now()
}

func (a *Auction) IsActive() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.Status == AuctionStatusActive && time.Now().Before(a.EndsAt)
}

func (a *Auction) IsFinished() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.Status == AuctionStatusFinished || time.Now().After(a.EndsAt)
}

func (a *Auction) ProcessBid(bid *Bid) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.Status != AuctionStatusActive {
		return false
	}

	if time.Now().After(a.EndsAt) {
		return false
	}

	if bid.Amount <= a.CurrentPrice {
		return false
	}

	a.CurrentPrice = bid.Amount
	return true
}

func (a *Auction) Finish(winnerID, winningBidID string, finalPrice int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Status = AuctionStatusFinished
	a.WinnerID = winnerID
	a.WinningBidID = winningBidID
	a.FinalPrice = finalPrice
}

func (a *Auction) GetCurrentPrice() int64 {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.CurrentPrice
}

func (a *Auction) GetStatus() AuctionStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.Status
}

