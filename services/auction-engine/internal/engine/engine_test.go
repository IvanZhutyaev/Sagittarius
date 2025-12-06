package engine

import (
	"context"
	"testing"
	"time"

	"github.com/sagittarius/auction/internal/shared/kafka"
	"github.com/sagittarius/auction/services/auction-engine/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_StartAuction(t *testing.T) {
	producer := kafka.NewProducer([]string{"localhost:9092"}, "test-topic")
	defer producer.Close()
	
	engine := NewEngine(producer)
	ctx := context.Background()
	
	auction := models.NewAuction(
		"test-auction-1",
		"item-1",
		"seller-1",
		1000,
		5*time.Minute,
		models.FirstPrice,
	)
	
	err := engine.StartAuction(ctx, auction)
	require.NoError(t, err)
	
	// Check auction is stored
	retrieved, exists := engine.GetAuction("test-auction-1")
	assert.True(t, exists)
	assert.Equal(t, "test-auction-1", retrieved.ID)
	assert.Equal(t, models.AuctionStatusActive, retrieved.GetStatus())
}

func TestEngine_ProcessBid(t *testing.T) {
	producer := kafka.NewProducer([]string{"localhost:9092"}, "test-topic")
	defer producer.Close()
	
	engine := NewEngine(producer)
	ctx := context.Background()
	
	// Start auction
	auction := models.NewAuction(
		"test-auction-2",
		"item-2",
		"seller-1",
		1000,
		5*time.Minute,
		models.FirstPrice,
	)
	
	err := engine.StartAuction(ctx, auction)
	require.NoError(t, err)
	
	// Process valid bid
	err = engine.ProcessBid(ctx, "bid-1", "test-auction-2", "user-1", 1500)
	require.NoError(t, err)
	
	// Process invalid bid (too low)
	err = engine.ProcessBid(ctx, "bid-2", "test-auction-2", "user-2", 500)
	assert.Error(t, err)
}

func TestEngine_ProcessBid_NonExistentAuction(t *testing.T) {
	producer := kafka.NewProducer([]string{"localhost:9092"}, "test-topic")
	defer producer.Close()
	
	engine := NewEngine(producer)
	ctx := context.Background()
	
	err := engine.ProcessBid(ctx, "bid-1", "non-existent", "user-1", 1500)
	assert.Error(t, err)
}

