package repositories

import (
	"swapngo-backend/internal/models"

	"gorm.io/gorm"
)

type SwapRepository interface {
	IBaseRepository[models.Swap]
}

type swapRepository struct {
	BaseRepository[models.Swap]
	db *gorm.DB
}

func NewSwapRepository(db *gorm.DB) SwapRepository {
	return &swapRepository{
		BaseRepository: *NewBaseRepository[models.Swap](db),
		db: db,
	}
}
