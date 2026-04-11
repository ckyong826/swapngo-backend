package bizs

import (
	"context"
	"log"
	"os"

	"swapngo-backend/internal/clients"
	"swapngo-backend/internal/fsm"
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
		go b.executeWeb3Workflow(depositID, accountID, amountMYRC)
	}

	return nil
}

// Asynchronous minting and update DB
func (b *depositBiz) executeWeb3Workflow(depositID, accountID string, amount float64) {
	// Create a background context since this outlives the HTTP request
	ctx := context.Background()

	// 1. Delegate to TokenService (Pure Web3 logic)
	txHash, err := b.tokenService.MintingMYRCBySUI(ctx, accountID, amount)
	
	// 2. Fetch the deposit again (outside the webhook transaction)
	deposit, dbErr := b.depositRepo.FindByID(ctx, uuid.Must(uuid.Parse(depositID)))
	if dbErr != nil {
		log.Printf("CRITICAL: Failed to fetch deposit %s after Web3 processing", depositID)
		return
	}

	// 3. Handle Web3 Result & Update DB
	// 3.1. If Web3 processing failed
	if err != nil {
		log.Printf("Web3 processing failed for deposit %s: %v", depositID, err)
		deposit.Status, _ = b.sm.Fire(deposit.Status, fsm.DepositEventWeb3Failed)
		_,err = b.depositRepo.Update(ctx, deposit)
		if err != nil {
			log.Printf("Failed to update deposit %s status to failed: %v", depositID, err)
		}
		return
	}

	// 3.2. If Web3 processing success
	deposit.Status, _ = b.sm.Fire(deposit.Status, fsm.DepositEventWeb3Success)
	deposit.TxHash = txHash
	_,err = b.depositRepo.Update(ctx, deposit)
	if err != nil {
		log.Printf("Failed to update deposit %s status to success: %v", depositID, err)
	}

	// 4. Notify Frontend
	user, err := b.accountRepo.FindByID(ctx, uuid.Must(uuid.Parse(accountID)))
	if err != nil {
		log.Printf("CRITICAL: Failed to fetch account %s after Web3 processing", accountID)
		return
	}
	b.hub.SendToUser(user.ID.String(), map[string]any{
		"type":    "DEPOSIT_SUCCESS",
		"amount":  amount,
		"tx_hash": txHash,
	})
}