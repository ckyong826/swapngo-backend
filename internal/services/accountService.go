package services

import (
	"context"
	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	authReq "swapngo-backend/pkg/requests/auth"

	"github.com/google/uuid"
)

type AccountService interface {
	CreateAccount(ctx context.Context, req *authReq.RegisterRequest, userID uuid.UUID) (models.Account, error)
}

type accountService struct {
	accountRepo repositories.AccountRepository
}

func NewAccountService(repo repositories.AccountRepository) AccountService {
	return &accountService{
		accountRepo: repo,
	}
}

func (s *accountService) CreateAccount(ctx context.Context, req *authReq.RegisterRequest, userID uuid.UUID) (models.Account, error) {
	
	account := models.Account{
		UserID: userID,
		AccountName: req.AccountName,
		CustodyType: models.CustodyType(req.CustodyType),
		Status: models.AccountActive,
	}

	acc, err := s.accountRepo.Create(ctx, &account)
	return *acc, err
}