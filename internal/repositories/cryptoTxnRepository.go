package repositories

import (
	"swapngo-backend/internal/models"

	"gorm.io/gorm"
)

type CryptoTxnRepository interface {
	IBaseRepository[models.CryptoTxn]
}

type cryptoTxnRepository struct {
	BaseRepository[models.CryptoTxn]
	db *gorm.DB
}

func NewCryptoTxnRepository(db *gorm.DB) CryptoTxnRepository {
	return &cryptoTxnRepository{
		BaseRepository: *NewBaseRepository[models.CryptoTxn](db),
		db:             db,
	}
}
