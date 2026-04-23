package bizs

import (
	"context"
	"errors"
	"fmt"
	"log"

	"swapngo-backend/internal/clients"
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

type WithdrawBiz interface {
	InitiateWithdrawal(ctx context.Context, userID string, amountMYRC float64, bankName, bankAccountNo string) (*models.Withdrawal, error)
	ProcessWithdrawEvent(ctx context.Context, withdrawID uuid.UUID, userID string, amountMYRC, amountMYR float64, bankName, bankAccountNo string) error
	ViewWithdraw(ctx context.Context, userID, id string) (*models.Withdrawal, error)
	ViewAllWithdraws(ctx context.Context, userID string) ([]*models.Withdrawal, error)
}

type withdrawBiz struct {
	db             *gorm.DB
	withdrawRepo   repositories.WithdrawRepository
	accountRepo    repositories.AccountRepository
	tokenService   services.TokenService
	walletService  services.WalletService
	paymentClient  clients.IPaymentClient
	hub            *ws.Hub
	sm             *fsm.StateMachine
}

func NewWithdrawBiz(db *gorm.DB, wr repositories.WithdrawRepository, ar repositories.AccountRepository, ts services.TokenService, ws services.WalletService, pc clients.IPaymentClient, hub *ws.Hub, sm *fsm.StateMachine) WithdrawBiz {
	return &withdrawBiz{
		db:             db,
		withdrawRepo:   wr,
		accountRepo:    ar,
		tokenService:   ts,
		walletService:  ws,
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

	// Check if the account has enough balance
	hasEnoughBalance, err := b.walletService.CheckBalanceByUserIDAndChain(ctx, userID, models.ChainSui, amountMYRC)
	if err != nil {
		log.Printf("CRITICAL: Failed to get balance for user %s", userID)
		return nil, err
	}
	if !hasEnoughBalance {
		log.Printf("CRITICAL: Insufficient balance for user %s", userID)
		return nil, errors.New("insufficient balance")
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
	event := kafka.WithdrawInitiated{
		WithdrawID:    withdraw.ID,
		UserID:        userID,
		AmountMYRC:    amountMYRC,
		AmountMYR:     amountMYR,
		BankName:      bankName,
		BankAccountNo: bankAccountNo,
	}
	if pubErr := kafka.PublishWithdrawInitiatedEvent(ctx, "withdraw_events_topic", event); pubErr != nil {
		log.Printf("Failed to publish withdraw event: %v", pubErr)
		return nil, pubErr
	}

	return withdraw, nil
}

// 2. 异步处理链：Web3 扣款 + Web2 法币打款
func (b *withdrawBiz) ProcessWithdrawEvent(ctx context.Context, wUUID uuid.UUID, userID string, amountMYRC, amountMYR float64, bankName, bankAccountNo string) error {
	// Check if withdraw oder existed
	withdrawal, err := b.withdrawRepo.FindByID(ctx, wUUID)
	if err != nil || withdrawal == nil {
		return fmt.Errorf("Withdrawal not found %s: %w", wUUID, err)
	}

	if withdrawal.Status == models.WithdrawStateSuccess || withdrawal.Status == models.WithdrawStateFailed {
		return nil // gracefully skip idempotency
	}

	// ==========================================
	// 步骤 A: 扣除 Web3 资产 (转移到平台国库 Treasury)
	// ==========================================
	txHash, err := b.tokenService.TransferToTreasury(ctx, withdrawal.AccountID.String(), amountMYRC)
	
	// 获取锁更新数据库
	var shouldProceedToFiat bool
	_ = database.RunInTx(b.db, ctx, func(txCtx context.Context) error {
		w, _ := b.withdrawRepo.LockById(txCtx, wUUID)
		
		if err != nil {
			log.Printf("Web3 Deduction failed for withdraw %s: %v", wUUID, err)
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
		return fmt.Errorf("web3 process failed: %w", err)
	}

	// ==========================================
	// 步骤 B: Web2 法币打款 (请求网关)
	// ==========================================
	// 调用网关 Payout API 打钱给用户银行卡
	fiatRefID, payoutErr := b.paymentClient.PayoutToBank(ctx, amountMYR, bankName, bankAccountNo)

	_ = database.RunInTx(b.db, ctx, func(txCtx context.Context) error {
		w, _ := b.withdrawRepo.LockById(txCtx, wUUID)

		if payoutErr != nil {
			log.Printf("CRITICAL: Fiat payout failed for withdraw %s: %v", wUUID, payoutErr)
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
		return nil
	} else {
		b.hub.SendToUser(userID, map[string]any{"type": "WITHDRAW_FAILED", "reason": "bank payout error"})
		return fmt.Errorf("fiat payout failed: %w", payoutErr)
	}
}

func (b *withdrawBiz) ViewWithdraw(ctx context.Context, userID string, id string) (*models.Withdrawal, error) {
	accounts, err := b.accountRepo.FindByUserID(ctx, uuid.Must(uuid.Parse(userID)))
	if err != nil || len(accounts) == 0 {
		return nil, fmt.Errorf("failed to fetch user account")
	}
	accountID := accounts[0].ID

	withdraw, err := b.withdrawRepo.FirstBy(ctx, "id = ? AND account_id = ?", uuid.Must(uuid.Parse(id)), accountID)
	if err != nil {
		return nil, err
	}
	if withdraw == nil {
		return nil, fmt.Errorf("withdraw not found")
	}
	return withdraw, nil
}

func (b *withdrawBiz) ViewAllWithdraws(ctx context.Context, userID string) ([]*models.Withdrawal, error) {
	accounts, err := b.accountRepo.FindByUserID(ctx, uuid.Must(uuid.Parse(userID)))
	if err != nil || len(accounts) == 0 {
		return nil, fmt.Errorf("failed to fetch user account")
	}
	accountID := accounts[0].ID

	withdraws, err := b.withdrawRepo.FindBy(ctx, "account_id = ?", accountID)
	if err != nil {
		return nil, err
	}
	return withdraws, nil
}