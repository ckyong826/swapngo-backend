package repositories

import (
	"swapngo-backend/internal/models"

	"gorm.io/gorm"
)

type AccountRepository interface {
	IBaseRepository[models.Account]
	
}

type accountRepository struct {
	BaseRepository[models.Account]
	db *gorm.DB
}

func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &accountRepository{
		BaseRepository: *NewBaseRepository[models.Account](db),
		db: db,
	}
}

