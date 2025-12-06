package handler

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sagittarius/auction/internal/shared/metrics"
	"github.com/sagittarius/auction/services/bidder/internal/service"
)

type Handler struct {
	service *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{service: svc}
}

type SubmitBidRequest struct {
	AuctionID      string `json:"auction_id" binding:"required"`
	Amount         int64  `json:"amount" binding:"required,min=100"`
	IdempotencyKey string `json:"idempotency_key"`
}

type SubmitBidResponse struct {
	BidID         string `json:"bid_id"`
	ReservationID string `json:"reservation_id"`
	Success       bool   `json:"success"`
	ErrorMessage  string `json:"error_message,omitempty"`
}

func (h *Handler) SubmitBid(c *gin.Context) {
	startTime := time.Now()

	var req SubmitBidRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		metrics.RequestTotal.WithLabelValues("POST", "/bids", "400").Inc()
		return
	}

	// Get user ID from JWT (in production)
	userID := c.GetString("user_id")
	if userID == "" {
		// For development, use header
		userID = c.GetHeader("X-User-ID")
		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			metrics.RequestTotal.WithLabelValues("POST", "/bids", "401").Inc()
			return
		}
	}

	// Generate idempotency key if not provided
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = c.GetHeader("X-Idempotency-Key")
	}

	response, err := h.service.SubmitBid(c.Request.Context(), service.SubmitBidRequest{
		AuctionID:      req.AuctionID,
		UserID:         userID,
		Amount:         req.Amount,
		IdempotencyKey: req.IdempotencyKey,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		metrics.RequestTotal.WithLabelValues("POST", "/bids", "500").Inc()
		metrics.RequestDuration.WithLabelValues("POST", "/bids", "500").Observe(time.Since(startTime).Seconds())
		return
	}

	statusCode := http.StatusOK
	if !response.Success {
		statusCode = http.StatusBadRequest
	}

	c.JSON(statusCode, SubmitBidResponse{
		BidID:         response.BidID,
		ReservationID: response.ReservationID,
		Success:       response.Success,
		ErrorMessage:  response.ErrorMessage,
	})

	statusStr := fmt.Sprintf("%d", statusCode)
	metrics.RequestTotal.WithLabelValues("POST", "/bids", statusStr).Inc()
	metrics.RequestDuration.WithLabelValues("POST", "/bids", statusStr).Observe(time.Since(startTime).Seconds())
}

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

