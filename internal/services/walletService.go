package services

import (
	"context"
	"errors"
	"fmt"
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
	walletRepo       repositories.WalletRepository
	accountRepo      repositories.AccountRepository
	userRepo         repositories.UserRepository
	walletClient     clients.WalletClient
	tokenBalanceRepo repositories.TokenBalanceRepository
}

func NewWalletService(repo repositories.WalletRepository, ar repositories.AccountRepository, ur repositories.UserRepository, client clients.WalletClient, tbr repositories.TokenBalanceRepository) WalletService {
	return &walletService{
		walletRepo:       repo,
		accountRepo:      ar,
		userRepo:         ur,
		walletClient:     client,
		tokenBalanceRepo: tbr,
	}
}

func (s *walletService) GenerateWalletsForAccount(ctx context.Context, accountId uuid.UUID) error {
	// 1. Define each chains.
	// CEX ledger model: balances live in token_balances, not per-user on-chain
	// wallets. Only SUI is a real chain (deposit/withdraw edge), so it's the only
	// per-user wallet we generate. ETH/SOLANA/POLYGON wallets were dead.
	chains := []models.ChainName{
		models.ChainSui,
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
	accountID := accounts[0].ID

	suiWallet, err := s.walletRepo.FindByAccountIdAndChain(ctx, accountID, string(models.ChainSui))
	if err != nil || suiWallet == nil {
		return wallet.WalletInfoResponse{}, errors.New("SUI wallet not found")
	}

	// MYRC has a real on-chain representation, so it follows the chain.
	// Other tokens (BTC/ETH/USDT/USDC, plus the ledger's SUI entry) have no
	// chain query available here and stay on the ledger.
	chainMYRCStr, err := s.walletClient.GetBalance(ctx, models.ChainSui, suiWallet.Address)
	if err != nil {
		return wallet.WalletInfoResponse{}, fmt.Errorf("failed to fetch on-chain MYRC balance: %w", err)
	}
	chainMYRC, err := strconv.ParseFloat(chainMYRCStr, 64)
	if err != nil {
		return wallet.WalletInfoResponse{}, errors.New("invalid on-chain MYRC balance format")
	}

	ledgerBalances, err := s.tokenBalanceRepo.GetAllForAccount(ctx, accountID)
	if err != nil {
		return wallet.WalletInfoResponse{}, err
	}

	// Live prices to calculate MYR values.
	clients.PriceMux.RLock()
	usdMyr, _ := strconv.ParseFloat(clients.LatestPrices["USDMYR"], 64)
	btcUSD, _ := strconv.ParseFloat(clients.LatestPrices["BTCUSDT"], 64)
	ethUSD, _ := strconv.ParseFloat(clients.LatestPrices["ETHUSDT"], 64)
	suiUSD, _ := strconv.ParseFloat(clients.LatestPrices["SUIUSDT"], 64)
	clients.PriceMux.RUnlock()

	tokenPriceMYR := map[models.TokenType]float64{
		models.MYRC: 1.0,
		models.BTC:  btcUSD * usdMyr,
		models.ETH:  ethUSD * usdMyr,
		models.USDT: usdMyr,
		models.USDC: usdMyr,
		models.SUI:  suiUSD * usdMyr,
	}

	var balances []wallet.TokenBalance
	var totalMYR float64
	for _, tb := range ledgerBalances {
		amount := tb.Balance
		if tb.Token == models.MYRC {
			amount = chainMYRC
		}
		valueMYR := amount * tokenPriceMYR[tb.Token]
		balances = append(balances, wallet.TokenBalance{
			Token:    string(tb.Token),
			Amount:   amount,
			ValueMYR: valueMYR,
		})
		totalMYR += valueMYR
	}

	return wallet.WalletInfoResponse{
		SUIAddress:    suiWallet.Address,
		Email:         user.Email,
		Balances:      balances,
		TotalValueMYR: totalMYR,
	}, nil
}

func (s *walletService) CheckBalanceByUserIDAndChain(ctx context.Context, userID string, chain models.ChainName, amount float64) (bool, error) {
	userUUID := uuid.Must(uuid.Parse(userID))
	accounts, err := s.accountRepo.FindByUserID(ctx, userUUID)
	if err != nil {
		return false, err
	}
	if len(accounts) == 0 {
		return false, errors.New("account not found")
	}

	// Check the live on-chain balance, not the ledger — a withdrawal must not
	// proceed unless the chain actually holds enough funds to move.
	w, err := s.walletRepo.FindByAccountIdAndChain(ctx, accounts[0].ID, string(chain))
	if err != nil || w == nil {
		return false, errors.New("wallet not found")
	}

	balanceStr, err := s.walletClient.GetBalance(ctx, chain, w.Address)
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

	// MYRC has a real on-chain representation, so it's read from the chain,
	// not the ledger.
	w, err := s.walletRepo.FindByAccountIdAndChain(ctx, accounts[0].ID, string(models.ChainSui))
	if err != nil || w == nil {
		return "0", errors.New("SUI wallet not found")
	}

	return s.walletClient.GetBalance(ctx, models.ChainSui, w.Address)
}
	