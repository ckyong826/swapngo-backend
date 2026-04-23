package bizs

import (
	"context"
	"fmt"

	"swapngo-backend/internal/fsm"
	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	"swapngo-backend/internal/services"
	"swapngo-backend/internal/ws"
	"swapngo-backend/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SwapBiz interface {
	InitiateSwap(ctx context.Context, userID, fromToken, toToken string, fromAmount, estimatedAmount, slippage float64) (*models.Swap, error)
	ViewSwap(ctx context.Context, userID, id string) (*models.Swap, error)
	ViewAllSwaps(ctx context.Context, userID string) ([]*models.Swap, error)
	ProcessSwapEvent(ctx context.Context, orderID uuid.UUID, userAddress, fromToken, toToken, txDigest string, amountPaid, expectedAmount float64) error
}

type swapBiz struct {
	db           *gorm.DB
	swapRepo     repositories.SwapRepository
	accountRepo  repositories.AccountRepository
	tokenService services.TokenService
	hub          *ws.Hub
	sm           *fsm.StateMachine
}

func NewSwapBiz(db *gorm.DB, sr repositories.SwapRepository, ar repositories.AccountRepository, ts services.TokenService, hub *ws.Hub, sm *fsm.StateMachine) SwapBiz {
	return &swapBiz{
		db:           db,
		swapRepo:     sr,
		accountRepo:  ar,
		tokenService: ts,
		hub:          hub,
		sm:           sm,
	}
}

func (b *swapBiz) InitiateSwap(ctx context.Context, userID, fromToken, toToken string, fromAmount, estimatedAmount, slippage float64) (*models.Swap, error) {
	userUUID := uuid.Must(uuid.Parse(userID))
	account, err := b.accountRepo.FindByUserID(ctx, userUUID)
	if err != nil || account == nil {
		return nil, fmt.Errorf("failed to fetch sender account")
	}

	swap := &models.Swap{
		AccountID:         account[0].ID,
		FromToken:         models.TokenType(fromToken),
		ToToken:           models.TokenType(toToken),
		FromAmount:        fromAmount,
		EstimatedToAmount: estimatedAmount,
		SlippageTolerance: slippage,
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

	// 2. 异步执行 Web3 链上智能合约调用
	err = b.tokenService.ExecuteSwap(ctx, swap.ID.String())
	if err != nil {
		return nil, err
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

	payoutTx, err := b.tokenService.ExecuteSwapPayout(ctx, userAddress, fromToken, toToken, txDigest, amountPaid, expectedAmount)
	if err != nil {
		swap.Status, _ = b.sm.Fire(swap.Status, fsm.SwapEventFailed)
		b.swapRepo.Update(ctx, swap)
		return fmt.Errorf("failed to process payout: %w", err)
	}

	swap.Status, _ = b.sm.Fire(swap.Status, fsm.SwapEventSuccess)
	swap.TxHash = payoutTx
	swap.ActualToAmount = expectedAmount
	if _, err := b.swapRepo.Update(ctx, swap); err != nil {
		return fmt.Errorf("db error setting success: %w", err)
	}

	b.hub.SendToUser(swap.AccountID.String(), map[string]any{"type": "SWAP_COMPLETED", "swap_id": swap.ID})
	return nil
}