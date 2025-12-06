package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type CachedRepository struct {
	repo   *Repository
	client *redis.Client
	ttl    time.Duration
}

func NewCachedRepository(repo *Repository, client *redis.Client) *CachedRepository {
	return &CachedRepository{
		repo:   repo,
		client: client,
		ttl:    5 * time.Minute,
	}
}

func (r *CachedRepository) GetBalance(ctx context.Context, userID string) (*Balance, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("balance:%s", userID)
	cached, err := r.client.Get(ctx, cacheKey).Bytes()
	if err == nil {
		var balance Balance
		if err := json.Unmarshal(cached, &balance); err == nil {
			return &balance, nil
		}
	}

	// Cache miss, get from DB
	balance, err := r.repo.GetBalance(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Cache the result
	if data, err := json.Marshal(balance); err == nil {
		r.client.Set(ctx, cacheKey, data, r.ttl)
	}

	return balance, nil
}

func (r *CachedRepository) ReserveFunds(ctx context.Context, userID string, amount int64, auctionID, bidID string) (string, error) {
	// Invalidate cache
	cacheKey := fmt.Sprintf("balance:%s", userID)
	r.client.Del(ctx, cacheKey)

	return r.repo.ReserveFunds(ctx, userID, amount, auctionID, bidID)
}

func (r *CachedRepository) ReleaseFunds(ctx context.Context, reservationID string) error {
	// Get reservation to invalidate user cache
	reservation, err := r.repo.GetReservation(ctx, reservationID)
	if err == nil {
		cacheKey := fmt.Sprintf("balance:%s", reservation.UserID)
		r.client.Del(ctx, cacheKey)
	}

	return r.repo.ReleaseFunds(ctx, reservationID)
}

func (r *CachedRepository) ChargeFunds(ctx context.Context, reservationID string) (int64, string, error) {
	// Get reservation to invalidate user cache
	reservation, err := r.repo.GetReservation(ctx, reservationID)
	if err == nil {
		cacheKey := fmt.Sprintf("balance:%s", reservation.UserID)
		r.client.Del(ctx, cacheKey)
	}

	return r.repo.ChargeFunds(ctx, reservationID)
}

