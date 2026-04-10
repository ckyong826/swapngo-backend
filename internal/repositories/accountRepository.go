package repositories

import (
	"context"
	"swapngo-backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AccountRepository interface {
	IBaseRepository[models.Account]
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]models.Account, error)
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

func (r *accountRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]models.Account, error) {
	var accounts []models.Account
	err := r.db.Where("user_id = ?", userID).Find(&accounts).Error
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

