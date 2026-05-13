package services

import (
	"context"
	"errors"
	"strconv"
	"swapngo-backend/internal/clients"
	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	"swapngo-backend/pkg/responses/wallet"

	"github.com/google/uuid"
)

type WalletService interface {
	GenerateWalletsForAccount(ctx context.Context, accountId uuid.UUID) error
	GetTotalBalanceByUserID(ctx context.Context, userID string) ([]wallet.WalletResponse, error)
	GetWalletInfo(ctx context.Context, userID string) (wallet.WalletInfoResponse, error)
	CheckBalanceByUserIDAndChain(ctx context.Context, userID string, chain models.ChainName, amount float64) (bool, error)
	GetMYRCBalanceByUserID(ctx context.Context, userID string) (string, error)
}

type walletService struct {
	walletRepo   repositories.WalletRepository
	accountRepo  repositories.AccountRepository
	userRepo     repositories.UserRepository
	walletClient clients.WalletClient
}

func NewWalletService(repo repositories.WalletRepository, ar repositories.AccountRepository, ur repositories.UserRepository, client clients.WalletClient) WalletService {
	return &walletService{
		walletRepo:   repo,
		accountRepo:  ar,
		userRepo:     ur,
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

func (s *walletService) GetWalletInfo(ctx context.Context, userID string) (wallet.WalletInfoResponse, error) {
	userUUID := uuid.Must(uuid.Parse(userID))

	user, err := s.userRepo.FindByID(ctx, userUUID)
	if err != nil || user == nil {
		return wallet.WalletInfoResponse{}, errors.New("user not found")
	}

	accounts, err := s.accountRepo.FindByUserID(ctx, userUUID)
	if err != nil || len(accounts) == 0 {
		return wallet.WalletInfoResponse{}, errors.New("account not found")
	}

	suiWallet, err := s.walletRepo.FindByAccountIdAndChain(ctx, accounts[0].ID, string(models.ChainSui))
	if err != nil || suiWallet == nil {
		return wallet.WalletInfoResponse{}, errors.New("SUI wallet not found")
	}

	balanceStr, _ := s.walletClient.GetBalance(ctx, suiWallet.ChainName, suiWallet.Address)
	myrcAmount, _ := strconv.ParseFloat(balanceStr, 64)

	return wallet.WalletInfoResponse{
		SUIAddress: suiWallet.Address,
		Email:      user.Email,
		Balances: []wallet.TokenBalance{
			{Token: "MYRC", Amount: myrcAmount, ValueMYR: myrcAmount},
		},
		TotalValueMYR: myrcAmount,
	}, nil
}

func (s *walletService) CheckBalanceByUserIDAndChain(ctx context.Context, userID string, chain models.ChainName, amount float64) (bool, error) {
	balanceStr, err := s.GetMYRCBalanceByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	balance, err := strconv.ParseFloat(balanceStr, 64)
	if err != nil {
		return false, errors.New("invalid balance format")
	}
	return balance >= amount, nil
}

func (s *walletService) GetMYRCBalanceByUserID(ctx context.Context, userID string) (string, error) {
	userUUID := uuid.Must(uuid.Parse(userID))
	accounts, err := s.accountRepo.FindByUserID(ctx, userUUID)
	if err != nil {
		return "0", err
	}
	if len(accounts) == 0 {
		return "0", errors.New("account not found")
	}

	// TODO : Temporarily only support one account for one user
	if len(accounts) > 1 {
		return "0", errors.New("multiple accounts found")
	}

	wallet, err := s.walletRepo.FindByAccountIdAndChain(ctx, accounts[0].ID, string(models.ChainSui))
	if err != nil {
		return "0", err
	}

	balance, err := s.walletClient.GetBalance(ctx, wallet.ChainName, wallet.Address)
	if err != nil {
		return "0", err
	}

	return balance,nil
}
	