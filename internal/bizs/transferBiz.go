package bizs

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"strings"

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

type ResolvedRecipient struct {
	Username   string `json:"username"`
	SuiAddress string `json:"sui_address"`
}

type TransferBiz interface {
	ResolveRecipient(ctx context.Context, userID, query string) (*ResolvedRecipient, error)
	InitiateTransfer(ctx context.Context, userID, toAddress string, amount float64, pin string) (*models.Transfer, error)
	ProcessTransferEvent(ctx context.Context, transferID uuid.UUID, senderID, fromAddress, toAddress string, amount float64) error
	ViewTransfer(ctx context.Context, userID, id string) (*models.Transfer, error)
	ViewAllTransfers(ctx context.Context, userID string) ([]*models.Transfer, error)
}

type transferBiz struct {
	db           *gorm.DB
	transferRepo repositories.TransferRepository
	walletRepo repositories.WalletRepository
	accountRepo    repositories.AccountRepository
	userRepo     repositories.UserRepository
	tokenService services.TokenService
	hub          *ws.Hub
	sm           *fsm.StateMachine
}

func NewTransferBiz(db *gorm.DB, tr repositories.TransferRepository, wr repositories.WalletRepository,ar repositories.AccountRepository, ur repositories.UserRepository, ts services.TokenService, hub *ws.Hub, sm *fsm.StateMachine) TransferBiz {
	return &transferBiz{
		db:           db,
		transferRepo: tr,
		walletRepo: wr,
		accountRepo: ar,
		userRepo:     ur,
		tokenService: ts,
		hub:          hub,
		sm:           sm,
	}
}

// resolveToAddress turns a recipient input (raw 0x address, username, or phone)
// into a Sui address. This is the single authoritative resolution path — never
// trust a client-supplied address that claims to map to a username.
// callerAddr blocks self-sends.
func (b *transferBiz) resolveToAddress(ctx context.Context, callerAddr, input string) (username, address string, err error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", fmt.Errorf("recipient is required")
	}

	if strings.HasPrefix(input, "0x") {
		if len(input) != 66 {
			return "", "", fmt.Errorf("invalid SUI address")
		}
		if _, hexErr := hex.DecodeString(input[2:]); hexErr != nil {
			return "", "", fmt.Errorf("invalid SUI address")
		}
		if strings.EqualFold(input, callerAddr) {
			return "", "", fmt.Errorf("cannot send to yourself")
		}
		return "", input, nil
	}

	// username or phone — let the DB match either. Empty email never matches a real user.
	u, err := b.userRepo.CheckExist(ctx, input, "", input)
	if err != nil || u == nil {
		return "", "", fmt.Errorf("recipient not found")
	}

	accounts, err := b.accountRepo.FindByUserID(ctx, u.ID)
	if err != nil || len(accounts) == 0 {
		return "", "", fmt.Errorf("recipient cannot receive yet")
	}
	wallet, err := b.walletRepo.FindByAccountIdAndChain(ctx, accounts[0].ID, string(models.ChainSui))
	if err != nil || wallet == nil || wallet.Address == "" {
		return "", "", fmt.Errorf("recipient cannot receive yet")
	}
	if strings.EqualFold(wallet.Address, callerAddr) {
		return "", "", fmt.Errorf("cannot send to yourself")
	}
	return u.Username, wallet.Address, nil
}

func (b *transferBiz) ResolveRecipient(ctx context.Context, userID, query string) (*ResolvedRecipient, error) {
	accounts, err := b.accountRepo.FindByUserID(ctx, uuid.Must(uuid.Parse(userID)))
	if err != nil || len(accounts) == 0 {
		return nil, fmt.Errorf("failed to fetch account")
	}
	fromWallet, err := b.walletRepo.FindByAccountIdAndChain(ctx, accounts[0].ID, string(models.ChainSui))
	if err != nil {
		return nil, fmt.Errorf("failed to fetch wallet")
	}

	username, address, err := b.resolveToAddress(ctx, fromWallet.Address, query)
	if err != nil {
		return nil, err
	}
	return &ResolvedRecipient{Username: username, SuiAddress: address}, nil
}

