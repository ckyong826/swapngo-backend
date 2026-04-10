package services

import (
	"swapngo-backend/internal/repositories"
)

type DepositService interface {
}

type depositService struct {
	repo repositories.DepositRepository
}

func NewDepositService(repo repositories.DepositRepository) DepositService {
	return &depositService{repo: repo}
}

