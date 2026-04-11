package services

import (
	"context"

	"github.com/google/uuid"
	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
)

type DepositService interface {
	CreatePendingDeposit(ctx context.Context, accountID uuid.UUID, amountMYR, amountMYRC float64, gatewayRefID string) (*models.Deposit, error)
}

type depositService struct {
	repo repositories.DepositRepository
}

func NewDepositService(repo repositories.DepositRepository) DepositService {
	return &depositService{repo: repo}
}

func (s *depositService) CreatePendingDeposit(ctx context.Context, accountID uuid.UUID, amountMYR, amountMYRC float64, gatewayRefID string) (*models.Deposit, error) {
	deposit := &models.Deposit{
		AccountID:    accountID,
		AmountMYR:    amountMYR,
		AmountMYRC:   amountMYRC,
		Status:       models.DepositStatePending,
		GatewayRefID: gatewayRefID,
	}

	_, err := s.repo.Create(ctx, deposit)
	if err != nil {
		return nil, err
	}
	return deposit, nil
}
