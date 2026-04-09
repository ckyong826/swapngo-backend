package repositories

import (
	"context"

	"swapngo-backend/internal/models"
	"swapngo-backend/pkg/utils"

	"gorm.io/gorm"
)

type UserRepository interface {
	IBaseRepository[models.User]
	CheckExist(ctx context.Context, phoneNumber string, email string, username string) (*models.User, error)
}

type userRepository struct {
	BaseRepository[models.User]
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{
		BaseRepository: *NewBaseRepository[models.User](db),
		db: db,
	}
}

func (r *userRepository) CheckExist(ctx context.Context, phoneNumber string, email string, username string) (*models.User, error) {
	query := r.db.WithContext(ctx).Where("phone_number = ? OR email = ? OR username = ?", phoneNumber, email, username)
	return utils.FirstOrNil[models.User](query)
}
