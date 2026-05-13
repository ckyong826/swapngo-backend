package services

import (
	"context"

	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"

	"github.com/google/uuid"
)

type WithdrawService interface {
	CreatePendingWithdrawal(ctx context.Context, accountID uuid.UUID, amountMYRC float64, bankName, bankAccountNo, gatewayRefID string) (*models.Withdrawal, error)
}

type withdrawService struct {
	repo repositories.WithdrawRepository
}

func NewWithdrawService(repo repositories.WithdrawRepository) WithdrawService {
	return &withdrawService{repo: repo}
}

func (s *withdrawService) CreatePendingWithdrawal(ctx context.Context, accountID uuid.UUID, amountMYRC float64, bankName, bankAccountNo, gatewayRefID string) (*models.Withdrawal, error) {
	withdrawal := &models.Withdrawal{
		AccountID:     accountID,
		AmountMYR:     amountMYRC,
		AmountMYRC:    amountMYRC,
		BankName:      bankName,
		BankAccountNo: bankAccountNo,
		Status:        models.WithdrawStatePending,
		GatewayRefID:  gatewayRefID,
	}

	_, err := s.repo.Create(ctx, withdrawal)
	if err != nil {
		return nil, err
	}
	return withdrawal, nil
}
