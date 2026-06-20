package services

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"

	"swapngo-backend/internal/clients"
	"swapngo-backend/internal/clients/chains"
	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	config "swapngo-backend/pkg/configs"
)

type TokenService interface {
	MintingMYRCBySUI(ctx context.Context, accountID string, amount float64) (txHash string, err error)
	TransferToTreasury(ctx context.Context, accountID string, amount float64) (txHash string, err error)
	TransferToAddress(ctx context.Context, senderUserID string, toAddress string, amount float64) (txHash string, err error)

	// CreditToken / DebitToken move a token's ledger balance. They are the single
	// source of truth for balances under the CEX ledger model.
	CreditToken(ctx context.Context, accountID string, token models.TokenType, amount float64) error
	DebitToken(ctx context.Context, accountID string, token models.TokenType, amount float64) error

	// ExecuteSwapLegs runs both legs of a swap (deduct "from", credit "to") for
	// the worker. MYRC legs move real on-chain MYRC (treasury<->user) before
	// touching the ledger, since MYRC balance must follow the chain; other
	// tokens are ledger-only. If toAmount is 0 it is resolved here from live
	// prices. Returns the resolved "to" amount and a representative tx hash.
	ExecuteSwapLegs(ctx context.Context, accountID, fromToken, toToken string, fromAmount, toAmount float64) (resolvedToAmount float64, txHash string, err error)
}

type tokenService struct {
	walletRepo       repositories.WalletRepository
	accountRepo      repositories.AccountRepository
	tokenBalanceRepo repositories.TokenBalanceRepository
	suiClient        chains.IChainClient
}

func NewTokenService(
	wr repositories.WalletRepository,
	ar repositories.AccountRepository,
	tbr repositories.TokenBalanceRepository,
	sc chains.IChainClient,
) TokenService {
	return &tokenService{
		walletRepo:       wr,
		accountRepo:      ar,
		tokenBalanceRepo: tbr,
		suiClient:        sc,
	}
}

// ── helpers ────────────────────────────────────────────────────────────────

// livePriceMYR returns the current MYR price for a given token symbol.
// Prices are kept in the in-memory cache populated by the price worker
// (CoinGecko + open.er-api.com); if a price isn't cached yet (e.g. CoinGecko
// unreachable), a fallback rate is used instead.
func livePriceMYR(token string) (float64, error) {
	clients.PriceMux.RLock()
	defer clients.PriceMux.RUnlock()

	switch token {
	case "MYRC":
		return 1.0, nil
	case "SUI":
		usdMyr := clients.PriceOrFallback("USDMYR", clients.FallbackUSDMYR)
		sui := clients.PriceOrFallback("SUIUSDT", clients.FallbackSUIUSDT)
		return sui * usdMyr, nil
	case "BTC":
		usdMyr := clients.PriceOrFallback("USDMYR", clients.FallbackUSDMYR)
		btc := clients.PriceOrFallback("BTCUSDT", clients.FallbackBTCUSDT)
		return btc * usdMyr, nil
	case "ETH":
		usdMyr := clients.PriceOrFallback("USDMYR", clients.FallbackUSDMYR)
		eth := clients.PriceOrFallback("ETHUSDT", clients.FallbackETHUSDT)
		return eth * usdMyr, nil
	case "USDT", "USDC":
		usdMyr := clients.PriceOrFallback("USDMYR", clients.FallbackUSDMYR)
		return usdMyr, nil
	default:
		return 0, fmt.Errorf("unknown token: %s", token)
	}
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

// ExecuteSwapLegs is called by the Kafka consumer worker after a swap order
// is created. MYRC legs move real on-chain MYRC (treasury<->user) before the
// ledger is touched, so MYRC balance always follows the chain; other tokens
// (no on-chain representation in this app) stay ledger-only. If the to-leg
// fails after the from-leg already succeeded, the from-leg is refunded so a
// partial failure doesn't strand the user's funds mid-swap.
func (s *tokenService) ExecuteSwapLegs(ctx context.Context, accountID, fromToken, toToken string, fromAmount, toAmount float64) (float64, string, error) {
	fromT := models.TokenType(fromToken)
	toT := models.TokenType(toToken)
	acctUUID := uuid.Must(uuid.Parse(accountID))

	if toAmount == 0 {
		fromPriceMYR, err := livePriceMYR(fromToken)
		if err != nil {
			return 0, "", fmt.Errorf("failed to get from-token price: %w", err)
		}
		toPriceMYR, err := livePriceMYR(toToken)
		if err != nil {
			return 0, "", fmt.Errorf("failed to get to-token price: %w", err)
		}
		if toPriceMYR == 0 {
			return 0, "", fmt.Errorf("to-token price is zero")
		}
		toAmount = (fromAmount * fromPriceMYR) / toPriceMYR
	}

	// From-leg: MYRC moves real on-chain funds to treasury first; the ledger
	// is only debited once that confirms. Other tokens are ledger-only.
	var fromTxHash string
	if fromT == models.MYRC {
		h, err := s.TransferToTreasury(ctx, accountID, fromAmount)
		if err != nil {
			return 0, "", fmt.Errorf("failed to transfer %s to treasury: %w", fromToken, err)
		}
		fromTxHash = h
	}
	if err := s.tokenBalanceRepo.DeductBalance(ctx, acctUUID, fromT, fromAmount); err != nil {
		return 0, "", fmt.Errorf("failed to deduct %s balance: %w", fromToken, err)
	}

	// To-leg: MYRC is minted/sent on-chain first; the ledger is only credited
	// once that confirms.
	var toTxHash string
	if toT == models.MYRC {
		h, err := s.MintingMYRCBySUI(ctx, accountID, toAmount)
		if err != nil {
			s.refundSwapFromLeg(ctx, accountID, fromT, fromAmount)
			return 0, "", fmt.Errorf("failed to mint %s: %w", toToken, err)
		}
		toTxHash = h
	}
	if err := s.tokenBalanceRepo.AddBalance(ctx, acctUUID, toT, toAmount); err != nil {
		s.refundSwapFromLeg(ctx, accountID, fromT, fromAmount)
		return 0, "", fmt.Errorf("failed to credit %s balance: %w", toToken, err)
	}

	txHash := "ledger"
	if toTxHash != "" {
		txHash = toTxHash
	} else if fromTxHash != "" {
		txHash = fromTxHash
	}
	return toAmount, txHash, nil
}

// refundSwapFromLeg reverses a successful from-leg deduction when the to-leg
// fails: credits the ledger back, and if the from-leg moved real on-chain
// MYRC to treasury, sends it back to the user too.
func (s *tokenService) refundSwapFromLeg(ctx context.Context, accountID string, fromToken models.TokenType, fromAmount float64) {
	if fromToken == models.MYRC {
		if _, err := s.MintingMYRCBySUI(ctx, accountID, fromAmount); err != nil {
			log.Printf("CRITICAL: failed to refund on-chain MYRC for account %s after failed swap: %v", accountID, err)
		}
	}
	if err := s.tokenBalanceRepo.AddBalance(ctx, uuid.Must(uuid.Parse(accountID)), fromToken, fromAmount); err != nil {
		log.Printf("CRITICAL: failed to refund ledger %s for account %s after failed swap: %v", fromToken, accountID, err)
	}
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
