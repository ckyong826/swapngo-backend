package services

import (
	"context"
	"swapngo-backend/internal/clients"
	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	"swapngo-backend/pkg/requests"

	"github.com/google/uuid"
)

type WalletService interface {
	GenerateWalletsForAccount(ctx context.Context, req *requests.RegisterRequest, accountId uuid.UUID) error
}

type walletService struct {
	walletRepo   repositories.WalletRepository
	walletClient clients.WalletClient
}

func NewWalletService(repo repositories.WalletRepository, client clients.WalletClient) WalletService {
	return &walletService{
		walletRepo:   repo,
		walletClient: client,
	}
}

func (s *walletService) GenerateWalletsForAccount(ctx context.Context, req *requests.RegisterRequest, accountId uuid.UUID) error {
	// 1. Define each chains
	chains := []models.ChainName{
		models.ChainSui,
		models.ChainEthereum,
		models.ChainSolana,
		models.ChainPolygon,
	}

	// 2. Repeatedly generate address for each chain
	for _, chain := range chains {
		address, privateKey, err := s.walletClient.GenerateAddress(chain)
		if err != nil {
			return err
		}

		wallet := &models.Wallet{
			AccountID:  accountId,
			ChainName:  chain,
			Address:    address,
			PrivateKey: privateKey,
			Status:     models.WalletActive,
		}

		_, err = s.walletRepo.Create(ctx, wallet)
		if err != nil {
			return err
		}
	}

	return nil
}