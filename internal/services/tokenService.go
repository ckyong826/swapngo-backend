package services

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"

	"swapngo-backend/internal/clients"
	"swapngo-backend/internal/clients/chains"
	"swapngo-backend/internal/kafka"
	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	config "swapngo-backend/pkg/configs"
)

type TokenService interface {
	MintingMYRCBySUI(ctx context.Context, accountID string, amount float64) (txHash string, err error)
	TransferToTreasury(ctx context.Context, accountID string, amount float64) (txHash string, err error)
	TransferToAddress(ctx context.Context, senderUserID string, toAddress string, amount float64) (txHash string, err error)
	ExecuteSwap(ctx context.Context, swapID string) error
	ExecuteSwapPayout(ctx context.Context, swapID, userAddress, fromToken, toToken, txDigest string, amountPaid, expectedAmount float64) (string, error)

	// CreditToken / DebitToken move a token's ledger balance. They are the single
	// source of truth for balances under the CEX ledger model.
	CreditToken(ctx context.Context, accountID string, token models.TokenType, amount float64) error
	DebitToken(ctx context.Context, accountID string, token models.TokenType, amount float64) error
}

type tokenService struct {
	walletRepo       repositories.WalletRepository
	swapRepo         repositories.SwapRepository
	accountRepo      repositories.AccountRepository
	tokenBalanceRepo repositories.TokenBalanceRepository
	suiClient        chains.IChainClient
}

func NewTokenService(
	wr repositories.WalletRepository,
	sr repositories.SwapRepository,
	ar repositories.AccountRepository,
	tbr repositories.TokenBalanceRepository,
	sc chains.IChainClient,
) TokenService {
	return &tokenService{
		walletRepo:       wr,
		swapRepo:         sr,
		accountRepo:      ar,
		tokenBalanceRepo: tbr,
		suiClient:        sc,
	}
}

// ── helpers ────────────────────────────────────────────────────────────────

// livePriceMYR returns the current MYR price for a given token symbol.
// Prices are kept in the in-memory cache populated by the Binance WS worker.
func livePriceMYR(token string) (float64, error) {
	clients.PriceMux.RLock()
	defer clients.PriceMux.RUnlock()

	switch token {
	case "MYRC":
		return 1.0, nil
	case "SUI":
		usdMyr := parsePrice(clients.LatestPrices["USDMYR"])
		sui := parsePrice(clients.LatestPrices["SUIUSDT"])
		if usdMyr == 0 || sui == 0 {
			return 0, fmt.Errorf("SUI price not yet available")
		}
		return sui * usdMyr, nil
	case "BTC":
		usdMyr := parsePrice(clients.LatestPrices["USDMYR"])
		btc := parsePrice(clients.LatestPrices["BTCUSDT"])
		if usdMyr == 0 || btc == 0 {
			return 0, fmt.Errorf("BTC price not yet available")
		}
		return btc * usdMyr, nil
	case "ETH":
		usdMyr := parsePrice(clients.LatestPrices["USDMYR"])
		eth := parsePrice(clients.LatestPrices["ETHUSDT"])
		if usdMyr == 0 || eth == 0 {
			return 0, fmt.Errorf("ETH price not yet available")
		}
		return eth * usdMyr, nil
	case "USDT", "USDC":
		usdMyr := parsePrice(clients.LatestPrices["USDMYR"])
		if usdMyr == 0 {
			return 0, fmt.Errorf("USD/MYR rate not yet available")
		}
		return usdMyr, nil
	default:
		return 0, fmt.Errorf("unknown token: %s", token)
	}
}

