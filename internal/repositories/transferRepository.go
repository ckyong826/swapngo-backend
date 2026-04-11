package repositories

import (
	"swapngo-backend/internal/models"

	"gorm.io/gorm"
)

type TransferRepository interface {
	IBaseRepository[models.Transfer]
}

type transferRepository struct {
	BaseRepository[models.Transfer]
	db *gorm.DB
}

func NewTransferRepository(db *gorm.DB) TransferRepository {
	return &transferRepository{
		BaseRepository: *NewBaseRepository[models.Transfer](db),
		db: db,
	}
}
