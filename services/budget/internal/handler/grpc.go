package handler

import (
	"context"

	"github.com/sagittarius/auction/proto/budget"
	"github.com/sagittarius/auction/services/budget/internal/repository"
	"github.com/sagittarius/auction/services/budget/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	budget.UnimplementedBudgetServiceServer
	service *service.Service
}

func NewHandler(svc *service.Service) *Handler {
	return &Handler{service: svc}
}

func (h *Handler) ReserveFunds(ctx context.Context, req *budget.ReserveFundsRequest) (*budget.ReserveFundsResponse, error) {
	reservationID, err := h.service.ReserveFunds(ctx, req.UserId, req.Amount, req.AuctionId, req.BidId)
	if err != nil {
		if err == repository.ErrInsufficientFunds {
			return &budget.ReserveFundsResponse{
				Success:      false,
				ErrorMessage: "insufficient funds",
			}, nil
		}
		return &budget.ReserveFundsResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, status.Error(codes.Internal, err.Error())
	}

	balance, _ := h.service.GetBalance(ctx, req.UserId)

	return &budget.ReserveFundsResponse{
		Success:         true,
		ReservationId:   reservationID,
		RemainingBalance: balance,
	}, nil
}

func (h *Handler) ChargeFunds(ctx context.Context, req *budget.ChargeFundsRequest) (*budget.ChargeFundsResponse, error) {
	amount, userID, err := h.service.ChargeFunds(ctx, req.ReservationId)
	if err != nil {
		return &budget.ChargeFundsResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, status.Error(codes.Internal, err.Error())
	}

	_ = userID // Use in production for logging

	return &budget.ChargeFundsResponse{
		Success:      true,
		ChargedAmount: amount,
	}, nil
}

func (h *Handler) ReleaseFunds(ctx context.Context, req *budget.ReleaseFundsRequest) (*budget.ReleaseFundsResponse, error) {
	err := h.service.ReleaseFunds(ctx, req.ReservationId)
	if err != nil {
		return &budget.ReleaseFundsResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, status.Error(codes.Internal, err.Error())
	}

	return &budget.ReleaseFundsResponse{
		Success: true,
	}, nil
}

func (h *Handler) GetBalance(ctx context.Context, req *budget.GetBalanceRequest) (*budget.GetBalanceResponse, error) {
	balance, err := h.service.GetBalance(ctx, req.UserId)
	if err != nil {
		return &budget.GetBalanceResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, status.Error(codes.Internal, err.Error())
	}

	return &budget.GetBalanceResponse{
		Success: true,
		Balance: balance,
	}, nil
}
