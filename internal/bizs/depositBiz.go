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

type DepositBiz interface {
	HandlePaymentWebhook(ctx context.Context, gatewayRefID string, isPaid bool) error
}

type depositBiz struct {
	db           *gorm.DB
	depositRepo  repositories.DepositRepository
	tokenService services.TokenService
	hub          *ws.Hub
	sm           *fsm.StateMachine
}

func NewDepositBiz(db *gorm.DB, dr repositories.DepositRepository, ts services.TokenService, hub *ws.Hub, sm *fsm.StateMachine) DepositBiz {
	return &depositBiz{
		db:           db,
		depositRepo:  dr,
		tokenService: ts,
		hub:          hub,
		sm:           sm,
	}
}

func (b *depositBiz) HandlePaymentWebhook(ctx context.Context, gatewayRefID string, isPaid bool) error {
	var depositID, userID string
	var amountMYRC float64
	var shouldTriggerWeb3 bool

	// 1. Transaction Block 
	err := database.RunInTx(b.db, ctx, func(txCtx context.Context) error {
		// 1. Lock Deposit
		deposit, err := b.depositRepo.LockByGatewayRef(txCtx, gatewayRefID)
		if err != nil {
			return err
		}

		// 2. Idempotency Check
		if deposit.Status != models.DepositStatePending {
			log.Printf("Idempotent Webhook: Deposit %s is already %s", deposit.ID, deposit.Status)
			return nil
		}

		// 3. Fire State Machine
		event := fsm.DepositEventPaymentFailed
		if isPaid {
			event = fsm.DepositEventPaymentSuccess
		}
		
		nextState, err := b.sm.Fire(deposit.Status, event)
		if err != nil {
			return err
		}

		// 4. Update DB
		deposit.Status = nextState
		if _,err := b.depositRepo.Update(txCtx, deposit); err != nil {
			return err
		}

		// 5. Prepare data for async Web3 execution
		if nextState == models.DepositStateProcessingWeb3 {
			shouldTriggerWeb3 = true
			depositID = deposit.ID.String()
			userID = deposit.UserID.String()
			amountMYRC = deposit.AmountMYRC
		}

		return nil
	})

	if err != nil {
		return err
	}

	// 2. Async Web3 Block (Out of DB Transaction)
	if shouldTriggerWeb3 {
		go b.executeWeb3Workflow(depositID, userID, amountMYRC)
	}

	return nil
}

// executeWeb3Workflow orchestrates the blockchain interactions and subsequent DB updates
func (b *depositBiz) executeWeb3Workflow(depositID, userID string, amount float64) {
	// Create a background context since this outlives the HTTP request
	ctx := context.Background()

	// 1. Delegate to TokenService (Pure Web3 logic)
	txHash, err := b.tokenService.Minting(ctx, userID, amount)
	
	// 2. Fetch the deposit again (outside the webhook transaction)
	deposit, dbErr := b.depositRepo.FindByID(ctx, uuid.Must(uuid.Parse(depositID)))
	if dbErr != nil {
		log.Printf("CRITICAL: Failed to fetch deposit %s after Web3 processing", depositID)
		return
	}

	// 3. Handle Web3 Result & Update DB
	if err != nil {
		log.Printf("Web3 processing failed for deposit %s: %v", depositID, err)
		deposit.Status, _ = b.sm.Fire(deposit.Status, fsm.DepositEventWeb3Failed)
		_,err = b.depositRepo.Update(ctx, deposit)
		if err != nil {
			log.Printf("Failed to update deposit %s status to failed: %v", depositID, err)
		}
		return
	}

	deposit.Status, _ = b.sm.Fire(deposit.Status, fsm.DepositEventWeb3Success)
	deposit.TxHash = txHash
	_,err = b.depositRepo.Update(ctx, deposit)
	if err != nil {
		log.Printf("Failed to update deposit %s status to success: %v", depositID, err)
	}

	// 4. Notify Frontend
	b.hub.SendToUser(userID, map[string]any{
		"type":    "DEPOSIT_SUCCESS",
		"amount":  amount,
		"tx_hash": txHash,
	})
}