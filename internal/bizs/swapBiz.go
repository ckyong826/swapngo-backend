package bizs

import (
	"context"
	"fmt"
	"math"
	"time"

	"swapngo-backend/internal/fsm"
	"swapngo-backend/internal/kafka"
	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	"swapngo-backend/internal/services"
	"swapngo-backend/internal/ws"
	"swapngo-backend/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// quoteValidFor is how long a server-locked swap rate stays valid before
// settlement must fail and force the user to re-quote.
const quoteValidFor = 30 * time.Second

type SwapBiz interface {
	InitiateSwap(ctx context.Context, userID, fromToken, toToken string, fromAmount, estimatedAmount, slippage float64) (*models.Swap, error)
	ViewSwap(ctx context.Context, userID, id string) (*models.Swap, error)
	ViewAllSwaps(ctx context.Context, userID string) ([]*models.Swap, error)
	ProcessSwapEvent(ctx context.Context, orderID uuid.UUID, userAddress, fromToken, toToken, txDigest string, amountPaid, expectedAmount float64) error
}

type swapBiz struct {
	db               *gorm.DB
	swapRepo         repositories.SwapRepository
	accountRepo      repositories.AccountRepository
	userRepo         repositories.UserRepository
	tokenBalanceRepo repositories.TokenBalanceRepository
	tokenService     services.TokenService
	hub              *ws.Hub
	sm               *fsm.StateMachine
}

func NewSwapBiz(db *gorm.DB, sr repositories.SwapRepository, ar repositories.AccountRepository, ur repositories.UserRepository, tbr repositories.TokenBalanceRepository, ts services.TokenService, hub *ws.Hub, sm *fsm.StateMachine) SwapBiz {
	return &swapBiz{
		db:               db,
		swapRepo:         sr,
		accountRepo:      ar,
		userRepo:         ur,
		tokenBalanceRepo: tbr,
		tokenService:     ts,
		hub:              hub,
		sm:               sm,
	}
}

func (b *swapBiz) InitiateSwap(ctx context.Context, userID, fromToken, toToken string, fromAmount, estimatedAmount, slippage float64) (*models.Swap, error) {
	userUUID := uuid.Must(uuid.Parse(userID))

	user, err := b.userRepo.FindByID(ctx, userUUID)
	if err != nil || user == nil {
		return nil, fmt.Errorf("failed to fetch user")
	}
	if user.KycStatus != models.KycApproved {
		return nil, fmt.Errorf("KYC not approved")
	}

	if fromToken == toToken {
		return nil, fmt.Errorf("cannot swap a token into itself")
	}

	account, err := b.accountRepo.FindByUserID(ctx, userUUID)
	if err != nil || account == nil {
		return nil, fmt.Errorf("failed to fetch sender account")
	}

	balance, err := b.tokenBalanceRepo.GetBalance(ctx, account[0].ID, models.TokenType(fromToken))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch balance: %w", err)
	}
	if balance < fromAmount {
		return nil, fmt.Errorf("insufficient balance")
	}

	// Server is the sole authority on the exchange rate — the client's
	// estimated_amount is only used to detect a stale frontend price (below).
	quotedToAmount, err := b.tokenService.QuoteSwap(fromToken, toToken, fromAmount)
	if err != nil {
		return nil, fmt.Errorf("failed to quote swap: %w", err)
	}
	if estimatedAmount > 0 && math.Abs(quotedToAmount-estimatedAmount)/quotedToAmount > slippage {
		return nil, fmt.Errorf("price moved beyond slippage tolerance, please refresh quote")
	}

	swap := &models.Swap{
		AccountID:         account[0].ID,
		FromToken:         models.TokenType(fromToken),
		ToToken:           models.TokenType(toToken),
		FromAmount:        fromAmount,
		EstimatedToAmount: quotedToAmount,
		SlippageTolerance: slippage,
		QuoteExpiresAt:    time.Now().Add(quoteValidFor),
		Status:            models.SwapStatePending,
	}

	// 1. 开启数据库事务锁定并创建
	err = database.RunInTx(b.db, ctx, func(txCtx context.Context) error {
		if _, err := b.swapRepo.Create(txCtx, swap); err != nil {
			return err
		}

		swap.Status, _ = b.sm.Fire(swap.Status, fsm.SwapEventStart)
		if _,err := b.swapRepo.Update(txCtx, swap); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 2. 发布事件，链上与账本操作均在 worker 中异步完成
	event := kafka.SwapInitiated{
		OrderID:        swap.ID,
		FromToken:      fromToken,
		ToToken:        toToken,
		AmountPaid:     fromAmount,
		ExpectedAmount: quotedToAmount,
	}
	if pubErr := kafka.PublishSwapInitiatedEvent(ctx, "swap_events_topic", event); pubErr != nil {
		return nil, pubErr
	}
	return swap, nil
}

func (b *swapBiz) ViewSwap(ctx context.Context, userID string, id string) (*models.Swap, error) {
	accounts, err := b.accountRepo.FindByUserID(ctx, uuid.Must(uuid.Parse(userID)))
	if err != nil || len(accounts) == 0 {
		return nil, fmt.Errorf("failed to fetch user account")
	}
	accountID := accounts[0].ID

	swap, err := b.swapRepo.FirstBy(ctx, "id = ? AND account_id = ?", uuid.Must(uuid.Parse(id)), accountID)
	if err != nil {
		return nil, err
	}
	if swap == nil {
		return nil, fmt.Errorf("swap not found")
	}
	return swap, nil
}

func (b *swapBiz) ViewAllSwaps(ctx context.Context, userID string) ([]*models.Swap, error) {
	accounts, err := b.accountRepo.FindByUserID(ctx, uuid.Must(uuid.Parse(userID)))
	if err != nil || len(accounts) == 0 {
		return nil, fmt.Errorf("failed to fetch user account")
	}
	accountID := accounts[0].ID

	swaps, err := b.swapRepo.FindBy(ctx, "account_id = ?", accountID)
	if err != nil {
		return nil, err
	}
	return swaps, nil
}

func (b *swapBiz) ProcessSwapEvent(ctx context.Context, orderID uuid.UUID, userAddress, fromToken, toToken, txDigest string, amountPaid, expectedAmount float64) error {
	swap, err := b.swapRepo.FirstBy(ctx, "id = ?", orderID)
	if err != nil {
		return fmt.Errorf("db error fetching swap: %w", err)
	}
	if swap == nil {
		return fmt.Errorf("swap order not found in DB: %s", orderID)
	}

	if swap.Status == models.SwapStateSuccess || swap.Status == models.SwapStateFailed {
		return nil // gracefully skip, already handled
	}

	if time.Now().After(swap.QuoteExpiresAt) {
		swap.Status, _ = b.sm.Fire(swap.Status, fsm.SwapEventFailed)
		b.swapRepo.Update(ctx, swap)

		if account, accErr := b.accountRepo.FindByID(ctx, swap.AccountID); accErr == nil && account != nil {
			b.hub.SendToUser(account.UserID.String(), ws.Event("SWAP_FAILED", map[string]any{
				"swap_id":    swap.ID,
				"from_token": swap.FromToken,
				"to_token":   swap.ToToken,
				"reason":     "quote_expired",
			}))
		}
		return fmt.Errorf("swap %s: locked quote expired before settlement", orderID)
	}

	resolvedToAmount, txHash, err := b.tokenService.ExecuteSwapLegs(ctx, swap.AccountID.String(), fromToken, toToken, amountPaid, expectedAmount)
	if err != nil {
		swap.Status, _ = b.sm.Fire(swap.Status, fsm.SwapEventFailed)
		b.swapRepo.Update(ctx, swap)

		if account, accErr := b.accountRepo.FindByID(ctx, swap.AccountID); accErr == nil && account != nil {
			b.hub.SendToUser(account.UserID.String(), ws.Event("SWAP_FAILED", map[string]any{
				"swap_id":    swap.ID,
				"from_token": swap.FromToken,
				"to_token":   swap.ToToken,
				"reason":     "blockchain error",
			}))
		}
		return fmt.Errorf("failed to process swap legs: %w", err)
	}

	swap.Status, _ = b.sm.Fire(swap.Status, fsm.SwapEventSuccess)
	swap.TxHash = txHash
	swap.ActualToAmount = resolvedToAmount
	if _, err := b.swapRepo.Update(ctx, swap); err != nil {
		return fmt.Errorf("db error setting success: %w", err)
	}

	account, accErr := b.accountRepo.FindByID(ctx, swap.AccountID)
	if accErr == nil && account != nil {
		b.hub.SendToUser(account.UserID.String(), ws.Event("SWAP_COMPLETED", map[string]any{
			"swap_id":    swap.ID,
			"from_token": swap.FromToken,
			"to_token":   swap.ToToken,
			"to_amount":  swap.ActualToAmount,
			"tx_hash":    swap.TxHash,
		}))
	}
	return nil
}