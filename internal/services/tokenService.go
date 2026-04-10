package services

import (
	"context"
	"fmt"

	"swapngo-backend/internal/clients"
	"swapngo-backend/internal/repositories"
)

type TokenService interface {
	Minting(ctx context.Context, userID string, amount float64) (txHash string, err error)
}

type tokenService struct {
	walletRepo repositories.WalletRepository
	suiClient  clients.IChainClient
}

func NewTokenService(wr repositories.WalletRepository, sc clients.IChainClient) TokenService {
	return &tokenService{
		walletRepo: wr,
		suiClient:  sc,
	}
}

func (s *tokenService) MintingSui(ctx context.Context, userID string, amount float64) (string, error) {
	// 1. Fetch the user's target wallet (e.g., SUI)
	// Assuming your repo has a method to get a specific chain's wallet
	wallet, err := s.walletRepo.FindByUserIdAndChain(ctx, userID, "SUI")
	if err != nil {
		return "", fmt.Errorf("failed to find user SUI wallet: %w", err)
	}
	if wallet == nil {
		return "", fmt.Errorf("user does not have a SUI wallet initialized")
	}

	// 2. Execute the on-chain transfer/minting
	// If you ever switch from SUI to another chain for MYRC, you only change code here!
	txHash, err := s.suiClient.TransferMYRC(ctx, wallet.Address, amount)
	if err != nil {
		return "", fmt.Errorf("chain transfer failed: %w", err)
	}

	return txHash, nil
}