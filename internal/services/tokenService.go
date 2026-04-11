package services

import (
	"context"
	"fmt"
	"github.com/google/uuid"

	"swapngo-backend/internal/clients/chains"
	"swapngo-backend/internal/repositories"
)

type TokenService interface {
	MintingMYRCBySUI(ctx context.Context, accountID string, amount float64) (txHash string, err error)
}

type tokenService struct {
	walletRepo repositories.WalletRepository
	suiClient  chains.IChainClient
}

func NewTokenService(wr repositories.WalletRepository, sc chains.IChainClient) TokenService {
	return &tokenService{
		walletRepo: wr,
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
	txHash, err := s.suiClient.TransferMYRC(ctx, wallet.Address, amount)
	if err != nil {
		return "", fmt.Errorf("chain transfer failed: %w", err)
	}

	return txHash, nil
}