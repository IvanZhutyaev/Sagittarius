package repository

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	// Use test database URL
	dbURL := "postgres://user:password@localhost:5432/sagittarius_test?sslmode=disable"
	
	config, err := pgxpool.ParseConfig(dbURL)
	require.NoError(t, err)
	
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	require.NoError(t, err)
	
	// Create tables for testing
	_, err = pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS balances (
			user_id VARCHAR(255) PRIMARY KEY,
			balance BIGINT NOT NULL DEFAULT 0,
			version BIGINT NOT NULL DEFAULT 0,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		
		CREATE TABLE IF NOT EXISTS reservations (
			id VARCHAR(255) PRIMARY KEY,
			user_id VARCHAR(255) NOT NULL,
			auction_id VARCHAR(255) NOT NULL,
			bid_id VARCHAR(255) NOT NULL,
			amount BIGINT NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'reserved',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`)
	require.NoError(t, err)
	
	return pool
}

func TestRepository_GetBalance(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	
	repo := NewRepository(pool)
	ctx := context.Background()
	
	// Test getting non-existent balance (should create)
	balance, err := repo.GetBalance(ctx, "test-user-1")
	require.NoError(t, err)
	assert.Equal(t, "test-user-1", balance.UserID)
	assert.Equal(t, int64(0), balance.Balance)
}

func TestRepository_ReserveFunds(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	
	repo := NewRepository(pool)
	ctx := context.Background()
	
	// Create balance first
	_, err := repo.GetBalance(ctx, "test-user-1")
	require.NoError(t, err)
	
	// Add funds
	_, err = pool.Exec(ctx, "UPDATE balances SET balance = 10000 WHERE user_id = $1", "test-user-1")
	require.NoError(t, err)
	
	// Reserve funds
	reservationID, err := repo.ReserveFunds(ctx, "test-user-1", 1000, "auction-1", "bid-1")
	require.NoError(t, err)
	assert.NotEmpty(t, reservationID)
	
	// Check balance decreased
	balance, err := repo.GetBalance(ctx, "test-user-1")
	require.NoError(t, err)
	assert.Equal(t, int64(9000), balance.Balance)
}

func TestRepository_ReserveFunds_InsufficientFunds(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	
	repo := NewRepository(pool)
	ctx := context.Background()
	
	// Create balance with 0 funds
	_, err := repo.GetBalance(ctx, "test-user-2")
	require.NoError(t, err)
	
	// Try to reserve more than available
	_, err = repo.ReserveFunds(ctx, "test-user-2", 1000, "auction-1", "bid-1")
	assert.Error(t, err)
	assert.Equal(t, ErrInsufficientFunds, err)
}

func TestRepository_ChargeFunds(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	
	repo := NewRepository(pool)
	ctx := context.Background()
	
	// Setup: create balance and reserve funds
	_, err := repo.GetBalance(ctx, "test-user-3")
	require.NoError(t, err)
	
	_, err = pool.Exec(ctx, "UPDATE balances SET balance = 10000 WHERE user_id = $1", "test-user-3")
	require.NoError(t, err)
	
	reservationID, err := repo.ReserveFunds(ctx, "test-user-3", 1000, "auction-1", "bid-1")
	require.NoError(t, err)
	
	// Charge funds
	amount, userID, err := repo.ChargeFunds(ctx, reservationID)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), amount)
	assert.Equal(t, "test-user-3", userID)
}

func TestRepository_ReleaseFunds(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()
	
	repo := NewRepository(pool)
	ctx := context.Background()
	
	// Setup: create balance and reserve funds
	_, err := repo.GetBalance(ctx, "test-user-4")
	require.NoError(t, err)
	
	_, err = pool.Exec(ctx, "UPDATE balances SET balance = 10000 WHERE user_id = $1", "test-user-4")
	require.NoError(t, err)
	
	reservationID, err := repo.ReserveFunds(ctx, "test-user-4", 1000, "auction-1", "bid-1")
	require.NoError(t, err)
	
	// Release funds
	err = repo.ReleaseFunds(ctx, reservationID)
	require.NoError(t, err)
	
	// Check balance restored
	balance, err := repo.GetBalance(ctx, "test-user-4")
	require.NoError(t, err)
	assert.Equal(t, int64(10000), balance.Balance)
}

