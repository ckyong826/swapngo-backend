package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"swapngo-backend/internal/clients/chains"
	"swapngo-backend/internal/repositories"
	config "swapngo-backend/pkg/configs"
)

type TokenService interface {
	MintingMYRCBySUI(ctx context.Context, accountID string, amount float64) (txHash string, err error)
	TransferToTreasury(ctx context.Context, accountID string, amount float64) (txHash string, err error)
	TransferToAddress(ctx context.Context, senderUserID string, toAddress string, amount float64) (txHash string, err error)
	ExecuteSwap(ctx context.Context, userID, fromToken, toToken string, amount, slippage float64) (txHash string, actualAmount float64, err error)
}

type tokenService struct {
	walletRepo repositories.WalletRepository
	accountRepo repositories.AccountRepository
	suiClient  chains.IChainClient
}

func NewTokenService(wr repositories.WalletRepository, ar repositories.AccountRepository, sc chains.IChainClient) TokenService {
	return &tokenService{
		walletRepo: wr,
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
	txHash, err := s.suiClient.TransferMYRC(ctx, config.Env.TreasuryAddress, wallet.Address, amount)
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

	txHash, err := s.suiClient.TransferMYRC(ctx, privateKey, config.Env.TreasuryAddress, amount)
	if err != nil {
		return "", fmt.Errorf("blockchain transfer to treasury failed: %w", err)
	}

	return txHash, nil
}

func (s *tokenService) TransferToAddress(ctx context.Context, senderUserID string, toAddress string, amount float64) (string, error) {
	// 1. 查找发送者的托管钱包
	account, err := s.accountRepo.FindByUserID(ctx, uuid.Must(uuid.Parse(senderUserID)))
	if err != nil || account == nil {
		return "", fmt.Errorf("failed to fetch sender account")
	}
	
	wallet, err := s.walletRepo.FindByAccountIdAndChain(ctx, account[0].ID, "SUI")
	if err != nil || wallet == nil {
		return "", fmt.Errorf("failed to fetch sender wallet")
	}

	// 2. 调用 SUI 链进行转账
	txHash, err := s.suiClient.TransferMYRC(ctx, wallet.PrivateKey, toAddress, amount)
	if err != nil {
		return "", fmt.Errorf("blockchain transfer failed: %w", err)
	}

	return txHash, nil
}

func (s *tokenService) ExecuteSwap(ctx context.Context, userID, fromToken, toToken string, amount, slippage float64) (string, float64, error) {
	account, err := s.accountRepo.FindByUserID(ctx, uuid.Must(uuid.Parse(userID)))
	if err != nil || account == nil {
		return "", 0, fmt.Errorf("failed to fetch sender account")
	}
	
	wallet, err := s.walletRepo.FindByAccountIdAndChain(ctx, account[0].ID, "SUI")
	if err != nil || wallet == nil {
		return "", 0, fmt.Errorf("failed to fetch user wallet")
	}

	// 🔒 生产环境逻辑：
	// 1. 调用 DEX 聚合器 API (例如 Sui 上的 Hop 或 Cetus) 获取最优路由和 CallData
	// 2. 组装 PTB (Programmable Transaction Block)
	// 3. 用 wallet.PrivateKey 签名并广播

	// 模拟链上交互返回 (比如由于滑点，实际获得的数量可能略少于预期)
	mockTxHash := "0xSuiSwap_" + uuid.New().String()[:8]
	mockActualAmount := amount * 0.99 // 假设 1:1 兑换但扣除 DEX 1% 的流动性费用

	return mockTxHash, mockActualAmount, nil
}