func (b *transferBiz) InitiateTransfer(ctx context.Context, userID string, toAddress string, amount float64, pin string) (*models.Transfer, error) {
	user, err := b.userRepo.FindByID(ctx, uuid.Must(uuid.Parse(userID)))
	if err != nil || user == nil {
		return nil, fmt.Errorf("failed to fetch user")
	}
	if user.KycStatus != models.KycApproved {
		return nil, fmt.Errorf("KYC not approved")
	}
	if err := verifyPin(user, pin); err != nil {
		return nil, err
	}

	// 1. Fetch sender account and wallet
	accounts, err := b.accountRepo.FindByUserID(ctx, uuid.Must(uuid.Parse(userID)))
	if err != nil || len(accounts) == 0 {
		log.Printf("CRITICAL: Failed to fetch account for user %s", userID)
		return nil, err
	}

	fromWallet, err := b.walletRepo.FindByAccountIdAndChain(ctx, accounts[0].ID, string(models.ChainSui))
	if err != nil {
		log.Printf("CRITICAL: Failed to fetch wallet for user %s", userID)
		return nil, err
	}

	// Authoritatively resolve recipient (raw address / username / phone) server-side.
	_, toAddress, err = b.resolveToAddress(ctx, fromWallet.Address, toAddress)
	if err != nil {
		return nil, err
	}

	transfer := &models.Transfer{
		SenderAccountID: accounts[0].ID,
		ToAddress:       toAddress,
		Amount:          amount,
		Status:          models.TransferStatePending,
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

	// Publish Event
	event := kafka.TransferInitiated{
		TransferID:  transfer.ID,
		SenderID:    userID,
		FromAddress: fromWallet.Address,
		ToAddress:   toAddress,
		Amount:      amount,
	}
	if pubErr := kafka.PublishTransferInitiatedEvent(ctx, "transfer_events_topic", event); pubErr != nil {
		log.Printf("Failed to publish transfer event: %v", pubErr)
		return nil, pubErr
	}

	return transfer, nil
}

func (b *transferBiz) ProcessTransferEvent(ctx context.Context, tUUID uuid.UUID, senderID, fromAddress, toAddress string, amount float64) error {
	transfer, dbErr := b.transferRepo.FindByID(ctx, tUUID)
	if dbErr != nil {
		return fmt.Errorf("CRITICAL: Failed to lock transfer %s: %v", tUUID, dbErr)
	}
	if transfer.Status == models.TransferStateSuccess || transfer.Status == models.TransferStateFailed {
		return nil
	}

	// 1. 纯 Web3 链上调用 (脱离数据库事务)
	txHash, err := b.tokenService.TransferToAddress(ctx, fromAddress, toAddress, amount)

	// 2. 使用悲观锁更新数据库状态
	_ = database.RunInTx(b.db, ctx, func(txCtx context.Context) error {
		t, dbErr := b.transferRepo.LockById(txCtx, tUUID)
		if dbErr != nil {
			log.Printf("CRITICAL: Failed to lock transfer %s: %v", tUUID, dbErr)
			return dbErr
		}

		if err != nil {
			log.Printf("Transfer %s failed on chain: %v", tUUID, err)
			t.Status, _ = b.sm.Fire(t.Status, fsm.TransferEventFailed)
			if _,err := b.transferRepo.Update(txCtx, t); err != nil {
				return err
			}
			return err
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
		b.hub.SendToUser(senderID, ws.Event("TRANSFER_SUCCESS", map[string]any{
			"transfer_id": tUUID,
			"amount":      amount,
			"tx_hash":     txHash,
		}))
		return nil
	} else {
		b.hub.SendToUser(senderID, ws.Event("TRANSFER_FAILED", map[string]any{
			"transfer_id": tUUID,
			"amount":      amount,
			"reason":      "blockchain error",
		}))
		return err
	}
}

func (b *transferBiz) ViewTransfer(ctx context.Context, userID string, id string) (*models.Transfer, error) {
	accounts, err := b.accountRepo.FindByUserID(ctx, uuid.Must(uuid.Parse(userID)))
	if err != nil || len(accounts) == 0 {
		return nil, fmt.Errorf("failed to fetch user account")
	}
	accountID := accounts[0].ID

	transfer, err := b.transferRepo.FirstBy(ctx, "id = ? AND (sender_account_id = ? OR receiver_account_id = ?)", uuid.Must(uuid.Parse(id)), accountID, accountID)
	if err != nil {
		return nil, err
	}
	if transfer == nil {
		return nil, fmt.Errorf("transfer not found")
	}
	return transfer, nil
}

func (b *transferBiz) ViewAllTransfers(ctx context.Context, userID string) ([]*models.Transfer, error) {
	accounts, err := b.accountRepo.FindByUserID(ctx, uuid.Must(uuid.Parse(userID)))
	if err != nil || len(accounts) == 0 {
		return nil, fmt.Errorf("failed to fetch user account")
	}
	accountID := accounts[0].ID

	transfers, err := b.transferRepo.FindBy(ctx, "sender_account_id = ? OR receiver_account_id = ?", accountID, accountID)
	if err != nil {
		return nil, err
	}
	return transfers, nil
}