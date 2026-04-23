package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"swapngo-backend/internal/clients/chains"
	"swapngo-backend/internal/kafka"
	"swapngo-backend/internal/repositories"
	config "swapngo-backend/pkg/configs"
)

type TokenService interface {
	MintingMYRCBySUI(ctx context.Context, accountID string, amount float64) (txHash string, err error)
	TransferToTreasury(ctx context.Context, accountID string, amount float64) (txHash string, err error)
	TransferToAddress(ctx context.Context, senderUserID string, toAddress string, amount float64) (txHash string, err error)
	ExecuteSwap(ctx context.Context, swapID string) error
	ExecuteSwapPayout(ctx context.Context, userAddress, fromToken, toToken, txDigest string, amountPaid, expectedAmount float64) (string, error)
}

type tokenService struct {
	walletRepo repositories.WalletRepository
	swapRepo repositories.SwapRepository
	accountRepo repositories.AccountRepository
	suiClient  chains.IChainClient
}

func NewTokenService(wr repositories.WalletRepository, sr repositories.SwapRepository, ar repositories.AccountRepository, sc chains.IChainClient) TokenService {
	return &tokenService{
		walletRepo: wr,
		swapRepo:   sr,
		accountRepo: ar,
		suiClient:  sc,
	}
}

func (s *tokenService) MintingMYRCBySUI(ctx context.Context, accountID string, amount float64) (string, error) {
	// 1. Fetch the user's target wallet
	wallet, err := s.walletRepo.FindByAccountIdAndChain(ctx, uuid.Must(uuid.Parse(accountID)), "SUI")
	if err != nil {
		return "", fmt.Errorf("failed to find user SUI wallet: %w", err)
	}
	if wallet == nil {
		return "", fmt.Errorf("user does not have a SUI wallet initialized")
	}

	// 2. Execute the on-chain minting
	txHash, err := s.suiClient.TransferMYRC(ctx, config.Env.SUIAdminPriv, config.Env.SUIAdminAddress, wallet.Address, amount)
	if err != nil {
		return "", fmt.Errorf("chain transfer failed: %w", err)
	}

	return txHash, nil
}

func (s *tokenService) TransferToTreasury(ctx context.Context, accountID string, amount float64) (string, error) {
	wallet, err := s.walletRepo.FindByAccountIdAndChain(ctx, uuid.Must(uuid.Parse(accountID)), "SUI")
	if err != nil {
		return "", fmt.Errorf("failed to fetch user wallet: %w", err)
	}
	if wallet == nil {
		return "", fmt.Errorf("user does not have a SUI wallet")
	}

	privateKey := wallet.PrivateKey

	txHash, err := s.suiClient.TransferMYRC(ctx, privateKey, wallet.Address, config.Env.SUIAdminAddress, amount)
	if err != nil {
		return "", fmt.Errorf("blockchain transfer to treasury failed: %w", err)
	}

	return txHash, nil
}

func (s *tokenService) TransferToAddress(ctx context.Context, fromAddress string, toAddress string, amount float64) (string, error) {
	// 1. 查找发送者的托管钱包
	wallet, err := s.walletRepo.FindByAddress(ctx, fromAddress)
	if err != nil || wallet == nil {
		return "", fmt.Errorf("failed to fetch sender wallet")
	}

	// 2. 调用 SUI 链进行转账
	txHash, err := s.suiClient.TransferMYRC(ctx, wallet.PrivateKey, wallet.Address, toAddress, amount)
	if err != nil {
		return "", fmt.Errorf("blockchain transfer failed: %w", err)
	}

	return txHash, nil
}

func (s *tokenService) ExecuteSwap(ctx context.Context, swapID string) error {
	swap, err := s.swapRepo.FindByID(ctx, uuid.Must(uuid.Parse(swapID)))
	if err != nil || swap == nil {
		return fmt.Errorf("failed to fetch sender account")
	}
	
	wallet, err := s.walletRepo.FindByAccountIdAndChain(ctx, swap.AccountID, "SUI")
	if err != nil || wallet == nil {
		return fmt.Errorf("failed to fetch user wallet")
	}

	// 1. Web3 Transfer User token to Treasury
	treasuryAddr := config.Env.SUITreasuryAddress
	if treasuryAddr == "" {
		return fmt.Errorf("CRITICAL: SUI treasury address missing in config")
	}

	var txDigest string
	var errTx error

	if string(swap.FromToken) == "SUI" {
		txDigest, errTx = s.suiClient.TransferCoin(ctx, wallet.PrivateKey, wallet.Address, treasuryAddr, swap.FromAmount)
	} else if string(swap.FromToken) == "MYRC" {
		txDigest, errTx = s.suiClient.TransferMYRC(ctx, wallet.PrivateKey, wallet.Address, treasuryAddr, swap.FromAmount)
	} else {
		return fmt.Errorf("unsupported token to swap from")
	}

	if errTx != nil {
		return fmt.Errorf("failed to execute on-chain transfer to treasury: %w", errTx)
	}

	// --- TRIGGER KAFKA EVENT ---
	event := kafka.SwapInitiated{
		OrderID:        swap.ID,
		UserAddress:    wallet.Address,
		FromToken:      string(swap.FromToken),
		ToToken:        string(swap.ToToken),
		AmountPaid:     swap.FromAmount,
		ExpectedAmount: swap.EstimatedToAmount,
		TxDigest:       txDigest,
	}

	err = kafka.PublishSwapInitiatedEvent(ctx, "swap_events_topic", event)
	if err != nil {
		return fmt.Errorf("failed to publish swap event: %w", err)
	}

	return nil
}

func (s *tokenService) ExecuteSwapPayout(ctx context.Context, userAddress, fromToken, toToken, txDigest string, amountPaid, expectedAmount float64) (string, error) {
	adminAddress := config.Env.SUIAdminAddress
	adminPriv := config.Env.SUIAdminPriv

	var isValid bool
	var err error

	if fromToken == "SUI" && toToken == "MYRC" {
		isValid, err = s.suiClient.VerifyTransfer(ctx, txDigest, adminAddress, amountPaid)
	} else if fromToken == "MYRC" && toToken == "SUI" {
		isValid, err = s.suiClient.VerifyMYRCTransfer(ctx, txDigest, adminAddress, amountPaid)
	} else {
		return "", fmt.Errorf("unsupported token pair: %s to %s", fromToken, toToken)
	}

	if err != nil {
		return "", fmt.Errorf("on-chain verification failed/pending: %w", err)
	}
	if !isValid {
		return "", fmt.Errorf("transfer verification rejected")
	}

	var payoutTx string
	if toToken == "MYRC" {
		payoutTx, err = s.suiClient.TransferMYRC(ctx, adminPriv, adminAddress, userAddress, expectedAmount)
	} else if toToken == "SUI" {
		payoutTx, err = s.suiClient.TransferCoin(ctx, adminPriv, adminAddress, userAddress, expectedAmount)
	}

	if err != nil {
		return "", fmt.Errorf("failed to execute payout token %s: %w", toToken, err)
	}

	return payoutTx, nil
}