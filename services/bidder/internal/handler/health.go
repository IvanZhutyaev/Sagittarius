package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sagittarius/auction/internal/shared/health"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

type HealthHandler struct {
	redisClient  *redis.Client
	grpcConn     *grpc.ClientConn
}

func NewHealthHandler(redisClient *redis.Client, grpcConn *grpc.ClientConn) *HealthHandler {
	return &HealthHandler{
		redisClient: redisClient,
		grpcConn:    grpcConn,
	}
}

func (h *HealthHandler) Liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
		"timestamp": time.Now(),
	})
}

func (h *HealthHandler) Readiness(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	checkers := []health.HealthChecker{}

	// Check Redis
	if h.redisClient != nil {
		checkers = append(checkers, health.NewRedisHealthChecker(h.redisClient))
	}

	// Check gRPC connection
	if h.grpcConn != nil {
		state := h.grpcConn.GetState()
		if state != connectivity.Ready && state != connectivity.Idle {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"reason": "grpc connection not ready",
				"timestamp": time.Now(),
			})
			return
		}
	}

	compositeChecker := health.NewCompositeHealthChecker(checkers...)
	status := health.CheckHealth(ctx, compositeChecker)

	if status.Status == "healthy" {
		c.JSON(http.StatusOK, status)
	} else {
		c.JSON(http.StatusServiceUnavailable, status)
	}
}

