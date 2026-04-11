package bizs

import (
	"context"
	"fmt"
	"log"

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
		FromToken:         fromToken,
		ToToken:           toToken,
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
	go b.executeWeb3Swap(swap.ID.String(), userID, fromToken, toToken, fromAmount, slippage)

	return swap, nil
}

func (b *swapBiz) executeWeb3Swap(swapID, userID, fromToken, toToken string, fromAmount, slippage float64) {
	ctx := context.Background()
	sUUID := uuid.Must(uuid.Parse(swapID))

	// 调用智能合约执行兑换
	txHash, actualAmount, err := b.tokenService.ExecuteSwap(ctx, userID, fromToken, toToken, fromAmount, slippage)

	// 使用悲观锁更新最终结果
	_ = database.RunInTx(b.db, ctx, func(txCtx context.Context) error {
		s, dbErr := b.swapRepo.LockById(txCtx, sUUID)
		if dbErr != nil {
			log.Printf("CRITICAL: Failed to lock swap %s: %v", swapID, dbErr)
			return dbErr
		}

		if err != nil {
			log.Printf("Swap %s failed on chain: %v", swapID, err)
			s.Status, _ = b.sm.Fire(s.Status, fsm.SwapEventFailed)
			if _,err := b.swapRepo.Update(txCtx, s); err != nil {
				return err
			}
			return nil
		}

		s.Status, _ = b.sm.Fire(s.Status, fsm.SwapEventSuccess)
		s.TxHash = txHash
		s.ActualToAmount = actualAmount
		if _,err := b.swapRepo.Update(txCtx, s); err != nil {
			return err
		}
		return nil
	})

	// 推送前端结果
	if err == nil {
		b.hub.SendToUser(userID, map[string]any{
			"type":          "SWAP_SUCCESS",
			"actual_amount": actualAmount,
			"tx_hash":       txHash,
		})
	} else {
		b.hub.SendToUser(userID, map[string]any{
			"type":   "SWAP_FAILED",
			"reason": "Slippage exceeded or network error",
		})
	}
}