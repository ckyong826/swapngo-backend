package bizs

import (
	"context"
	"fmt"
	"log"
	"os"

	"swapngo-backend/internal/clients"
	"swapngo-backend/internal/fsm"
	"swapngo-backend/internal/kafka"
	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	"swapngo-backend/internal/services"
	"swapngo-backend/internal/ws"
	"swapngo-backend/pkg/database"
	requests "swapngo-backend/pkg/requests/deposit"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DepositBiz interface {
	InitiateDepositMYRC(ctx context.Context, req *requests.InitiateDepositReq, userID string) (any, error)
	HandlePaymentWebhook(ctx context.Context, gatewayRefID string, isPaid bool) error
	ProcessDepositEvent(ctx context.Context, depositID uuid.UUID, accountID string, amount float64) error
	ViewDeposit(ctx context.Context, userID string, id string) (*models.Deposit, error)
	ViewAllDeposits(ctx context.Context, userID string) ([]*models.Deposit, error)
}

type depositBiz struct {
	db             *gorm.DB
	depositRepo    repositories.DepositRepository
	tokenService   services.TokenService
	accountRepo    repositories.AccountRepository
	hub            *ws.Hub
	sm             *fsm.StateMachine
	paymentClient  clients.IPaymentClient
	depositService services.DepositService
}

func NewDepositBiz(db *gorm.DB, dr repositories.DepositRepository, ts services.TokenService, ar repositories.AccountRepository, hub *ws.Hub, sm *fsm.StateMachine, pc clients.IPaymentClient, ds services.DepositService) DepositBiz {
	return &depositBiz{
		db:             db,
		depositRepo:    dr,
		tokenService:   ts,
		accountRepo:    ar,
		hub:            hub,
		sm:             sm,
		paymentClient:  pc,
		depositService: ds,
	}
}

func (b *depositBiz) InitiateDepositMYRC(ctx context.Context, req *requests.InitiateDepositReq, userID string) (any, error) {
	// 1. Get user account
	accounts, err := b.accountRepo.FindByUserID(ctx, uuid.Must(uuid.Parse(userID)))
	if err != nil || len(accounts) == 0 {
		log.Printf("CRITICAL: Failed to fetch account for user %s", userID)
		return nil, err
	}

	// 2. Fetch basic profile info
	email := "user@example.com"
	name := "SwapNGo User"
	description := "Deposit to SwapNGo Wallet"

	callbackURL := os.Getenv("BILLPLZ_CALLBACK_URL")
	collectionID := os.Getenv("BILLPLZ_COLLECTION_ID")

	// 3. Create Bill via paymentClient
	billRes, err := b.paymentClient.CreateBill(ctx, email, name, req.Amount, description, callbackURL, collectionID)
	if err != nil {
		return nil, err
	}

	// 4. Create proper deposit using depositService
	accountID := uuid.Must(uuid.Parse(accounts[0].ID.String()))
	deposit, err := b.depositService.CreatePendingDeposit(ctx, accountID, req.Amount, req.Amount, billRes.ID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"deposit_id":  deposit.ID,
		"payment_url": billRes.URL,
	}, nil
}

func (b *depositBiz) HandlePaymentWebhook(ctx context.Context, gatewayRefID string, isPaid bool) error {
	var depositID, accountID string
	var amountMYRC float64
	var shouldTriggerWeb3 bool

	// 1. Transaction Block 
	err := database.RunInTx(b.db, ctx, func(txCtx context.Context) error {
		// 1.1. Lock Deposit
		deposit, err := b.depositRepo.LockByGatewayRef(txCtx, gatewayRefID)
		if err != nil {
			return err
		}

		// 1.2. Idempotency Check
		if deposit.Status != models.DepositStatePending {
			log.Printf("Idempotent Webhook: Deposit %s is already %s", deposit.ID, deposit.Status)
			return nil
		}

		// 1.3. Update Event
		event := fsm.DepositEventPaymentFailed
		if isPaid {
			event = fsm.DepositEventPaymentSuccess
		}
		
		nextState, err := b.sm.Fire(deposit.Status, event)
		if err != nil {
			return err
		}

		// 1.4. Update DB
		deposit.Status = nextState
		if _,err := b.depositRepo.Update(txCtx, deposit); err != nil {
			return err
		}

		// 1.5. Prepare data for async Web3 execution
		if nextState == models.DepositStateProcessingWeb3 {
			shouldTriggerWeb3 = true
			depositID = deposit.ID.String()
			accountID = deposit.AccountID.String()
			amountMYRC = deposit.AmountMYRC
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 2. Async Web3 Block (Out of DB Transaction)
	if shouldTriggerWeb3 {
		event := kafka.DepositWeb3Initiated{
			DepositID: uuid.Must(uuid.Parse(depositID)),
			AccountID: accountID,
			Amount:    amountMYRC,
		}
		if pubErr := kafka.PublishDepositWeb3InitiatedEvent(ctx, "deposit_events_topic", event); pubErr != nil {
			log.Printf("Failed to publish Web3 deposit event: %v", pubErr)
			return pubErr
		}
	}

	return nil
}

// Asynchronous minting and update DB
func (b *depositBiz) ProcessDepositEvent(ctx context.Context, depositID uuid.UUID, accountID string, amount float64) error {
	// 1. Delegate to TokenService (Pure Web3 logic)
	txHash, err := b.tokenService.MintingMYRCBySUI(ctx, accountID, amount)
	
	// 2. Fetch the deposit again
	deposit, dbErr := b.depositRepo.FindByID(ctx, depositID)
	if dbErr != nil {
		return fmt.Errorf("CRITICAL: Failed to fetch deposit %s after Web3 processing", depositID)
	}
	if deposit.Status == models.DepositStateSuccess || deposit.Status == models.DepositStateFailed {
		return nil // gracefully skip idempotency
	}

	// 3. Handle Web3 Result & Update DB
	if err != nil {
		log.Printf("Web3 processing failed for deposit %s: %v", depositID, err)
		deposit.Status, _ = b.sm.Fire(deposit.Status, fsm.DepositEventWeb3Failed)
		if _, updateErr := b.depositRepo.Update(ctx, deposit); updateErr != nil {
			return fmt.Errorf("failed to update deposit to failed: %w", updateErr)
		}
		return err
	}

	deposit.Status, _ = b.sm.Fire(deposit.Status, fsm.DepositEventWeb3Success)
	deposit.TxHash = txHash
	if _, updateErr := b.depositRepo.Update(ctx, deposit); updateErr != nil {
		return fmt.Errorf("failed to update deposit to success: %w", updateErr)
	}

	// 4. Notify Frontend (hub is keyed by user ID, not account ID)
	account, accErr := b.accountRepo.FindByID(ctx, uuid.Must(uuid.Parse(accountID)))
	if accErr == nil && account != nil {
		b.hub.SendToUser(account.UserID.String(), map[string]any{
			"type":    "DEPOSIT_SUCCESS",
			"amount":  amount,
			"tx_hash": txHash,
		})
	}
	return nil
}

func (b *depositBiz) ViewDeposit(ctx context.Context, userID string, id string) (*models.Deposit, error) {
	accounts, err := b.accountRepo.FindByUserID(ctx, uuid.Must(uuid.Parse(userID)))
	if err != nil || len(accounts) == 0 {
		return nil, fmt.Errorf("failed to fetch user account")
	}
	accountID := accounts[0].ID

	deposit, err := b.depositRepo.FirstBy(ctx, "id = ? AND account_id = ?", uuid.Must(uuid.Parse(id)), accountID)
	if err != nil {
		return nil, err
	}
	if deposit == nil {
		return nil, fmt.Errorf("deposit not found")
	}
	return deposit, nil
}

func (b *depositBiz) ViewAllDeposits(ctx context.Context, userID string) ([]*models.Deposit, error) {
	accounts, err := b.accountRepo.FindByUserID(ctx, uuid.Must(uuid.Parse(userID)))
	if err != nil || len(accounts) == 0 {
		return nil, fmt.Errorf("failed to fetch user account")
	}
	accountID := accounts[0].ID

	deposits, err := b.depositRepo.FindBy(ctx, "account_id = ?", accountID)
	if err != nil {
		return nil, err
	}
	return deposits, nil
}