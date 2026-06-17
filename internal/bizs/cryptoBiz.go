package bizs

import (
	"context"
	"fmt"

	"swapngo-backend/internal/clients/chains"
	"swapngo-backend/internal/models"
	"swapngo-backend/internal/repositories"
	"swapngo-backend/internal/services"
	config "swapngo-backend/pkg/configs"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CryptoBiz handles crypto deposit/withdraw rails on top of the ledger.
// SUI is a real testnet chain (verify-by-digest deposit, real on-chain withdraw);
// BTC/ETH/USDT/USDC are simulated (ledger-only movements).
type CryptoBiz interface {
	DepositSUI(ctx context.Context, userID, txDigest string, amount float64) (*models.CryptoTxn, error)
	SimulateDeposit(ctx context.Context, userID, token string, amount float64) (*models.CryptoTxn, error)
	Withdraw(ctx context.Context, userID, token string, amount float64, toAddress string) (*models.CryptoTxn, error)
	ViewAll(ctx context.Context, userID string) ([]*models.CryptoTxn, error)
}

type cryptoBiz struct {
	db           *gorm.DB
	cryptoRepo   repositories.CryptoTxnRepository
	accountRepo  repositories.AccountRepository
	tokenService services.TokenService
	suiClient    chains.IChainClient
}

func NewCryptoBiz(db *gorm.DB, cr repositories.CryptoTxnRepository, ar repositories.AccountRepository, ts services.TokenService, sc chains.IChainClient) CryptoBiz {
	return &cryptoBiz{
		db:           db,
		cryptoRepo:   cr,
		accountRepo:  ar,
		tokenService: ts,
		suiClient:    sc,
	}
}

// simulatedTokens are the tokens whose chain is mocked.
var simulatedTokens = map[models.TokenType]bool{
	models.BTC:  true,
	models.ETH:  true,
	models.USDT: true,
	models.USDC: true,
}

func (b *cryptoBiz) resolveAccountID(ctx context.Context, userID string) (uuid.UUID, error) {
	accounts, err := b.accountRepo.FindByUserID(ctx, uuid.Must(uuid.Parse(userID)))
	if err != nil || len(accounts) == 0 {
		return uuid.Nil, fmt.Errorf("failed to fetch user account")
	}
	return accounts[0].ID, nil
}

// DepositSUI verifies a real on-chain SUI transfer to the treasury address and,
// if valid and not already claimed, credits the user's SUI ledger balance.
func (b *cryptoBiz) DepositSUI(ctx context.Context, userID, txDigest string, amount float64) (*models.CryptoTxn, error) {
	accountID, err := b.resolveAccountID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Replay guard: a digest can only be credited once.
	existing, err := b.cryptoRepo.FirstBy(ctx, "tx_ref = ? AND direction = ?", txDigest, models.CryptoDirectionDeposit)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("deposit already credited for tx digest %s", txDigest)
	}

	treasury := config.Env.SUITreasuryAddress
	if treasury == "" {
		return nil, fmt.Errorf("CRITICAL: SUI treasury address missing in config")
	}

	// Verify the on-chain transfer landed at the treasury for at least `amount`.
	ok, err := b.suiClient.VerifyTransfer(ctx, txDigest, treasury, amount)
	if err != nil {
		return nil, fmt.Errorf("on-chain SUI deposit verification failed: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("on-chain SUI deposit verification rejected")
	}

	if err := b.tokenService.CreditToken(ctx, accountID.String(), models.SUI, amount); err != nil {
		return nil, fmt.Errorf("failed to credit SUI ledger balance: %w", err)
	}

	txn := &models.CryptoTxn{
		AccountID: accountID,
		Token:     models.SUI,
		Direction: models.CryptoDirectionDeposit,
		Amount:    amount,
		TxRef:     txDigest,
		Simulated: false,
		Status:    "SUCCESS",
	}
	if _, err := b.cryptoRepo.Create(ctx, txn); err != nil {
		return nil, err
	}
	return txn, nil
}

// SimulateDeposit credits a mocked-chain token (BTC/ETH/USDT/USDC) to the ledger.
func (b *cryptoBiz) SimulateDeposit(ctx context.Context, userID, token string, amount float64) (*models.CryptoTxn, error) {
	accountID, err := b.resolveAccountID(ctx, userID)
	if err != nil {
		return nil, err
	}

	t := models.TokenType(token)
	if !simulatedTokens[t] {
		return nil, fmt.Errorf("simulated deposit not supported for token %q (use the SUI deposit or fiat deposit endpoints)", token)
	}

	if err := b.tokenService.CreditToken(ctx, accountID.String(), t, amount); err != nil {
		return nil, fmt.Errorf("failed to credit %s ledger balance: %w", t, err)
	}

	txn := &models.CryptoTxn{
		AccountID: accountID,
		Token:     t,
		Direction: models.CryptoDirectionDeposit,
		Amount:    amount,
		TxRef:     fmt.Sprintf("sim-deposit-%s", uuid.NewString()),
		Simulated: true,
		Status:    "SUCCESS",
	}
	if _, err := b.cryptoRepo.Create(ctx, txn); err != nil {
		return nil, err
	}
	return txn, nil
}

// Withdraw debits the ledger and moves crypto out. SUI is sent on-chain for real;
// the simulated tokens just debit the ledger and record a synthetic txid.
func (b *cryptoBiz) Withdraw(ctx context.Context, userID, token string, amount float64, toAddress string) (*models.CryptoTxn, error) {
	accountID, err := b.resolveAccountID(ctx, userID)
	if err != nil {
		return nil, err
	}

	t := models.TokenType(token)
	if t == models.MYRC {
		return nil, fmt.Errorf("MYRC is withdrawn to a bank via the fiat withdraw endpoint, not here")
	}
	if t != models.SUI && !simulatedTokens[t] {
		return nil, fmt.Errorf("unsupported withdraw token %q", token)
	}

	// Debit the ledger first (source of truth). Fails if balance is insufficient.
	if err := b.tokenService.DebitToken(ctx, accountID.String(), t, amount); err != nil {
		return nil, fmt.Errorf("failed to debit %s ledger balance: %w", t, err)
	}

	var txRef string
	simulated := true

	if t == models.SUI {
		if toAddress == "" {
			// Refund: nothing left the chain, undo the debit.
			_ = b.tokenService.CreditToken(ctx, accountID.String(), t, amount)
			return nil, fmt.Errorf("to_address is required for SUI withdrawal")
		}
		// Real on-chain send from the treasury hot wallet to the user's address.
		digest, err := b.suiClient.TransferCoin(ctx, config.Env.SUITreasuryPriv, config.Env.SUITreasuryAddress, toAddress, amount)
		if err != nil {
			// Refund the ledger debit if the on-chain send failed.
			_ = b.tokenService.CreditToken(ctx, accountID.String(), t, amount)
			return nil, fmt.Errorf("on-chain SUI withdrawal failed: %w", err)
		}
		txRef = digest
		simulated = false
	} else {
		txRef = fmt.Sprintf("sim-withdraw-%s", uuid.NewString())
	}

	txn := &models.CryptoTxn{
		AccountID: accountID,
		Token:     t,
		Direction: models.CryptoDirectionWithdraw,
		Amount:    amount,
		TxRef:     txRef,
		Simulated: simulated,
		Status:    "SUCCESS",
	}
	if _, err := b.cryptoRepo.Create(ctx, txn); err != nil {
		return nil, err
	}
	return txn, nil
}

func (b *cryptoBiz) ViewAll(ctx context.Context, userID string) ([]*models.CryptoTxn, error) {
	accountID, err := b.resolveAccountID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return b.cryptoRepo.FindBy(ctx, "account_id = ?", accountID)
}