func parsePrice(s string) float64 {
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// ── existing methods (unchanged) ──────────────────────────────────────────

func (s *tokenService) MintingMYRCBySUI(ctx context.Context, accountID string, amount float64) (string, error) {
	wallet, err := s.walletRepo.FindByAccountIdAndChain(ctx, uuid.Must(uuid.Parse(accountID)), "SUI")
	if err != nil {
		return "", fmt.Errorf("failed to find user SUI wallet: %w", err)
	}
	if wallet == nil {
		return "", fmt.Errorf("user does not have a SUI wallet initialized")
	}
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
	txHash, err := s.suiClient.TransferMYRC(ctx, wallet.PrivateKey, wallet.Address, config.Env.SUIAdminAddress, amount)
	if err != nil {
		return "", fmt.Errorf("blockchain transfer to treasury failed: %w", err)
	}
	return txHash, nil
}

func (s *tokenService) TransferToAddress(ctx context.Context, fromAddress string, toAddress string, amount float64) (string, error) {
	wallet, err := s.walletRepo.FindByAddress(ctx, fromAddress)
	if err != nil || wallet == nil {
		return "", fmt.Errorf("failed to fetch sender wallet")
	}
	txHash, err := s.suiClient.TransferMYRC(ctx, wallet.PrivateKey, wallet.Address, toAddress, amount)
	if err != nil {
		return "", fmt.Errorf("blockchain transfer failed: %w", err)
	}
	return txHash, nil
}

// ── swap ───────────────────────────────────────────────────────────────────

// ExecuteSwap is called by swapBiz after the swap record is created.
// It moves the "from" tokens (on-chain or off-chain) and publishes a Kafka event
// so the worker can complete the payout asynchronously.
func (s *tokenService) ExecuteSwap(ctx context.Context, swapID string) error {
	swap, err := s.swapRepo.FindByID(ctx, uuid.Must(uuid.Parse(swapID)))
	if err != nil || swap == nil {
		return fmt.Errorf("swap not found: %s", swapID)
	}

	// Wallet address is only used as a routing key in the Kafka event so the
	// worker can resolve the account again. Balances themselves live in the ledger.
	wallet, err := s.walletRepo.FindByAccountIdAndChain(ctx, swap.AccountID, "SUI")
	if err != nil || wallet == nil {
		return fmt.Errorf("failed to fetch user SUI wallet")
	}

	fromToken := models.TokenType(swap.FromToken)
	toToken := models.TokenType(swap.ToToken)

	// Calculate estimated payout amount using live rate if not already set
	expectedAmount := swap.EstimatedToAmount
	if expectedAmount == 0 {
		fromPriceMYR, err := livePriceMYR(string(fromToken))
		if err != nil {
			return fmt.Errorf("failed to get from-token price: %w", err)
		}
		toPriceMYR, err := livePriceMYR(string(toToken))
		if err != nil {
			return fmt.Errorf("failed to get to-token price: %w", err)
		}
		if toPriceMYR == 0 {
			return fmt.Errorf("to-token price is zero")
		}
		expectedAmount = (swap.FromAmount * fromPriceMYR) / toPriceMYR
	}

	// CEX ledger model: every token's tradeable balance lives in token_balances.
	// Deduct the "from" leg here; the worker credits the "to" leg on payout.
	// No on-chain transfer and no gas — swaps are pure ledger math.
	if err := s.tokenBalanceRepo.DeductBalance(ctx, swap.AccountID, fromToken, swap.FromAmount); err != nil {
		return fmt.Errorf("failed to deduct %s balance: %w", fromToken, err)
	}
	txDigest := "ledger" // no on-chain tx for swaps

	// Publish Kafka event for the worker to handle payout
	event := kafka.SwapInitiated{
		OrderID:        swap.ID,
		UserAddress:    wallet.Address,
		FromToken:      string(fromToken),
		ToToken:        string(toToken),
		AmountPaid:     swap.FromAmount,
		ExpectedAmount: expectedAmount,
		TxDigest:       txDigest,
	}

	if err := kafka.PublishSwapInitiatedEvent(ctx, "swap_events_topic", event); err != nil {
		return fmt.Errorf("failed to publish swap event: %w", err)
	}

	return nil
}

// ExecuteSwapPayout is called by the Kafka consumer worker after it picks up
// a SwapInitiated event. It delivers the "to" tokens to the user.
func (s *tokenService) ExecuteSwapPayout(ctx context.Context, swapID, userAddress, fromToken, toToken, txDigest string, amountPaid, expectedAmount float64) (string, error) {
	toT := models.TokenType(toToken)

	// Find accountID for ledger balance operations
	wallet, err := s.walletRepo.FindByAddress(ctx, userAddress)
	if err != nil || wallet == nil {
		return "", fmt.Errorf("failed to find wallet for address %s", userAddress)
	}
	accountID := wallet.AccountID

	// CEX ledger payout: credit the "to" leg to the user's ledger balance.
	// The "from" leg was already deducted in ExecuteSwap. No on-chain verify
	// or transfer — swaps never touch the chain.
	if err := s.tokenBalanceRepo.AddBalance(ctx, accountID, toT, expectedAmount); err != nil {
		return "", fmt.Errorf("failed to credit %s balance: %w", toT, err)
	}

	return "ledger", nil
}

// CreditToken credits a token amount to the account's ledger balance.
func (s *tokenService) CreditToken(ctx context.Context, accountID string, token models.TokenType, amount float64) error {
	return s.tokenBalanceRepo.AddBalance(ctx, uuid.Must(uuid.Parse(accountID)), token, amount)
}

// DebitToken debits a token amount from the account's ledger balance.
// Returns an error if the balance would go negative.
func (s *tokenService) DebitToken(ctx context.Context, accountID string, token models.TokenType, amount float64) error {
	return s.tokenBalanceRepo.DeductBalance(ctx, uuid.Must(uuid.Parse(accountID)), token, amount)
}
