package repositories

import (
	"swapngo-backend/internal/models"

	"gorm.io/gorm"
)

type WalletRepository interface {
	IBaseRepository[models.Wallet]
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



