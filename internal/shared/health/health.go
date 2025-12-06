package health

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type HealthChecker interface {
	Check(ctx context.Context) error
}

type DatabaseHealthChecker struct {
	db *pgxpool.Pool
}

func NewDatabaseHealthChecker(db *pgxpool.Pool) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{db: db}
}

func (h *DatabaseHealthChecker) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.db.Ping(ctx)
}

type RedisHealthChecker struct {
	client *redis.Client
}

func NewRedisHealthChecker(client *redis.Client) *RedisHealthChecker {
	return &RedisHealthChecker{client: client}
}

func (h *RedisHealthChecker) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return h.client.Ping(ctx).Err()
}

type CompositeHealthChecker struct {
	checkers []HealthChecker
}

func NewCompositeHealthChecker(checkers ...HealthChecker) *CompositeHealthChecker {
	return &CompositeHealthChecker{checkers: checkers}
}

func (c *CompositeHealthChecker) Check(ctx context.Context) error {
	for _, checker := range c.checkers {
		if err := checker.Check(ctx); err != nil {
			return fmt.Errorf("health check failed: %w", err)
		}
	}
	return nil
}

type HealthStatus struct {
	Status    string            `json:"status"`
	Checks    map[string]string `json:"checks,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

func CheckHealth(ctx context.Context, checker HealthChecker) HealthStatus {
	err := checker.Check(ctx)
	status := "healthy"
	if err != nil {
		status = "unhealthy"
	}

	return HealthStatus{
		Status:    status,
		Timestamp: time.Now(),
	}
}

