package repositories

import (
	"context"
	"swapngo-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WalletRepository interface {
	IBaseRepository[models.Wallet]
	FindByAccountId(ctx context.Context, accountID uuid.UUID) ([]models.Wallet, error)
	FindByAccountIdAndChain(ctx context.Context, accountID uuid.UUID, chain string) (*models.Wallet, error)
}

type walletRepository struct {
	BaseRepository[models.Wallet]
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) WalletRepository {
	return &walletRepository{
		BaseRepository: *NewBaseRepository[models.Wallet](db),
		db: db,
	}
}

func (r *walletRepository) FindByAccountId(ctx context.Context, accountID uuid.UUID) ([]models.Wallet, error) {
	var wallets []models.Wallet
	err := r.db.Where("account_id = ?", accountID).Find(&wallets).Error
	if err != nil {
		return nil, err
	}
	return wallets, nil
}

func (r *walletRepository) FindByAccountIdAndChain(ctx context.Context, accountID uuid.UUID, chain string) (*models.Wallet, error) {
	var wallet models.Wallet
	err := r.db.Where("account_id = ? AND chain_name = ?", accountID, chain).First(&wallet).Error
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}





