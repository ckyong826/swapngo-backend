package bizs

import (
	"context"
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

type TransferBiz interface {
	InitiateTransfer(ctx context.Context, userID string, fromAddress string, toAddress string, amount float64) (*models.Transfer, error)
}

type transferBiz struct {
	db           *gorm.DB
	transferRepo repositories.TransferRepository
	walletRepo repositories.WalletRepository
	tokenService services.TokenService
	hub          *ws.Hub
	sm           *fsm.StateMachine
}

func NewTransferBiz(db *gorm.DB, tr repositories.TransferRepository, wr repositories.WalletRepository, ts services.TokenService, hub *ws.Hub, sm *fsm.StateMachine) TransferBiz {
	return &transferBiz{
		db:           db,
		transferRepo: tr,
		walletRepo: wr,
		tokenService: ts,
		hub:          hub,
		sm:           sm,
	}
}

func (b *transferBiz) InitiateTransfer(ctx context.Context, userID string, fromAddress string, toAddress string, amount float64) (*models.Transfer, error) {
	// 1. Check if wallet existed
	fromWallet, err := b.walletRepo.FindByAddress(ctx, fromAddress)
	if err != nil || fromWallet == nil {
		return nil, err
	}

	transfer := &models.Transfer{
		SenderAccountID:  fromWallet.AccountID,
		ToAddress: toAddress,
		Amount:    amount,
		Status:    models.TransferStatePending,
	}

	// 开启事务进行落库并流转状态
	err = database.RunInTx(b.db, ctx, func(txCtx context.Context) error {
		if _, err := b.transferRepo.Create(txCtx, transfer); err != nil {
			return err
		}

		transfer.Status, _ = b.sm.Fire(transfer.Status, fsm.TransferEventStart)
		if _,err := b.transferRepo.Update(txCtx, transfer); err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// 异步执行 Web3 链上转账
	go b.executeWeb3Transfer(transfer.ID.String(), userID, fromAddress, toAddress, amount)

	return transfer, nil
}

func (b *transferBiz) executeWeb3Transfer(transferID, senderID, fromAddress, toAddress string, amount float64) {
	ctx := context.Background()
	tUUID := uuid.Must(uuid.Parse(transferID))

	// 1. 纯 Web3 链上调用 (脱离数据库事务)
	txHash, err := b.tokenService.TransferToAddress(ctx, fromAddress, toAddress, amount)

	// 2. 使用悲观锁更新数据库状态
	_ = database.RunInTx(b.db, ctx, func(txCtx context.Context) error {
		t, dbErr := b.transferRepo.LockById(txCtx, tUUID)
		if dbErr != nil {
			log.Printf("CRITICAL: Failed to lock transfer %s: %v", transferID, dbErr)
			return dbErr
		}

		if err != nil {
			log.Printf("Transfer %s failed on chain: %v", transferID, err)
			t.Status, _ = b.sm.Fire(t.Status, fsm.TransferEventFailed)
			if _,err := b.transferRepo.Update(txCtx, t); err != nil {
				return err
			}
			return nil
		}

		t.Status, _ = b.sm.Fire(t.Status, fsm.TransferEventSuccess)
		t.TxHash = txHash
		if _,err := b.transferRepo.Update(txCtx, t); err != nil {
			return err
		}
		return nil
	})

	// 3. WebSocket 通知发送方
	if err == nil {
		b.hub.SendToUser(senderID, map[string]any{
			"type":    "TRANSFER_SUCCESS",
			"amount":  amount,
			"tx_hash": txHash,
		})
	} else {
		b.hub.SendToUser(senderID, map[string]any{
			"type":   "TRANSFER_FAILED",
			"reason": "blockchain error",
		})
	}
}