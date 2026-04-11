package bizs

import (
	"context"
	"log"

	"swapngo-backend/internal/clients"
	"swapngo-backend/internal/fsm"
	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	"swapngo-backend/internal/services"
	"swapngo-backend/internal/ws"
	"swapngo-backend/pkg/database"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WithdrawBiz interface {
	InitiateWithdrawal(ctx context.Context, userID string, amountMYRC float64, bankName, bankAccountNo string) (*models.Withdrawal, error)
}

type withdrawBiz struct {
	db             *gorm.DB
	withdrawRepo   repositories.WithdrawRepository
	accountRepo    repositories.AccountRepository
	tokenService   services.TokenService
	paymentClient  clients.IPaymentClient
	hub            *ws.Hub
	sm             *fsm.StateMachine
}

func NewWithdrawBiz(db *gorm.DB, wr repositories.WithdrawRepository, ar repositories.AccountRepository, ts services.TokenService, pc clients.IPaymentClient, hub *ws.Hub, sm *fsm.StateMachine) WithdrawBiz {
	return &withdrawBiz{
		db:             db,
		withdrawRepo:   wr,
		accountRepo:    ar,
		tokenService:   ts,
		paymentClient:  pc,
		hub:            hub,
		sm:             sm,
	}
}

// InitiateWithdrawal 
func (b *withdrawBiz) InitiateWithdrawal(ctx context.Context, userID string, amountMYRC float64, bankName, bankAccountNo string) (*models.Withdrawal, error) {
	accounts, err := b.accountRepo.FindByUserID(ctx, uuid.Must(uuid.Parse(userID)))
	if err != nil || len(accounts) == 0 {
		log.Printf("CRITICAL: Failed to fetch account for user %s", userID)
		return nil, err
	}

	// 假设 1 MYRC = 1 MYR
	amountMYR := amountMYRC

	withdraw := &models.Withdrawal{
		AccountID:     accounts[0].ID,
		AmountMYRC:    amountMYRC,
		AmountMYR:     amountMYR,
		BankName:      bankName,
		BankAccountNo: bankAccountNo,
		Status:        models.WithdrawStatePending,
	}

	// 事务块：创建订单并立即流转到 PROCESSING_WEB3
	err = database.RunInTx(b.db, ctx, func(txCtx context.Context) error {
		if _, err := b.withdrawRepo.Create(txCtx, withdraw); err != nil {
			return err
		}
		
		withdraw.Status, _ = b.sm.Fire(withdraw.Status, fsm.WithdrawEventStartWeb3)
		if _, err := b.withdrawRepo.Update(txCtx, withdraw); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Asynchronously execute the withdrawal workflow
	go b.executeWithdrawWorkflow(withdraw.ID.String(), userID, amountMYRC, amountMYR, bankName, bankAccountNo)

	return withdraw, nil
}

// 2. 异步处理链：Web3 扣款 + Web2 法币打款
func (b *withdrawBiz) executeWithdrawWorkflow(withdrawID, userID string, amountMYRC, amountMYR float64, bankName, bankAccountNo string) {
	ctx := context.Background()
	wUUID := uuid.Must(uuid.Parse(withdrawID))

	// ==========================================
	// 步骤 A: 扣除 Web3 资产 (转移到平台国库 Treasury)
	// ==========================================
	txHash, err := b.tokenService.TransferToTreasury(ctx, userID, amountMYRC)
	
	// 获取锁更新数据库
	var shouldProceedToFiat bool
	_ = database.RunInTx(b.db, ctx, func(txCtx context.Context) error {
		w, _ := b.withdrawRepo.LockById(txCtx, wUUID)
		
		if err != nil {
			log.Printf("Web3 Deduction failed for withdraw %s: %v", withdrawID, err)
			w.Status, _ = b.sm.Fire(w.Status, fsm.WithdrawEventWeb3Failed)
			if _, err := b.withdrawRepo.Update(txCtx, w); err != nil {
				return err
			}
			return err
		}

		w.Status, _ = b.sm.Fire(w.Status, fsm.WithdrawEventWeb3Success)
		w.TxHash = txHash
		if _, err := b.withdrawRepo.Update(txCtx, w); err != nil {
			return err
		}
		
		shouldProceedToFiat = true
		return nil
	})

	if !shouldProceedToFiat {
		b.hub.SendToUser(userID, map[string]any{"type": "WITHDRAW_FAILED", "reason": "blockchain error"})
		return
	}

	// ==========================================
	// 步骤 B: Web2 法币打款 (请求网关)
	// ==========================================
	// 调用网关 Payout API 打钱给用户银行卡
	fiatRefID, payoutErr := b.paymentClient.PayoutToBank(ctx, amountMYR, bankName, bankAccountNo)

	_ = database.RunInTx(b.db, ctx, func(txCtx context.Context) error {
		w, _ := b.withdrawRepo.LockById(txCtx, wUUID)

		if payoutErr != nil {
			log.Printf("CRITICAL: Fiat payout failed for withdraw %s: %v", withdrawID, payoutErr)
			w.Status, _ = b.sm.Fire(w.Status, fsm.WithdrawEventFiatFailed)
			if _, err := b.withdrawRepo.Update(txCtx, w); err != nil {
				return err
			}
			return err
		}

		w.Status, _ = b.sm.Fire(w.Status, fsm.WithdrawEventFiatSuccess)
		w.GatewayRefID = fiatRefID
		if _, err := b.withdrawRepo.Update(txCtx, w); err != nil {
			return err
		}
		return nil
	})

	// 推送最终结果给前端
	if payoutErr == nil {
		b.hub.SendToUser(userID, map[string]any{
			"type":    "WITHDRAW_SUCCESS",
			"amount":  amountMYR,
			"tx_hash": txHash,
		})
	} else {
		b.hub.SendToUser(userID, map[string]any{"type": "WITHDRAW_FAILED", "reason": "bank payout error"})
	}
}