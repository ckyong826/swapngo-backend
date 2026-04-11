package services

import (
	"context"

	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	withdrawalReq "swapngo-backend/pkg/requests/withdrawal"

	"github.com/google/uuid"
)

type WithdrawService interface {
	CreatePendingWithdrawal(ctx context.Context,req *withdrawalReq.WithdrawalRequest, accountID uuid.UUID, gatewayRefID string) (*models.Withdrawal, error)
}

type withdrawService struct {
	repo repositories.WithdrawRepository
}

func NewWithdrawalService(repo repositories.WithdrawRepository) WithdrawService {
	return &withdrawService{repo: repo}
}

func (s *withdrawService) CreatePendingWithdrawal(ctx context.Context,req *withdrawalReq.WithdrawalRequest, accountID uuid.UUID,  gatewayRefID string) (*models.Withdrawal, error) {
	withdrawal := &models.Withdrawal{
		AccountID:    accountID,
		AmountMYR:    req.AmountMYRC,
		AmountMYRC:   req.AmountMYRC,
		Status:       models.WithdrawStatePending,
		GatewayRefID: gatewayRefID,
		BankName:     req.BankName,
		BankAccountNo: req.BankAccountNo,
	}

	_, err := s.repo.Create(ctx, withdrawal)
	if err != nil {
		return nil, err
	}
	return withdrawal, nil
}
