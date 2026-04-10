package services

import (
	"context"
	"errors"
	"swapngo-backend/internal/clients"
	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	"swapngo-backend/pkg/responses/wallet"

	"github.com/google/uuid"
)

type WalletService interface {
	GenerateWalletsForAccount(ctx context.Context, accountId uuid.UUID) error
	GetTotalBalanceByUserID(ctx context.Context, userID string) ([]wallet.WalletResponse, error)
}

type walletService struct {
	walletRepo   repositories.WalletRepository
	accountRepo  repositories.AccountRepository
	walletClient clients.WalletClient
}

func NewWalletService(repo repositories.WalletRepository, client clients.WalletClient) WalletService {
	return &walletService{
		walletRepo:   repo,
		walletClient: client,
	}
}

func (s *walletService) GenerateWalletsForAccount(ctx context.Context, accountId uuid.UUID) error {
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

func (s *walletService) GetTotalBalanceByUserID(ctx context.Context, userID string) ([]wallet.WalletResponse, error) {
	var walletResponses []wallet.WalletResponse
	// 1. Find account for user
	accounts, err := s.accountRepo.FindByUserID(ctx, uuid.Must(uuid.Parse(userID)))
	if err != nil {
		return walletResponses, err
	}
	if len(accounts) == 0 {
		return walletResponses, errors.New("account not found")
	}

	// TODO : Temporarily only support one account for one user
	if len(accounts) > 1 {
		return walletResponses, errors.New("multiple accounts found")
	}
	
	// 2. Find all wallets for the user
	wallets, err := s.walletRepo.FindByAccountId(ctx, accounts[0].ID)
	if err != nil {
		return walletResponses, err
	}

	// 3. Get balance for each wallet
	for _, w := range wallets {
    balance, err := s.walletClient.GetBalance(ctx, w.ChainName, w.Address)
    if err != nil {
        continue 
    }
    walletResponses = append(walletResponses, wallet.WalletResponse{
        ChainName:     string(w.ChainName),
        PublicAddress: w.Address,
        Balance:       balance, 
    })
	}

	return walletResponses, nil
}